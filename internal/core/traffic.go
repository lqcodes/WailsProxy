package core

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"wailsproxy/internal/config"
)

// TrafficCollector 负责每秒采集内核上下行流量与活跃连接数并推送事件
type TrafficCollector struct {
	totalUpload   int64
	totalDownload int64
	lastUpload    int64
	lastDownload  int64
	activeConns   int64
	simTelemetry  bool
	stopCh        chan struct{}
	stopped       int32
}

// NewTrafficCollector 创建流量采集器
func NewTrafficCollector(simTelemetry bool) *TrafficCollector {
	return &TrafficCollector{
		simTelemetry: simTelemetry,
		stopCh:       make(chan struct{}),
	}
}

// AddTraffic 累加实际转发的上下行字节数
func (tc *TrafficCollector) AddTraffic(upBytes, downBytes int64) {
	if upBytes > 0 {
		atomic.AddInt64(&tc.totalUpload, upBytes)
	}
	if downBytes > 0 {
		atomic.AddInt64(&tc.totalDownload, downBytes)
	}
}

// SetActiveConns 更新活跃连接数
func (tc *TrafficCollector) SetActiveConns(n int) {
	atomic.StoreInt64(&tc.activeConns, int64(n))
}

// IncrConn 增加活跃连接计数
func (tc *TrafficCollector) IncrConn(delta int64) {
	atomic.AddInt64(&tc.activeConns, delta)
}

// Start 启动每秒一次的流量统计采集与推送
func (tc *TrafficCollector) Start(onStats func(stats config.TrafficStats)) {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		rng := rand.New(rand.NewSource(time.Now().UnixNano()))

		for {
			select {
			case <-tc.stopCh:
				return
			case <-ticker.C:
				// 尝试从 Sing-box Clash API 获取连接数，若未开启则使用本地统计
				tc.probeClashConnections()

				curUp := atomic.LoadInt64(&tc.totalUpload)
				curDown := atomic.LoadInt64(&tc.totalDownload)
				prevUp := atomic.LoadInt64(&tc.lastUpload)
				prevDown := atomic.LoadInt64(&tc.lastDownload)

				upSpeed := curUp - prevUp
				downSpeed := curDown - prevDown

				// 当处于内置守护模拟模式且本秒无外部代理请求时，提供平滑心跳流量便于仪表盘波形图实时观测
				if tc.simTelemetry && upSpeed == 0 && downSpeed == 0 {
					simUp := int64(18*1024 + rng.Intn(65*1024))
					simDown := int64(120*1024 + rng.Intn(680*1024))
					curUp = atomic.AddInt64(&tc.totalUpload, simUp)
					curDown = atomic.AddInt64(&tc.totalDownload, simDown)
					upSpeed = simUp
					downSpeed = simDown
					if atomic.LoadInt64(&tc.activeConns) == 0 {
						atomic.StoreInt64(&tc.activeConns, int64(6+rng.Intn(12)))
					}
				}

				atomic.StoreInt64(&tc.lastUpload, curUp)
				atomic.StoreInt64(&tc.lastDownload, curDown)

				conns := int(atomic.LoadInt64(&tc.activeConns))
				if conns < 0 {
					conns = 0
				}

				if onStats != nil {
					onStats(config.TrafficStats{
						UploadSpeed:   upSpeed,
						DownloadSpeed: downSpeed,
						TotalUpload:   curUp,
						TotalDownload: curDown,
						ActiveConns:   conns,
					})
				}
			}
		}
	}()
}

func (tc *TrafficCollector) probeClashConnections() {
	client := &http.Client{Timeout: 250 * time.Millisecond}
	resp, err := client.Get("http://127.0.0.1:19090/connections")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var payload struct {
		DownloadTotal int64           `json:"downloadTotal"`
		UploadTotal   int64           `json:"uploadTotal"`
		Connections   []interface{}   `json:"connections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
		if payload.UploadTotal > 0 || payload.DownloadTotal > 0 {
			atomic.StoreInt64(&tc.totalUpload, payload.UploadTotal)
			atomic.StoreInt64(&tc.totalDownload, payload.DownloadTotal)
		}
		atomic.StoreInt64(&tc.activeConns, int64(len(payload.Connections)))
	}
}

// Stop 停止流量采集器
func (tc *TrafficCollector) Stop() {
	if atomic.CompareAndSwapInt32(&tc.stopped, 0, 1) {
		close(tc.stopCh)
	}
}
