package parser

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVMess(t *testing.T) {
	// 1. 标准 VMess 链接 (WS + TLS)
	vmessStdJSON := `{"v":"2","ps":"香港BGP-01","add":"hk1.example.com","port":"443","id":"23ad6b10-8d1a-40f7-8ad0-e3e35cd38297","aid":"0","scy":"auto","net":"ws","type":"none","host":"cdn.example.com","path":"/vmess-ws","tls":"tls","sni":"cdn.example.com","alpn":"h2,http/1.1","fp":"chrome"}`
	link1 := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessStdJSON))
	n1, err := ParseLink(link1)
	require.NoError(t, err)
	assert.Equal(t, "vmess", n1.Protocol)
	assert.Equal(t, "香港BGP-01", n1.Name)
	assert.Equal(t, "hk1.example.com", n1.Server)
	assert.Equal(t, 443, n1.Port)
	assert.Equal(t, "23ad6b10-8d1a-40f7-8ad0-e3e35cd38297", n1.Uuid)
	assert.Equal(t, "ws", n1.Network)
	assert.Equal(t, "/vmess-ws", n1.Path)
	assert.Equal(t, "cdn.example.com", n1.Host)
	assert.True(t, n1.Tls)
	assert.Equal(t, "chrome", n1.Fingerprint)

	// 2. 特殊字符与 숫자类型 port/aid + gRPC + RawBase64
	vmessGrpcJSON := `{"v":2,"ps":"🇯🇵 东京 #1 (高速 & 稳定)","add":"jp.example.com","port":8443,"id":"11111111-2222-3333-4444-555555555555","aid":2,"net":"grpc","path":"gunService","tls":"tls"}`
	link2 := "vmess://" + base64.RawURLEncoding.EncodeToString([]byte(vmessGrpcJSON))
	n2, err := ParseLink(link2)
	require.NoError(t, err)
	assert.Equal(t, "🇯🇵 东京 #1 (高速 & 稳定)", n2.Name)
	assert.Equal(t, 8443, n2.Port)
	assert.Equal(t, 2, n2.AlterId)
	assert.Equal(t, "grpc", n2.Network)
	assert.Equal(t, "gunService", n2.ServiceName)

	// 3. 残缺/非法 VMess 链接
	_, err = ParseLink("vmess://not_valid_json_base64!!!")
	assert.Error(t, err)

	badJSON := `{"v":"2","ps":"broken","add":"","port":0}`
	_, err = ParseLink("vmess://" + base64.StdEncoding.EncodeToString([]byte(badJSON)))
	assert.Error(t, err)
}

func TestParseVLESS(t *testing.T) {
	// 1. 标准 VLESS + Reality + XTLS Vision
	link1 := "vless://550e8400-e29b-41d4-a716-446655440000@us.example.com:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=www.microsoft.com&fp=chrome&pbk=mB7_publicKey_test123&sid=a1b2c3d4&type=tcp#US-Reality-01"
	n1, err := ParseLink(link1)
	require.NoError(t, err)
	assert.Equal(t, "vless", n1.Protocol)
	assert.Equal(t, "US-Reality-01", n1.Name)
	assert.Equal(t, "us.example.com", n1.Server)
	assert.Equal(t, 443, n1.Port)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", n1.Uuid)
	assert.Equal(t, "reality", n1.Security)
	assert.Equal(t, "mB7_publicKey_test123", n1.PublicKey)
	assert.Equal(t, "a1b2c3d4", n1.ShortId)
	assert.Equal(t, "xtls-rprx-vision", n1.Flow)
	assert.True(t, n1.Tls)

	// 2. VLESS + WS + URL 编码特殊字符备注
	link2 := "vless://550e8400-e29b-41d4-a716-446655440000@sg.example.com:2053?type=ws&security=tls&path=%2Fvless-ws%3Fed%3D2048&host=sg.example.com&sni=sg.example.com#%F0%9F%87%B8%F0%9F%87%AC%20%E6%96%B0%E5%8A%A0%E5%9D%A1%20%5BIEPL%5D"
	n2, err := ParseLink(link2)
	require.NoError(t, err)
	assert.Equal(t, "🇸🇬 新加坡 [IEPL]", n2.Name)
	assert.Equal(t, "/vless-ws?ed=2048", n2.Path)
	assert.Equal(t, "ws", n2.Network)

	// 3. 非法 VLESS 链接（缺少 UUID 或端口）
	_, err = ParseLink("vless://@example.com:443#NoUUID")
	assert.Error(t, err)
	_, err = ParseLink("vless://uuid-123@example.com:99999#BadPort")
	assert.Error(t, err)
}

