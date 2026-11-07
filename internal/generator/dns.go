package generator

import (
	"net/url"
	"strings"

	"wailsproxy/internal/config"
)

// buildSingBoxDNS 构建 Sing-box (1.12+ / 1.14+ 标准格式) 的 DNS 配置块
func buildSingBoxDNS(cfg config.AppConfig) map[string]interface{} {
	directDNS := strings.TrimSpace(cfg.DirectDNS)
	if directDNS == "" {
		directDNS = "223.5.5.5"
	}
	directDNS = strings.TrimPrefix(strings.TrimPrefix(directDNS, "udp://"), "https://")
	if idx := strings.Index(directDNS, "/"); idx > 0 {
		directDNS = directDNS[:idx]
	}

	remoteDNS := strings.TrimSpace(cfg.RemoteDNS)
	if remoteDNS == "" {
		if cfg.EnableDoH {
			remoteDNS = "https://1.1.1.1/dns-query"
		} else {
			remoteDNS = "1.1.1.1"
		}
	}

	remoteDetour := "proxy"
	finalServer := "remote-dns"
	if cfg.RoutingMode == config.RoutingModeDirect {
		remoteDetour = "direct"
		finalServer = "direct-dns"
	}

	remoteServerObj := map[string]interface{}{
		"tag":    "remote-dns",
		"detour": remoteDetour,
	}

	if cfg.EnableDoH || strings.HasPrefix(strings.ToLower(remoteDNS), "https://") {
		u, err := url.Parse(remoteDNS)
		host := "1.1.1.1"
		path := "/dns-query"
		if err == nil && u.Hostname() != "" {
			host = u.Hostname()
			if u.Path != "" {
				path = u.Path
			}
		} else if !strings.Contains(remoteDNS, "://") && remoteDNS != "" {
			host = remoteDNS
		}
		remoteServerObj["type"] = "https"
		remoteServerObj["server"] = host
		remoteServerObj["path"] = path
		remoteServerObj["domain_resolver"] = "direct-dns"
	} else {
		host := strings.TrimPrefix(remoteDNS, "udp://")
		if strings.HasPrefix(strings.ToLower(host), "https://") {
			host = "1.1.1.1"
		}
		remoteServerObj["type"] = "udp"
		remoteServerObj["server"] = host
	}

	rules := []map[string]interface{}{}
	if d, _ := classifyCustomRules(cfg.CustomBlockRules); len(d) > 0 {
		rules = append(rules, map[string]interface{}{
			"domain_suffix": normalizeDomainSuffixes(d),
			"action":        "reject",
		})
	}
	if cfg.RoutingMode == config.RoutingModeRule {
		if d, _ := classifyCustomRules(cfg.CustomProxyRules); len(d) > 0 {
			rules = append(rules, map[string]interface{}{
				"domain_suffix": normalizeDomainSuffixes(d),
				"server":        "remote-dns",
			})
		}
		directSuffixes := []string{".cn", "baidu.com", "qq.com", "taobao.com", "jd.com", "bilibili.com", "163.com", "aliyun.com", "apple.com", "icloud.com"}
		if d, _ := classifyCustomRules(cfg.CustomDirectRules); len(d) > 0 {
			directSuffixes = appendUniqueStrings(directSuffixes, normalizeDomainSuffixes(d)...)
		}
		rules = append(rules,
			map[string]interface{}{
				"domain_suffix": directSuffixes,
				"server":        "direct-dns",
			},
		)
	}

	return map[string]interface{}{
		"servers": []map[string]interface{}{
			{
				"type":   "udp",
				"tag":    "direct-dns",
				"server": directDNS,
			},
			remoteServerObj,
		},
		"rules":    rules,
		"final":    finalServer,
		"strategy": "prefer_ipv4",
	}
}

func normalizeDomainSuffixes(domains []string) []string {
	out := make([]string, 0, len(domains))
	for _, d := range domains {
		d = strings.TrimSpace(strings.TrimPrefix(d, "domain:"))
		if d == "" {
			continue
		}
		if d == "cn" {
			d = ".cn"
		}
		out = appendUniqueStrings(out, d)
	}
	return out
}

func appendUniqueStrings(base []string, items ...string) []string {
	seen := make(map[string]bool, len(base))
	for _, b := range base {
		seen[b] = true
	}
	for _, item := range items {
		if item != "" && !seen[item] {
			seen[item] = true
			base = append(base, item)
		}
	}
	return base
}

