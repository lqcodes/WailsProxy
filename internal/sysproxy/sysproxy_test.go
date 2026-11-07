package sysproxy

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRunner struct {
	mu       sync.Mutex
	commands []string
}

func (m *mockRunner) Run(name string, args ...string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd := name + " " + strings.Join(args, " ")
	m.commands = append(m.commands, cmd)
	if name == "networksetup" && len(args) > 0 && args[0] == "-listallnetworkservices" {
		return "An asterisk (*) denotes that a network service is disabled.\nWi-Fi\nThunderbolt Bridge", nil
	}
	return "OK", nil
}

func TestSysProxy_EnableReadAndRestore_AllPlatforms(t *testing.T) {
	platforms := []string{"darwin", "windows", "linux"}

	for _, osName := range platforms {
		runner := &mockRunner{}
		mgr := NewManagerWithRunner(osName, runner)

		// 1. 初始状态未开启
		st0 := mgr.GetState()
		assert.False(t, st0.Enabled)

		// 2. 开启系统代理
		err := mgr.Enable("127.0.0.1", 10809, 10808)
		require.NoError(t, err, "platform: %s", osName)

		// 3. 读回状态验证已开启
		st1 := mgr.GetState()
		assert.True(t, st1.Enabled)
		assert.Equal(t, "127.0.0.1", st1.Host)
		assert.Equal(t, 10809, st1.HttpPort)
		assert.Equal(t, 10808, st1.SocksPort)

		// 4. 模拟退出/关闭时清除代理并读回验证已还原
		err = mgr.Disable()
		require.NoError(t, err, "platform: %s", osName)

		st2 := mgr.GetState()
		assert.False(t, st2.Enabled)
		assert.NotEmpty(t, runner.commands)
	}
}
