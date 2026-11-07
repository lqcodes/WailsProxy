package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Manager 管理应用配置、节点和订阅的加载与原子持久化
type Manager struct {
	mu          sync.RWMutex
	dataDir     string
	configPath  string
	nodesPath   string
	subsPath    string
	appConfig   AppConfig
	nodes       []NodeConfig
	subs        []Subscription
}

// NewManager 初始化配置管理器并从磁盘加载数据（如不存在则创建默认配置）
func NewManager(dataDir string) (*Manager, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	m := &Manager{
		dataDir:    dataDir,
		configPath: filepath.Join(dataDir, "app_config.json"),
		nodesPath:  filepath.Join(dataDir, "nodes.json"),
		subsPath:   filepath.Join(dataDir, "subscriptions.json"),
		appConfig:  DefaultAppConfig(),
		nodes:      []NodeConfig{},
		subs:       []Subscription{},
	}

	if err := m.loadOrInit(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) loadOrInit() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var cfg AppConfig
	if err := ReadJSON(m.configPath, &cfg); err == nil {
		m.appConfig = normalizeConfig(cfg)
		_ = AtomicWriteJSON(m.configPath, m.appConfig)
	} else {
		m.appConfig = DefaultAppConfig()
		if err := AtomicWriteJSON(m.configPath, m.appConfig); err != nil {
			return err
		}
	}

	var subs []Subscription
	if err := ReadJSON(m.subsPath, &subs); err == nil {
		m.subs = subs
	}
	m.ensureDefaultGroupLocked()

	var nodes []NodeConfig
	if err := ReadJSON(m.nodesPath, &nodes); err == nil {
		for i := range nodes {
			if nodes[i].RawLink != "" && (nodes[i].Mldsa65Verify == "" || nodes[i].SpiderX == "") {
				if u, err := url.Parse(nodes[i].RawLink); err == nil {
					q := u.Query()
					if nodes[i].Mldsa65Verify == "" && q.Get("pqv") != "" {
						nodes[i].Mldsa65Verify = q.Get("pqv")
					}
					if nodes[i].SpiderX == "" && q.Get("spx") != "" {
						nodes[i].SpiderX = q.Get("spx")
					}
				}
			}
			// 过滤旧版本演示占位节点（避免真实连接误选到 1.1.1.1 演示节点）
			_ = strings.TrimSpace(nodes[i].Id)
		}
		m.nodes = nodes
	}

	m.recalculateSubCountsLocked()
	_ = AtomicWriteJSON(m.subsPath, m.subs)
	_ = AtomicWriteJSON(m.nodesPath, m.nodes)
	return nil
}

func normalizeConfig(cfg AppConfig) AppConfig {
	def := DefaultAppConfig()
	if cfg.SocksPort <= 0 || cfg.SocksPort > 65535 {
		cfg.SocksPort = def.SocksPort
	}
	if cfg.HttpPort <= 0 || cfg.HttpPort > 65535 {
		cfg.HttpPort = def.HttpPort
	}
	if cfg.SocksPort == cfg.HttpPort {
		if cfg.SocksPort < 65535 {
			cfg.HttpPort = cfg.SocksPort + 1
		} else {
			cfg.HttpPort = cfg.SocksPort - 1
		}
	}
	if cfg.ProxyMode != ProxyModeSystem && cfg.ProxyMode != ProxyModeManual && cfg.ProxyMode != ProxyModeTun {
		cfg.ProxyMode = def.ProxyMode
	}
	if cfg.RoutingMode != RoutingModeRule && cfg.RoutingMode != RoutingModeGlobal && cfg.RoutingMode != RoutingModeDirect {
		cfg.RoutingMode = def.RoutingMode
	}
	if cfg.DirectDNS == "" {
		cfg.DirectDNS = def.DirectDNS
	}
	if cfg.RemoteDNS == "" {
		cfg.RemoteDNS = def.RemoteDNS
	}
	if cfg.ThemeMode != "light" && cfg.ThemeMode != "dark" && cfg.ThemeMode != "auto" {
		cfg.ThemeMode = def.ThemeMode
	}
	if cfg.ThemeColor == "" {
		cfg.ThemeColor = def.ThemeColor
	}
	if cfg.LatencyTestUrl == "" {
		cfg.LatencyTestUrl = def.LatencyTestUrl
	}
	if cfg.LatencyTimeout <= 0 {
		cfg.LatencyTimeout = def.LatencyTimeout
	}
	if cfg.LatencyMaxConc <= 0 {
		cfg.LatencyMaxConc = def.LatencyMaxConc
	}
	if cfg.CoreType != "sing-box" && cfg.CoreType != "xray" {
		cfg.CoreType = def.CoreType
	}
	if cfg.ActiveGroupId == "" {
		cfg.ActiveGroupId = DefaultGroupID
	}
	if len(cfg.CustomDirectRules) == 0 && len(cfg.CustomProxyRules) == 0 && len(cfg.CustomBlockRules) == 0 {
		cfg.CustomDirectRules = DefaultDirectRules()
		cfg.CustomProxyRules = DefaultProxyRules()
		cfg.CustomBlockRules = DefaultBlockRules()
	} else {
		if cfg.CustomDirectRules == nil {
			cfg.CustomDirectRules = []string{}
		}
		if cfg.CustomProxyRules == nil {
			cfg.CustomProxyRules = []string{}
		}
		if cfg.CustomBlockRules == nil {
			cfg.CustomBlockRules = []string{}
		}
	}
	return cfg
}

