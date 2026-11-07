package subscription

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseContent_ClashYAML(t *testing.T) {
	yamlText := `
port: 7890
socks-port: 7891
proxies:
  - name: "香港 VLESS Reality"
    type: vless
    server: hk.reality.example.com
    port: 443
    uuid: 550e8400-e29b-41d4-a716-446655440000
    network: tcp
    tls: true
    udp: true
    flow: xtls-rprx-vision
    servername: www.apple.com
    client-fingerprint: chrome
    reality-opts:
      public-key: pubKey_example_123
      short-id: 01020304
  - name: "东京 VMess WS"
    type: vmess
    server: jp.vmess.example.com
    port: 8443
    uuid: 11111111-2222-3333-4444-555555555555
    alterId: 0
    cipher: auto
    tls: true
    network: ws
    ws-opts:
      path: /ws-path
      headers:
        Host: cdn.jp.example.com
  - name: "新加坡 SS"
    type: ss
    server: sg.ss.example.com
    port: 8388
    cipher: aes-256-gcm
    password: my_ss_password
  - name: "韩国 Trojan gRPC"
    type: trojan
    server: kr.trojan.example.com
    port: 443
    password: trojan_secret_pw
    network: grpc
    sni: kr.trojan.example.com
    grpc-opts:
      grpc-service-name: krGrpc
  - name: "德国 Hysteria2"
    type: hysteria2
    server: de.hy2.example.com
    port: 4433
    password: hy2_password
    sni: de.hy2.example.com
    obfs: salamander
    obfs-password: salamander_pw
`
	nodes, err := ParseContent(yamlText)
	require.NoError(t, err)
	require.Len(t, nodes, 5)

	assert.Equal(t, "vless", nodes[0].Protocol)
	assert.Equal(t, "reality", nodes[0].Security)
	assert.Equal(t, "pubKey_example_123", nodes[0].PublicKey)
	assert.Equal(t, "01020304", nodes[0].ShortId)

	assert.Equal(t, "vmess", nodes[1].Protocol)
	assert.Equal(t, "/ws-path", nodes[1].Path)
	assert.Equal(t, "cdn.jp.example.com", nodes[1].Host)

	assert.Equal(t, "ss", nodes[2].Protocol)
	assert.Equal(t, "aes-256-gcm", nodes[2].Cipher)

	assert.Equal(t, "trojan", nodes[3].Protocol)
	assert.Equal(t, "krGrpc", nodes[3].ServiceName)

	assert.Equal(t, "hysteria2", nodes[4].Protocol)
	assert.Equal(t, "salamander", nodes[4].ObfsType)
}

func TestParseContent_SingBoxJSON(t *testing.T) {
	jsonText := `{
  "outbounds": [
    {
      "type": "vless",
      "tag": "US-SingBox-VLESS",
      "server": "us.sb.example.com",
      "server_port": 443,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "flow": "xtls-rprx-vision",
      "tls": {
        "enabled": true,
        "server_name": "www.cloudflare.com",
        "utls": { "enabled": true, "fingerprint": "chrome" },
        "reality": { "enabled": true, "public_key": "sb_pub_key", "short_id": "aabb" }
      }
    },
    {
      "type": "shadowsocks",
      "tag": "HK-SingBox-SS",
      "server": "hk.sb.example.com",
      "server_port": 9001,
      "method": "chacha20-ietf-poly1305",
      "password": "sb_ss_password"
    },
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}`
	nodes, err := ParseContent(jsonText)
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	assert.Equal(t, "US-SingBox-VLESS", nodes[0].Name)
	assert.Equal(t, "reality", nodes[0].Security)
	assert.Equal(t, "sb_pub_key", nodes[0].PublicKey)
	assert.Equal(t, "ss", nodes[1].Protocol)
	assert.Equal(t, "chacha20-ietf-poly1305", nodes[1].Cipher)
}

func TestFetchSubscription_HTTPMock(t *testing.T) {
	links := "trojan://pw123@hk.example.com:443?sni=hk.example.com#HK-01\nhy2://auth@sg.example.com:4433#SG-01"
	encoded := base64.StdEncoding.EncodeToString([]byte(links))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(encoded))
	}))
	defer srv.Close()

	nodes, err := FetchSubscription(context.Background(), srv.URL, 5*time.Second)
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	assert.Equal(t, "HK-01", nodes[0].Name)
	assert.Equal(t, "SG-01", nodes[1].Name)
}
