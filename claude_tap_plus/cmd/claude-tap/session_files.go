package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// BaseDir 返回 ~/.claude-tap-plus/ 根目录。
func BaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude-tap-plus"
	}
	return filepath.Join(home, ".claude-tap-plus")
}

// --- backend.json ---

// BackendInfo 记录后端服务和代理的连接信息，存储在 ~/.claude-tap-plus/backend.json 中。
type BackendInfo struct {
	Host      string `json:"host"`       // 后端监听地址，如 127.0.0.1
	Port      int    `json:"port"`       // 后端监听端口，如 8080
	ProxyURL  string `json:"proxy_url"`  // 本地代理地址，如 http://127.0.0.1:64902
}

// BackendJSONPath 返回 backend.json 的完整路径。
func BackendJSONPath() string {
	return filepath.Join(BaseDir(), "backend.json")
}

// ReadBackendInfo 读取 backend.json，失败返回 nil。
func ReadBackendInfo() *BackendInfo {
	data, err := os.ReadFile(BackendJSONPath())
	if err != nil {
		return nil
	}
	var info BackendInfo
	if json.Unmarshal(data, &info) != nil {
		return nil
	}
	return &info
}

// WriteBackendInfo 写入 backend.json（自动创建目录）。
func WriteBackendInfo(info BackendInfo) error {
	dir := filepath.Dir(BackendJSONPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(BackendJSONPath(), data, 0o644)
}

// WriteProxyURL 将代理 URL 合并写入已有的 backend.json（保留后端的 host/port）。
func WriteProxyURL(proxyURL string) error {
	info := ReadBackendInfo()
	if info == nil {
		info = &BackendInfo{}
	}
	info.ProxyURL = proxyURL
	return WriteBackendInfo(*info)
}

// RemoveBackendInfo 删除 backend.json。
func RemoveBackendInfo() error {
	err := os.Remove(BackendJSONPath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// --- proxy.json ---

// ProxySession 记录一个活跃的代理会话。
type ProxySession struct {
	StartedAt string `json:"started_at"` // 会话启动时间（ISO 8601）
}

// ProxyJSONPath 返回 proxy.json 的完整路径。
func ProxyJSONPath() string {
	return filepath.Join(BaseDir(), "proxy.json")
}

var proxyMu sync.Mutex // 保护 proxy.json 并发读写

// ProxySessionKey 生成 proxy.json 中的唯一键：{projectSlug}_{sessionID}。
func ProxySessionKey(projectSlug, sessionID string) string {
	return projectSlug + "_" + sessionID
}

// RegisterProxySession 向 proxy.json 注册一个会话。
// key 格式为 {projectSlug}_{sessionID}，确保同一项目同一会话唯一。
func RegisterProxySession(key string, startedAt string) error {
	proxyMu.Lock()
	defer proxyMu.Unlock()

	sessions := readProxySessions()
	sessions[key] = ProxySession{StartedAt: startedAt}
	return writeProxySessions(sessions)
}

// UnregisterProxySession 从 proxy.json 注销一个会话。
func UnregisterProxySession(key string) error {
	proxyMu.Lock()
	defer proxyMu.Unlock()

	sessions := readProxySessions()
	delete(sessions, key)
	return writeProxySessions(sessions)
}

// ReadProxySessions 读取 proxy.json 中的所有会话（不加锁，供 watchdog 使用）。
func ReadProxySessions() map[string]ProxySession {
	proxyMu.Lock()
	defer proxyMu.Unlock()
	return readProxySessions()
}

// readProxySessions 内部读取函数，不加锁。
func readProxySessions() map[string]ProxySession {
	result := make(map[string]ProxySession)
	data, err := os.ReadFile(ProxyJSONPath())
	if err != nil {
		return result
	}
	json.Unmarshal(data, &result)
	return result
}

// writeProxySessions 内部写入函数，不加锁。
func writeProxySessions(sessions map[string]ProxySession) error {
	dir := filepath.Dir(ProxyJSONPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	logger.Debug("session_files", "proxy.json updated: %d sessions", len(sessions))
	return os.WriteFile(ProxyJSONPath(), data, 0o644)
}