func (m *Manager) ensureDefaultGroupLocked() {
	hasDefault := false
	for _, s := range m.subs {
		if s.Id == DefaultGroupID {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		m.subs = append([]Subscription{{
			Id:             DefaultGroupID,
			Name:           DefaultGroupName,
			Url:            "",
			AutoUpdate:     false,
			UpdateInterval: 0,
			LastUpdated:    0,
			NodeCount:      0,
		}}, m.subs...)
	}
}

func (m *Manager) recalculateSubCountsLocked() {
	counts := make(map[string]int)
	groupNames := make(map[string]string)
	for _, s := range m.subs {
		groupNames[s.Id] = s.Name
	}

	for i := range m.nodes {
		if m.nodes[i].SubId == "" {
			m.nodes[i].SubId = DefaultGroupID
		}
		if name, ok := groupNames[m.nodes[i].SubId]; ok {
			m.nodes[i].GroupName = name
		} else if m.nodes[i].GroupName == "" {
			m.nodes[i].GroupName = DefaultGroupName
		}
		counts[m.nodes[i].SubId]++
	}

	for i := range m.subs {
		m.subs[i].NodeCount = counts[m.subs[i].Id]
	}

	// Ensure ActiveNodeId points to a valid node if nodes exist
	if len(m.nodes) == 0 {
		m.appConfig.ActiveNodeId = ""
	} else {
		found := false
		for _, n := range m.nodes {
			if n.Id == m.appConfig.ActiveNodeId {
				found = true
				break
			}
		}
		if !found {
			m.appConfig.ActiveNodeId = m.nodes[0].Id
		}
	}
}

// DataDir 返回数据目录路径
func (m *Manager) DataDir() string {
	return m.dataDir
}

// GetConfig 获取当前配置的副本
func (m *Manager) GetConfig() AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg := m.appConfig
	cfg.CustomDirectRules = append([]string(nil), m.appConfig.CustomDirectRules...)
	cfg.CustomProxyRules = append([]string(nil), m.appConfig.CustomProxyRules...)
	cfg.CustomBlockRules = append([]string(nil), m.appConfig.CustomBlockRules...)
	return cfg
}

// SaveConfig 保存全局配置（原子写入）
func (m *Manager) SaveConfig(cfg AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.appConfig = normalizeConfig(cfg)
	return AtomicWriteJSON(m.configPath, m.appConfig)
}

// GetNodes 获取全部节点列表
func (m *Manager) GetNodes() []NodeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]NodeConfig, len(m.nodes))
	copy(out, m.nodes)
	return out
}

// GetNodesByGroup 按分组 ID 获取节点（subId 为空或 "all" 时返回全部）
func (m *Manager) GetNodesByGroup(subId string) []NodeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if subId == "" || subId == "all" {
		out := make([]NodeConfig, len(m.nodes))
		copy(out, m.nodes)
		return out
	}
	var out []NodeConfig
	for _, n := range m.nodes {
		if n.SubId == subId {
			out = append(out, n)
		}
	}
	if out == nil {
		return []NodeConfig{}
	}
	return out
}

