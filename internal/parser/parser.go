package parser

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"wailsproxy/internal/config"
)

// DecodeBase64 兼容带填充、无填充、标准 Base64 与 URL-Safe Base64 解码
func DecodeBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return nil, fmt.Errorf("empty base64 input")
	}

	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}

	for _, enc := range encodings {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}

	// 尝试补齐 '=' 后解码
	if rem := len(s) % 4; rem != 0 {
		padded := s + strings.Repeat("=", 4-rem)
		if b, err := base64.StdEncoding.DecodeString(padded); err == nil {
			return b, nil
		}
		if b, err := base64.URLEncoding.DecodeString(padded); err == nil {
			return b, nil
		}
	}

	return nil, fmt.Errorf("invalid base64 encoding")
}

// ParseLink 解析单条节点分享链接（支持 vmess, vless, trojan, ss, hysteria2/hy2, tuic）
func ParseLink(rawLink string) (config.NodeConfig, error) {
	link := strings.TrimSpace(rawLink)
	if link == "" {
		return config.NodeConfig{}, fmt.Errorf("empty share link")
	}

	lower := strings.ToLower(link)
	switch {
	case strings.HasPrefix(lower, "vmess://"):
		return parseVMess(link)
	case strings.HasPrefix(lower, "vless://"):
		return parseVLESS(link)
	case strings.HasPrefix(lower, "trojan://"):
		return parseTrojan(link)
	case strings.HasPrefix(lower, "ss://"):
		return parseShadowsocks(link)
	case strings.HasPrefix(lower, "hysteria2://"), strings.HasPrefix(lower, "hy2://"):
		return parseHysteria2(link)
	case strings.HasPrefix(lower, "tuic://"):
		return parseTUIC(link)
	default:
		return config.NodeConfig{}, fmt.Errorf("unsupported protocol scheme in link: %s", link)
	}
}

// ParseMultipleLinks 批量解析多行分享链接（或 Base64 编码的多行文本）
func ParseMultipleLinks(input string) ([]config.NodeConfig, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return nil, fmt.Errorf("输入内容为空")
	}

	// 如果不包含 ://，尝试先进行 Base64 解码
	if !strings.Contains(text, "://") {
		if decoded, err := DecodeBase64(text); err == nil && strings.Contains(string(decoded), "://") {
			text = string(decoded)
		}
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var nodes []config.NodeConfig
	var lastErr error

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		node, err := ParseLink(line)
		if err != nil {
			lastErr = err
			continue
		}
		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("未识别到有效的代理节点链接")
	}
	return nodes, nil
}

func parseHostPort(u *url.URL) (string, int, error) {
	host := u.Hostname()
	portStr := u.Port()
	if host == "" || portStr == "" {
		return "", 0, fmt.Errorf("missing host or port")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("invalid port: %s", portStr)
	}
	return host, port, nil
}

func decodeFragment(frag string, fallback string) string {
	if frag == "" {
		return fallback
	}
	if unescaped, err := url.QueryUnescape(frag); err == nil && unescaped != "" {
		return strings.TrimSpace(unescaped)
	}
	return strings.TrimSpace(frag)
}

func newNodeID() string {
	return uuid.NewString()
}
