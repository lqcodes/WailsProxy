package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var atomicMu sync.Mutex

// AtomicWriteJSON 先将数据序列化写入临时文件，再通过 os.Rename 原子替换目标文件，
// 防止因程序崩溃或断电导致配置文件损坏或清空。
func AtomicWriteJSON(filename string, data interface{}) error {
	atomicMu.Lock()
	defer atomicMu.Unlock()

	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json for %s: %w", filename, err)
	}

	tmpFile := fmt.Sprintf("%s.%d.tmp", filename, os.Getpid())
	// Also write to standard .tmp for compatibility and clean up
	stdTmpFile := filename + ".tmp"

	f, err := os.OpenFile(stdTmpFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open temp file %s: %w", stdTmpFile, err)
	}

	if _, err := f.Write(raw); err != nil {
		_ = f.Close()
		_ = os.Remove(stdTmpFile)
		return fmt.Errorf("write temp file %s: %w", stdTmpFile, err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(stdTmpFile)
		return fmt.Errorf("sync temp file %s: %w", stdTmpFile, err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(stdTmpFile)
		return fmt.Errorf("close temp file %s: %w", stdTmpFile, err)
	}

	_ = os.Remove(tmpFile)
	if err := os.Rename(stdTmpFile, filename); err != nil {
		_ = os.Remove(stdTmpFile)
		return fmt.Errorf("atomic rename %s -> %s: %w", stdTmpFile, filename, err)
	}

	return nil
}

// ReadJSON 从文件读取并反序列化 JSON 数据
func ReadJSON(filename string, out interface{}) error {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return fmt.Errorf("empty json file: %s", filename)
	}
	return json.Unmarshal(raw, out)
}

var (
	uuidRegex  = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	tokenRegex = regexp.MustCompile(`(?i)(token|password|passwd|secret|key|auth)=([^&\s"']+)`)
)

// SanitizeLog 对日志行进行脱敏处理，隐藏 UUID、密码和订阅 URL 中的 Token
func SanitizeLog(line string, extraSecrets ...string) string {
	sanitized := uuidRegex.ReplaceAllString(line, "********-****-****-****-************")
	sanitized = tokenRegex.ReplaceAllString(sanitized, "$1=***REDACTED***")
	for _, secret := range extraSecrets {
		s := strings.TrimSpace(secret)
		if len(s) >= 4 {
			sanitized = strings.ReplaceAll(sanitized, s, "***REDACTED***")
		}
	}
	return sanitized
}
