// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ProxyHandler 处理代理注册、trace-init 转发和代理列表查询。
type ProxyHandler struct {
	mu            sync.RWMutex
	proxies       map[string]string   // pid -> proxy_url，注册的代理列表（内存）
	sessions      map[string]string   // session_id -> proxy_url，会话到代理的精确路由映射
	proxySessions map[string][]string // pid -> []session_id，代理注销时清理映射
	traceMu       sync.Mutex          // trace-init 串行锁，确保一对一消费
	svc           *service.ProxyService // 代理列表服务（数据库）
}

// NewProxyHandler 创建代理处理器（无数据库依赖，仅内存功能）。
func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		proxies:       make(map[string]string),
		sessions:      make(map[string]string),
		proxySessions: make(map[string][]string),
	}
}

// NewProxyHandlerWithService 创建带数据库服务的代理处理器。
func NewProxyHandlerWithService(svc *service.ProxyService) *ProxyHandler {
	return &ProxyHandler{
		proxies:       make(map[string]string),
		sessions:      make(map[string]string),
		proxySessions: make(map[string][]string),
		svc:           svc,
	}
}

// bindSession 将会话绑定到代理 URL（内部调用需持有写锁）。
func (h *ProxyHandler) bindSession(pid, sessionID, proxyURL string) {
	// 解绑该 session_id 之前可能绑定的代理
	if oldURL, ok := h.sessions[sessionID]; ok && oldURL != proxyURL {
		logger.Debug("api.proxy", "rebind session %s: %s -> %s", sessionID, oldURL, proxyURL)
	}
	h.sessions[sessionID] = proxyURL
	// 记录到 proxy 索引，用于注销时清理
	found := false
	for _, sid := range h.proxySessions[pid] {
		if sid == sessionID {
			found = true
			break
		}
	}
	if !found {
		h.proxySessions[pid] = append(h.proxySessions[pid], sessionID)
	}
}

// unbindProxySessions 清理指定 pid 下的所有 session 绑定（内部调用需持有写锁）。
func (h *ProxyHandler) unbindProxySessions(pid string) {
	for _, sessionID := range h.proxySessions[pid] {
		if h.sessions[sessionID] == h.proxies[pid] {
			delete(h.sessions, sessionID)
			logger.Debug("api.proxy", "unbind session %s on proxy %s unregister", sessionID, pid)
		}
	}
	delete(h.proxySessions, pid)
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
	h.unbindProxySessions(req.PID)
	delete(h.proxies, req.PID)
	h.mu.Unlock()

	logger.Info("api.proxy", "unregistered: pid=%s", req.PID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TraceInit 转发 trace-init 请求到指定代理。
// POST /api/proxy/trace-init
// 路由优先级：
//  1. proxy_pid 精确路由（init 锁保证一对一）
//  2. session_id 已绑定代理的精确路由
//  3. 广播探测（兜底，proxy 侧 guard 防止误投）
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

	// 解析请求
	var tiReq TraceInitRequest
	if err := json.Unmarshal(body, &tiReq); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if tiReq.SessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	// 串行锁：确保同一时刻只处理一个 trace-init（配合 init PID 文件一对一消费）
	h.traceMu.Lock()
	defer h.traceMu.Unlock()

	// 优先级 1：proxy_pid 精确路由
	if tiReq.ProxyPID != "" {
		h.mu.RLock()
		proxyURL, found := h.proxies[tiReq.ProxyPID]
		h.mu.RUnlock()

		if found {
			logger.Debug("api.proxy", "trace-init: PID route session=%s to pid=%s url=%s", tiReq.SessionID, tiReq.ProxyPID, proxyURL)
			resp, err := relayTraceInit(proxyURL, body)
			if err == nil {
				h.mu.Lock()
				h.bindSession(tiReq.ProxyPID, tiReq.SessionID, proxyURL)
				h.mu.Unlock()
				logger.Info("api.proxy", "trace-init: session=%s bound to proxy pid=%s", tiReq.SessionID, tiReq.ProxyPID)
				writeJSON(w, http.StatusOK, resp)
				return
			}
			logger.Warn("api.proxy", "trace-init: PID route failed for pid=%s: %v, falling back", tiReq.ProxyPID, err)
		} else {
			logger.Warn("api.proxy", "trace-init: proxy_pid=%s not registered, falling back", tiReq.ProxyPID)
		}
	}

	// 优先级 2：session_id 已绑定代理
	h.mu.RLock()
	boundURL, bound := h.sessions[tiReq.SessionID]
	proxiesCopy := make(map[string]string, len(h.proxies))
	for pid, url := range h.proxies {
		proxiesCopy[pid] = url
	}
	h.mu.RUnlock()

	if bound {
		logger.Debug("api.proxy", "trace-init: direct route session=%s to %s", tiReq.SessionID, boundURL)
		resp, err := relayTraceInit(boundURL, body)
		if err == nil {
			writeJSON(w, http.StatusOK, resp)
			return
		}
		logger.Warn("api.proxy", "trace-init: bound proxy %s unreachable for session=%s: %v", boundURL, tiReq.SessionID, err)
	}

	if len(proxiesCopy) == 0 {
		logger.Warn("api.proxy", "trace-init: no proxies registered")
		http.Error(w, "no proxies registered", http.StatusServiceUnavailable)
		return
	}

	// 优先级 3：广播探测（兜底）
	for pid, proxyURL := range proxiesCopy {
		resp, err := relayTraceInit(proxyURL, body)
		if err != nil {
			logger.Debug("api.proxy", "trace-init relay to %s failed: %v", proxyURL, err)
			continue
		}
		h.mu.Lock()
		h.bindSession(pid, tiReq.SessionID, proxyURL)
		h.mu.Unlock()
		logger.Info("api.proxy", "trace-init: session=%s bound to proxy pid=%s url=%s (broadcast)", tiReq.SessionID, pid, proxyURL)
		writeJSON(w, http.StatusOK, resp)
		return
	}

	logger.Warn("api.proxy", "trace-init: all proxies failed for session=%s", tiReq.SessionID)
	http.Error(w, "all proxies unreachable", http.StatusBadGateway)
}

// List 处理获取代理列表的请求。
// GET /api/proxies
// 查询参数：status（可选）、project（可选）
func (h *ProxyHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	var filter store.ProxyFilter
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("project"); v != "" {
		filter.Project = &v
	}

	logger.Debug("api.proxy", "GET /api/proxies status=%v project=%v", filter.Status, filter.Project)

	proxies, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list proxies")
		return
	}

	// 确保返回空数组而非 null
	if proxies == nil {
		proxies = []store.Proxy{}
	}

	items := make([]ProxyItem, len(proxies))
	for i, p := range proxies {
		items[i] = ProxyItem{
			ProxyID:      p.ProxyID,
			ProjectSlug:  p.ProjectSlug,
			Status:       p.Status,
			RegisteredAt: p.RegisteredAt,
			LastPingAt:   p.LastPingAt,
		}
	}

	writeJSON(w, http.StatusOK, ProxiesResponse{
		Proxies: items,
		Total:   len(items),
	})
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
