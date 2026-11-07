package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"wailsproxy/internal/config"
)

// GenerateConfig 根据 AppConfig 与活动节点生成对应内核（Sing-box 或 Xray）的完整 JSON 配置文件内容
func GenerateConfig(cfg config.AppConfig, node config.NodeConfig) ([]byte, error) {
	if err := validateInputs(cfg, node); err != nil {
		return nil, err
	}

	if strings.EqualFold(cfg.CoreType, "xray") {
		return GenerateXrayConfig(cfg, node)
	}
	return GenerateSingBoxConfig(cfg, node)
}

func validateInputs(cfg config.AppConfig, node config.NodeConfig) error {
	if cfg.SocksPort <= 0 || cfg.SocksPort > 65535 {
		return fmt.Errorf("invalid socksPort: %d", cfg.SocksPort)
	}
	if cfg.HttpPort <= 0 || cfg.HttpPort > 65535 {
		return fmt.Errorf("invalid httpPort: %d", cfg.HttpPort)
	}
	if cfg.SocksPort == cfg.HttpPort {
		return fmt.Errorf("socksPort and httpPort cannot be identical: %d", cfg.SocksPort)
	}
	if strings.TrimSpace(node.Server) == "" || node.Port <= 0 || node.Port > 65535 {
		return fmt.Errorf("invalid node server or port: %s:%d", node.Server, node.Port)
	}
	return nil
}

// GenerateSingBoxConfig 生成 Sing-box 标准 JSON 配置
func GenerateSingBoxConfig(cfg config.AppConfig, node config.NodeConfig) ([]byte, error) {
	listenAddr := "127.0.0.1"
	if cfg.AllowLan {
		listenAddr = "0.0.0.0"
	}

	inbounds := []map[string]interface{}{
		{
			"type":        "socks",
			"tag":         "socks-in",
			"listen":      listenAddr,
			"listen_port": cfg.SocksPort,
		},
		{
			"type":        "http",
			"tag":         "http-in",
			"listen":      listenAddr,
			"listen_port": cfg.HttpPort,
		},
	}

	if cfg.ProxyMode == config.ProxyModeTun {
		inbounds = append(inbounds, map[string]interface{}{
			"type":         "tun",
			"tag":          "tun-in",
			"address":      []string{"172.19.0.1/30"},
			"mtu":          9000,
			"auto_route":   true,
			"strict_route": true,
			"stack":        "system",
		})
	}

	proxyOutbound, err := buildSingBoxProxyOutbound(node)
	if err != nil {
		return nil, err
	}

	outbounds := []map[string]interface{}{
		proxyOutbound,
		{
			"type": "direct",
			"tag":  "direct",
		},
	}

	root := map[string]interface{}{
		"log": map[string]interface{}{
			"level":     "info",
			"timestamp": true,
		},
		"dns":       buildSingBoxDNS(cfg),
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route":     buildSingBoxRoute(cfg),
		"experimental": map[string]interface{}{
			"clash_api": map[string]interface{}{
				"external_controller": "127.0.0.1:19090",
			},
		},
	}

	return json.MarshalIndent(root, "", "  ")
}

