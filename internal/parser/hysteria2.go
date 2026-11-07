package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"wailsproxy/internal/config"
)

func parseHysteria2(rawLink string) (config.NodeConfig, error) {
	u, err := url.Parse(rawLink)
	if err != nil {
		return config.NodeConfig{}, fmt.Errorf("invalid hysteria2 url: %w", err)
	}

	password := u.User.Username()
	if pwd, ok := u.User.Password(); ok && pwd != "" {
		password = password + ":" + pwd
	}
	if unescaped, err := url.QueryUnescape(password); err == nil && unescaped != "" {
		password = unescaped
	}

	host, port, err := parseHostPort(u)
	if err != nil {
		return config.NodeConfig{}, fmt.Errorf("invalid hysteria2 host/port: %w", err)
	}

	q := u.Query()
	if password == "" {
		password = q.Get("auth")
	}
	if strings.TrimSpace(password) == "" {
		return config.NodeConfig{}, fmt.Errorf("missing hysteria2 password/auth")
	}

	sni := q.Get("sni")
	if sni == "" {
		sni = host
	}

	upMbps, _ := strconv.Atoi(q.Get("upmbps"))
	downMbps, _ := strconv.Atoi(q.Get("downmbps"))

	name := decodeFragment(u.Fragment, fmt.Sprintf("Hy2-%s:%d", host, port))

	return config.NodeConfig{
		Id:           newNodeID(),
		SubId:        config.DefaultGroupID,
		GroupName:    config.DefaultGroupName,
		Name:         name,
		Protocol:     "hysteria2",
		Server:       host,
		Port:         port,
		Password:     password,
		Network:      "quic",
		Tls:          true,
		Sni:          sni,
		Alpn:         q.Get("alpn"),
		Fingerprint:  q.Get("fp"),
		Security:     "tls",
		ObfsType:     q.Get("obfs"),
		ObfsPassword: q.Get("obfs-password"),
		UpMbps:       upMbps,
		DownMbps:     downMbps,
		RawLink:      rawLink,
	}, nil
}

func parseTUIC(rawLink string) (config.NodeConfig, error) {
	u, err := url.Parse(rawLink)
	if err != nil {
		return config.NodeConfig{}, fmt.Errorf("invalid tuic url: %w", err)
	}

	userUUID := u.User.Username()
	password, _ := u.User.Password()
	if userUUID == "" {
		return config.NodeConfig{}, fmt.Errorf("missing tuic uuid")
	}

	host, port, err := parseHostPort(u)
	if err != nil {
		return config.NodeConfig{}, fmt.Errorf("invalid tuic host/port: %w", err)
	}

	q := u.Query()
	sni := q.Get("sni")
	if sni == "" {
		sni = host
	}

	name := decodeFragment(u.Fragment, fmt.Sprintf("TUIC-%s:%d", host, port))

	return config.NodeConfig{
		Id:          newNodeID(),
		SubId:       config.DefaultGroupID,
		GroupName:   config.DefaultGroupName,
		Name:        name,
		Protocol:    "tuic",
		Server:      host,
		Port:        port,
		Uuid:        userUUID,
		Password:    password,
		Network:     "quic",
		Tls:         true,
		Sni:         sni,
		Alpn:        q.Get("alpn"),
		Security:    "tls",
		RawLink:     rawLink,
	}, nil
}
