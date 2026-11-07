package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"wailsproxy/internal/config"
	"wailsproxy/internal/core"
	"wailsproxy/internal/parser"
	"wailsproxy/internal/ping"
	"wailsproxy/internal/subscription"
	"wailsproxy/internal/sysproxy"
	"wailsproxy/internal/tray"
)

// App 结构体暴露给 Wails 前端的所有绑定方法
type App struct {
	ctx        context.Context
	mu         sync.RWMutex
	cfgMgr     *config.Manager
	coreMgr    *core.Manager
	sysProxy   *sysproxy.Manager
	forceQuit  bool
	isTestMode bool
}

// NewApp 创建默认的应用实例
func NewApp() *App {
	binDir, dataDir := resolveRuntimeDirs()
	app := NewAppWithDirs(binDir, dataDir, false)
	return app
}

func defaultOSBaseDir() string {
	if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
		return filepath.Join(cfgDir, "WailsProxy")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".wailsproxy")
	}
	return filepath.Join(os.TempDir(), "WailsProxy")
}

func resolveRuntimeDirs() (string, string) {
	cwd, err := os.Getwd()
	if err == nil && cwd != "/" && cwd != "" {
		if _, err := os.Stat(filepath.Join(cwd, "wails.json")); err == nil {
			binDir := filepath.Join(cwd, "bin")
			dataDir := filepath.Join(cwd, "data")
			_ = os.MkdirAll(binDir, 0755)
			_ = os.MkdirAll(dataDir, 0755)
			return binDir, dataDir
		}
		if _, err := os.Stat(filepath.Join(cwd, "bin")); err == nil {
			binDir := filepath.Join(cwd, "bin")
			dataDir := filepath.Join(cwd, "data")
			_ = os.MkdirAll(binDir, 0755)
			_ = os.MkdirAll(dataDir, 0755)
			return binDir, dataDir
		}
	}

	// 根据不同操作系统 (macOS / Windows / Linux) 自动定位并创建默认用户数据与内核托管目录
	base := defaultOSBaseDir()
	binDir := filepath.Join(base, "bin")
	dataDir := filepath.Join(base, "data")
	_ = os.MkdirAll(binDir, 0755)
	_ = os.MkdirAll(dataDir, 0755)
	return binDir, dataDir
}

// NewAppWithDirs 使用指定目录创建 App 实例（便于单元测试与端到端集成测试）
func NewAppWithDirs(binDir, dataDir string, isTestMode bool) *App {
	cfgMgr, err := config.NewManager(dataDir)
	if err != nil {
		panic(fmt.Sprintf("failed to init config manager: %v", err))
	}

	var sysMgr *sysproxy.Manager
	if isTestMode {
		sysMgr = sysproxy.NewManagerWithRunner("noop", nil)
	} else {
		sysMgr = sysproxy.NewManager()
	}

	app := &App{
		cfgMgr:     cfgMgr,
		sysProxy:   sysMgr,
		isTestMode: isTestMode,
	}

	app.coreMgr = core.NewManager(binDir, dataDir, core.Callbacks{
		OnStatusChange: func(status config.ConnectionStatus) {
			app.emitEvent("connection:status", status)
		},
		OnTraffic: func(stats config.TrafficStats) {
			app.emitEvent("traffic:stats", stats)
		},
		OnLog: func(line string) {
			app.emitEvent("core:log", line)
		},
	})

	return app
}

