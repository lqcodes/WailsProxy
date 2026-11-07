package parser

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"wailsproxy/internal/config"
)

type vmessJSON struct {
	V    interface{} `json:"v"`
	Ps   string      `json:"ps"`
	Add  string      `json:"add"`
	Port interface{} `json:"port"`
	Id   string      `json:"id"`
	Aid  interface{} `json:"aid"`
	Scy  string      `json:"scy"`
	Net  string      `json:"net"`
	Type string      `json:"type"`
	Host string      `json:"host"`
	Path string      `json:"path"`
	Tls  string      `json:"tls"`
	Sni  string      `json:"sni"`
	Alpn string      `json:"alpn"`
	Fp   string      `json:"fp"`
}

func parseFlexInt(val interface{}) int {
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		return 0
	}
}

func parseVMess(rawLink string) (config.NodeConfig, error) {
	payload := strings.TrimPrefix(rawLink, "vmess://")
	payload = strings.TrimPrefix(payload, "VMESS://")
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return config.NodeConfig{}, fmt.Errorf("empty vmess payload")
	}

	decoded, err := DecodeBase64(payload)
	if err != nil {
		return config.NodeConfig{}, fmt.Errorf("decode vmess base64: %w", err)
	}

	var vj vmessJSON
	if err := json.Unmarshal(decoded, &vj); err != nil {
		return config.NodeConfig{}, fmt.Errorf("unmarshal vmess json: %w", err)
	}

	port := parseFlexInt(vj.Port)
	if vj.Add == "" || port <= 0 || port > 65535 || vj.Id == "" {
		return config.NodeConfig{}, fmt.Errorf("invalid vmess parameters: missing server, port, or uuid")
	}

	network := strings.ToLower(strings.TrimSpace(vj.Net))
	if network == "" {
		network = "tcp"
	}

	cipher := strings.TrimSpace(vj.Scy)
	if cipher == "" {
		cipher = "auto"
	}

	isTLS := strings.EqualFold(vj.Tls, "tls") || vj.Tls == "1" || strings.EqualFold(vj.Tls, "true")
	name := strings.TrimSpace(vj.Ps)
	if name == "" {
		name = fmt.Sprintf("VMess-%s:%d", vj.Add, port)
	}

	node := config.NodeConfig{
		Id:          newNodeID(),
		SubId:       config.DefaultGroupID,
		GroupName:   config.DefaultGroupName,
		Name:        name,
		Protocol:    "vmess",
		Server:      strings.TrimSpace(vj.Add),
		Port:        port,
		Uuid:        strings.TrimSpace(vj.Id),
		AlterId:     parseFlexInt(vj.Aid),
		Cipher:      cipher,
		Network:     network,
		Path:        vj.Path,
		Host:        vj.Host,
		Tls:         isTLS,
		Sni:         vj.Sni,
		Alpn:        vj.Alpn,
		Fingerprint: vj.Fp,
		Security:     vj.Tls,
		RawLink:     rawLink,
	}
	if network == "grpc" && node.ServiceName == "" {
		node.ServiceName = vj.Path
	}
	return node, nil
}
