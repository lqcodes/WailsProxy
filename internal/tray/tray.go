package tray

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// FormatTerminalProxyCmd 生成一键复制终端代理环境变量命令（支持 bash/zsh, powershell, cmd）
func FormatTerminalProxyCmd(host string, httpPort, socksPort int, shell string) string {
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "powershell", "ps1", "pwsh":
		return fmt.Sprintf(`$env:http_proxy="http://%s:%d"; $env:https_proxy="http://%s:%d"; $env:all_proxy="socks5://%s:%d"`,
			host, httpPort, host, httpPort, host, socksPort)
	case "cmd", "bat":
		return fmt.Sprintf(`set http_proxy=http://%s:%d && set https_proxy=http://%s:%d && set all_proxy=socks5://%s:%d`,
			host, httpPort, host, httpPort, host, socksPort)
	default: // bash / zsh
		return fmt.Sprintf(`export http_proxy=http://%s:%d https_proxy=http://%s:%d all_proxy=socks5://%s:%d`,
			host, httpPort, host, httpPort, host, socksPort)
	}
}

// BuildDarwinLaunchAgent 生成 macOS LaunchAgent plist 内容
func BuildDarwinLaunchAgent(execPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.wailsproxy.app</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`, execPath)
}

// BuildLinuxDesktopEntry 生成 Linux ~/.config/autostart/wailsproxy.desktop 内容
func BuildLinuxDesktopEntry(execPath string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=WailsProxy
Comment=Modern Cross-Platform Proxy Client
Exec=%s
Terminal=false
StartupNotify=false
`, execPath)
}

// ConfigureAutoStart 设置或取消开机自启动（baseDir 可在测试中传入临时目录）
func ConfigureAutoStart(enabled bool, baseDir string) error {
	execPath, err := os.Executable()
	if err != nil || execPath == "" {
		execPath = "/Applications/WailsProxy.app/Contents/MacOS/WailsProxy"
	}

	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		baseDir = home
	}

	switch runtime.GOOS {
	case "darwin":
		plistDir := filepath.Join(baseDir, "Library", "LaunchAgents")
		plistPath := filepath.Join(plistDir, "com.wailsproxy.app.plist")
		if !enabled {
			_ = os.Remove(plistPath)
			return nil
		}
		if err := os.MkdirAll(plistDir, 0755); err != nil {
			return err
		}
		return os.WriteFile(plistPath, []byte(BuildDarwinLaunchAgent(execPath)), 0644)

	case "linux":
		autoDir := filepath.Join(baseDir, ".config", "autostart")
		desktopPath := filepath.Join(autoDir, "wailsproxy.desktop")
		if !enabled {
			_ = os.Remove(desktopPath)
			return nil
		}
		if err := os.MkdirAll(autoDir, 0755); err != nil {
			return err
		}
		return os.WriteFile(desktopPath, []byte(BuildLinuxDesktopEntry(execPath)), 0644)

	default:
		return nil
	}
}
