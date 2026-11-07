package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"wailsproxy/internal/config"
)

func parseShadowsocks(rawLink string) (config.NodeConfig, error) {
	trimmed := strings.TrimSpace(rawLink)
	body := trimmed[5:] // remove "ss://"
	if body == "" {
		return config.NodeConfig{}, fmt.Errorf("empty shadowsocks link")
	}

	var fragment string
	if hashIdx := strings.Index(body, "#"); hashIdx != -1 {
		fragment = body[hashIdx+1:]
		body = body[:hashIdx]
	}

	// Remove optional query string for SIP002 plugins if present
	if qIdx := strings.Index(body, "?"); qIdx != -1 {
		body = body[:qIdx]
	}

	var cipher, password, host string
	var port int

	if atIdx := strings.LastIndex(body, "@"); atIdx != -1 {
		// SIP002 format: base64(method:password)@host:port or method:password@host:port
		userPart := body[:atIdx]
		hostPortPart := body[atIdx+1:]

		if strings.Contains(userPart, ":") {
			parts := strings.SplitN(userPart, ":", 2)
			cipher = parts[0]
			password, _ = url.QueryUnescape(parts[1])
		} else {
			decoded, err := DecodeBase64(userPart)
			if err != nil {
				return config.NodeConfig{}, fmt.Errorf("invalid ss userinfo base64: %w", err)
			}
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) != 2 {
				return config.NodeConfig{}, fmt.Errorf("invalid ss userinfo format")
			}
			cipher = parts[0]
			password = parts[1]
		}

		u, err := url.Parse("http://" + hostPortPart)
		if err != nil {
			return config.NodeConfig{}, fmt.Errorf("invalid ss host:port: %w", err)
		}
		host, port, err = parseHostPort(u)
		if err != nil {
			return config.NodeConfig{}, err
		}
	} else {
		// Legacy format: base64(method:password@host:port)
		decoded, err := DecodeBase64(body)
		if err != nil {
			return config.NodeConfig{}, fmt.Errorf("invalid legacy ss base64: %w", err)
		}
		decodedStr := string(decoded)
		atIdx := strings.LastIndex(decodedStr, "@")
		if atIdx == -1 {
			return config.NodeConfig{}, fmt.Errorf("missing '@' in legacy ss payload")
		}
		userPart := decodedStr[:atIdx]
		hostPortPart := decodedStr[atIdx+1:]

		parts := strings.SplitN(userPart, ":", 2)
		if len(parts) != 2 {
			return config.NodeConfig{}, fmt.Errorf("invalid legacy ss cipher:password")
		}
		cipher = parts[0]
		password = parts[1]

		colonIdx := strings.LastIndex(hostPortPart, ":")
		if colonIdx == -1 {
			return config.NodeConfig{}, fmt.Errorf("missing port in legacy ss payload")
		}
		host = strings.Trim(hostPortPart[:colonIdx], "[]")
		p, err := strconv.Atoi(hostPortPart[colonIdx+1:])
		if err != nil || p <= 0 || p > 65535 {
			return config.NodeConfig{}, fmt.Errorf("invalid port in legacy ss payload")
		}
		port = p
	}

	if cipher == "" || password == "" || host == "" || port <= 0 {
		return config.NodeConfig{}, fmt.Errorf("incomplete shadowsocks configuration")
	}

	name := decodeFragment(fragment, fmt.Sprintf("SS-%s:%d", host, port))
	return config.NodeConfig{
		Id:        newNodeID(),
		SubId:     config.DefaultGroupID,
		GroupName: config.DefaultGroupName,
		Name:      name,
		Protocol:  "ss",
		Server:    host,
		Port:      port,
		Cipher:    cipher,
		Password:  password,
		Network:   "tcp",
		RawLink:   rawLink,
	}, nil
}
