package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"wailsproxy/internal/config"
	"wailsproxy/internal/parser"
)

// ParseContent 自动识别订阅内容格式（Base64、多行 URI、Clash YAML、Sing-box JSON）并解析为统一节点列表
func ParseContent(rawContent string) ([]config.NodeConfig, error) {
	trimmed := strings.TrimSpace(rawContent)
	if trimmed == "" {
		return nil, fmt.Errorf("订阅内容为空")
	}

	// 1. 尝试识别 Sing-box JSON 格式（以 { 开头并包含 "outbounds"）
	if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, `"outbounds"`) {
		if nodes, err := parseSingBoxJSON(trimmed); err == nil && len(nodes) > 0 {
			return nodes, nil
		}
	}

	// 2. 尝试识别 Clash / Mihomo YAML 格式（包含 proxies: 关键字）
	if strings.Contains(trimmed, "proxies:") {
		if nodes, err := parseClashYAML(trimmed); err == nil && len(nodes) > 0 {
			return nodes, nil
		}
	}

	// 3. 尝试 Base64 解码后再识别（有些机场将 YAML 或多行链接做了整体 Base64）
	if !strings.Contains(trimmed, "://") {
		if decodedBytes, err := parser.DecodeBase64(trimmed); err == nil {
			decodedStr := strings.TrimSpace(string(decodedBytes))
			if strings.Contains(decodedStr, "proxies:") {
				if nodes, err := parseClashYAML(decodedStr); err == nil && len(nodes) > 0 {
					return nodes, nil
				}
			}
			if strings.HasPrefix(decodedStr, "{") && strings.Contains(decodedStr, `"outbounds"`) {
				if nodes, err := parseSingBoxJSON(decodedStr); err == nil && len(nodes) > 0 {
					return nodes, nil
				}
			}
			trimmed = decodedStr
		}
	}

	// 4. 按多行分享链接解析
	return parser.ParseMultipleLinks(trimmed)
}

type clashConfig struct {
	Proxies []clashProxy `yaml:"proxies"`
}

type clashProxy struct {
	Name              string            `yaml:"name"`
	Type              string            `yaml:"type"`
	Server            string            `yaml:"server"`
	Port              int               `yaml:"port"`
	UUID              string            `yaml:"uuid"`
	Password          string            `yaml:"password"`
	AlterID           int               `yaml:"alterId"`
	Cipher            string            `yaml:"cipher"`
	UDP               bool              `yaml:"udp"`
	TLS               bool              `yaml:"tls"`
	SkipCertVerify    bool              `yaml:"skip-cert-verify"`
	ServerName        string            `yaml:"servername"`
	SNI               string            `yaml:"sni"`
	Flow              string            `yaml:"flow"`
	Network           string            `yaml:"network"`
	ClientFingerprint string            `yaml:"client-fingerprint"`
	ALPN              []string          `yaml:"alpn"`
	Obfs              string            `yaml:"obfs"`
	ObfsPassword      string            `yaml:"obfs-password"`
	Up                string            `yaml:"up"`
	Down              string            `yaml:"down"`
	WSOpts            *clashWSOpts      `yaml:"ws-opts"`
	GRPCOpts          *clashGRPCOpts    `yaml:"grpc-opts"`
	RealityOpts       *clashRealityOpts `yaml:"reality-opts"`
}

type clashWSOpts struct {
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
}

type clashGRPCOpts struct {
	GrpcServiceName string `yaml:"grpc-service-name"`
}

type clashRealityOpts struct {
	PublicKey string `yaml:"public-key"`
	ShortID   string `yaml:"short-id"`
}

