package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicWriteJSON_ConcurrentAndCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "app_config.json")

	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cfg := DefaultAppConfig()
			cfg.SocksPort = 10800 + idx
			err := AtomicWriteJSON(target, cfg)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	// Verify target file is valid non-empty JSON
	var loaded AppConfig
	err := ReadJSON(target, &loaded)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, loaded.SocksPort, 10800)
	assert.Less(t, loaded.SocksPort, 10825)

	// Verify .tmp file was cleaned up after atomic rename
	_, err = os.Stat(target + ".tmp")
	assert.True(t, os.IsNotExist(err), ".tmp file should be removed after atomic rename")
}

func TestSanitizeLog(t *testing.T) {
	raw := `Connecting user 23ad6b10-8d1a-40f7-8ad0-e3e35cd38297 with password=my_super_secret_pw and https://sub.example.com/api?token=abcdef123456`
	cleaned := SanitizeLog(raw, "customSecret999")
	assert.NotContains(t, cleaned, "23ad6b10-8d1a-40f7-8ad0-e3e35cd38297")
	assert.NotContains(t, cleaned, "my_super_secret_pw")
	assert.NotContains(t, cleaned, "abcdef123456")
	assert.Contains(t, cleaned, "********-****-****-****-************")
	assert.Contains(t, cleaned, "***REDACTED***")
}

func TestManager_CRUDAndSort(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Default group should exist
	subs := mgr.GetSubscriptions()
	require.Len(t, subs, 1)
	assert.Equal(t, DefaultGroupID, subs[0].Id)

	// Add subscription group
	sub, err := mgr.AddSubscription(Subscription{
		Name: "香港专线",
		Url:  "https://example.com/sub",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, sub.Id)

	// Add nodes
	nodes, err := mgr.AddNodes([]NodeConfig{
		{Name: "HK-Slow", Protocol: "vless", Server: "1.1.1.1", Port: 443, TcpDelay: 250},
		{Name: "HK-Timeout", Protocol: "vmess", Server: "2.2.2.2", Port: 443, TcpDelay: -1},
		{Name: "HK-Fast", Protocol: "trojan", Server: "3.3.3.3", Port: 443, TcpDelay: 35, RealDelay: 60},
	}, sub.Id)
	require.NoError(t, err)
	require.Len(t, nodes, 3)

	// Sort by delay
	require.NoError(t, mgr.SortNodesByDelay())
	sorted := mgr.GetNodesByGroup(sub.Id)
	require.Len(t, sorted, 3)
	assert.Equal(t, "HK-Fast", sorted[0].Name)
	assert.Equal(t, "HK-Slow", sorted[1].Name)
	assert.Equal(t, "HK-Timeout", sorted[2].Name)

	// Delete invalid/timeout nodes
	removed, err := mgr.DeleteInvalidNodes(sub.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	assert.Len(t, mgr.GetNodesByGroup(sub.Id), 2)
}