func buildSingBoxProxyOutbound(node config.NodeConfig) (map[string]interface{}, error) {
	proto := strings.ToLower(strings.TrimSpace(node.Protocol))
	ob := map[string]interface{}{
		"tag":         "proxy",
		"server":      node.Server,
		"server_port": node.Port,
	}

	switch proto {
	case "vmess":
		cipher := node.Cipher
		if cipher == "" {
			cipher = "auto"
		}
		ob["type"] = "vmess"
		ob["uuid"] = node.Uuid
		ob["security"] = cipher
		ob["alter_id"] = node.AlterId
	case "vless":
		ob["type"] = "vless"
		ob["uuid"] = node.Uuid
		if node.Flow != "" {
			ob["flow"] = node.Flow
		}
	case "trojan":
		ob["type"] = "trojan"
		ob["password"] = node.Password
	case "ss", "shadowsocks":
		cipher := node.Cipher
		if cipher == "" {
			cipher = "aes-256-gcm"
		}
		ob["type"] = "shadowsocks"
		ob["method"] = cipher
		ob["password"] = node.Password
	case "hysteria2", "hy2":
		ob["type"] = "hysteria2"
		ob["password"] = node.Password
		if node.UpMbps > 0 {
			ob["up_mbps"] = node.UpMbps
		}
		if node.DownMbps > 0 {
			ob["down_mbps"] = node.DownMbps
		}
		if node.ObfsType != "" {
			ob["obfs"] = map[string]interface{}{
				"type":     node.ObfsType,
				"password": node.ObfsPassword,
			}
		}
	case "tuic":
		ob["type"] = "tuic"
		ob["uuid"] = node.Uuid
		ob["password"] = node.Password
		ob["congestion_control"] = "bbr"
	default:
		return nil, fmt.Errorf("unsupported node protocol for sing-box: %s", node.Protocol)
	}

	// TLS / Reality 配置
	if node.Tls || node.Security == "tls" || node.Security == "reality" || proto == "trojan" || proto == "hysteria2" || proto == "tuic" {
		sni := node.Sni
		if sni == "" {
			sni = node.Host
		}
		if sni == "" {
			sni = node.Server
		}
		tlsCfg := map[string]interface{}{
			"enabled":     true,
			"server_name": sni,
		}
		if node.Alpn != "" {
			parts := strings.Split(node.Alpn, ",")
			var alpn []string
			for _, p := range parts {
				if trimmed := strings.TrimSpace(p); trimmed != "" {
					alpn = append(alpn, trimmed)
				}
			}
			if len(alpn) > 0 {
				tlsCfg["alpn"] = alpn
			}
		}
		if node.Fingerprint != "" || node.Security == "reality" {
			fp := node.Fingerprint
			if fp == "" {
				fp = "chrome"
			}
			tlsCfg["utls"] = map[string]interface{}{
				"enabled":     true,
				"fingerprint": fp,
			}
		}
		if node.Security == "reality" {
			tlsCfg["reality"] = map[string]interface{}{
				"enabled":    true,
				"public_key": node.PublicKey,
				"short_id":   node.ShortId,
			}
		}
		ob["tls"] = tlsCfg
	}

	// Transport 传输层配置 (ws / grpc / h2 / httpupgrade)
	netType := strings.ToLower(strings.TrimSpace(node.Network))
	switch netType {
	case "ws":
		tr := map[string]interface{}{
			"type": "ws",
		}
		if node.Path != "" {
			tr["path"] = node.Path
		}
		if node.Host != "" {
			tr["headers"] = map[string]string{"Host": node.Host}
		}
		ob["transport"] = tr
	case "grpc":
		svc := node.ServiceName
		if svc == "" {
			svc = node.Path
		}
		ob["transport"] = map[string]interface{}{
			"type":         "grpc",
			"service_name": svc,
		}
	case "h2", "http":
		tr := map[string]interface{}{
			"type": "http",
		}
		if node.Path != "" {
			tr["path"] = node.Path
		}
		if node.Host != "" {
			tr["host"] = []string{node.Host}
		}
		ob["transport"] = tr
	case "httpupgrade":
		tr := map[string]interface{}{
			"type": "httpupgrade",
		}
		if node.Path != "" {
			tr["path"] = node.Path
		}
		if node.Host != "" {
			tr["host"] = node.Host
		}
		ob["transport"] = tr
	}

	return ob, nil
}

// GenerateXrayConfig 生成 Xray-core 标准 JSON 配置
func GenerateXrayConfig(cfg config.AppConfig, node config.NodeConfig) ([]byte, error) {
	listenAddr := "127.0.0.1"
	if cfg.AllowLan {
		listenAddr = "0.0.0.0"
	}

	inbounds := []map[string]interface{}{
		{
			"tag":      "socks-in",
			"port":     cfg.SocksPort,
			"listen":   listenAddr,
			"protocol": "socks",
			"settings": map[string]interface{}{
				"auth": "noauth",
				"udp":  true,
			},
			"sniffing": map[string]interface{}{
				"enabled":      true,
				"destOverride": []string{"http", "tls", "quic"},
			},
		},
		{
			"tag":      "http-in",
			"port":     cfg.HttpPort,
			"listen":   listenAddr,
			"protocol": "http",
			"sniffing": map[string]interface{}{
				"enabled":      true,
				"destOverride": []string{"http", "tls", "quic"},
			},
		},
	}

	proxyOutbound, err := buildXrayProxyOutbound(node)
	if err != nil {
		return nil, err
	}

	directOutbound := map[string]interface{}{
		"tag":      "direct",
		"protocol": "freedom",
	}
	if iface := DetectPhysicalInterface(); iface != "" {
		directOutbound["streamSettings"] = map[string]interface{}{
			"sockopt": map[string]interface{}{
				"interface": iface,
			},
		}
	}

	outbounds := []map[string]interface{}{
		proxyOutbound,
		directOutbound,
		{
			"tag":      "block",
			"protocol": "blackhole",
		},
	}

	root := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "info",
		},
		"stats":    map[string]interface{}{},
		"dns":      buildXrayDNS(cfg),
		"inbounds": inbounds,
		"outbounds": outbounds,
		"routing":  buildXrayRouting(cfg),
	}

	return json.MarshalIndent(root, "", "  ")
}