func (a *App) seedDemoNodesIfEmpty() {
	if len(a.cfgMgr.GetNodes()) > 0 {
		return
	}
	sub, err := a.cfgMgr.AddSubscription(config.Subscription{
		Id:             "sub-iepl-vip",
		Name:           "亚太专线订阅",
		Url:            "https://sub.wailsproxy.local/api/v1/client/subscribe",
		AutoUpdate:     true,
		UpdateInterval: 12,
		LastUpdated:    time.Now().Unix(),
	})
	if err != nil {
		return
	}

	demoNodes := []config.NodeConfig{
		{
			Id:          "node-hk-01",
			SubId:       sub.Id,
			GroupName:   sub.Name,
			Name:        "🇭🇰 香港 IEPL 01 [VLESS Reality]",
			Protocol:    "vless",
			Server:      "1.1.1.1",
			Port:        443,
			Uuid:        "550e8400-e29b-41d4-a716-446655440000",
			Network:     "tcp",
			Tls:         true,
			Sni:         "www.cloudflare.com",
			Fingerprint: "chrome",
			Security:    "reality",
			PublicKey:   "mB7_demo_public_key_001",
			ShortId:     "a1b2c3d4",
			Flow:        "xtls-rprx-vision",
			TcpDelay:    28,
			RealDelay:   49,
			RawLink:     "vless://550e8400-e29b-41d4-a716-446655440000@1.1.1.1:443?security=reality&sni=www.cloudflare.com&fp=chrome&pbk=mB7_demo_public_key_001&sid=a1b2c3d4&flow=xtls-rprx-vision&type=tcp#%F0%9F%87%AD%F0%9F%87%B0%20%E9%A6%99%E6%B8%AF%20IEPL%2001",
		},
		{
			Id:          "node-jp-01",
			SubId:       sub.Id,
			GroupName:   sub.Name,
			Name:        "🇯🇵 东京 BGP 01 [VMess WS]",
			Protocol:    "vmess",
			Server:      "1.0.0.1",
			Port:        443,
			Uuid:        "23ad6b10-8d1a-40f7-8ad0-e3e35cd38297",
			Cipher:      "auto",
			Network:     "ws",
			Path:        "/vmess-ws",
			Host:        "jp.cdn.example.com",
			Tls:         true,
			Sni:         "jp.cdn.example.com",
			Fingerprint: "chrome",
			Security:    "tls",
			TcpDelay:    46,
			RealDelay:   78,
		},
		{
			Id:           "node-sg-01",
			SubId:        sub.Id,
			GroupName:    sub.Name,
			Name:         "🇸🇬 新加坡极速 [Hysteria2]",
			Protocol:     "hysteria2",
			Server:       "8.8.8.8",
			Port:         443,
			Password:     "demo_hy2_auth",
			Network:      "quic",
			Tls:          true,
			Sni:          "sg.hy2.example.com",
			Security:     "tls",
			ObfsType:     "salamander",
			ObfsPassword: "obfs_password",
			UpMbps:       100,
			DownMbps:     500,
			TcpDelay:     58,
			RealDelay:    92,
		},
		{
			Id:        "node-us-01",
			SubId:     config.DefaultGroupID,
			GroupName: config.DefaultGroupName,
			Name:      "🇺🇸 洛杉矶 CN2 GIA [Trojan]",
			Protocol:  "trojan",
			Server:    "8.8.4.4",
			Port:      443,
			Password:  "demo_trojan_pass",
			Network:   "grpc",
			ServiceName: "trojan-grpc",
			Tls:       true,
			Sni:       "us.trojan.example.com",
			Security:  "tls",
			TcpDelay:  138,
			RealDelay: 195,
		},
	}

	_, _ = a.cfgMgr.AddNodes(demoNodes[:3], sub.Id)
	_, _ = a.cfgMgr.AddNodes(demoNodes[3:], config.DefaultGroupID)
	_ = a.cfgMgr.SetActiveNode("node-hk-01")
}

// startup 在 Wails 启动时调用，初始化信号监听与系统代理快照备份
func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()

	if !a.isTestMode && ctx != nil {
		wailsRuntime.WindowSetMinSize(ctx, 1020, 680)
	}

	a.sysProxy.BackupSnapshot()

	// 监听 SIGINT / SIGTERM 信号，保障 Ctrl+C 或终止信号时立即还原系统代理并退出，绝不卡死
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		a.mu.Lock()
		a.forceQuit = true
		a.mu.Unlock()
		a.ShutdownCleanup()
		os.Exit(0)
	}()

	// 后台定时自动更新订阅（针对开启了 AutoUpdate 且到期需要同步的订阅源）
	if !a.isTestMode && ctx != nil {
		go a.startAutoUpdateLoop(ctx)
	}
}

