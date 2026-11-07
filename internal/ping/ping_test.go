package ping

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"wailsproxy/internal/config"
)

func TestTCPPing_SuccessAndTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, _ := strconv.Atoi(portStr)

	delay := TCPPing(host, port, 1*time.Second)
	assert.Greater(t, delay, int64(0))

	// 测试未开放端口或不可达地址应返回 -1
	badDelay := TCPPing("127.0.0.1", 1, 200*time.Millisecond)
	assert.Equal(t, int64(-1), badDelay)
}

func TestHTTPDelay_Mock204Server(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	delay := HTTPDelay("", srv.URL, 2*time.Second)
	assert.GreaterOrEqual(t, delay, int64(10))

	// 测试超时返回 -1
	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer slowSrv.Close()

	timeoutDelay := HTTPDelay("", slowSrv.URL, 50*time.Millisecond)
	assert.Equal(t, int64(-1), timeoutDelay)
}

func TestBatchTest_HighConcurrency(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	nodes := make([]config.NodeConfig, 30)
	for i := 0; i < 30; i++ {
		if i%5 == 0 {
			nodes[i] = config.NodeConfig{Id: "timeout-" + strconv.Itoa(i), Server: "127.0.0.1", Port: 1}
		} else {
			nodes[i] = config.NodeConfig{Id: "ok-" + strconv.Itoa(i), Server: host, Port: port}
		}
	}

	var callbackCount int32
	results := BatchTest(
		context.Background(),
		nodes,
		10,
		300*time.Millisecond,
		srv.URL,
		"",
		"",
		func(res config.PingResultEvent) {
			atomic.AddInt32(&callbackCount, 1)
		},
	)

	assert.Len(t, results, 30)
	assert.Equal(t, int32(30), atomic.LoadInt32(&callbackCount))
}
