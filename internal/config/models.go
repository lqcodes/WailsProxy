package config

// ConnectionStatus 连接状态机
// 状态流转: Disconnected → Connecting → Connected → Disconnecting → Disconnected
type ConnectionStatus = string

const (
	StatusDisconnected  ConnectionStatus = "disconnected"
	StatusConnecting    ConnectionStatus = "connecting"
	StatusConnected     ConnectionStatus = "connected"
	StatusDisconnecting ConnectionStatus = "disconnecting"
	StatusError         ConnectionStatus = "error"
)

// ProxyMode 连接模式常量
const (
	ProxyModeSystem = "system"
	ProxyModeManual = "manual"
	ProxyModeTun    = "tun"
)

// RoutingMode 路由分流模式常量
const (
	RoutingModeRule   = "rule"
	RoutingModeGlobal = "global"
	RoutingModeDirect = "direct"
)

// DefaultGroupID 默认分组 ID
const (
	DefaultGroupID   = "default"
	DefaultGroupName = "默认分组"
)

// AppConfig 全局应用配置（持久化到磁盘 app_config.json）
type AppConfig struct {
	// 本地代理端口
	SocksPort int  `json:"socksPort"` // 默认 10808
	HttpPort  int  `json:"httpPort"`  // 默认 10809
	AllowLan  bool `json:"allowLan"`  // true 则监听 0.0.0.0

	// 连接模式与路由模式
	ProxyMode   string `json:"proxyMode"`   // "system" | "manual" | "tun"
	RoutingMode string `json:"routingMode"` // "rule" | "global" | "direct"

	// DNS 配置
	DirectDNS string `json:"directDns"` // 默认 "223.5.5.5"
	RemoteDNS string `json:"remoteDns"` // 默认 "https://1.1.1.1/dns-query"
	EnableDoH bool   `json:"enableDoh"`

	// 主题
	ThemeMode  string `json:"themeMode"`  // "light" | "dark" | "auto"
	ThemeColor string `json:"themeColor"` // "blue" | "green" | "purple" | "orange" | "pink"

	// 桌面行为
	CloseToTray bool `json:"closeToTray"` // true：关闭窗口 → 最小化到托盘
	AutoStart   bool `json:"autoStart"`   // 开机自启动

	// 延迟测试
	LatencyTestUrl string `json:"latencyTestUrl"` // 默认 "http://cp.cloudflare.com/generate_204"
	LatencyTimeout int    `json:"latencyTimeout"` // 测速超时毫秒，默认 3000
	LatencyMaxConc int    `json:"latencyMaxConc"` // 最大并发测速数，默认 20

	// 活动选择
	ActiveNodeId  string `json:"activeNodeId"`
	ActiveGroupId string `json:"activeGroupId"`

	// 内核选择
	CoreType string `json:"coreType"` // "sing-box" | "xray"

	// 自定义分流规则
	CustomDirectRules []string `json:"customDirectRules"`
	CustomProxyRules  []string `json:"customProxyRules"`
	CustomBlockRules  []string `json:"customBlockRules"`
}

// DefaultDirectRules 返回默认直连常用规则（国内域名、苹果服务与局域网段）
func DefaultDirectRules() []string {
	return []string{
		"cn",
		"baidu.com",
		"qq.com",
		"weixin.qq.com",
		"taobao.com",
		"tmall.com",
		"jd.com",
		"alicdn.com",
		"aliyun.com",
		"bilibili.com",
		"163.com",
		"126.com",
		"zhihu.com",
		"douyin.com",
		"weibo.com",
		"meituan.com",
		"apple.com",
		"icloud.com",
		"192.168.0.0/16",
		"10.0.0.0/8",
		"172.16.0.0/12",
	}
}

// DefaultProxyRules 返回默认强制代理常用规则（防 DNS 污染与高频海外服务）
func DefaultProxyRules() []string {
	return []string{
		"google.com",
		"googleapis.com",
		"gstatic.com",
		"youtube.com",
		"ytimg.com",
		"ggpht.com",
		"github.com",
		"githubusercontent.com",
		"openai.com",
		"chatgpt.com",
		"anthropic.com",
		"claude.ai",
		"x.com",
		"twitter.com",
		"t.me",
		"telegram.org",
		"cloudflare.com",
		"wikipedia.org",
		"docker.com",
		"huggingface.co",
	}
}

// DefaultBlockRules 返回默认广告与追踪拦截常用规则
func DefaultBlockRules() []string {
	return []string{
		"doubleclick.net",
		"googlesyndication.com",
		"googleadservices.com",
		"adservice.google.com",
		"cpro.baidu.com",
		"pos.baidu.com",
		"tanx.com",
		"miaozhen.com",
		"umeng.com",
	}
}

// DefaultAppConfig 返回带合理默认值的 AppConfig
func DefaultAppConfig() AppConfig {
	return AppConfig{
		SocksPort:         10808,
		HttpPort:          10809,
		AllowLan:          false,
		ProxyMode:         ProxyModeSystem,
		RoutingMode:       RoutingModeRule,
		DirectDNS:         "223.5.5.5",
		RemoteDNS:         "https://1.1.1.1/dns-query",
		EnableDoH:         true,
		ThemeMode:         "dark",
		ThemeColor:        "blue",
		CloseToTray:       false,
		AutoStart:         false,
		LatencyTestUrl:    "http://cp.cloudflare.com/generate_204",
		LatencyTimeout:    3000,
		LatencyMaxConc:    20,
		ActiveNodeId:      "",
		ActiveGroupId:     DefaultGroupID,
		CoreType:          "xray",
		CustomDirectRules: DefaultDirectRules(),
		CustomProxyRules:  DefaultProxyRules(),
		CustomBlockRules:  DefaultBlockRules(),
	}
}