func (a *App) startAutoUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().Unix()
			for _, sub := range a.cfgMgr.GetSubscriptions() {
				if !sub.AutoUpdate || sub.UpdateInterval <= 0 || strings.TrimSpace(sub.Url) == "" {
					continue
				}
				intervalSec := int64(sub.UpdateInterval) * 3600
				if now-sub.LastUpdated >= intervalSec {
					_ = a.UpdateSubscription(sub.Id)
				}
			}
		}
	}
}

// beforeClose 拦截窗口关闭事件，当 CloseToTray=true 且非主动退出时隐藏到系统托盘
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	a.mu.RLock()
	force := a.forceQuit
	a.mu.RUnlock()
	if force {
		return false
	}

	cfg := a.cfgMgr.GetConfig()
	if cfg.CloseToTray && ctx != nil {
		wailsRuntime.WindowHide(ctx)
		return true
	}
	return false
}

// shutdown 在 Wails 退出时执行 5 步安全退出链
func (a *App) shutdown(ctx context.Context) {
	a.ShutdownCleanup()
}

// ShutdownCleanup 无条件停止内核并还原系统代理（零断网保护）
func (a *App) ShutdownCleanup() {
	_ = a.coreMgr.Stop()
	_ = a.sysProxy.Disable()
}

func (a *App) emitEvent(eventName string, data interface{}) {
	a.mu.RLock()
	ctx := a.ctx
	isTest := a.isTestMode
	a.mu.RUnlock()

	if ctx != nil && !isTest {
		wailsRuntime.EventsEmit(ctx, eventName, data)
	}
}

// ==================== 5.1 连接控制 ====================

// StartProxy 启动内核并根据 ProxyMode 设置系统代理/TUN
func (a *App) StartProxy() error {
	cfg := a.cfgMgr.GetConfig()
	node, err := a.cfgMgr.GetActiveNode()
	if err != nil {
		return err
	}

	if err := a.coreMgr.Start(cfg, *node); err != nil {
		return err
	}

	// 根据连接模式处理系统代理（System 模式或非 root 下的 TUN 模式均开启系统全局代理，确保浏览器外网立即生效）
	if cfg.ProxyMode == config.ProxyModeSystem || (cfg.ProxyMode == config.ProxyModeTun && os.Geteuid() != 0) {
		if err := a.sysProxy.Enable("127.0.0.1", cfg.HttpPort, cfg.SocksPort); err != nil {
			a.coreMgr.AppendLog(fmt.Sprintf("WARN 设置系统代理遇到告警: %v", err))
		} else {
			a.coreMgr.AppendLog(fmt.Sprintf("INFO 已自动开启系统全局代理 (HTTP/HTTPS=127.0.0.1:%d, SOCKS5=127.0.0.1:%d)", cfg.HttpPort, cfg.SocksPort))
		}
	} else {
		_ = a.sysProxy.Disable()
	}

	return nil
}

// StopProxy 停止内核、还原系统代理、清理子进程
func (a *App) StopProxy() error {
	_ = a.sysProxy.Disable()
	return a.coreMgr.Stop()
}

// GetConnectionStatus 获取当前连接状态机状态
func (a *App) GetConnectionStatus() config.ConnectionStatus {
	return a.coreMgr.GetStatus()
}

// ==================== 5.2 节点与订阅管理 ====================

// GetNodes 获取全部节点列表
func (a *App) GetNodes() []config.NodeConfig {
	return a.cfgMgr.GetNodes()
}

// GetNodesByGroup 按分组获取节点
func (a *App) GetNodesByGroup(subId string) []config.NodeConfig {
	return a.cfgMgr.GetNodesByGroup(subId)
}

// ImportNodes 从粘贴的多行分享链接批量导入到当前或默认分组
func (a *App) ImportNodes(links string) ([]config.NodeConfig, error) {
	cfg := a.cfgMgr.GetConfig()
	targetSubId := cfg.ActiveGroupId
	if targetSubId == "" || targetSubId == "all" {
		targetSubId = config.DefaultGroupID
	}
	return a.ImportNodesToGroup(links, targetSubId)
}