func parseClashYAML(content string) ([]config.NodeConfig, error) {
	var cc clashConfig
	if err := yaml.Unmarshal([]byte(content), &cc); err != nil {
		return nil, fmt.Errorf("parse clash yaml: %w", err)
	}

	var nodes []config.NodeConfig
	for _, p := range cc.Proxies {
		if p.Server == "" || p.Port <= 0 {
			continue
		}
		proto := strings.ToLower(strings.TrimSpace(p.Type))
		switch proto {
		case "ss", "shadowsocks":
			proto = "ss"
		case "vmess", "vless", "trojan", "hysteria2", "tuic":
		case "hy2":
			proto = "hysteria2"
		default:
			continue
		}

		network := strings.ToLower(p.Network)
		if network == "" {
			if proto == "hysteria2" || proto == "tuic" {
				network = "quic"
			} else {
				network = "tcp"
			}
		}

		sni := p.SNI
		if sni == "" {
			sni = p.ServerName
		}

		var path, host, serviceName string
		if p.WSOpts != nil {
			path = p.WSOpts.Path
			if h, ok := p.WSOpts.Headers["Host"]; ok {
				host = h
			} else if h, ok := p.WSOpts.Headers["host"]; ok {
				host = h
			}
		}
		if p.GRPCOpts != nil {
			serviceName = p.GRPCOpts.GrpcServiceName
		}

		security := "none"
		isTLS := p.TLS || proto == "trojan" || proto == "hysteria2" || proto == "tuic"
		if isTLS {
			security = "tls"
		}
		var pubKey, shortID string
		if p.RealityOpts != nil && p.RealityOpts.PublicKey != "" {
			security = "reality"
			isTLS = true
			pubKey = p.RealityOpts.PublicKey
			shortID = p.RealityOpts.ShortID
		}

		name := strings.TrimSpace(p.Name)
		if name == "" {
			name = fmt.Sprintf("%s-%s:%d", strings.ToUpper(proto), p.Server, p.Port)
		}

		nodes = append(nodes, config.NodeConfig{
			Id:           uuid.NewString(),
			SubId:        config.DefaultGroupID,
			GroupName:    config.DefaultGroupName,
			Name:         name,
			Protocol:     proto,
			Server:       p.Server,
			Port:         p.Port,
			Uuid:         p.UUID,
			Password:     p.Password,
			AlterId:      p.AlterID,
			Cipher:       p.Cipher,
			Network:      network,
			Path:         path,
			Host:         host,
			ServiceName:  serviceName,
			Tls:          isTLS,
			Sni:          sni,
			Alpn:         strings.Join(p.ALPN, ","),
			Fingerprint:  p.ClientFingerprint,
			Security:     security,
			PublicKey:    pubKey,
			ShortId:      shortID,
			Flow:         p.Flow,
			ObfsType:     p.Obfs,
			ObfsPassword: p.ObfsPassword,
		})
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no valid proxy nodes found in Clash YAML")
	}
	return nodes, nil
}

type singBoxConfig struct {
	Outbounds []singBoxOutbound `json:"outbounds"`
}

type singBoxOutbound struct {
	Type       string            `json:"type"`
	Tag        string            `json:"tag"`
	Server     string            `json:"server"`
	ServerPort int               `json:"server_port"`
	UUID       string            `json:"uuid"`
	Password   string            `json:"password"`
	Method     string            `json:"method"`
	Security   string            `json:"security"`
	AlterID    int               `json:"alter_id"`
	Flow       string            `json:"flow"`
	UpMbps     int               `json:"up_mbps"`
	DownMbps   int               `json:"down_mbps"`
	Obfs       *singBoxObfs      `json:"obfs"`
	TLS        *singBoxTLS       `json:"tls"`
	Transport  *singBoxTransport `json:"transport"`
}

type singBoxObfs struct {
	Type     string `json:"type"`
	Password string `json:"password"`
}

type singBoxTLS struct {
	Enabled    bool            `json:"enabled"`
	ServerName string          `json:"server_name"`
	ALPN       []string        `json:"alpn"`
	UTLS       *singBoxUTLS    `json:"utls"`
	Reality    *singBoxReality `json:"reality"`
}

type singBoxUTLS struct {
	Enabled     bool   `json:"enabled"`
	Fingerprint string `json:"fingerprint"`
}

type singBoxReality struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"public_key"`
	ShortID   string `json:"short_id"`
}

type singBoxTransport struct {
	Type        string            `json:"type"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
	ServiceName string            `json:"service_name"`
}

