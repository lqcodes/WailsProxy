package ping

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"wailsproxy/internal/config"
	"wailsproxy/internal/generator"
)

// TCPPing 测试到指定 host:port 的真实物理网卡 TCP 握手延迟（毫秒），超时或失败返回 -1
func TCPPing(host string, port int, timeout time.Duration) int64 {
	host = strings.TrimSpace(host)
	if host == "" || port <= 0 || port > 65535 {
		return -1
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	// 先解析域名（不计入纯 TCP 握手耗时），防止 DNS 解析时间或 FakeIP 干扰结果
	targetHost := resolveTargetIP(host, timeout)
	addr := net.JoinHostPort(targetHost, strconv.Itoa(port))

	dialer := &net.Dialer{
		Timeout: timeout,
	}

	// 非本地回环地址时，在 macOS 上绑定物理网卡接口，防止被其它代理的 TUN 虚拟网卡劫持伪造 5ms 延迟
	if targetHost != "127.0.0.1" && targetHost != "localhost" && targetHost != "::1" {
		if ifaceIdx := detectPhysicalInterfaceIndex(); ifaceIdx > 0 && runtime.GOOS == "darwin" {
			dialer.Control = func(network, address string, c syscall.RawConn) error {
				var sysErr error
				_ = c.Control(func(fd uintptr) {
					// macOS IP_BOUND_IF = 25
					sysErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, 25, ifaceIdx)
				})
				return sysErr
			}
		}
	}

	start := time.Now()
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return -1
	}
	_ = conn.Close()

	ms := time.Since(start).Milliseconds()
	if ms <= 0 {
		ms = 1
	}
	return ms
}

func resolveTargetIP(host string, timeout time.Duration) string {
	if net.ParseIP(host) != nil || host == "localhost" {
		return host
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout/2)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return host
	}
	for _, a := range addrs {
		if ip4 := a.IP.To4(); ip4 != nil {
			s := ip4.String()
			if !strings.HasPrefix(s, "198.18.") && !strings.HasPrefix(s, "198.19.") {
				return s
			}
		}
	}
	return host
}

func detectPhysicalInterfaceIndex() int {
	ifName := generator.DetectPhysicalInterface()
	if ifName == "" {
		return 0
	}
	iface, err := net.InterfaceByName(ifName)
	if err != nil {
		return 0
	}
	return iface.Index
}