func buildXrayProxyOutbound(node config.NodeConfig) (map[string]interface{}, error) {
	proto := strings.ToLower(strings.TrimSpace(node.Protocol))
	streamSettings := map[string]interface{}{
		"network": "tcp",
	}
	if node.Network != "" && node.Network != "quic" {
		streamSettings["network"] = node.Network
	}
	if iface := DetectPhysicalInterface(); iface != "" {
		streamSettings["sockopt"] = map[string]interface{}{
			"interface": iface,
		}
	}

	if node.Security == "reality" {
		streamSettings["security"] = "reality"
		realityCfg := map[string]interface{}{
			"serverName":  node.Sni,
			"fingerprint": firstNonEmpty(node.Fingerprint, "chrome"),
			"show":        false,
			"publicKey":   node.PublicKey,
			"shortId":     node.ShortId,
		}
		if node.SpiderX != "" {
			realityCfg["spiderX"] = node.SpiderX
		}
		if node.Mldsa65Verify != "" {
			realityCfg["mldsa65Verify"] = node.Mldsa65Verify
		}
		streamSettings["realitySettings"] = realityCfg
	} else if node.Tls || proto == "trojan" {
		streamSettings["security"] = "tls"
		streamSettings["tlsSettings"] = map[string]interface{}{
			"serverName":  firstNonEmpty(node.Sni, node.Server),
			"fingerprint": firstNonEmpty(node.Fingerprint, "chrome"),
		}
	}

	if node.Network == "ws" {
		streamSettings["wsSettings"] = map[string]interface{}{
			"path": firstNonEmpty(node.Path, "/"),
			"headers": map[string]string{
				"Host": firstNonEmpty(node.Host, node.Server),
			},
		}
	} else if node.Network == "grpc" {
		streamSettings["grpcSettings"] = map[string]interface{}{
			"serviceName": firstNonEmpty(node.ServiceName, node.Path),
		}
	}

	serverAddr := resolveRealServerIP(node.Server)

	ob := map[string]interface{}{
		"tag":            "proxy",
		"streamSettings": streamSettings,
	}

	switch proto {
	case "vmess":
		ob["protocol"] = "vmess"
		ob["settings"] = map[string]interface{}{
			"vnext": []map[string]interface{}{
				{
					"address": serverAddr,
					"port":    node.Port,
					"users": []map[string]interface{}{
						{
							"id":       node.Uuid,
							"alterId":  node.AlterId,
							"security": firstNonEmpty(node.Cipher, "auto"),
						},
					},
				},
			},
		}
	case "vless":
		ob["protocol"] = "vless"
		ob["settings"] = map[string]interface{}{
			"vnext": []map[string]interface{}{
				{
					"address": serverAddr,
					"port":    node.Port,
					"users": []map[string]interface{}{
						{
							"id":         node.Uuid,
							"encryption": "none",
							"flow":       node.Flow,
						},
					},
				},
			},
		}
	case "trojan":
		ob["protocol"] = "trojan"
		ob["settings"] = map[string]interface{}{
			"servers": []map[string]interface{}{
				{
					"address":  serverAddr,
					"port":     node.Port,
					"password": node.Password,
				},
			},
		}
	case "ss", "shadowsocks":
		ob["protocol"] = "shadowsocks"
		ob["settings"] = map[string]interface{}{
			"servers": []map[string]interface{}{
				{
					"address":  serverAddr,
					"port":     node.Port,
					"method":   firstNonEmpty(node.Cipher, "aes-256-gcm"),
					"password": node.Password,
				},
			},
		}
	case "hysteria2", "hy2", "tuic":
		// Xray 兼容回退或统一结构生成
		ob["protocol"] = "socks"
		ob["settings"] = map[string]interface{}{
			"servers": []map[string]interface{}{
				{
					"address": serverAddr,
					"port":    node.Port,
				},
			},
		}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", node.Protocol)
	}

	return ob, nil
}

