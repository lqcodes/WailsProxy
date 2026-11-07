package main

import (
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"wailsproxy/internal/config"
)

func getTestFreePort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	p, _ := strconv.Atoi(portStr)
	return p
}

func TestApp_EndToEndIntegration(t *testing.T) {
	binDir := t.TempDir()
	dataDir := t.TempDir()
	app := NewAppWithDirs(binDir, dataDir, true)
	defer app.ShutdownCleanup()

	// 1. 初始状态应为 Disconnected
	assert.Equal(t, config.StatusDisconnected, app.GetConnectionStatus())

	// 2. 配置免费随机端口
	cfg := app.GetConfig()
	cfg.SocksPort = getTestFreePort(t)
	cfg.HttpPort = getTestFreePort(t)
	cfg.ProxyMode = config.ProxyModeManual
	cfg.RoutingMode = config.RoutingModeRule
	require.NoError(t, app.SaveConfig(cfg))

	// 3. 启动本地 TCP 节点模拟服务器供测速与连接验证
	mockNodeLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer mockNodeLn.Close()
	go func() {
		for {
			conn, err := mockNodeLn.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	nodeHost, nodePortStr, _ := net.SplitHostPort(mockNodeLn.Addr().String())
	nodePort, _ := strconv.Atoi(nodePortStr)

	// 4. 批量导入多行分享链接
	rawLinks := "vless://550e8400-e29b-41d4-a716-446655440000@" + nodeHost + ":" + strconv.Itoa(nodePort) + "?security=tls&type=tcp#Local-VLESS\n" +
		"trojan://secret123@" + nodeHost + ":" + strconv.Itoa(nodePort) + "?sni=example.com#Local-Trojan"
	imported, err := app.ImportNodes(rawLinks)
	require.NoError(t, err)
	require.Len(t, imported, 2)

	allNodes := app.GetNodes()
	require.Len(t, allNodes, 2)

	// 5. 设置活动节点并启动连接
	require.NoError(t, app.SetActiveNode(imported[0].Id))
	require.NoError(t, app.StartProxy())
	assert.Equal(t, config.StatusConnected, app.GetConnectionStatus())

	// 6. 在连接中切换连接模式 (system / tun / manual) 与路由模式 (rule / global / direct)
	require.NoError(t, app.SetProxyMode(config.ProxyModeSystem))
	assert.Equal(t, config.ProxyModeSystem, app.GetConfig().ProxyMode)
	assert.Equal(t, config.StatusConnected, app.GetConnectionStatus())

	require.NoError(t, app.SetRoutingMode(config.RoutingModeGlobal))
	assert.Equal(t, config.RoutingModeGlobal, app.GetConfig().RoutingMode)
	assert.Equal(t, config.StatusConnected, app.GetConnectionStatus())

	// 7. 测试单个与批量节点延迟
	delay, err := app.TestNodeDelay(imported[0].Id)
	require.NoError(t, err)
	assert.Greater(t, delay, int64(0))

	require.NoError(t, app.TestAllDelay())
	require.NoError(t, app.SortNodesByDelay())

	// 8. 测试新增订阅与远程拉取更新
	subContent := base64.StdEncoding.EncodeToString([]byte(
		"hy2://pass@" + nodeHost + ":" + strconv.Itoa(nodePort) + "#Sub-Hy2-01",
	))
	subSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(subContent))
	}))
	defer subSrv.Close()

	require.NoError(t, app.AddSubscription(config.Subscription{
		Id:   "sub-test-1",
		Name: "测试机场订阅",
		Url:  subSrv.URL,
	}))
	require.NoError(t, app.UpdateAllSubscriptions())
	subNodes := app.GetNodesByGroup("sub-test-1")
	require.Len(t, subNodes, 1)
	assert.Equal(t, "Sub-Hy2-01", subNodes[0].Name)

	// 验证手动新导入节点到已有订阅分组后，点击刷新节点不会把刚添加的节点丢失
	manualAdded, err := app.ImportNodesToGroup(
		"vless://550e8400-e29b-41d4-a716-446655440000@1.2.3.4:8443?security=tls&type=tcp#Manual-Added-Node",
		"sub-test-1",
	)
	require.NoError(t, err)
	require.Len(t, manualAdded, 1)
	require.NoError(t, app.UpdateSubscription("sub-test-1"))
	subNodesAfterRefresh := app.GetNodesByGroup("sub-test-1")
	require.Len(t, subNodesAfterRefresh, 2)

	// 9. 停止代理并验证状态还原
	require.NoError(t, app.StopProxy())
	assert.Equal(t, config.StatusDisconnected, app.GetConnectionStatus())
}