// ImportNodesToGroup 从分享链接、远程订阅 URL (http/https) 或订阅文本批量导入到指定分组
func (a *App) ImportNodesToGroup(links string, subId string) ([]config.NodeConfig, error) {
	if subId == "" || subId == "all" {
		subId = config.DefaultGroupID
	}

	trimmed := strings.TrimSpace(links)
	if trimmed == "" {
		return nil, fmt.Errorf("输入内容为空")
	}

	// 检测是否包含 http:// 或 https:// 远程订阅地址（支持直接在导入弹窗粘贴订阅链接）
	lines := strings.Split(strings.ReplaceAll(trimmed, "\r\n", "\n"), "\n")
	var httpURLs []string
	var nonHTTPLines []string
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		lower := strings.ToLower(l)
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
			httpURLs = append(httpURLs, l)
		} else {
			nonHTTPLines = append(nonHTTPLines, l)
		}
	}

	var allParsed []config.NodeConfig
	var lastErr error

	for _, subURL := range httpURLs {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		fetched, err := subscription.FetchSubscription(ctx, subURL, 20*time.Second)
		cancel()
		if err != nil {
			lastErr = err
			a.coreMgr.AppendLog(fmt.Sprintf("ERROR 从订阅链接拉取失败: %v", err), subURL)
			continue
		}
		allParsed = append(allParsed, fetched...)

		// 若目标分组尚未绑定订阅 URL，则自动将该 URL 绑定到当前目标分组，确保节点直接进入用户当前选中的分组且支持后续一键刷新
		for _, s := range a.cfgMgr.GetSubscriptions() {
			if s.Id == subId && strings.TrimSpace(s.Url) == "" {
				s.Url = subURL
				s.LastUpdated = time.Now().Unix()
				_, _ = a.cfgMgr.AddSubscription(s)
				break
			}
		}
	}

	if len(nonHTTPLines) > 0 {
		remaining := strings.Join(nonHTTPLines, "\n")
		parsedNodes, err := subscription.ParseContent(remaining)
		if err != nil {
			parsedNodes, err = parser.ParseMultipleLinks(remaining)
		}
		if err != nil {
			lastErr = err
		} else {
			allParsed = append(allParsed, parsedNodes...)
		}
	}

	if len(allParsed) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("未识别到有效的代理节点或订阅内容")
	}

	added, err := a.cfgMgr.AddNodes(allParsed, subId)
	if err != nil {
		return nil, err
	}
	if len(added) == 1 {
		_ = a.cfgMgr.SetActiveNode(added[0].Id)
	}
	a.coreMgr.AppendLog(fmt.Sprintf("INFO 成功导入 %d 个节点到分组 [%s]", len(added), subId))
	return added, nil
}

// DeleteNode 删除单个节点
func (a *App) DeleteNode(id string) error {
	return a.cfgMgr.DeleteNode(id)
}

// UpdateNode 更新单个节点的完整配置（若当前正在连接该节点则自动热重载）
func (a *App) UpdateNode(node config.NodeConfig) error {
	if err := a.cfgMgr.UpdateNode(node); err != nil {
		return err
	}
	a.coreMgr.AppendLog(fmt.Sprintf("INFO 已更新节点配置: %s (%s %s:%d)", node.Name, strings.ToUpper(node.Protocol), node.Server, node.Port))

	if a.coreMgr.GetStatus() == config.StatusConnected {
		cfg := a.cfgMgr.GetConfig()
		if cfg.ActiveNodeId == node.Id {
			if updatedNode, err := a.cfgMgr.GetNodeByID(node.Id); err == nil {
				return a.coreMgr.Reload(cfg, *updatedNode)
			}
		}
	}
	return nil
}

// DeleteInvalidNodes 清理指定分组或全部的超时失效节点
func (a *App) DeleteInvalidNodes(subId string) (int, error) {
	return a.cfgMgr.DeleteInvalidNodes(subId)
}

// SortNodesByDelay 一键按延迟升序排序节点
func (a *App) SortNodesByDelay() error {
	return a.cfgMgr.SortNodesByDelay()
}

