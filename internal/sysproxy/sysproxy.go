package sysproxy

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// CommandRunner 抽象系统命令执行接口，便于跨平台测试与安全防护
type CommandRunner interface {
	Run(name string, args ...string) (string, error)
}

type osCommandRunner struct{}

func (osCommandRunner) Run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// State 表示当前系统代理状态快照
type State struct {
	Enabled   bool   `json:"enabled"`
	Host      string `json:"host"`
	HttpPort  int    `json:"httpPort"`
	SocksPort int    `json:"socksPort"`
	Service   string `json:"service"` // macOS 网络服务名，如 "Wi-Fi"
}

// Manager 负责跨平台系统代理的开启、状态查询与崩溃安全还原
type Manager struct {
	mu       sync.Mutex
	osName   string
	runner   CommandRunner
	snapshot *State
	current  State
}

// NewManager 创建系统代理管理器
func NewManager() *Manager {
	return &Manager{
		osName: runtime.GOOS,
		runner: osCommandRunner{},
	}
}

type noopCommandRunner struct{}

func (noopCommandRunner) Run(name string, args ...string) (string, error) {
	return "OK", nil
}

// NewManagerWithRunner 创建带自定义命令执行器的管理器（用于测试）
func NewManagerWithRunner(osName string, runner CommandRunner) *Manager {
	if runner == nil {
		runner = noopCommandRunner{}
	}
	return &Manager{
		osName: osName,
		runner: runner,
	}
}

// BackupSnapshot 在修改系统代理前记录原始系统代理快照
func (m *Manager) BackupSnapshot() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.snapshot == nil {
		cp := m.current
		m.snapshot = &cp
	}
	return *m.snapshot
}

// Enable 开启系统代理（HTTP + SOCKS5），自动备份原始状态
func (m *Manager) Enable(host string, httpPort, socksPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if host == "" {
		host = "127.0.0.1"
	}
	if m.snapshot == nil {
		cp := m.current
		m.snapshot = &cp
	}

	var err error
	switch m.osName {
	case "darwin":
		err = m.enableDarwinLocked(host, httpPort, socksPort)
	case "windows":
		err = m.enableWindowsLocked(host, httpPort)
	case "linux":
		err = m.enableLinuxLocked(host, httpPort, socksPort)
	default:
		err = nil
	}
	if err != nil {
		return err
	}

	m.current = State{
		Enabled:   true,
		Host:      host,
		HttpPort:  httpPort,
		SocksPort: socksPort,
		Service:   m.detectDarwinServiceLocked(),
	}
	return nil
}

// Disable 关闭系统代理并还原到启动前快照状态（零断网保护）
func (m *Manager) Disable() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.current.Enabled {
		m.snapshot = nil
		return nil
	}

	var err error
	switch m.osName {
	case "darwin":
		err = m.disableDarwinLocked()
	case "windows":
		err = m.disableWindowsLocked()
	case "linux":
		err = m.disableLinuxLocked()
	}

	m.current = State{Enabled: false}
	m.snapshot = nil
	return err
}

// GetState 获取当前系统代理状态
func (m *Manager) GetState() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

func (m *Manager) listDarwinServicesLocked() []string {
	out, err := m.runner.Run("networksetup", "-listallnetworkservices")
	if err != nil || out == "" {
		return []string{"Wi-Fi"}
	}
	var services []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "*") || strings.Contains(line, "asterisk") {
			continue
		}
		services = append(services, line)
	}
	if len(services) == 0 {
		return []string{"Wi-Fi"}
	}
	return services
}

func (m *Manager) detectDarwinServiceLocked() string {
	svcs := m.listDarwinServicesLocked()
	for _, svc := range svcs {
		if strings.EqualFold(svc, "Wi-Fi") || strings.Contains(svc, "Ethernet") || strings.Contains(svc, "LAN") {
			return svc
		}
	}
	return svcs[0]
}

