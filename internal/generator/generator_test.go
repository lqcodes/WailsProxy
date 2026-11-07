package generator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"wailsproxy/internal/config"
)

func TestGenerateConfig_CombinatorialMatrix(t *testing.T) {
	nodes := []config.NodeConfig{
		{
			Id:       "n-vmess",
			Name:     "VMess-WS-TLS",
			Protocol: "vmess",
			Server:   "hk.vmess.example.com",
			Port:     443,
			Uuid:     "23ad6b10-8d1a-40f7-8ad0-e3e35cd38297",
			Cipher:   "auto",
			Network:  "ws",
			Path:     "/vmess",
			Host:     "cdn.example.com",
			Tls:      true,
			Sni:      "cdn.example.com",
		},
		{
			Id:          "n-vless-reality",
			Name:        "VLESS-Reality",
			Protocol:    "vless",
			Server:      "us.reality.example.com",
			Port:        443,
			Uuid:        "550e8400-e29b-41d4-a716-446655440000",
			Network:     "tcp",
			Tls:         true,
			Sni:         "www.microsoft.com",
			Fingerprint: "chrome",
			Security:    "reality",
			PublicKey:   "y46_fdM2j1h1o1B7t8y7x0z1A2b3C4d5E6f7G8h9I0k",
			ShortId:     "a1b2c3d4",
			Flow:        "xtls-rprx-vision",
		},
		{
			Id:       "n-ss",
			Name:     "SS-Node",
			Protocol: "ss",
			Server:   "sg.ss.example.com",
			Port:     8388,
			Cipher:   "aes-256-gcm",
			Password: "ss_password",
		},
		{
			Id:          "n-trojan",
			Name:        "Trojan-gRPC",
			Protocol:    "trojan",
			Server:      "kr.trojan.example.com",
			Port:        443,
			Password:    "trojan_pw",
			Network:     "grpc",
			ServiceName: "grpcSvc",
			Tls:         true,
			Sni:         "kr.trojan.example.com",
		},
		{
			Id:           "n-hy2",
			Name:         "Hysteria2-Node",
			Protocol:     "hysteria2",
			Server:       "de.hy2.example.com",
			Port:         4433,
			Password:     "hy2_auth",
			Tls:          true,
			Sni:          "de.hy2.example.com",
			ObfsType:     "salamander",
			ObfsPassword: "obfs_password",
			UpMbps:       100,
			DownMbps:     500,
		},
	}

	routingModes := []string{config.RoutingModeRule, config.RoutingModeGlobal, config.RoutingModeDirect}
	proxyModes := []string{config.ProxyModeSystem, config.ProxyModeManual, config.ProxyModeTun}
	dohStates := []bool{true, false}
	coreTypes := []string{"sing-box", "xray"}

	for _, node := range nodes {
		for _, rMode := range routingModes {
			for _, pMode := range proxyModes {
				for _, doh := range dohStates {
					for _, coreType := range coreTypes {
						cfg := config.DefaultAppConfig()
						cfg.RoutingMode = rMode
						cfg.ProxyMode = pMode
						cfg.EnableDoH = doh
						cfg.CoreType = coreType
						cfg.CustomDirectRules = []string{"internal.corp.com", "10.20.0.0/16"}
						cfg.CustomProxyRules = []string{"openai.com"}
						cfg.CustomBlockRules = []string{"ads.example.com"}

						raw, err := GenerateConfig(cfg, node)
						require.NoError(t, err, "failed for %s/%s/%s/doh=%v/%s", node.Protocol, rMode, pMode, doh, coreType)
						require.NoError(t, ValidateConfig(raw, coreType, ""))
						if coreType == "xray" {
							require.NoError(t, ValidateConfig(raw, coreType, "../../bin/xray"))
						} else if node.Protocol != "vmess" {
							require.NoError(t, ValidateConfig(raw, coreType, "../../bin/sing-box"))
						}

						var parsed map[string]interface{}
						require.NoError(t, json.Unmarshal(raw, &parsed))

						// TUN mode on sing-box should include tun-in inbound
						if coreType == "sing-box" && pMode == config.ProxyModeTun {
							inbounds := parsed["inbounds"].([]interface{})
							assert.Len(t, inbounds, 3, "TUN mode should have socks, http, and tun inbounds")
						}
					}
				}
			}
		}
	}
}
