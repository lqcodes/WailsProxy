package parser

import (
	"fmt"
	"net/url"
	"strings"

	"wailsproxy/internal/config"
)

func parseVLESS(rawLink string) (config.NodeConfig, error) {
	u, err := url.Parse(rawLink)
	if err != nil {
		return config.NodeConfig{}, fmt.Errorf("invalid vless url: %w", err)
	}

	userUUID := strings.TrimSpace(u.User.Username())
	if userUUID == "" {
		return config.NodeConfig{}, fmt.Errorf("missing vless uuid")
	}

	host, port, err := parseHostPort(u)
	if err != nil {
		return config.NodeConfig{}, fmt.Errorf("invalid vless host/port: %w", err)
	}

	q := u.Query()
	network := strings.ToLower(q.Get("type"))
	if network == "" {
		network = "tcp"
	}

	security := strings.ToLower(q.Get("security"))
	if security == "" {
		security = "none"
	}
	isTLS := security == "tls" || security == "reality" || security == "xtls"

	sni := q.Get("sni")
	if sni == "" {
		sni = q.Get("peer")
	}

	name := decodeFragment(u.Fragment, fmt.Sprintf("VLESS-%s:%d", host, port))
	serviceName := q.Get("serviceName")
	if serviceName == "" && network == "grpc" {
		serviceName = q.Get("path")
	}

	return config.NodeConfig{
		Id:          newNodeID(),
		SubId:       config.DefaultGroupID,
		GroupName:   config.DefaultGroupName,
		Name:        name,
		Protocol:    "vless",
		Server:      host,
		Port:        port,
		Uuid:        userUUID,
		Network:     network,
		Path:        q.Get("path"),
		Host:        q.Get("host"),
		ServiceName: serviceName,
		Tls:         isTLS,
		Sni:         sni,
		Alpn:        q.Get("alpn"),
		Fingerprint: q.Get("fp"),
		Security:      security,
		PublicKey:     q.Get("pbk"),
		ShortId:       q.Get("sid"),
		SpiderX:       q.Get("spx"),
		Mldsa65Verify: q.Get("pqv"),
		Flow:          q.Get("flow"),
		RawLink:       rawLink,
	}, nil
}
