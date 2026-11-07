package core

import (
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"wailsproxy/internal/config"
)

func getFreePort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	p, _ := strconv.Atoi(portStr)
	return p
}

func TestCoreManager_StartReloadStopLifecycle(t *testing.T) {
	binDir := t.TempDir()
	dataDir := t.TempDir()

	var trafficCount int32
	var logCount int32

	mgr := NewManager(binDir, dataDir, Callbacks{
		OnStatusChange: func(status config.ConnectionStatus) {},
		OnTraffic: func(stats config.TrafficStats) {
			atomic.AddInt32(&trafficCount, 1)
		},
		OnLog: func(line string) {
			atomic.AddInt32(&logCount, 1)
		},
	})

	cfg := config.DefaultAppConfig()
	cfg.SocksPort = getFreePort(t)
	cfg.HttpPort = getFreePort(t)
	cfg.ProxyMode = config.ProxyModeManual
	cfg.RoutingMode = config.RoutingModeRule

	node := config.NodeConfig{
		Id:       "test-node-1",
		Name:     "HK-VLESS",
		Protocol: "vless",
		Server:   "hk.example.com",
		Port:     443,
		Uuid:     "550e8400-e29b-41d4-a716-446655440000",
		Security: "tls",
		Tls:      true,
	}

	// 1. 启动代理
	err := mgr.Start(cfg, node)
	require.NoError(t, err)
	assert.Equal(t, config.StatusConnected, mgr.GetStatus())

	// 等待 1.1 秒验证流量采集推送
	time.Sleep(1150 * time.Millisecond)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&trafficCount), int32(1))

	// 2. 热重载切换至 Global 模式
	cfg.RoutingMode = config.RoutingModeGlobal
	err = mgr.Reload(cfg, node)
	require.NoError(t, err)
	assert.Equal(t, config.StatusConnected, mgr.GetStatus())

	// 3. 验证日志脱敏（不应包含明文 UUID）
	logs := mgr.GetLogs()
	assert.NotEmpty(t, logs)
	for _, l := range logs {
		assert.NotContains(t, l, "550e8400-e29b-41d4-a716-446655440000")
	}

	// 4. 停止代理
	err = mgr.Stop()
	require.NoError(t, err)
	assert.Equal(t, config.StatusDisconnected, mgr.GetStatus())
}
