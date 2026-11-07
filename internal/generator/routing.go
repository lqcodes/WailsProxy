package generator

import (
	"strings"

	"wailsproxy/internal/config"
)

func classifyCustomRules(items []string) (domains []string, cidrs []string) {
	for _, raw := range items {
		r := strings.TrimSpace(raw)
		if r == "" || strings.HasPrefix(r, "#") {
			continue
		}
		if strings.Contains(r, "/") || isLikelyIP(r) {
			if !strings.Contains(r, "/") {
				r = r + "/32"
			}
			cidrs = append(cidrs, r)
		} else {
			domains = append(domains, strings.TrimPrefix(r, "domain:"))
		}
	}
	return domains, cidrs
}

func isLikelyIP(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 3 {
			return false
		}
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

// buildSingBoxRoute 构建 Sing-box (1.12+ / 1.14+ 标准格式) 的路由规则配置块
func buildSingBoxRoute(cfg config.AppConfig) map[string]interface{} {
	rules := []map[string]interface{}{
		{
			"action": "sniff",
		},
		{
			"protocol": "dns",
			"action":   "hijack-dns",
		},
		{
			"ip_is_private": true,
			"outbound":      "direct",
		},
	}

	// 用户自定义拦截规则
	if d, c := classifyCustomRules(cfg.CustomBlockRules); len(d) > 0 || len(c) > 0 {
		if nd := normalizeDomainSuffixes(d); len(nd) > 0 {
			rules = append(rules, map[string]interface{}{
				"domain_suffix": nd,
				"action":        "reject",
			})
		}
		if len(c) > 0 {
			rules = append(rules, map[string]interface{}{
				"ip_cidr": c,
				"action":  "reject",
			})
		}
	}

	// 用户自定义代理规则（优先于泛直连规则）
	if cfg.RoutingMode != config.RoutingModeDirect {
		if d, c := classifyCustomRules(cfg.CustomProxyRules); len(d) > 0 || len(c) > 0 {
			if nd := normalizeDomainSuffixes(d); len(nd) > 0 {
				rules = append(rules, map[string]interface{}{
					"domain_suffix": nd,
					"outbound":      "proxy",
				})
			}
			if len(c) > 0 {
				rules = append(rules, map[string]interface{}{
					"ip_cidr":  c,
					"outbound": "proxy",
				})
			}
		}
	}

	// 用户自定义直连规则
	if cfg.RoutingMode == config.RoutingModeRule {
		if d, c := classifyCustomRules(cfg.CustomDirectRules); len(d) > 0 || len(c) > 0 {
			if nd := normalizeDomainSuffixes(d); len(nd) > 0 {
				rules = append(rules, map[string]interface{}{
					"domain_suffix": nd,
					"outbound":      "direct",
				})
			}
			if len(c) > 0 {
				rules = append(rules, map[string]interface{}{
					"ip_cidr":  c,
					"outbound": "direct",
				})
			}
		}
	}

	finalOutbound := "proxy"
	switch cfg.RoutingMode {
	case config.RoutingModeDirect:
		finalOutbound = "direct"
	case config.RoutingModeGlobal:
		finalOutbound = "proxy"
	default: // RoutingModeRule (Bypass-CN)
		rules = append(rules,
			map[string]interface{}{
				"domain_suffix": []string{".cn", "baidu.com", "qq.com", "weixin.qq.com", "taobao.com", "tmall.com", "alicdn.com", "aliyun.com", "jd.com", "bilibili.com", "163.com", "126.com", "sohu.com", "sina.com.cn", "zhihu.com", "douyin.com"},
				"outbound":      "direct",
			},
		)
		finalOutbound = "proxy"
	}

	return map[string]interface{}{
		"default_domain_resolver": map[string]interface{}{
			"server": "direct-dns",
		},
		"auto_detect_interface": true,
		"rules":                 rules,
		"final":                 finalOutbound,
	}
}

func toXrayDomains(domains []string) []string {
	out := make([]string, 0, len(domains))
	for _, d := range domains {
		d = strings.TrimSpace(strings.TrimPrefix(d, "domain:"))
		d = strings.TrimPrefix(d, ".")
		if d == "" {
			continue
		}
		out = appendUniqueStrings(out, "domain:"+d)
	}
	return out
}

// buildXrayRouting 构建 Xray-core 的路由配置块（内置防 DNS 泄漏与常用规则优先级保护）
func buildXrayRouting(cfg config.AppConfig) map[string]interface{} {
	rules := []map[string]interface{}{
		// 拦截 QUIC (UDP 443)，强制浏览器回退至 TLS TCP，防止部分环境下 QUIC 绕过代理产生真实 IP/DNS 泄漏
		{
			"type":        "field",
			"port":        "443",
			"network":     "udp",
			"outboundTag": "block",
		},
		// 局域网私有地址直连
		{
			"type":        "field",
			"ip":          []string{"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7"},
			"outboundTag": "direct",
		},
		// 国内公共 DNS 服务器强制直连
		{
			"type":        "field",
			"ip":          []string{"223.5.5.5/32", "223.6.6.6/32", "119.29.29.29/32", "114.114.114.114/32", "1.12.12.12/32"},
			"outboundTag": "direct",
		},
	}

	// 非直连模式下，强制海外 DoH / 公共 DNS 服务器流量走代理隧道，防止 DNS 请求明文或直连泄漏
	if cfg.RoutingMode != config.RoutingModeDirect {
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"ip":          []string{"1.1.1.1/32", "1.0.0.1/32", "8.8.8.8/32", "8.8.4.4/32", "9.9.9.9/32", "208.67.222.222/32"},
			"outboundTag": "proxy",
		})
	}

	// 1. 广告与恶意追踪拦截规则（最高业务优先级）
	if d, c := classifyCustomRules(cfg.CustomBlockRules); len(d) > 0 || len(c) > 0 {
		if xd := toXrayDomains(d); len(xd) > 0 {
			rules = append(rules, map[string]interface{}{"type": "field", "domain": xd, "outboundTag": "block"})
		}
		if len(c) > 0 {
			rules = append(rules, map[string]interface{}{"type": "field", "ip": c, "outboundTag": "block"})
		}
	}

	// 2. 自定义与默认强制代理规则（优先于泛直连规则，确保 Google/GitHub/OpenAI 等绝不误走直连）
	if cfg.RoutingMode != config.RoutingModeDirect {
		if d, c := classifyCustomRules(cfg.CustomProxyRules); len(d) > 0 || len(c) > 0 {
			if xd := toXrayDomains(d); len(xd) > 0 {
				rules = append(rules, map[string]interface{}{"type": "field", "domain": xd, "outboundTag": "proxy"})
			}
			if len(c) > 0 {
				rules = append(rules, map[string]interface{}{"type": "field", "ip": c, "outboundTag": "proxy"})
			}
		}
	}

	// 3. 自定义直连规则（仅在规则模式下生效，全局模式下除私有 IP 外全部走代理）
	if cfg.RoutingMode == config.RoutingModeRule {
		if d, c := classifyCustomRules(cfg.CustomDirectRules); len(d) > 0 || len(c) > 0 {
			if xd := toXrayDomains(d); len(xd) > 0 {
				rules = append(rules, map[string]interface{}{"type": "field", "domain": xd, "outboundTag": "direct"})
			}
			if len(c) > 0 {
				rules = append(rules, map[string]interface{}{"type": "field", "ip": c, "outboundTag": "direct"})
			}
		}
	}

	switch cfg.RoutingMode {
	case config.RoutingModeDirect:
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"port":        "0-65535",
			"outboundTag": "direct",
		})
	case config.RoutingModeGlobal:
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"port":        "0-65535",
			"outboundTag": "proxy",
		})
	default: // Rule Mode (Bypass-CN)
		rules = append(rules,
			map[string]interface{}{
				"type": "field",
				"domain": []string{
					"domain:cn",
					"domain:baidu.com",
					"domain:qq.com",
					"domain:weixin.qq.com",
					"domain:taobao.com",
					"domain:tmall.com",
					"domain:alicdn.com",
					"domain:aliyun.com",
					"domain:jd.com",
					"domain:bilibili.com",
					"domain:163.com",
					"domain:126.com",
					"domain:zhihu.com",
					"domain:douyin.com",
					"domain:weibo.com",
				},
				"outboundTag": "direct",
			},
			map[string]interface{}{
				"type":        "field",
				"port":        "0-65535",
				"outboundTag": "proxy",
			},
		)
	}

	return map[string]interface{}{
		"domainStrategy": "AsIs",
		"rules":          rules,
	}
}
