---
name: wails-proxy-dev
description: Guides the step-by-step development, debugging, testing, and architecture verification for the WailsProxy (Go + Wails + AntD v2rayN-like proxy client) application. Use when implementing real-time traffic monitors, multi-theme switching, 3-mode proxy (System/Manual/TUN), custom DNS/ports, subscription parsing, or AntD UI bindings.
---

# WailsProxy 客户端开发技能指南

本技能（Skill）为开发和维护 `WailsProxy` 提供针对全套业务特性的端到端实现步骤、测试方案与排错指南。

---

## 1. 核心架构与模块路线图

```text
               ┌── [1. 开源内核集成管理] (Sing-box / Xray-core 二进制调度)
               ├── [2. 订阅分组与全协议解析] (vmess, vless, trojan, ss, hy2, clash/sing-box 订阅)
               ├── [3. 路由模式与配置生成] (全局 / 规则 / 直连，DNS 与端口自定义)
[核心服务层] ──┼── [4. 三模连接与主控开关] (连接/断开按钮, 系统代理 / 纯代理 / TUN)
               ├── [5. 批量测速与一键刷新] (TCP Ping + HTTP 204 延迟, 订阅拉取更新)
               ├── [6. 流量监控] (每秒 EventsEmit 推送, Canvas 波形图)
               ├── [7. 多主题系统] (AntD v5 ConfigProvider 动态 Token 切换)
               ├── [8. 托盘与桌面生命周期] (最小化到托盘, 系统代理安全还原, 开机自启)
               └── [9. 全量功能自动化测试] (单测 + 配置预检 + Mock 测速 + 集成冒烟)
```

---

## 2. 核心特性开发与执行指南

### 特性 1：开源内核集成与管理（严禁自研核心）
1. **内核选型**：默认 **Sing-box**（轻量、TUN 与 DoH 支持好），备选 **Xray-core**。
2. **二进制存储**：放置在 `bin/` 目录，按平台区分文件名（`sing-box` / `sing-box.exe`）。
3. **启动检测**：程序启动时检测 `bin/` 中的内核文件是否存在且具有执行权限，缺失时弹窗引导或触发自动下载。
4. **配置预检**：任何新配置生效前必须先调用 `sing-box check -c <file>` 或 `xray run -test -config <file>` 做语法校验。

---

### 特性 2：订阅分组管理与一键刷新
1. **分组数据结构 (`internal/subscription/`)**：
   - 每个 `Subscription` 对象代表一个分组，其 `Id` 即节点的 `SubId`。
   - 内置一个 `default` 分组用于存放用户手动导入的单条链接。
2. **刷新逻辑 (`RefreshSubscription(subId)` / `UpdateAllSubscriptions()`)**：
   - HTTP GET 拉取订阅 URL 内容。
   - 自动识别订阅格式：Base64 多行纯文本 → 逐行解析协议链接；YAML 含 `proxies:` 关键字 → Clash 格式转换；JSON 含 `outbounds` → Sing-box 格式转换。
   - 更新后保持用户已有备注/自定义名称，仅更新连接参数。
   - 通过 `subscription:updated` 事件通知前端刷新列表。
3. **前端交互**：
   - AntD `Tabs` 显示分组标签页，每个 Tab 标题后带节点数量 Badge。
   - 每个分组 Tab 工具栏包含：刷新节点、批量测速、按延迟排序、删除失效节点。

---

### 特性 3：双重延迟测试与结果排序
1. **TCP Ping (`internal/ping/tcp.go`)**：
   - Goroutine Worker Pool（最大并发由 `AppConfig.LatencyMaxConc` 控制，默认 20）。
   - `net.DialTimeout(tcp, server:port, timeout)` 测量 TCP 握手耗时。
   - 超时或连接失败标记 `TcpDelay = -1`。
2. **HTTP 真实延迟 (`internal/ping/http.go`)**：
   - 通过已启动的本地代理端口构造 `http.Client`（设置 Proxy），请求 `LatencyTestUrl`。
   - 测量从发送请求到收到响应头的总耗时（含 DNS + TLS + 代理路由链路）。
   - 超时标记 `RealDelay = -1`。
3. **实时推送**：每完成一个节点测速，通过 `ping:result` 事件实时推送到前端，前端即时更新对应行。
4. **前端排序**：表头支持点击 `TcpDelay` / `RealDelay` 列排序，一键按延迟升序排列。

---

### 特性 4：主控"连接 / 断开"按钮与连接状态机
1. **状态机 (`ConnectionStatus`)**：
   ```text
   Disconnected ──点击连接──→ Connecting ──内核启动成功──→ Connected
        ↑                                                    │
        └──────────── Disconnecting ←──点击断开/出错──────────┘
   ```
   - 出现错误时进入 `Error` 状态，前端展示错误详情，允许重试或查看日志。