// SetActiveNode 设置当前活动节点（若当前处于已连接状态，则自动平滑切换到新节点）
func (a *App) SetActiveNode(nodeId string) error {
	if err := a.cfgMgr.SetActiveNode(nodeId); err != nil {
		return err
	}

	if a.coreMgr.GetStatus() == config.StatusConnected {
		cfg := a.cfgMgr.GetConfig()
		node, err := a.cfgMgr.GetActiveNode()
		if err == nil {
			return a.coreMgr.Reload(cfg, *node)
		}
	}
	return nil
}

// GetSubscriptions 获取全部订阅源
func (a *App) GetSubscriptions() []config.Subscription {
	return a.cfgMgr.GetSubscriptions()
}

// AddSubscription 新增或更新订阅
func (a *App) AddSubscription(sub config.Subscription) error {
	if strings.TrimSpace(sub.Name) == "" && strings.TrimSpace(sub.Url) != "" {
		rawURL := strings.TrimSpace(sub.Url)
		if idx := strings.Index(rawURL, "://"); idx > 0 {
			rest := rawURL[idx+3:]
			if slash := strings.Index(rest, "/"); slash > 0 {
				rest = rest[:slash]
			}
			if colon := strings.Index(rest, ":"); colon > 0 {
				rest = rest[:colon]
			}
			if rest != "" {
				sub.Name = "订阅 - " + rest
			}
		}
	}
	saved, err := a.cfgMgr.AddSubscription(sub)
	if err != nil {
		return err
	}
	// 如果填了 URL，立即拉取订阅节点
	if strings.TrimSpace(saved.Url) != "" && !strings.Contains(saved.Url, "wailsproxy.local") && !a.isTestMode {
		return a.UpdateSubscription(saved.Id)
	}
	return nil
}