func parseSingBoxJSON(content string) ([]config.NodeConfig, error) {
	var sb singBoxConfig
	if err := json.Unmarshal([]byte(content), &sb); err != nil {
		return nil, fmt.Errorf("parse sing-box json: %w", err)
	}

	var nodes []config.NodeConfig
	for _, ob := range sb.Outbounds {
		if ob.Server == "" || ob.ServerPort <= 0 {
			continue
		}
		proto := strings.ToLower(ob.Type)
		switch proto {
		case "shadowsocks":
			proto = "ss"
		case "vmess", "vless", "trojan", "hysteria2", "tuic":
		default:
			continue
		}

		network := "tcp"
		var path, host, serviceName string
		if ob.Transport != nil && ob.Transport.Type != "" {
			network = ob.Transport.Type
			path = ob.Transport.Path
			serviceName = ob.Transport.ServiceName
			if h, ok := ob.Transport.Headers["Host"]; ok {
				host = h
			}
		} else if proto == "hysteria2" || proto == "tuic" {
			network = "quic"
		}

		var isTLS bool
		var sni, alpn, fp, sec, pubKey, shortID string
		sec = "none"
		if ob.TLS != nil && ob.TLS.Enabled {
			isTLS = true
			sec = "tls"
			sni = ob.TLS.ServerName
			alpn = strings.Join(ob.TLS.ALPN, ",")
			if ob.TLS.UTLS != nil && ob.TLS.UTLS.Enabled {
				fp = ob.TLS.UTLS.Fingerprint
			}
			if ob.TLS.Reality != nil && ob.TLS.Reality.Enabled {
				sec = "reality"
				pubKey = ob.TLS.Reality.PublicKey
				shortID = ob.TLS.Reality.ShortID
			}
		}

		cipher := ob.Method
		if cipher == "" {
			cipher = ob.Security
		}

		var obfsType, obfsPass string
		if ob.Obfs != nil {
			obfsType = ob.Obfs.Type
			obfsPass = ob.Obfs.Password
		}

		name := strings.TrimSpace(ob.Tag)
		if name == "" {
			name = fmt.Sprintf("%s-%s:%d", strings.ToUpper(proto), ob.Server, ob.ServerPort)
		}

		nodes = append(nodes, config.NodeConfig{
			Id:           uuid.NewString(),
			SubId:        config.DefaultGroupID,
			GroupName:    config.DefaultGroupName,
			Name:         name,
			Protocol:     proto,
			Server:       ob.Server,
			Port:         ob.ServerPort,
			Uuid:         ob.UUID,
			Password:     ob.Password,
			AlterId:      ob.AlterID,
			Cipher:       cipher,
			Network:      network,
			Path:         path,
			Host:         host,
			ServiceName:  serviceName,
			Tls:          isTLS,
			Sni:          sni,
			Alpn:         alpn,
			Fingerprint:  fp,
			Security:     sec,
			PublicKey:    pubKey,
			ShortId:      shortID,
			Flow:         ob.Flow,
			ObfsType:     obfsType,
			ObfsPassword: obfsPass,
			UpMbps:       ob.UpMbps,
			DownMbps:     ob.DownMbps,
		})
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no valid proxy outbounds found in Sing-box JSON")
	}
	return nodes, nil
}

// FetchSubscription 从远程 URL 拉取订阅内容并解析为节点列表
// 优先使用 v2rayN User-Agent 获取包含完整参数（如 VLESS Reality pqv / spx）的原始链接，若无法解析则自动回退 Clash.Meta User-Agent
func FetchSubscription(ctx context.Context, subURL string, timeout time.Duration) ([]config.NodeConfig, error) {
	subURL = strings.TrimSpace(subURL)
	if subURL == "" {
		return nil, fmt.Errorf("订阅 URL 不能为空")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	userAgents := []string{
		"v2rayN/7.0 (WailsProxy/2.0)",
		"WailsProxy/2.0 (Clash.Meta; sing-box; v2rayN)",
	}

	var lastErr error
	for _, ua := range userAgents {
		nodes, err := fetchWithUserAgent(ctx, subURL, ua, timeout)
		if err == nil && len(nodes) > 0 {
			return nodes, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("未从订阅地址解析到任何有效节点")
}

func fetchWithUserAgent(ctx context.Context, subURL string, userAgent string, timeout time.Duration) ([]config.NodeConfig, error) {
	// 优先直连拉取（避免系统残留失效代理导致拉取超时），若直连失败再回退使用环境/系统代理
	transports := []*http.Transport{
		{Proxy: nil},
		http.DefaultTransport.(*http.Transport).Clone(),
	}

	var lastErr error
	for _, tr := range transports {
		client := &http.Client{
			Timeout:   timeout,
			Transport: tr,
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, subURL, nil)
		if err != nil {
			return nil, fmt.Errorf("构造订阅请求失败: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "*/*")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("拉取订阅失败: %w", err)
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // max 10MB
		_ = resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("订阅服务器返回 HTTP %d", resp.StatusCode)
			continue
		}
		if readErr != nil {
			lastErr = fmt.Errorf("读取订阅响应失败: %w", readErr)
			continue
		}

		return ParseContent(string(body))
	}
	return nil, lastErr
}