// DetectPhysicalInterface 检测当前系统活动的物理网卡接口（如 macOS 上的 en0 / en1），防止流量陷入虚拟网卡回环
func DetectPhysicalInterface() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if !strings.HasPrefix(iface.Name, "en") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
				ipStr := ipnet.IP.String()
				if !strings.HasPrefix(ipStr, "169.254.") {
					return iface.Name
				}
			}
		}
	}
	return ""
}

func resolveRealServerIP(host string) string {
	host = strings.TrimSpace(host)
	if host == "" || net.ParseIP(host) != nil {
		return host
	}
	if strings.HasSuffix(host, ".example.com") || strings.HasSuffix(host, ".local") || host == "localhost" {
		return host
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return host
	}
	for _, a := range addrs {
		if ip4 := a.IP.To4(); ip4 != nil {
			ipStr := ip4.String()
			// 排除 FakeIP 地址段 (198.18.0.0/15)
			if !strings.HasPrefix(ipStr, "198.18.") && !strings.HasPrefix(ipStr, "198.19.") {
				return ipStr
			}
		}
	}
	return host
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// ValidateConfig 预检生成的配置语法合法性；若存在真实内核二进制则调用 check / -test 校验
func ValidateConfig(rawJSON []byte, coreType string, coreBinaryPath string) error {
	if len(rawJSON) == 0 {
		return fmt.Errorf("empty configuration bytes")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(rawJSON, &parsed); err != nil {
		return fmt.Errorf("invalid json syntax: %w", err)
	}

	inbounds, ok := parsed["inbounds"].([]interface{})
	if !ok || len(inbounds) == 0 {
		return fmt.Errorf("missing or empty inbounds in generated config")
	}

	outbounds, ok := parsed["outbounds"].([]interface{})
	if !ok || len(outbounds) == 0 {
		return fmt.Errorf("missing or empty outbounds in generated config")
	}

	if _, hasDNS := parsed["dns"]; !hasDNS {
		return fmt.Errorf("missing dns block in generated config")
	}

	if strings.EqualFold(coreType, "xray") {
		if _, hasRouting := parsed["routing"]; !hasRouting {
			return fmt.Errorf("missing routing block in xray config")
		}
	} else {
		if _, hasRoute := parsed["route"]; !hasRoute {
			return fmt.Errorf("missing route block in sing-box config")
		}
	}

	// 若指定了开源内核二进制文件且存在可执行文件，则调用内核命令预检
	if coreBinaryPath != "" {
		if absBin, err := filepath.Abs(coreBinaryPath); err == nil {
			coreBinaryPath = absBin
		}
		if info, err := os.Stat(coreBinaryPath); err == nil && !info.IsDir() {
			tmpFile, err := os.CreateTemp("", "wailsproxy-check-*.json")
			if err != nil {
				return fmt.Errorf("create temp check file: %w", err)
			}
			tmpPath := tmpFile.Name()
			defer os.Remove(tmpPath)

			if _, err := tmpFile.Write(rawJSON); err != nil {
				_ = tmpFile.Close()
				return err
			}
			_ = tmpFile.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var cmd *exec.Cmd
			if strings.EqualFold(coreType, "xray") {
				cmd = exec.CommandContext(ctx, coreBinaryPath, "run", "-test", "-config", tmpPath)
			} else {
				cmd = exec.CommandContext(ctx, coreBinaryPath, "check", "-c", tmpPath)
			}
			cmd.Dir = filepath.Dir(coreBinaryPath)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("core check failed: %v (%s)", err, strings.TrimSpace(string(out)))
			}
		}
	}

	return nil
}