// GetActiveNode 获取当前选中的活动节点
func (m *Manager) GetActiveNode() (*NodeConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.nodes) == 0 {
		return nil, fmt.Errorf("当前没有可用节点，请先导入节点或更新订阅")
	}
	for _, n := range m.nodes {
		if n.Id == m.appConfig.ActiveNodeId {
			nodeCopy := n
			return &nodeCopy, nil
		}
	}
	nodeCopy := m.nodes[0]
	return &nodeCopy, nil
}

// GetNodeByID 根据 ID 获取节点
func (m *Manager) GetNodeByID(id string) (*NodeConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, n := range m.nodes {
		if n.Id == id {
			nodeCopy := n
			return &nodeCopy, nil
		}
	}
	return nil, fmt.Errorf("未找到节点: %s", id)
}

// AddNodes 批量添加节点到指定分组并持久化
func (m *Manager) AddNodes(newNodes []NodeConfig, subId string) ([]NodeConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if subId == "" {
		subId = DefaultGroupID
	}

	groupName := DefaultGroupName
	for _, s := range m.subs {
		if s.Id == subId {
			groupName = s.Name
			break
		}
	}

	added := make([]NodeConfig, 0, len(newNodes))
	for _, n := range newNodes {
		if n.Id == "" {
			n.Id = uuid.NewString()
		}
		n.SubId = subId
		n.GroupName = groupName

		updatedExisting := false
		for i := range m.nodes {
			if m.nodes[i].SubId == subId &&
				m.nodes[i].Protocol == n.Protocol &&
				m.nodes[i].Server == n.Server &&
				m.nodes[i].Port == n.Port &&
				m.nodes[i].Name == n.Name {
				n.Id = m.nodes[i].Id
				m.nodes[i] = n
				updatedExisting = true
				break
			}
		}
		if !updatedExisting {
			m.nodes = append(m.nodes, n)
		}
		added = append(added, n)
	}

	m.recalculateSubCountsLocked()
	if err := AtomicWriteJSON(m.nodesPath, m.nodes); err != nil {
		return nil, err
	}
	if err := AtomicWriteJSON(m.subsPath, m.subs); err != nil {
		return nil, err
	}
	_ = AtomicWriteJSON(m.configPath, m.appConfig)
	return added, nil
}

// UpdateNode 更新单个节点的全部配置参数并原子持久化
func (m *Manager) UpdateNode(updated NodeConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if strings.TrimSpace(updated.Id) == "" {
		return fmt.Errorf("节点 ID 不能为空")
	}
	if strings.TrimSpace(updated.Server) == "" || updated.Port <= 0 || updated.Port > 65535 {
		return fmt.Errorf("服务器地址或端口不合法")
	}

	idx := -1
	for i := range m.nodes {
		if m.nodes[i].Id == updated.Id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("未找到要更新的节点: %s", updated.Id)
	}

	if updated.SubId == "" || updated.SubId == "all" {
		updated.SubId = m.nodes[idx].SubId
		if updated.SubId == "" {
			updated.SubId = DefaultGroupID
		}
	}

	groupName := DefaultGroupName
	for _, s := range m.subs {
		if s.Id == updated.SubId {
			groupName = s.Name
			break
		}
	}
	updated.GroupName = groupName
	if strings.TrimSpace(updated.Name) == "" {
		updated.Name = fmt.Sprintf("%s-%s:%d", strings.ToUpper(updated.Protocol), updated.Server, updated.Port)
	}

	m.nodes[idx] = updated
	m.recalculateSubCountsLocked()

	if err := AtomicWriteJSON(m.nodesPath, m.nodes); err != nil {
		return err
	}
	return AtomicWriteJSON(m.subsPath, m.subs)
}