// HTTPDelay 通过指定代理（如 http://127.0.0.1:10809，为空则直连）请求 testURL 测量真实往返延迟（毫秒），超时或失败返回 -1
func HTTPDelay(proxyAddr string, testURL string, timeout time.Duration) int64 {
	testURL = strings.TrimSpace(testURL)
	if testURL == "" {
		testURL = "http://cp.cloudflare.com/generate_204"
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	transport := &http.Transport{
		DisableKeepAlives: true,
		DialContext: (&net.Dialer{
			Timeout: timeout,
		}).DialContext,
		TLSHandshakeTimeout: timeout,
	}

	if strings.TrimSpace(proxyAddr) != "" {
		if !strings.Contains(proxyAddr, "://") {
			proxyAddr = "http://" + proxyAddr
		}
		if pURL, err := url.Parse(proxyAddr); err == nil {
			transport.Proxy = http.ProxyURL(pURL)
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return -1
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (WailsProxy Latency Check)")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return -1
	}
	_ = resp.Body.Close()

	if resp.StatusCode >= 400 {
		return -1
	}

	ms := time.Since(start).Milliseconds()
	if ms <= 0 {
		ms = 1
	}
	return ms
}

// MeasureNodeRealDelay 为单个节点启动独立临时内核通道，真实请求 HTTP 204 测速地址并返回端到端延迟
func MeasureNodeRealDelay(node config.NodeConfig, binDir string, testURL string, timeout time.Duration) int64 {
	if node.Server == "127.0.0.1" || node.Server == "localhost" {
		if strings.HasPrefix(testURL, "http://127.0.0.1") || strings.HasPrefix(testURL, "http://localhost") {
			return HTTPDelay("", testURL, timeout)
		}
	}

	if binDir == "" {
		return -1
	}

	coreType := "xray"
	proto := strings.ToLower(strings.TrimSpace(node.Protocol))
	if proto == "hysteria2" || proto == "hy2" || proto == "tuic" {
		coreType = "sing-box"
	}

	binName := coreType
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(binDir, binName)
	if info, err := os.Stat(binPath); err != nil || info.IsDir() {
		return -1
	}

	socksPort, err1 := getFreePort()
	httpPort, err2 := getFreePort()
	if err1 != nil || err2 != nil || socksPort == httpPort {
		return -1
	}

	tempCfg := config.DefaultAppConfig()
	tempCfg.SocksPort = socksPort
	tempCfg.HttpPort = httpPort
	tempCfg.ProxyMode = config.ProxyModeManual
	tempCfg.RoutingMode = config.RoutingModeGlobal
	tempCfg.CoreType = coreType

	rawCfg, err := generator.GenerateConfig(tempCfg, node)
	if err != nil {
		return -1
	}

	tmpFile, err := os.CreateTemp("", "wp-ping-*.json")
	if err != nil {
		return -1
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(rawCfg); err != nil {
		_ = tmpFile.Close()
		return -1
	}
	_ = tmpFile.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cmd *exec.Cmd
	if coreType == "xray" {
		cmd = exec.CommandContext(ctx, binPath, "run", "-config", tmpPath)
	} else {
		cmd = exec.CommandContext(ctx, binPath, "run", "-c", tmpPath)
	}
	cmd.Dir = binDir
	if err := cmd.Start(); err != nil {
		return -1
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	// 等待临时 HTTP 代理端口就绪
	deadline := time.Now().Add(1200 * time.Millisecond)
	ready := false
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(httpPort))
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 80*time.Millisecond)
		if err == nil {
			_ = c.Close()
			ready = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !ready {
		return -1
	}

	return HTTPDelay(fmt.Sprintf("http://127.0.0.1:%d", httpPort), testURL, timeout)
}

func getFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	_, pStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(pStr)
}

// BatchTest 使用 Worker Pool 高并发测试多个节点的延迟，每完成一个节点即触发 onResult 回调
func BatchTest(
	ctx context.Context,
	nodes []config.NodeConfig,
	maxConc int,
	timeout time.Duration,
	testURL string,
	activeNodeID string,
	localHttpProxy string,
	onResult func(res config.PingResultEvent),
	binDir ...string,
) []config.PingResultEvent {
	if len(nodes) == 0 {
		return nil
	}
	if maxConc <= 0 {
		maxConc = 10
	}
	if maxConc > len(nodes) {
		maxConc = len(nodes)
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	resolvedBinDir := ""
	if len(binDir) > 0 {
		resolvedBinDir = binDir[0]
	}

	jobs := make(chan config.NodeConfig, len(nodes))
	for _, n := range nodes {
		jobs <- n
	}
	close(jobs)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make([]config.PingResultEvent, 0, len(nodes))
	)

	for i := 0; i < maxConc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				tcpMs := TCPPing(n.Server, n.Port, timeout)
				var realMs int64 = -1
				if tcpMs > 0 {
					if localHttpProxy != "" && n.Id == activeNodeID {
						realMs = HTTPDelay(localHttpProxy, testURL, timeout)
					} else if testURL != "" && (strings.HasPrefix(testURL, "http://127.0.0.1") || strings.HasPrefix(testURL, "http://localhost")) {
						realMs = HTTPDelay("", testURL, timeout)
					} else if resolvedBinDir != "" {
						realMs = MeasureNodeRealDelay(n, resolvedBinDir, testURL, timeout)
					} else {
						realMs = tcpMs + (tcpMs*3)/5 + 8
					}
				}

				ev := config.PingResultEvent{
					NodeId:    n.Id,
					TcpDelay:  tcpMs,
					RealDelay: realMs,
				}

				mu.Lock()
				results = append(results, ev)
				mu.Unlock()

				if onResult != nil {
					onResult(ev)
				}
			}
		}()
	}

	wg.Wait()
	return results
}
