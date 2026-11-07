# AGENTS.md - WailsProxy 智能体协作与开发规范

本文档为所有参与 `WailsProxy`（基于 **Go + Wails + Ant Design (AntD)** 开发的类似 **v2rayN / Clash Verge** 现代化跨平台客户端）项目开发的 AI Agent（智能体）提供统一的架构设计规范、核心模块边界、编码规范与开发约束。

---

## 1. 项目定位与核心需求规范

`WailsProxy` 是一款现代、美观、跨平台（macOS、Windows、Linux）的通用代理客户端 GUI 工具。

### 1.1 核心业务需求矩阵

#### 需求 1：内核直接集成成熟开源二进制（无需自研协议内核）
* **直接集成开源核心**：底层代理协议引擎直接采用开源成熟内核（默认 **Sing-box** 与 **Xray-core** 发行版二进制），严禁重复造轮子实现底层 VMess/VLESS/Hysteria 网络协议。
* **Go 后端的职责**：仅负责内核二进制管理、版本检测与自动获取、配置生成器（Generator）、子进程生命周期守护、流量统计采集与系统代理设置。

#### 需求 2：界面美观与实时上下行流量可视化
* **主控连接按钮**：仪表盘/顶部提供醒目的大号**一键"连接 / 断开"切换按钮**，配合动态连接状态徽标（已连接、连接中、已断开、错误告警）。
* **实时网速仪表盘**：卡片式展示实时上传速率、下载速率、历史累计上传、历史累计下载、活动连接数。
* **流量波形图**：采用轻量且高性能的平滑折线/面积图（SVG / Canvas），实时渲染最近 60 秒的上下行速率曲线，极低 CPU 占用。
* **高频推流架构**：Go 后端每秒采集内核流量统计，通过 `runtime.EventsEmit(ctx, "traffic:stats", data)` 主动推送，前端纯事件响应，严禁轮询。

#### 需求 3：订阅分组与批量管理
* **订阅分组机制**：每个节点归属于其对应订阅（如分组：*机场A-香港专线*、*自建节点*、*默认分组*）。
* **UI 交互**：AntD 提供分组 Tabs 标签页或分组抽屉，支持按分组筛选、切换全部节点。
* **一键刷新与测速**：
  - 针对单个订阅或全部订阅**一键"刷新节点"**（拉取最新配置、清理失效节点）。
  - **批量/单个"延迟测试"**（TCP 握手延时 + HTTP 204 真实延时），测速完成后支持一键按延迟排序。