// MoveNodesToGroup 将一个或多个节点移动到指定订阅分组并原子持久化
func (m *Manager) MoveNodesToGroup(nodeIds []string, targetSubId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(nodeIds) == 0 {
		return nil
	}
	if targetSubId == "" || targetSubId == "all" {
		targetSubId = DefaultGroupID
	}

	targetGroupName := DefaultGroupName
	foundGroup := targetSubId == DefaultGroupID
	for _, s := range m.subs {
		if s.Id == targetSubId {
			targetGroupName = s.Name
			foundGroup = true
			break
		}
	}
	if !foundGroup {
		return fmt.Errorf("未找到目标分组: %s", targetSubId)
	}

	idSet := make(map[string]bool, len(nodeIds))
	for _, id := range nodeIds {
		idSet[id] = true
	}

	moved := 0
	for i := range m.nodes {
		if idSet[m.nodes[i].Id] {
			m.nodes[i].SubId = targetSubId
			m.nodes[i].GroupName = targetGroupName
			moved++
		}
	}
	if moved == 0 {
		return fmt.Errorf("未找到要移动的节点")
	}

	m.recalculateSubCountsLocked()
	if err := AtomicWriteJSON(m.nodesPath, m.nodes); err != nil {
		return err
	}
	return AtomicWriteJSON(m.subsPath, m.subs)
}

// ReplaceGroupNodes 刷新某个订阅分组下的节点（增量合并更新已有同端点节点，并完整保留用户手动添加的节点，绝不丢失）
func (m *Manager) ReplaceGroupNodes(subId string, newNodes []NodeConfig, lastErr string, updatedAt int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	groupName := DefaultGroupName
	for i := range m.subs {
		if m.subs[i].Id == subId {
			groupName = m.subs[i].Name
			m.subs[i].LastError = lastErr
			if lastErr == "" && updatedAt > 0 {
				m.subs[i].LastUpdated = updatedAt
			}
			break
		}
	}

	// 同步该分组下所有现有节点的分组名称
	for i := range m.nodes {
		if m.nodes[i].SubId == subId {
			m.nodes[i].GroupName = groupName
		}
	}

	if lastErr == "" && len(newNodes) > 0 {
		for _, n := range newNodes {
			matchedIdx := -1
			for i := range m.nodes {
				if m.nodes[i].SubId != subId {
					continue
				}
				if (n.Id != "" && m.nodes[i].Id == n.Id) ||
					(m.nodes[i].Protocol == n.Protocol &&
						strings.EqualFold(m.nodes[i].Server, n.Server) &&
						m.nodes[i].Port == n.Port) {
					matchedIdx = i
					break
				}
			}

			if matchedIdx >= 0 {
				old := m.nodes[matchedIdx]
				n.Id = old.Id
				if strings.TrimSpace(n.Name) == "" {
					n.Name = old.Name
				}
				if n.TcpDelay == 0 {
					n.TcpDelay = old.TcpDelay
				}
				if n.RealDelay == 0 {
					n.RealDelay = old.RealDelay
				}
				n.SubId = subId
				n.GroupName = groupName
				m.nodes[matchedIdx] = n
			} else {
				if n.Id == "" {
					n.Id = uuid.NewString()
				}
				n.SubId = subId
				n.GroupName = groupName
				m.nodes = append(m.nodes, n)
			}
		}
	}

	m.recalculateSubCountsLocked()
	if err := AtomicWriteJSON(m.nodesPath, m.nodes); err != nil {
		return err
	}
	if err := AtomicWriteJSON(m.subsPath, m.subs); err != nil {
		return err
	}
	return AtomicWriteJSON(m.configPath, m.appConfig)
}

// DeleteNode 删除单个节点
func (m *Manager) DeleteNode(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, n := range m.nodes {
		if n.Id == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("节点不存在: %s", id)
	}

	m.nodes = append(m.nodes[:idx], m.nodes[idx+1:]...)
	m.recalculateSubCountsLocked()
	if err := AtomicWriteJSON(m.nodesPath, m.nodes); err != nil {
		return err
	}
	if err := AtomicWriteJSON(m.subsPath, m.subs); err != nil {
		return err
	}
	return AtomicWriteJSON(m.configPath, m.appConfig)
}

// DeleteInvalidNodes 一键清理超时/失效节点（TcpDelay == -1 或 RealDelay == -1）
func (m *Manager) DeleteInvalidNodes(subId string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	kept := make([]NodeConfig, 0, len(m.nodes))
	removed := 0
	for _, n := range m.nodes {
		matchGroup := subId == "" || subId == "all" || n.SubId == subId
		if matchGroup && (n.TcpDelay < 0 || n.RealDelay < 0) {
			removed++
			continue
		}
		kept = append(kept, n)
	}

	m.nodes = kept
	m.recalculateSubCountsLocked()
	if err := AtomicWriteJSON(m.nodesPath, m.nodes); err != nil {
		return 0, err
	}
	if err := AtomicWriteJSON(m.subsPath, m.subs); err != nil {
		return 0, err
	}
	_ = AtomicWriteJSON(m.configPath, m.appConfig)
	return removed, nil
}