// NodeConfig 统一节点模型（支持全部主流协议的完整参数）
type NodeConfig struct {
	Id        string `json:"id"`
	SubId     string `json:"subId"`     // 所属订阅分组 ID
	GroupName string `json:"groupName"` // 分组显示名称
	Name      string `json:"name"`      // 节点别名
	Protocol  string `json:"protocol"`  // vmess | vless | trojan | ss | hysteria2 | tuic

	// 基础连接
	Server string `json:"server"`
	Port   int    `json:"port"`

	// 认证（按协议使用不同字段）
	Uuid     string `json:"uuid,omitempty"`     // VMess/VLESS/Trojan UUID
	Password string `json:"password,omitempty"` // SS/Trojan/Hysteria2 密码
	AlterId  int    `json:"alterId,omitempty"`  // VMess AlterID
	Cipher   string `json:"cipher,omitempty"`   // SS 加密算法 (aes-256-gcm, chacha20-ietf-poly1305 等)

	// 传输层 (Transport)
	Network     string `json:"network,omitempty"`     // tcp | ws | grpc | quic | h2 | httpupgrade
	Path        string `json:"path,omitempty"`        // WebSocket/HTTP path
	Host        string `json:"host,omitempty"`        // WebSocket/HTTP Host 头
	ServiceName string `json:"serviceName,omitempty"` // gRPC serviceName

	// TLS / Reality
	Tls           bool   `json:"tls"`
	Sni           string `json:"sni,omitempty"`
	Alpn          string `json:"alpn,omitempty"`          // 逗号分隔: "h2,http/1.1"
	Fingerprint   string `json:"fingerprint,omitempty"`   // uTLS 指纹 (chrome, firefox, safari 等)
	Security      string `json:"security,omitempty"`      // VLESS: "reality" | "tls" | "none"
	PublicKey     string `json:"publicKey,omitempty"`     // Reality 公钥
	ShortId       string `json:"shortId,omitempty"`       // Reality ShortId
	SpiderX       string `json:"spiderX,omitempty"`       // Reality SpiderX (spx)
	Mldsa65Verify string `json:"mldsa65Verify,omitempty"` // Reality ML-DSA-65 后量子验证公钥 (pqv)
	Flow          string `json:"flow,omitempty"`          // VLESS XTLS flow: "xtls-rprx-vision"

	// Hysteria2 专属
	ObfsType     string `json:"obfsType,omitempty"` // "salamander"
	ObfsPassword string `json:"obfsPassword,omitempty"`
	UpMbps       int    `json:"upMbps,omitempty"`   // 上行带宽 Mbps
	DownMbps     int    `json:"downMbps,omitempty"` // 下行带宽 Mbps

	// 测速结果
	TcpDelay  int64 `json:"tcpDelay"`  // TCP 握手延迟 ms (-1 超时, 0 未测)
	RealDelay int64 `json:"realDelay"` // HTTP 204 真实延迟 ms (-1 超时, 0 未测)

	// 原始链接（便于导出分享）
	RawLink string `json:"rawLink,omitempty"`
}

// Subscription 订阅源定义
type Subscription struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Url            string `json:"url"`
	AutoUpdate     bool   `json:"autoUpdate"`
	UpdateInterval int    `json:"updateInterval"`      // 更新间隔（小时），0 表示手动
	LastUpdated    int64  `json:"lastUpdated"`         // 上次成功更新 Unix 时间戳
	NodeCount      int    `json:"nodeCount"`           // 当前组内节点数量
	LastError      string `json:"lastError,omitempty"` // 最近一次拉取失败的错误信息
}

// TrafficStats 实时流量统计（由 Go 每秒推送给前端）
type TrafficStats struct {
	UploadSpeed   int64 `json:"uploadSpeed"`   // 字节/秒
	DownloadSpeed int64 `json:"downloadSpeed"` // 字节/秒
	TotalUpload   int64 `json:"totalUpload"`   // 本次连接累计上传字节
	TotalDownload int64 `json:"totalDownload"` // 本次连接累计下载字节
	ActiveConns   int   `json:"activeConns"`   // 活跃连接数
}

// PingResultEvent 单个节点测速完成事件载荷
type PingResultEvent struct {
	NodeId    string `json:"nodeId"`
	TcpDelay  int64  `json:"tcpDelay"`
	RealDelay int64  `json:"realDelay"`
}

// SubscriptionUpdatedEvent 订阅更新完成通知载荷
type SubscriptionUpdatedEvent struct {
	SubId     string `json:"subId"`
	NodeCount int    `json:"nodeCount"`
	Error     string `json:"error,omitempty"`
}

// DependencyItem 单个系统/核心依赖项检测状态
type DependencyItem struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Installed   bool   `json:"installed"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Description string `json:"description"`
	InstallCmd  string `json:"installCmd,omitempty"`
	Error       string `json:"error,omitempty"`
}

// DependencyCheckResult 启动依赖项全面自检结果
type DependencyCheckResult struct {
	Ready    bool             `json:"ready"`
	Platform string           `json:"platform"`
	BinDir   string           `json:"binDir"`
	Items    []DependencyItem `json:"items"`
	Message  string           `json:"message"`
}