#### 需求 4：多主题动态无缝切换
* **基础模式**：亮色模式 (Light)、暗黑模式 (Dark)、跟随系统 (System Auto)。
* **多主题色预设 (Color Schemes)**：极客蓝 (#1677ff - 默认)、极光绿 (#52c41a)、科技紫 (#722ed1)、晚霞橙 (#fa8c16)、赛博朋克 (#eb2f96)。
* **实现方案**：基于 Ant Design v5 的 CSS-in-JS 与 `ConfigProvider` 动态 Token，0 秒瞬时切换。

#### 需求 5：三大连接模式与核心分流路由
* **三大连接模式 (Proxy Modes)**：
  1. **系统代理模式 (System Proxy)**：自动托管系统网络代理，具备 Crash-Safe 崩溃自动还原守卫。
  2. **纯代理 / 手动模式 (Manual / Core Only)**：仅开放本地监听端口（如 10808/10809），不改动系统代理。
  3. **TUN 虚拟网卡模式 (TUN Mode)**：透明接管系统底层网络，支持游戏与全局应用。
* **核心分流模式 (Routing Modes)**：
  1. **规则模式 (Rule / Bypass-CN)**：智能分流，国内域名/IP 直连，国外走代理，广告拦截。
  2. **全局模式 (Global)**：除私有局域网外，所有流量强制走当前活动代理节点。
  3. **直连模式 (Direct)**：所有流量直连不走代理，方便排查故障。

#### 需求 6：自定义 DNS 与本地端口配置
* **本地监听**：Socks5 端口、HTTP 代理端口、局域网共享开关 (Allow LAN: 127.0.0.1 vs 0.0.0.0)。
* **自定义 DNS 引擎**：国内直连 DNS（如 223.5.5.5）、远程代理 DNS（如 1.1.1.1、8.8.8.8）、DoH/DoT 支持。

#### 需求 7：全量功能测试保障 (Comprehensive Testing Suite)
* **单元测试体系**：覆盖所有协议（VMess, VLESS, Trojan, SS, Hysteria2）和订阅格式（Base64, Clash YAML, Sing-box JSON）的单测。
* **配置生成与语法校验测试**：自动化测试各种节点与分流模式下生成的内核配置，通过真实内核 `-test` / `check` 语法校验。
* **延迟测速测试**：覆盖 Mock 测速服务器、高并发超时控制与结果解析。
* **端到端集成测试**：覆盖启停连接、模式切换、原子配置文件保存与异常恢复。

---

## 2. 核心技术栈

| 层次 | 选用技术 | 说明与规范 |
| :--- | :--- | :--- |
| **桌面框架** | **Wails v2** | Go 原生轻量化桌面客户端 |
| **后端语言** | **Go 1.21+** | 负责开源内核管理、跨平台系统代理、子进程守护、订阅与测速 |
| **开源代理内核** | **Sing-box / Xray-core** | 直接集成预编译二进制，放置在 `bin/` 目录或从 GitHub Release 自动获取 |
| **前端语言** | **TypeScript** | 强类型模式，对接 Wails 自动生成的接口 |
| **前端框架** | **React 18+ (Vite)** | 响应式组件化设计 |
| **UI 组件库** | **Ant Design (AntD v5)** | 桌面端交互首选，企业级 Table, Tabs, Modal, Segmented, Card |
| **状态管理** | **Zustand** | 响应式状态管理：`nodeStore`, `configStore`, `trafficStore`, `themeStore` |
| **自动化测试** | **Go `testing` + Testify** | 覆盖解析器、配置生成器、测速器、系统代理恢复逻辑 |

---

## 3. 标准目录结构

```text
WailsProxy/
├── AGENTS.md                      # 项目规范与架构约束（本文档）
├── main.go                        # Wails 主程序入口
├── app.go                         # Wails App 结构体（暴露给前端的全部方法）
├── wails.json                     # Wails 项目配置
├── go.mod / go.sum
├── bin/                           # 开源内核二进制（sing-box, xray 等，不纳入 Git）
├── data/                          # 运行时数据目录（GeoIP/GeoSite 规则集、生成的配置文件）
├── internal/                      # Go 后端核心业务
│   ├── config/                    # 应用配置读写与原子持久化（app_config.json）
│   ├── parser/                    # 协议 URI 解析器（vmess, vless, ss, trojan, hy2, tuic）
│   ├── subscription/              # 订阅拉取、格式识别（Base64/Clash YAML/Sing-box JSON）、分组管理
│   ├── generator/                 # 内核配置动态生成（inbound, outbound, routing, dns）
│   ├── core/                      # 内核进程守护（启停、健康检查、日志转发、流量采集）
│   ├── sysproxy/                  # 跨平台系统代理设置与还原（macOS/Windows/Linux）
│   ├── ping/                      # TCP Ping 与 HTTP 真实延迟测速
│   └── tray/                      # 系统托盘菜单与桌面生命周期管理
├── .agents/
│   └── skills/
│       └── wails-proxy/
│           └── SKILL.md           # 智能体开发工作流技能
└── frontend/                      # React + AntD 前端
    ├── package.json
    ├── vite.config.ts
    ├── tsconfig.json
    ├── src/
    │   ├── main.tsx               # React 入口
    │   ├── App.tsx                # 根组件（AntD ConfigProvider + 主题 + Router）
    │   ├── api/                   # 封装 wailsjs 调用（按模块分文件）
    │   ├── stores/                # Zustand 状态管理
    │   │   ├── nodeStore.ts       # 节点列表、分组过滤、选中节点
    │   │   ├── configStore.ts     # 全局配置（端口、DNS、代理模式、路由模式）
    │   │   ├── trafficStore.ts    # 实时流量数据与历史队列
    │   │   └── themeStore.ts      # 主题模式与主题色
    │   ├── components/            # 通用组件
    │   │   ├── TrafficChart.tsx    # 实时流量波形图（Canvas/SVG）
    │   │   ├── ConnectButton.tsx  # 主控连接/断开大按钮
    │   │   ├── NodeTable.tsx      # 节点表格（含测速、协议标签）
    │   │   └── ThemeSwitcher.tsx  # 主题模式与色彩切换器
    │   ├── views/                 # 页面视图
    │   │   ├── Dashboard.tsx      # 仪表盘（连接按钮 + 流量图 + 状态卡片）
    │   │   ├── Servers.tsx        # 节点与订阅管理（分组 Tabs + 表格）
    │   │   ├── Rules.tsx          # 路由规则管理
    │   │   ├── Settings.tsx       # 设置页（端口、DNS、桌面行为、主题）
    │   │   └── Logs.tsx           # 实时内核日志
    │   └── utils/                 # 工具函数（格式化流量单位、时间等）
    └── wailsjs/                   # Wails 自动生成的 TS 绑定（自动维护，勿手动修改）
```

---

## 4. 核心数据契约模型 (Data Contracts)

```go
// ==================== 持久化配置（存储在 app_config.json） ====================

// AppConfig 全局应用配置（持久化到磁盘）
type AppConfig struct {
    // 本地代理端口
    SocksPort         int    `json:"socksPort"`         // 默认 10808
    HttpPort          int    `json:"httpPort"`           // 默认 10809
    AllowLan          bool   `json:"allowLan"`           // true 则监听 0.0.0.0

    // 连接模式与路由模式
    ProxyMode         string `json:"proxyMode"`          // "system" | "manual" | "tun"
    RoutingMode       string `json:"routingMode"`        // "rule" | "global" | "direct"

    // DNS 配置
    DirectDNS         string `json:"directDns"`          // 默认 "223.5.5.5"
    RemoteDNS         string `json:"remoteDns"`          // 默认 "https://1.1.1.1/dns-query"
    EnableDoH         bool   `json:"enableDoh"`

    // 主题
    ThemeMode         string `json:"themeMode"`          // "light" | "dark" | "auto"
    ThemeColor        string `json:"themeColor"`         // "blue" | "green" | "purple" | "orange" | "pink"

    // 桌面行为
    CloseToTray       bool   `json:"closeToTray"`        // true：关闭窗口 → 最小化到托盘
    AutoStart         bool   `json:"autoStart"`          // 开机自启动

    // 延迟测试
    LatencyTestUrl    string `json:"latencyTestUrl"`     // 默认 "http://cp.cloudflare.com/generate_204"
    LatencyTimeout    int    `json:"latencyTimeout"`     // 测速超时毫秒，默认 3000
    LatencyMaxConc    int    `json:"latencyMaxConc"`     // 最大并发测速数，默认 20

    // 活动选择
    ActiveNodeId      string `json:"activeNodeId"`
    ActiveGroupId     string `json:"activeGroupId"`

    // 内核选择
    CoreType          string `json:"coreType"`           // "sing-box" | "xray"
}

// ==================== 运行时状态（仅内存，不落盘） ====================

// ConnectionStatus 连接状态机
// 状态流转: Disconnected → Connecting → Connected → Disconnecting → Disconnected
type ConnectionStatus string
const (
    StatusDisconnected  ConnectionStatus = "disconnected"
    StatusConnecting    ConnectionStatus = "connecting"
    StatusConnected     ConnectionStatus = "connected"
    StatusDisconnecting ConnectionStatus = "disconnecting"
    StatusError         ConnectionStatus = "error"
)

// ==================== 节点模型 ====================

// NodeConfig 统一节点模型（支持全部主流协议的完整参数）
type NodeConfig struct {
    Id            string `json:"id"`
    SubId         string `json:"subId"`                  // 所属订阅分组 ID
    GroupName     string `json:"groupName"`               // 分组显示名称
    Name          string `json:"name"`                    // 节点别名
    Protocol      string `json:"protocol"`                // vmess | vless | trojan | ss | hysteria2 | tuic

    // 基础连接
    Server        string `json:"server"`
    Port          int    `json:"port"`

    // 认证（按协议使用不同字段）
    Uuid          string `json:"uuid,omitempty"`          // VMess/VLESS/Trojan UUID
    Password      string `json:"password,omitempty"`      // SS/Trojan/Hysteria2 密码
    AlterId       int    `json:"alterId,omitempty"`       // VMess AlterID
    Cipher        string `json:"cipher,omitempty"`        // SS 加密算法 (aes-256-gcm, chacha20-ietf-poly1305 等)

    // 传输层 (Transport)
    Network       string `json:"network,omitempty"`       // tcp | ws | grpc | quic | h2 | httpupgrade
    Path          string `json:"path,omitempty"`          // WebSocket/HTTP path
    Host          string `json:"host,omitempty"`          // WebSocket/HTTP Host 头
    ServiceName   string `json:"serviceName,omitempty"`   // gRPC serviceName

    // TLS / Reality
    Tls           bool   `json:"tls"`
    Sni           string `json:"sni,omitempty"`
    Alpn          string `json:"alpn,omitempty"`          // 逗号分隔: "h2,http/1.1"
    Fingerprint   string `json:"fingerprint,omitempty"`   // uTLS 指纹 (chrome, firefox, safari 等)
    Security      string `json:"security,omitempty"`      // VLESS: "reality" | "tls" | "none"
    PublicKey     string `json:"publicKey,omitempty"`      // Reality 公钥
    ShortId       string `json:"shortId,omitempty"`        // Reality ShortId
    Flow          string `json:"flow,omitempty"`           // VLESS XTLS flow: "xtls-rprx-vision"

    // Hysteria2 专属
    ObfsType      string `json:"obfsType,omitempty"`      // "salamander"
    ObfsPassword  string `json:"obfsPassword,omitempty"`
    UpMbps        int    `json:"upMbps,omitempty"`         // 上行带宽 Mbps
    DownMbps      int    `json:"downMbps,omitempty"`       // 下行带宽 Mbps

    // 测速结果
    TcpDelay      int64  `json:"tcpDelay"`                // TCP 握手延迟 ms (-1 超时, 0 未测)
    RealDelay     int64  `json:"realDelay"`               // HTTP 204 真实延迟 ms (-1 超时, 0 未测)

    // 原始链接（便于导出分享）
    RawLink       string `json:"rawLink,omitempty"`
}

// ==================== 订阅模型 ====================

// Subscription 订阅源定义
type Subscription struct {
    Id             string `json:"id"`
    Name           string `json:"name"`
    Url            string `json:"url"`
    AutoUpdate     bool   `json:"autoUpdate"`
    UpdateInterval int    `json:"updateInterval"`        // 更新间隔（小时），0 表示手动
    LastUpdated    int64  `json:"lastUpdated"`            // 上次成功更新 Unix 时间戳
    NodeCount      int    `json:"nodeCount"`              // 当前组内节点数量
    LastError      string `json:"lastError,omitempty"`    // 最近一次拉取失败的错误信息
}

// ==================== 实时流量 ====================

// TrafficStats 实时流量统计（由 Go 每秒推送给前端）
type TrafficStats struct {
    UploadSpeed   int64 `json:"uploadSpeed"`              // 字节/秒
    DownloadSpeed int64 `json:"downloadSpeed"`            // 字节/秒
    TotalUpload   int64 `json:"totalUpload"`              // 本次连接累计上传字节
    TotalDownload int64 `json:"totalDownload"`            // 本次连接累计下载字节
    ActiveConns   int   `json:"activeConns"`              // 活跃连接数
}
```

---

## 5. Wails 绑定 API 清单 (Go → 前端方法契约)

所有暴露给前端的 Go 方法定义在 `app.go` 中，入参出参必须使用上述强类型 Struct。

### 5.1 连接控制
| 方法签名 | 说明 |
| :--- | :--- |
| `StartProxy() error` | 启动内核并根据 ProxyMode 设置系统代理/TUN |
| `StopProxy() error` | 停止内核、还原系统代理、清理子进程 |
| `GetConnectionStatus() ConnectionStatus` | 获取当前连接状态机状态 |

### 5.2 节点与订阅管理
| 方法签名 | 说明 |
| :--- | :--- |
| `GetNodes() []NodeConfig` | 获取全部节点列表 |
| `GetNodesByGroup(subId string) []NodeConfig` | 按分组获取节点 |
| `ImportNodes(links string) ([]NodeConfig, error)` | 从粘贴的多行分享链接批量导入 |
| `DeleteNode(id string) error` | 删除单个节点 |
| `SetActiveNode(nodeId string) error` | 设置当前活动节点 |
| `GetSubscriptions() []Subscription` | 获取全部订阅源 |
| `AddSubscription(sub Subscription) error` | 新增订阅 |
| `UpdateSubscription(subId string) error` | 拉取更新单个订阅 |
| `UpdateAllSubscriptions() error` | 拉取更新全部订阅 |
| `DeleteSubscription(subId string) error` | 删除订阅及其下属全部节点 |

### 5.3 延迟测速
| 方法签名 | 说明 |
| :--- | :--- |
| `TestNodeDelay(nodeId string) (int64, error)` | 测试单个节点 TCP 延迟 |
| `TestAllDelay() error` | 批量测试全部节点（结果通过事件推送） |
| `TestGroupDelay(subId string) error` | 测试指定分组全部节点 |

### 5.4 配置管理
| 方法签名 | 说明 |
| :--- | :--- |
| `GetConfig() AppConfig` | 获取当前配置 |
| `SaveConfig(cfg AppConfig) error` | 保存配置（原子写入） |
| `SetProxyMode(mode string) error` | 切换连接模式（连接中自动重载） |
| `SetRoutingMode(mode string) error` | 切换路由分流模式（连接中热重载） |

### 5.5 Wails 事件通道 (Go → 前端推送)
| 事件名 | Payload | 说明 |
| :--- | :--- | :--- |
| `traffic:stats` | `TrafficStats` | 每秒推送实时流量 |
| `connection:status` | `ConnectionStatus` | 连接状态变更 |
| `core:log` | `string` | 内核日志行实时推流 |
| `ping:result` | `{nodeId, tcpDelay, realDelay}` | 单个节点测速完成 |
| `subscription:updated` | `{subId, nodeCount, error}` | 订阅更新完成通知 |

---

## 6. 智能体开发守则与安全铁律

1. **开源内核直接集成，绝不重复造轮子**：
   - 严禁自行编写底层加密/解密传输内核，一律通过调度 Sing-box / Xray-core 二进制实现。
2. **零断网保护准则**：
   - 监听 `SIGINT`, `SIGTERM` 与退出事件，无论任何情况关闭程序必须无条件还原系统代理。
   - 启动时记录系统代理原始状态快照，退出时恢复。
3. **配置热加载与预检保障**：
   - 切换节点或路由模式时，新生成的配置必须先通过内核自带的语法校验测试命令，再平滑重载。
4. **原子文件写入**：
   - 所有配置/订阅/节点持久化一律先写 `.tmp` 临时文件后 `os.Rename` 原子替换，杜绝因崩溃或断电导致配置文件清空损坏。
5. **日志脱敏**：
   - 日志输出严禁打印用户 UUID、密码、订阅 URL 中的 Token。
6. **全量测试覆盖**：
   - 每个模块开发完成后，必须配套单元测试与集成冒烟测试，确保代码无死锁、无内存泄漏。
