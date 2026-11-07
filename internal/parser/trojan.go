package parser

import (
	"fmt"
	"net/url"
	"strings"

	"wailsproxy/internal/config"
)

func parseTrojan(rawLink string) (config.NodeConfig, error) {
	u, err := url.Parse(rawLink)
	if err != nil {
		return config.NodeConfig{}, fmt.Errorf("invalid trojan url: %w", err)
	}

	password := u.User.Username()
	if unescaped, err := url.QueryUnescape(password); err == nil && unescaped != "" {
		password = unescaped
	}
	if strings.TrimSpace(password) == "" {
		return config.NodeConfig{}, fmt.Errorf("missing trojan password")
	}

	host, port, err := parseHostPort(u)
	if err != nil {
		return config.NodeConfig{}, fmt.Errorf("invalid trojan host/port: %w", err)
	}

	q := u.Query()
	network := strings.ToLower(q.Get("type"))
	if network == "" {
		network = "tcp"
	}

	security := strings.ToLower(q.Get("security"))
	if security == "" {
		security = "tls"
	}
	isTLS := security != "none"

	sni := q.Get("sni")
	if sni == "" {
		sni = q.Get("peer")
	}
	if sni == "" {
		sni = host
	}

	name := decodeFragment(u.Fragment, fmt.Sprintf("Trojan-%s:%d", host, port))

	return config.NodeConfig{
		Id:          newNodeID(),
		SubId:       config.DefaultGroupID,
		GroupName:   config.DefaultGroupName,
		Name:        name,
		Protocol:    "trojan",
		Server:      host,
		Port:        port,
		Password:    password,
		Network:     network,
		Path:        q.Get("path"),
		Host:        q.Get("host"),
		ServiceName: q.Get("serviceName"),
		Tls:         isTLS,
		Sni:         sni,
		Alpn:        q.Get("alpn"),
		Fingerprint: q.Get("fp"),
		Security:    security,
		PublicKey:   q.Get("pbk"),
		ShortId:     q.Get("sid"),
		RawLink:     rawLink,
	}, nil
}
