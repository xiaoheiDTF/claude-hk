// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ProxyHandler 处理代理注册和 trace-init 转发。
type ProxyHandler struct {
	mu      sync.RWMutex
	proxies map[string]string // pid -> proxy_url，注册的代理列表
}

// NewProxyHandler 创建代理处理器。
func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		proxies: make(map[string]string),
	}
}

// Register 处理代理注册请求。
// POST /api/proxy/register
// Body: {"pid": "28364", "proxy_url": "http://127.0.0.1:64902"}
func (h *ProxyHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PID      string `json:"pid"`
		ProxyURL string `json:"proxy_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.PID == "" || req.ProxyURL == "" {
		http.Error(w, "pid and proxy_url required", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	h.proxies[req.PID] = req.ProxyURL
	h.mu.Unlock()

	logger.Info("api.proxy", "registered: pid=%s url=%s", req.PID, req.ProxyURL)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Unregister 处理代理注销请求。
// POST /api/proxy/unregister
// Body: {"pid": "28364"}
func (h *ProxyHandler) Unregister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PID string `json:"pid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	delete(h.proxies, req.PID)
	h.mu.Unlock()

	logger.Info("api.proxy", "unregistered: pid=%s", req.PID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TraceInit 转发 trace-init 请求到所有注册的代理。
// POST /api/proxy/trace-init
// 将请求体原样转发给每个代理的 /_internal/trace-init，返回第一个成功的响应。
func (h *ProxyHandler) TraceInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	// 获取所有注册的代理
	h.mu.RLock()
	urls := make([]string, 0, len(h.proxies))
	for _, url := range h.proxies {
		urls = append(urls, url)
	}
	h.mu.RUnlock()

	if len(urls) == 0 {
		logger.Warn("api.proxy", "trace-init: no proxies registered")
		http.Error(w, "no proxies registered", http.StatusServiceUnavailable)
		return
	}

	// 转发到每个代理，返回第一个成功的响应
	for _, proxyURL := range urls {
		resp, err := relayTraceInit(proxyURL, body)
		if err != nil {
			logger.Debug("api.proxy", "trace-init relay to %s failed: %v", proxyURL, err)
			continue
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	logger.Warn("api.proxy", "trace-init: all proxies failed")
	http.Error(w, "all proxies unreachable", http.StatusBadGateway)
}

// relayTraceInit 将 trace-init 请求转发到指定代理。
func relayTraceInit(proxyURL string, body []byte) (map[string]any, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", proxyURL+"/_internal/trace-init", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("proxy returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode proxy response: %w", err)
	}
	return result, nil
}