func TestParseTrojan(t *testing.T) {
	// 1. 标准 Trojan 链接
	link1 := "trojan://myTrojanPass123@kr.example.com:443?security=tls&sni=kr.example.com&alpn=h2,http/1.1&type=tcp#KR-Seoul-01"
	n1, err := ParseLink(link1)
	require.NoError(t, err)
	assert.Equal(t, "trojan", n1.Protocol)
	assert.Equal(t, "KR-Seoul-01", n1.Name)
	assert.Equal(t, "myTrojanPass123", n1.Password)
	assert.Equal(t, "kr.example.com", n1.Server)
	assert.Equal(t, 443, n1.Port)
	assert.True(t, n1.Tls)

	// 2. 含特殊字符密码与 gRPC 传输层
	link2 := "trojan://p%40ss%3Aword%2123@tw.example.com:8443?type=grpc&serviceName=trojan-grpc&sni=tw.example.com#%E5%8F%B0%E6%B9%BE%E4%B8%93%E7%BA%BF"
	n2, err := ParseLink(link2)
	require.NoError(t, err)
	assert.Equal(t, "p@ss:word!23", n2.Password)
	assert.Equal(t, "grpc", n2.Network)
	assert.Equal(t, "trojan-grpc", n2.ServiceName)
	assert.Equal(t, "台湾专线", n2.Name)

	// 3. 残缺 Trojan 链接
	_, err = ParseLink("trojan://@example.com:443")
	assert.Error(t, err)
}

func TestParseShadowsocks(t *testing.T) {
	// 1. SIP002 标准格式
	userInfo := base64.RawURLEncoding.EncodeToString([]byte("aes-256-gcm:ss_secret_2026"))
	link1 := "ss://" + userInfo + "@hk.ss.example.com:8388#HK-SS-01"
	n1, err := ParseLink(link1)
	require.NoError(t, err)
	assert.Equal(t, "ss", n1.Protocol)
	assert.Equal(t, "aes-256-gcm", n1.Cipher)
	assert.Equal(t, "ss_secret_2026", n1.Password)
	assert.Equal(t, "hk.ss.example.com", n1.Server)
	assert.Equal(t, 8388, n1.Port)
	assert.Equal(t, "HK-SS-01", n1.Name)

	// 2. 旧版全 Base64 格式 + 特殊字符密码
	legacyRaw := "chacha20-ietf-poly1305:p@ss:w0rd!@198.51.100.10:1080"
	link2 := "ss://" + base64.StdEncoding.EncodeToString([]byte(legacyRaw)) + "#Legacy-SS%20Node"
	n2, err := ParseLink(link2)
	require.NoError(t, err)
	assert.Equal(t, "chacha20-ietf-poly1305", n2.Cipher)
	assert.Equal(t, "p@ss:w0rd!", n2.Password)
	assert.Equal(t, "198.51.100.10", n2.Server)
	assert.Equal(t, 1080, n2.Port)
	assert.Equal(t, "Legacy-SS Node", n2.Name)

	// 3. 非法 SS 链接
	_, err = ParseLink("ss://invalid_base64_no_at")
	assert.Error(t, err)
}

func TestParseHysteria2AndTUIC(t *testing.T) {
	// 1. 标准 Hysteria2 链接（含 salamander 混淆与上下行带宽）
	link1 := "hysteria2://hy2_auth_pass@de.example.com:4433?sni=de.example.com&obfs=salamander&obfs-password=obfs_secret&upmbps=100&downmbps=500#DE-Hy2-01"
	n1, err := ParseLink(link1)
	require.NoError(t, err)
	assert.Equal(t, "hysteria2", n1.Protocol)
	assert.Equal(t, "hy2_auth_pass", n1.Password)
	assert.Equal(t, "salamander", n1.ObfsType)
	assert.Equal(t, "obfs_secret", n1.ObfsPassword)
	assert.Equal(t, 100, n1.UpMbps)
	assert.Equal(t, 500, n1.DownMbps)

	// 2. hy2:// 简写链接与 TUIC 链接
	link2 := "hy2://pass%40123@uk.example.com:8443?sni=uk.example.com#UK-Hy2"
	n2, err := ParseLink(link2)
	require.NoError(t, err)
	assert.Equal(t, "hysteria2", n2.Protocol)
	assert.Equal(t, "pass@123", n2.Password)

	link3 := "tuic://550e8400-e29b-41d4-a716-446655440000:tuic_pass@fr.example.com:443?sni=fr.example.com&alpn=h3#FR-TUIC"
	n3, err := ParseLink(link3)
	require.NoError(t, err)
	assert.Equal(t, "tuic", n3.Protocol)
	assert.Equal(t, "tuic_pass", n3.Password)

	// 3. 非法 Hysteria2 链接
	_, err = ParseLink("hy2://@example.com:443")
	assert.Error(t, err)
}

func TestParseMultipleLinks_Base64AndPlain(t *testing.T) {
	rawLinks := strings.Join([]string{
		"vless://550e8400-e29b-41d4-a716-446655440000@us.example.com:443?security=tls&type=tcp#US-Node",
		"trojan://pass123@kr.example.com:443?sni=kr.example.com#KR-Node",
		"hy2://auth123@jp.example.com:4433#JP-Node",
	}, "\n")

	encoded := base64.StdEncoding.EncodeToString([]byte(rawLinks))
	nodes, err := ParseMultipleLinks(encoded)
	require.NoError(t, err)
	assert.Len(t, nodes, 3)
}