// UpdateSubscription 拉取更新单个订阅
func (a *App) UpdateSubscription(subId string) error {
	subs := a.cfgMgr.GetSubscriptions()
	var target *config.Subscription
	for _, s := range subs {
		if s.Id == subId {
			cp := s
			target = &cp
			break
		}
	}
	if target == nil {
		return fmt.Errorf("未找到订阅分组: %s", subId)
	}

	// 纯本地无 URL 分组或本地演示订阅支持直接刷新状态并通知完成
	if strings.TrimSpace(target.Url) == "" || strings.Contains(target.Url, "wailsproxy.local") {
		existing := a.cfgMgr.GetNodesByGroup(target.Id)
		_ = a.cfgMgr.ReplaceGroupNodes(target.Id, existing, "", time.Now().Unix())
		ev := config.SubscriptionUpdatedEvent{
			SubId:     target.Id,
			NodeCount: len(existing),
		}
		a.emitEvent("subscription:updated", ev)
		a.coreMgr.AppendLog(fmt.Sprintf("INFO 分组 [%s] 节点已刷新 (共 %d 个节点)", target.Name, len(existing)))
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	nodes, err := subscription.FetchSubscription(ctx, target.Url, 15*time.Second)
	if err != nil {
		_ = a.cfgMgr.ReplaceGroupNodes(target.Id, nil, err.Error(), 0)
		a.emitEvent("subscription:updated", config.SubscriptionUpdatedEvent{
			SubId:     target.Id,
			NodeCount: target.NodeCount,
			Error:     err.Error(),
		})
		a.coreMgr.AppendLog(fmt.Sprintf("ERROR 订阅 [%s] 更新失败: %v", target.Name, err), target.Url)
		return err
	}

	if err := a.cfgMgr.ReplaceGroupNodes(target.Id, nodes, "", time.Now().Unix()); err != nil {
		return err
	}

	totalGroupNodes := len(a.cfgMgr.GetNodesByGroup(target.Id))
	ev := config.SubscriptionUpdatedEvent{
		SubId:     target.Id,
		NodeCount: totalGroupNodes,
	}
	a.emitEvent("subscription:updated", ev)
	a.coreMgr.AppendLog(fmt.Sprintf("INFO 订阅 [%s] 刷新成功，同步 %d 个订阅节点 (当前分组共 %d 个节点)", target.Name, len(nodes), totalGroupNodes), target.Url)
	return nil
}

// UpdateAllSubscriptions 拉取更新全部订阅
func (a *App) UpdateAllSubscriptions() error {
	subs := a.cfgMgr.GetSubscriptions()
	var lastErr error
	for _, s := range subs {
		if err := a.UpdateSubscription(s.Id); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// DeleteSubscription 删除订阅及其下属全部节点
func (a *App) DeleteSubscription(subId string) error {
	return a.cfgMgr.DeleteSubscription(subId)
}

// MoveNodesToGroup 将一个或多个节点移动到另一个订阅分组
func (a *App) MoveNodesToGroup(nodeIds []string, targetSubId string) error {
	return a.cfgMgr.MoveNodesToGroup(nodeIds, targetSubId)
}

// ==================== 5.3 延迟测速 ====================

// TestNodeDelay 测试单个节点 TCP 延迟与真实延迟
func (a *App) TestNodeDelay(nodeId string) (int64, error) {
	node, err := a.cfgMgr.GetNodeByID(nodeId)
	if err != nil {
		return -1, err
	}
	cfg := a.cfgMgr.GetConfig()
	timeout := time.Duration(cfg.LatencyTimeout) * time.Millisecond

	tcpMs := ping.TCPPing(node.Server, node.Port, timeout)
	var realMs int64 = -1
	if tcpMs > 0 {
		if a.coreMgr.GetStatus() == config.StatusConnected && cfg.ActiveNodeId == node.Id {
			localProxy := fmt.Sprintf("http://127.0.0.1:%d", cfg.HttpPort)
			realMs = ping.HTTPDelay(localProxy, cfg.LatencyTestUrl, timeout)
		} else {
			realMs = ping.MeasureNodeRealDelay(*node, a.coreMgr.GetBinDir(), cfg.LatencyTestUrl, timeout)
		}
	}

	_ = a.cfgMgr.UpdateNodeDelay(node.Id, tcpMs, realMs)
	a.emitEvent("ping:result", config.PingResultEvent{
		NodeId:    node.Id,
		TcpDelay:  tcpMs,
		RealDelay: realMs,
	})
	return tcpMs, nil
}

// TestAllDelay 批量测试全部节点（结果通过 ping:result 事件推送）
func (a *App) TestAllDelay() error {
	return a.TestGroupDelay("all")
}

// TestGroupDelay 测试指定分组全部节点
func (a *App) TestGroupDelay(subId string) error {
	nodes := a.cfgMgr.GetNodesByGroup(subId)
	if len(nodes) == 0 {
		return nil
	}

	cfg := a.cfgMgr.GetConfig()
	timeout := time.Duration(cfg.LatencyTimeout) * time.Millisecond
	var localProxy string
	if a.coreMgr.GetStatus() == config.StatusConnected {
		localProxy = fmt.Sprintf("http://127.0.0.1:%d", cfg.HttpPort)
	}

	a.coreMgr.AppendLog(fmt.Sprintf("INFO 开始并发测试 %d 个节点的延迟 (最大并发=%d)...", len(nodes), cfg.LatencyMaxConc))

	ping.BatchTest(
		context.Background(),
		nodes,
		cfg.LatencyMaxConc,
		timeout,
		cfg.LatencyTestUrl,
		cfg.ActiveNodeId,
		localProxy,
		func(res config.PingResultEvent) {
			_ = a.cfgMgr.UpdateNodeDelay(res.NodeId, res.TcpDelay, res.RealDelay)
			a.emitEvent("ping:result", res)
		},
		a.coreMgr.GetBinDir(),
	)

	a.coreMgr.AppendLog("INFO 批量节点延迟测试完成")
	return nil
}

// ==================== 5.4 配置管理 ====================

// GetConfig 获取当前配置
func (a *App) GetConfig() config.AppConfig {
	return a.cfgMgr.GetConfig()
}

// SaveConfig 保存配置（原子写入），若连接中且核心参数变化则自动重载
func (a *App) SaveConfig(cfg config.AppConfig) error {
	oldCfg := a.cfgMgr.GetConfig()
	if err := a.cfgMgr.SaveConfig(cfg); err != nil {
		return err
	}

	if oldCfg.AutoStart != cfg.AutoStart && !a.isTestMode {
		_ = tray.ConfigureAutoStart(cfg.AutoStart, "")
	}

	newCfg := a.cfgMgr.GetConfig()
	if a.coreMgr.GetStatus() == config.StatusConnected {
		needReload := oldCfg.SocksPort != newCfg.SocksPort ||
			oldCfg.HttpPort != newCfg.HttpPort ||
			oldCfg.AllowLan != newCfg.AllowLan ||
			oldCfg.ProxyMode != newCfg.ProxyMode ||
			oldCfg.RoutingMode != newCfg.RoutingMode ||
			oldCfg.DirectDNS != newCfg.DirectDNS ||
			oldCfg.RemoteDNS != newCfg.RemoteDNS ||
			oldCfg.EnableDoH != newCfg.EnableDoH ||
			oldCfg.CoreType != newCfg.CoreType ||
			strings.Join(oldCfg.CustomDirectRules, "\n") != strings.Join(newCfg.CustomDirectRules, "\n") ||
			strings.Join(oldCfg.CustomProxyRules, "\n") != strings.Join(newCfg.CustomProxyRules, "\n") ||
			strings.Join(oldCfg.CustomBlockRules, "\n") != strings.Join(newCfg.CustomBlockRules, "\n")

		if needReload {
			if node, err := a.cfgMgr.GetActiveNode(); err == nil {
				if err := a.coreMgr.Reload(newCfg, *node); err != nil {
					return err
				}
			}
			if newCfg.ProxyMode == config.ProxyModeSystem || (newCfg.ProxyMode == config.ProxyModeTun && os.Geteuid() != 0) {
				_ = a.sysProxy.Enable("127.0.0.1", newCfg.HttpPort, newCfg.SocksPort)
			} else {
				_ = a.sysProxy.Disable()
			}
		}
	}
	return nil
}

// SetProxyMode 切换连接模式（system | manual | tun，连接中自动重载）
func (a *App) SetProxyMode(mode string) error {
	if mode != config.ProxyModeSystem && mode != config.ProxyModeManual && mode != config.ProxyModeTun {
		return fmt.Errorf("不支持的连接模式: %s", mode)
	}
	cfg := a.cfgMgr.GetConfig()
	cfg.ProxyMode = mode
	return a.SaveConfig(cfg)
}

// SetRoutingMode 切换路由分流模式（rule | global | direct，连接中热重载）
func (a *App) SetRoutingMode(mode string) error {
	if mode != config.RoutingModeRule && mode != config.RoutingModeGlobal && mode != config.RoutingModeDirect {
		return fmt.Errorf("不支持的路由模式: %s", mode)
	}
	cfg := a.cfgMgr.GetConfig()
	cfg.RoutingMode = mode
	return a.SaveConfig(cfg)
}

// ==================== 辅助方法 ====================

// GetLogs 获取当前内核日志历史
func (a *App) GetLogs() []string {
	return a.coreMgr.GetLogs()
}

// ClearLogs 清空内核日志
func (a *App) ClearLogs() {
	a.coreMgr.ClearLogs()
}

// GetTerminalProxyCommand 获取当前端口对应的终端代理环境变量导出命令
func (a *App) GetTerminalProxyCommand(shell string) string {
	cfg := a.cfgMgr.GetConfig()
	return tray.FormatTerminalProxyCmd("127.0.0.1", cfg.HttpPort, cfg.SocksPort, shell)
}

// CheckDependencies 启动时后台检测系统与开源代理内核依赖项状态
func (a *App) CheckDependencies() config.DependencyCheckResult {
	return a.coreMgr.CheckDependencies()
}

// InstallDependencies 一键安装/修复缺失的代理内核与分流规则库
func (a *App) InstallDependencies() (config.DependencyCheckResult, error) {
	return a.coreMgr.InstallDependencies()
}
