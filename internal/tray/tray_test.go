package tray

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatTerminalProxyCmd(t *testing.T) {
	bashCmd := FormatTerminalProxyCmd("127.0.0.1", 10809, 10808, "zsh")
	assert.Contains(t, bashCmd, "export http_proxy=http://127.0.0.1:10809")
	assert.Contains(t, bashCmd, "all_proxy=socks5://127.0.0.1:10808")

	psCmd := FormatTerminalProxyCmd("127.0.0.1", 10809, 10808, "powershell")
	assert.Contains(t, psCmd, `$env:http_proxy="http://127.0.0.1:10809"`)

	cmdCmd := FormatTerminalProxyCmd("0.0.0.0", 10809, 10808, "cmd")
	assert.Contains(t, cmdCmd, `set http_proxy=http://127.0.0.1:10809`)
}

func TestConfigureAutoStart(t *testing.T) {
	tmpHome := t.TempDir()
	err := ConfigureAutoStart(true, tmpHome)
	require.NoError(t, err)

	if runtime.GOOS == "darwin" {
		plist := filepath.Join(tmpHome, "Library", "LaunchAgents", "com.wailsproxy.app.plist")
		_, err := os.Stat(plist)
		assert.NoError(t, err)

		require.NoError(t, ConfigureAutoStart(false, tmpHome))
		_, err = os.Stat(plist)
		assert.True(t, os.IsNotExist(err))
	}
}