2. **大按钮 UI (`ConnectButton.tsx`)**：
   - `Disconnected`：灰色/暗色，文案"连接"。
   - `Connecting`：主题色 + 旋转加载动画。
   - `Connected`：主题色 + 呼吸灯脉冲 CSS 动画，文案"已连接"。
   - `Disconnecting`：加载旋转，禁用点击。
   - `Error`：红色警告色 + 错误提示。
3. **连接流程**：
   1. 校验是否已选择活动节点，未选择时弹窗提示。
   2. 根据 `ActiveNodeId` + `RoutingMode` + DNS/端口配置生成内核配置文件。
   3. 调用内核 `check` 命令预检配置。
   4. 启动内核子进程，等待端口就绪（轮询 `net.Dial` 本地端口）。
   5. 根据 `ProxyMode` 决定是否设置系统代理或启动 TUN。
   6. 启动流量采集定时器。
   7. 状态切换至 `Connected`。

---

### 特性 5：全局模式与规则模式路由分流
1. **配置生成器 (`internal/generator/routing.go`)**：
   - **规则模式 (Rule)**：
     - `geosite:cn`, `geoip:cn`, `geoip:private` → direct
     - `geosite:category-ads-all` → block
     - 其余 → proxy
   - **全局模式 (Global)**：
     - `geoip:private` → direct
     - 其余全部 → proxy
   - **直连模式 (Direct)**：全部 → direct
2. **热重载**：
   - 用户在连接状态下切换路由模式时，后端重新生成配置 → 预检 → 平滑重启内核（先启新进程确认端口可用，再杀旧进程），无感切换。
3. **前端控件**：AntD `Segmented` 组件，三段可选：`规则` / `全局` / `直连`，放置于仪表盘顶部状态栏。

---

### 特性 6：实时上下行流量波形图
1. **Go 流量采集 (`internal/core/traffic.go`)**：
   - 启动 1 秒定时器 `time.NewTicker(time.Second)`。
   - 从内核 API / 日志解析当前上下行累计字节，差值计算本秒速率。
   - `runtime.EventsEmit(ctx, "traffic:stats", TrafficStats{...})`。
2. **前端 `TrafficChart.tsx`**：
   - 维护长度为 60 的 `ringBuffer`（最近 60 秒速率历史）。
   - 使用 Canvas 2D 或 SVG `<polyline>` 绘制平滑面积折线图。
   - 上传线使用主题色半透明，下载线使用互补色半透明，填充区域渐变。
3. **速率格式化**：
   - `formatSpeed(bytes)`: 自动显示为 `B/s`, `KB/s`, `MB/s`, `GB/s`。
   - `formatTraffic(bytes)`: 自动显示为 `B`, `KB`, `MB`, `GB`, `TB`。
4. **性能要求**：
   - `trafficStore` 使用 Zustand 独立 slice，`TrafficChart` 组件使用 `useStore(trafficStore, selector)` 精确订阅，严禁引起其他组件重渲染。
   - Canvas 刷新使用 `requestAnimationFrame` 节流。

---

### 特性 7：AntD v5 多主题动态切换
1. **主题 Store (`themeStore.ts`)**：
   ```typescript
   interface ThemeState {
     mode: 'light' | 'dark' | 'auto';       // 亮色 / 暗色 / 跟随系统
     color: 'blue' | 'green' | 'purple' | 'orange' | 'pink';
     isDark: boolean;                         // 计算属性：最终是否为暗色
   }
   ```
   - `auto` 模式通过 `window.matchMedia('(prefers-color-scheme: dark)')` 监听系统变化。
2. **颜色映射表**：
   ```typescript
   const COLOR_MAP = {
     blue:   '#1677ff',
     green:  '#52c41a',
     purple: '#722ed1',
     orange: '#fa8c16',
     pink:   '#eb2f96',
   };
   ```
3. **App.tsx 集成**：
   ```tsx
   <ConfigProvider
     theme={{
       algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
       token: { colorPrimary: COLOR_MAP[themeColor], borderRadius: 8 },
     }}
   >
   ```
4. **全局 CSS 变量**：为 Canvas 波形图与自定义组件注入 CSS 变量 `--primary-color`、`--bg-color`，确保非 AntD 元素也跟随主题。
5. **持久化**：切换后立即调用 `SaveConfig()` 保存到 Go 端 `app_config.json`。

---