// buildXrayDNS 构建 Xray-core 的防 DNS 泄漏配置块
// 核心策略：
// 1. 内置 DoH 引导 hosts，避免解析 DoH 域名本身产生本地 UDP DNS 泄漏
// 2. 非直连模式下，优先使用远程加密 DoH/DNS，国内直连 DNS 严格限定在大陆域名白名单且开启 skipFallback
// 3. 彻底杜绝海外域名向本地运营商 DNS 发起查询导致的 DNS 泄漏与抢答污染
func buildXrayDNS(cfg config.AppConfig) map[string]interface{} {
	directDNS := strings.TrimSpace(cfg.DirectDNS)
	if directDNS == "" {
		directDNS = "223.5.5.5"
	}
	remoteDNS := strings.TrimSpace(cfg.RemoteDNS)
	if remoteDNS == "" {
		if cfg.EnableDoH {
			remoteDNS = "https://1.1.1.1/dns-query"
		} else {
			remoteDNS = "1.1.1.1"
		}
	}
	if !cfg.EnableDoH && strings.HasPrefix(strings.ToLower(remoteDNS), "https://") {
		remoteDNS = "1.1.1.1"
	}

	hosts := map[string]interface{}{
		"dns.google":              "8.8.8.8",
		"cloudflare-dns.com":      "1.1.1.1",
		"one.one.one.one":         "1.1.1.1",
		"security.cloudflare-dns.com": "1.1.1.2",
		"dns.quad9.net":           "9.9.9.9",
		"dns.alidns.com":          "223.5.5.5",
		"doh.pub":                 "1.12.12.12",
		"dot.pub":                 "1.12.12.12",
	}

	var servers []interface{}
	switch cfg.RoutingMode {
	case config.RoutingModeDirect:
		servers = []interface{}{
			directDNS,
			remoteDNS,
		}
	case config.RoutingModeGlobal:
		// 全局代理模式：100% 使用远程 DNS，完全不向国内 DNS 发送任何查询（零泄漏）
		servers = []interface{}{
			remoteDNS,
		}
	default: // RoutingModeRule (智能分流 + 严格防 DNS 泄漏)
		proxyDomains := []string{
			"domain:google.com",
			"domain:googleapis.com",
			"domain:gstatic.com",
			"domain:youtube.com",
			"domain:ytimg.com",
			"domain:github.com",
			"domain:githubusercontent.com",
			"domain:openai.com",
			"domain:chatgpt.com",
			"domain:anthropic.com",
			"domain:claude.ai",
			"domain:x.com",
			"domain:twitter.com",
			"domain:t.me",
			"domain:telegram.org",
			"domain:cloudflare.com",
		}
		if customProxy, _ := classifyCustomRules(cfg.CustomProxyRules); len(customProxy) > 0 {
			for _, d := range customProxy {
				proxyDomains = appendUniqueStrings(proxyDomains, "domain:"+strings.TrimPrefix(d, "."))
			}
		}

		directDomains := []string{
			"domain:cn",
			"domain:baidu.com",
			"domain:qq.com",
			"domain:weixin.qq.com",
			"domain:taobao.com",
			"domain:tmall.com",
			"domain:jd.com",
			"domain:alicdn.com",
			"domain:aliyun.com",
			"domain:bilibili.com",
			"domain:163.com",
			"domain:126.com",
			"domain:zhihu.com",
			"domain:douyin.com",
			"domain:weibo.com",
			"domain:apple.com",
			"domain:icloud.com",
		}
		if customDirect, _ := classifyCustomRules(cfg.CustomDirectRules); len(customDirect) > 0 {
			for _, d := range customDirect {
				directDomains = appendUniqueStrings(directDomains, "domain:"+strings.TrimPrefix(d, "."))
			}
		}

		servers = []interface{}{
			// 1. 海外高频与自定义代理域名：强制走远程 DoH/DNS
			map[string]interface{}{
				"address":      remoteDNS,
				"domains":      proxyDomains,
				"skipFallback": false,
			},
			// 2. 默认全局回退服务器：所有未匹配域名一律走远程加密 DNS，防止未知海外域名泄漏给本地运营商
			remoteDNS,
			// 3. 国内白名单域名：仅对明确匹配的大陆域名使用国内直连 DNS，且设置 skipFallback=true 严禁参与海外域名回退
			map[string]interface{}{
				"address":      directDNS,
				"domains":      directDomains,
				"skipFallback": true,
			},
		}
	}

	return map[string]interface{}{
		"hosts":                  hosts,
		"servers":                servers,
		"queryStrategy":          "UseIPv4",
		"disableFallbackIfMatch": true,
	}
}