func (m *Manager) enableDarwinLocked(host string, httpPort, socksPort int) error {
	svcs := m.listDarwinServicesLocked()
	httpPortStr := strconv.Itoa(httpPort)
	socksPortStr := strconv.Itoa(socksPort)

	var lastErr error
	successCount := 0
	for _, svc := range svcs {
		if _, err := m.runner.Run("networksetup", "-setwebproxy", svc, host, httpPortStr); err != nil {
			lastErr = err
			continue
		}
		_, _ = m.runner.Run("networksetup", "-setsecurewebproxy", svc, host, httpPortStr)
		if socksPort > 0 {
			_, _ = m.runner.Run("networksetup", "-setsocksfirewallproxy", svc, host, socksPortStr)
		}
		_, _ = m.runner.Run("networksetup", "-setproxybypassdomains", svc, "127.0.0.1", "localhost", "192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12", "*.local")
		_, _ = m.runner.Run("networksetup", "-setwebproxystate", svc, "on")
		_, _ = m.runner.Run("networksetup", "-setsecurewebproxystate", svc, "on")
		if socksPort > 0 {
			_, _ = m.runner.Run("networksetup", "-setsocksfirewallproxystate", svc, "on")
		}
		successCount++
	}
	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("macOS setwebproxy failed: %w", lastErr)
	}
	return nil
}

func (m *Manager) disableDarwinLocked() error {
	svcs := m.listDarwinServicesLocked()
	var lastErr error
	for _, svc := range svcs {
		if _, err := m.runner.Run("networksetup", "-setwebproxystate", svc, "off"); err != nil {
			lastErr = err
		}
		_, _ = m.runner.Run("networksetup", "-setsecurewebproxystate", svc, "off")
		_, _ = m.runner.Run("networksetup", "-setsocksfirewallproxystate", svc, "off")
	}
	return lastErr
}

func (m *Manager) enableWindowsLocked(host string, httpPort int) error {
	regPath := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	proxyServer := fmt.Sprintf("%s:%d", host, httpPort)
	if _, err := m.runner.Run("reg", "add", regPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f"); err != nil {
		return fmt.Errorf("windows enable proxy reg failed: %w", err)
	}
	if _, err := m.runner.Run("reg", "add", regPath, "/v", "ProxyServer", "/t", "REG_SZ", "/d", proxyServer, "/f"); err != nil {
		return fmt.Errorf("windows set proxy server reg failed: %w", err)
	}
	return nil
}

func (m *Manager) disableWindowsLocked() error {
	regPath := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	_, err := m.runner.Run("reg", "add", regPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f")
	return err
}

func (m *Manager) enableLinuxLocked(host string, httpPort, socksPort int) error {
	httpPortStr := strconv.Itoa(httpPort)
	socksPortStr := strconv.Itoa(socksPort)
	if _, err := m.runner.Run("gsettings", "set", "org.gnome.system.proxy", "mode", "manual"); err != nil {
		return fmt.Errorf("linux gsettings proxy mode failed: %w", err)
	}
	_, _ = m.runner.Run("gsettings", "set", "org.gnome.system.proxy.http", "host", host)
	_, _ = m.runner.Run("gsettings", "set", "org.gnome.system.proxy.http", "port", httpPortStr)
	_, _ = m.runner.Run("gsettings", "set", "org.gnome.system.proxy.https", "host", host)
	_, _ = m.runner.Run("gsettings", "set", "org.gnome.system.proxy.https", "port", httpPortStr)
	if socksPort > 0 {
		_, _ = m.runner.Run("gsettings", "set", "org.gnome.system.proxy.socks", "host", host)
		_, _ = m.runner.Run("gsettings", "set", "org.gnome.system.proxy.socks", "port", socksPortStr)
	}
	return nil
}

func (m *Manager) disableLinuxLocked() error {
	_, err := m.runner.Run("gsettings", "set", "org.gnome.system.proxy", "mode", "none")
	return err
}