// SortNodesByDelay 按延迟升序排序节点（未测或超时节点排在末尾）
func (m *Manager) SortNodesByDelay() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sort.SliceStable(m.nodes, func(i, j int) bool {
		di := effectiveDelay(m.nodes[i])
		dj := effectiveDelay(m.nodes[j])
		return di < dj
	})

	return AtomicWriteJSON(m.nodesPath, m.nodes)
}

func effectiveDelay(n NodeConfig) int64 {
	if n.RealDelay > 0 {
		return n.RealDelay
	}
	if n.TcpDelay > 0 {
		return n.TcpDelay
	}
	if n.TcpDelay == 0 && n.RealDelay == 0 {
		return 999998 // 未测试排在超时前
	}
	return 999999 // 超时(-1)排在最后
}

// UpdateNodeDelay 更新单个节点的延迟数据
func (m *Manager) UpdateNodeDelay(nodeId string, tcpDelay, realDelay int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.nodes {
		if m.nodes[i].Id == nodeId {
			if tcpDelay != 0 {
				m.nodes[i].TcpDelay = tcpDelay
			}
			if realDelay != 0 {
				m.nodes[i].RealDelay = realDelay
			}
			return AtomicWriteJSON(m.nodesPath, m.nodes)
		}
	}
	return fmt.Errorf("未找到节点: %s", nodeId)
}

// SetActiveNode 设置当前活动节点
func (m *Manager) SetActiveNode(nodeId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	for _, n := range m.nodes {
		if n.Id == nodeId {
			found = true
			m.appConfig.ActiveNodeId = nodeId
			if n.SubId != "" {
				m.appConfig.ActiveGroupId = n.SubId
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("节点不存在: %s", nodeId)
	}
	return AtomicWriteJSON(m.configPath, m.appConfig)
}

// GetSubscriptions 获取全部订阅分组
func (m *Manager) GetSubscriptions() []Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Subscription, len(m.subs))
	copy(out, m.subs)
	return out
}

// AddSubscription 新增或更新订阅源
func (m *Manager) AddSubscription(sub Subscription) (Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sub.Id == "" {
		sub.Id = uuid.NewString()
	}
	if sub.Name == "" {
		sub.Name = "未命名订阅"
	}

	updated := false
	for i := range m.subs {
		if m.subs[i].Id == sub.Id {
			m.subs[i].Name = sub.Name
			m.subs[i].Url = sub.Url
			m.subs[i].AutoUpdate = sub.AutoUpdate
			m.subs[i].UpdateInterval = sub.UpdateInterval
			sub = m.subs[i]
			updated = true
			break
		}
	}
	if !updated {
		m.subs = append(m.subs, sub)
	}

	m.recalculateSubCountsLocked()
	if err := AtomicWriteJSON(m.subsPath, m.subs); err != nil {
		return sub, err
	}
	return sub, nil
}

// DeleteSubscription 删除订阅及其下属全部节点（默认分组只清空节点不删除分组本身）
func (m *Manager) DeleteSubscription(subId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	keptNodes := make([]NodeConfig, 0, len(m.nodes))
	for _, n := range m.nodes {
		if n.SubId != subId {
			keptNodes = append(keptNodes, n)
		}
	}
	m.nodes = keptNodes

	if subId != DefaultGroupID {
		keptSubs := make([]Subscription, 0, len(m.subs))
		for _, s := range m.subs {
			if s.Id != subId {
				keptSubs = append(keptSubs, s)
			}
		}
		m.subs = keptSubs
	}

	m.ensureDefaultGroupLocked()
	m.recalculateSubCountsLocked()

	if err := AtomicWriteJSON(m.nodesPath, m.nodes); err != nil {
		return err
	}
	if err := AtomicWriteJSON(m.subsPath, m.subs); err != nil {
		return err
	}
	return AtomicWriteJSON(m.configPath, m.appConfig)
}