### 特性 8：系统托盘与桌面生命周期
1. **托盘菜单 (`internal/tray/` + `app.go`)**：
   - 显示主窗口 / 隐藏主窗口
   - 代理模式切换（系统代理 / 纯代理 / TUN / 停用代理）→ 单选菜单
   - 路由模式切换（规则 / 全局 / 直连）→ 单选菜单
   - 当前节点名称（只读展示）
   - 复制终端代理环境变量
   - 彻底退出
2. **关闭窗口行为**：
   - `CloseToTray = true`（默认）：拦截窗口关闭事件，调用 `runtime.WindowHide(ctx)` 隐藏窗口。
   - `CloseToTray = false`：正常退出，触发完整关闭流程。
3. **安全退出链 (Shutdown Chain)**：
   ```text
   用户点击退出 / 收到 SIGINT/SIGTERM
     → 停止流量采集定时器
     → 停止内核子进程（发送 SIGTERM → 等待 3s → 强制 SIGKILL）
     → 恢复系统代理原始状态（从启动时备份的快照还原）
     → 清理临时配置文件
     → 退出程序
   ```
4. **开机自启**：
   - macOS：写入 `~/Library/LaunchAgents/com.wailsproxy.app.plist`。
   - Windows：写入 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`。
   - Linux：写入 `~/.config/autostart/wailsproxy.desktop`。

---

### 特性 9：全量功能自动化测试套件

1. **协议解析单测 (`internal/parser/*_test.go`)**：
   - 每种协议至少 3 个用例：标准链接、含特殊字符链接、非法/残缺链接（测试错误处理）。
   - 订阅格式测试：Base64 纯文本、Clash YAML（含多种协议混合）、Sing-box JSON。
   ```bash
   go test -v ./internal/parser/...
   ```

2. **配置生成测试 (`internal/generator/*_test.go`)**：
   - 枚举组合矩阵：{VMess, VLESS+Reality, SS, Trojan, Hy2} × {Rule, Global, Direct} × {有DoH, 无DoH}。
   - 生成配置后调用内核 `check` 命令验证语法合法性。
   ```bash
   go test -v ./internal/generator/...
   ```

3. **延迟测速测试 (`internal/ping/*_test.go`)**：
   - 启动本地 Mock TCP/HTTP Server，验证：正常延迟返回值合理、超时返回 -1、并发安全无 data race。
   ```bash
   go test -v -race ./internal/ping/...
   ```

4. **系统代理测试 (`internal/sysproxy/*_test.go`)**：
   - 设置代理 → 读回验证 → 清除代理 → 读回验证已还原。
   - 模拟异常退出场景，验证 defer / signal handler 中的还原逻辑。

5. **配置原子写入测试 (`internal/config/*_test.go`)**：
   - 验证并发写入不会产生空文件或损坏 JSON。
   - 验证 `.tmp` 文件在成功 Rename 后被清理。

6. **一键全量测试**：
   ```bash
   go test -v -race -cover -count=1 ./...
   ```

---

## 3. 常见排错与验证清单

| 异常现象 | 排查定位 | 解决方案 |
| :--- | :--- | :--- |
| **内核启动失败** | `bin/sing-box` 不存在或无执行权限 | `chmod +x bin/sing-box`，确认 CPU 架构与当前系统一致 |
| **配置预检报错** | 生成的 JSON 配置存在语法错误或字段缺失 | 查看 `sing-box check` 的具体报错行，检查 generator 对应字段映射 |
| **规则模式下国内网站打不开** | 缺少 GeoIP/GeoSite 规则集文件 | 检查 `data/` 目录规则集完整性，或触发自动下载更新 |
| **退出后浏览器无法上网** | 系统代理未被正确还原 | 检查 Shutdown Chain 日志；手动执行系统代理清除命令恢复网络 |
| **切换分组后节点表格为空** | `subId` 过滤条件与选中 Tab key 不匹配 | 检查 `nodeStore` 的分组过滤逻辑与 `Tabs` 组件 `activeKey` 绑定 |
| **测速全部超时 (-1)** | 并发数过高被防火墙拦截、或网络不通 | 降低 `LatencyMaxConc`，检查本地防火墙出站规则 |
| **暗色主题下部分区域颜色异常** | 自定义样式使用了硬编码 Hex 颜色 | 统一使用 AntD Design Token（`colorText`, `colorBgContainer` 等） |
| **配置文件变空/损坏** | 写入过程中程序崩溃 | 检查是否使用了原子写入（写 `.tmp` → `os.Rename`） |
| **TUN 模式启动失败** | 缺少管理员/root 权限 | 检查提权提示是否正确触发，macOS 需 root helper 或网络扩展权限 |
