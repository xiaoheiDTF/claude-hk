// Package proxy 提供 HTTP 代理功能，包括请求转发、SSE 流式处理与 Trace 记录。
package proxy

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/sse"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
)

const (
	maxUpstreamRetries = 5                    // 上游请求最大重试次数
	retryInterval      = 30 * time.Second     // 重试间隔
	proxyDialTimeout   = 10 * time.Second     // TCP 连接超时
	proxyTLSHandshake  = 10 * time.Second     // TLS 握手超时
	proxyResponseHdr   = 60 * time.Second     // 等待响应头超时
	proxyIdleConn      = 90 * time.Second     // 空闲连接超时
)

// ReverseProxy 拦截 HTTP 请求并转发到上游 API，同时记录请求/响应到 JSONL Trace，并处理 SSE 流式响应。
type ReverseProxy struct {
	target             string              // 上游 API 目标地址
	baseDir            string              // Trace 文件存放的基础目录
	writer             *trace.TraceWriter  // 当前会话的 Trace 写入器（由 _internal/trace-init 创建）
	client             *http.Client        // 转发请求的 HTTP 客户端
	turn               atomic.Int64        // 请求计数器，用于标记 Turn 编号
	server             *http.Server        // 本地代理 HTTP 服务器
	startOnce          sync.Once           // 确保 Start 只执行一次
	sessionID          string              // 当前会话 ID（由 trace-init 设置）
	projectSlug        string              // 当前项目标识（由 trace-init 设置）
	OnSessionInit      func(sessionID, projectSlug string) // 会话初始化回调（注册 proxy.json）
	OnSessionClose     func()                              // 会话关闭回调（注销 proxy.json，由 /_internal/session-close 触发）
	model              string              // 强制替换的模型名（空=不改写）
	upstreamAvailable  bool                // 上游可用性标记，初始 true
	fallbackConfigs    []*FallbackConfig   // 兜底配置列表（上游不可用时轮询）
	availableMu        sync.Mutex          // 保护 upstreamAvailable 和 fallbackConfigs 的并发安全
	actualAddr         string              // 实际监听地址（启动后设置）
	reasoningCache     *ReasoningCache     // reasoning_content 缓存（仅 kimi 模式）
	kimiMode           bool                // 是否为 kimi 上游
	fallbackIndex      int                 // fallback 轮询索引
	retryInterval      time.Duration       // 重试间隔（默认 retryInterval，测试可覆盖）
	// === 别名路由模式（aliases != nil 时启用，绕过下面的单一 target/fallback 状态机）===
	aliases      map[string]*Alias // name→别名；nil 表示处于 bypass（单一 target）模式
	aliasOrder   []string          // 别名数组顺序（fallback 顺序与 F6.2 可用列表用）
	defaultAlias string            // 请求 model 未命中时的兜底别名 name
}

// FallbackConfig 存储上游不可用时的兜底配置。
type FallbackConfig struct {
	BaseURL   string // 兜底上游 API 地址
	Model     string // 兜底模型名
	AuthToken string // 兜底认证 Token
	APIKey    string // 兜底 API Key
}

// Alias 是 proxy 侧的别名定义（由 main.go 从 config.Alias 转换而来，保持 proxy 包不依赖 config）。
// proxy 收到请求后按请求体 model（=别名 name）查表：改写为真实 model、选用对应 base_url/凭证转发。
type Alias struct {
	Name      string // 别名，Claude Code 发来的 model 名
	Model     string // 真实模型名，转发时改写进请求体
	BaseURL   string // 后端 API 地址
	APIKey    string // API Key（与 AuthToken 互斥；config 层已保证不会同时非空）
	AuthToken string // OAuth token（优先于 APIKey）
	Provider  string // anthropic/openai/gemini，决定 APIKey 的鉴权头格式
	KimiMode  *bool  // 显式指定 reasoning 注入；nil 时按 BaseURL 自动判断
}

// NewReverseProxy 创建一个代理实例，将请求转发到指定的 target URL。
// traceDir 为 Trace 文件存放的基础目录。
func NewReverseProxy(target, traceDir string) *ReverseProxy {
	logger.Debug("proxy", "new proxy: target=%s baseDir=%s", target, traceDir)
	return &ReverseProxy{
		target:            strings.TrimRight(target, "/"),
		baseDir:           traceDir,
		upstreamAvailable: true,
		client:            newProxyHTTPClient(),
	}
}

// newProxyHTTPClient 创建带 Transport 超时的 HTTP 客户端。
// 流式场景整体不设超时，但连接建立阶段有保护。
func newProxyHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 0, // 流式场景不设整体超时
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: proxyDialTimeout,
			}).DialContext,
			TLSHandshakeTimeout:   proxyTLSHandshake,
			ResponseHeaderTimeout: proxyResponseHdr,
			IdleConnTimeout:       proxyIdleConn,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不自动跟随重定向
		},
	}
}

// Start 启动代理 HTTP 服务器，监听指定的 host:port。
// 若 port 为 0，则自动选择可用端口。返回实际监听的端口号。
func (p *ReverseProxy) Start(host string, port int) (int, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.serveHTTP)
	p.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: mux,
	}

	listener, err := ioListener(p.server.Addr)
	if err != nil {
		return 0, fmt.Errorf("listen: %w", err)
	}

	// 在后台 goroutine 中启动服务
	go func() {
		if err := p.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Error("proxy", "server error: %v", err)
		}
	}()

	actualPort := listener.Addr().(*netTCPAddr).Port
	p.actualAddr = fmt.Sprintf("http://127.0.0.1:%d", actualPort)
	logger.Info("proxy", "listening on 127.0.0.1:%d", actualPort)
	return actualPort, nil
}

// Stop 优雅地关闭代理服务器，并关闭 Trace 写入器。
func (p *ReverseProxy) Stop() {
	logger.Info("proxy", "stopping")
	if p.writer != nil {
		p.writer.Close()
	}
	if p.server != nil {
		_ = p.server.Close()
	}
}

// URL 返回代理服务器的 HTTP 地址（如 http://127.0.0.1:8080）。
func (p *ReverseProxy) URL() string {
	return p.actualAddr
}

// serveHTTP 是代理的核心 HTTP 处理函数。
// 处理流程：请求预处理 → 尝试主上游 → 失败则尝试 fallback 链 → 全部失败则友好错误。
func (p *ReverseProxy) serveHTTP(w http.ResponseWriter, r *http.Request) {
	// 处理内部端点（绕过路径白名单）
	if strings.HasPrefix(r.URL.Path, "/_internal/") {
		p.handleInternal(w, r)
		return
	}

	// 拒绝未知路径
	if !IsAllowedPath(r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	// 读取请求体
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	// 保存原始请求体（改写 model 前），用于 fallback 时重新改写
	originalBody := make([]byte, len(bodyBytes))
	copy(originalBody, bodyBytes)

	// 解析请求体为 JSON（用于后续提取 model、stream 等字段）
	var reqBody any
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &reqBody)
	}

	// 请求体 model 改写（契约 3）
	bodyBytes = rewriteModel(bodyBytes, reqBody, p.model)

	// 重新解析改写后的请求体
	reqBody = nil
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &reqBody)
	}

	// 为 thinking 模式下的 assistant tool_call 消息注入 reasoning_content
	bodyBytes = injectReasoningContentCached(bodyBytes, reqBody, p.reasoningCache, p.kimiMode)
	reqBody = nil
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &reqBody)
	}

	// 生成请求 ID 与 Turn 编号
	reqID := fmt.Sprintf("req_%d", time.Now().UnixNano()%1e12)
	turn := int(p.turn.Add(1))
	t0 := time.Now()

	logger.Debug("proxy", "[Turn %d] request: %s %s (%d bytes)", turn, r.Method, r.URL.Path, len(bodyBytes))

	// 别名路由模式：按请求 model 查表改写 + 按别名选凭证 + 同真实 model 容错（F1/F2/F6）。
	// aliases != nil 即启用；否则走下面的 bypass（单一 target + 粘性 fallback 状态机）路径。
	if p.aliases != nil {
		p.serveAlias(w, r, originalBody, reqBody, reqID, turn, t0)
		return
	}

	// 读取当前状态快照
	p.availableMu.Lock()
	upstreamAvailable := p.upstreamAvailable
	fbConfigs := make([]*FallbackConfig, len(p.fallbackConfigs))
	copy(fbConfigs, p.fallbackConfigs)
	fbIdx := p.fallbackIndex
	p.availableMu.Unlock()
	hasFallback := len(fbConfigs) > 0

	// ========== Phase 1: 尝试主上游 ==========
	if upstreamAvailable {
		upstreamURL := p.target + r.URL.Path
		if r.URL.RawQuery != "" {
			upstreamURL += "?" + r.URL.RawQuery
		}
		logger.Info("proxy", "[Turn %d] -> %s %s (primary)", turn, r.Method, upstreamURL)

		upstreamReq, err := http.NewRequest(r.Method, upstreamURL, bytes.NewReader(bodyBytes))
		if err != nil {
			logProxyError(w, turn, err, 0)
			return
		}

		// 复制请求头，过滤逐跳头
		for k, vals := range r.Header {
			lower := strings.ToLower(k)
			if hopByHopHeaders[lower] {
				continue
			}
			for _, v := range vals {
				upstreamReq.Header.Add(k, v)
			}
		}
		upstreamReq.Header.Del("Host")

		resp, err := p.doWithRetry(upstreamReq, turn, hasFallback)
		if err != nil {
			// 连接错误 → 标记不可用，进入 fallback 链
			logger.Warn("proxy", "[Turn %d] primary connection error: %v", turn, err)
			p.markUnavailable()
			// fall through to fallback chain
		} else if !shouldFallback(resp.StatusCode) {
			// 主上游成功（或非降级错误如 400），直接处理响应
			defer resp.Body.Close()
			p.dispatchResponse(w, resp, reqID, turn, r, reqBody, t0, p.target)
			return
		} else {
			// 主上游返回需要降级的状态码 → 标记不可用，进入 fallback 链
			logger.Warn("proxy", "[Turn %d] primary returned %d, triggering fallback", turn, resp.StatusCode)
			resp.Body.Close()
			p.markUnavailable()
			// fall through to fallback chain
		}
	}

	// ========== Phase 2: 尝试 Fallback 链 ==========
	if hasFallback {
		for i := 0; i < len(fbConfigs); i++ {
			idx := (fbIdx + i) % len(fbConfigs)
			fb := fbConfigs[idx]

			// 从原始请求体重新改写 model
			fbBody := make([]byte, len(originalBody))
			copy(fbBody, originalBody)
			var fbParsed any
			if len(fbBody) > 0 {
				_ = json.Unmarshal(fbBody, &fbParsed)
			}
			if fb.Model != "" {
				fbBody = rewriteModel(fbBody, fbParsed, fb.Model)
				fbParsed = nil
				if len(fbBody) > 0 {
					_ = json.Unmarshal(fbBody, &fbParsed)
				}
			}
			fbBody = injectReasoningContentCached(fbBody, fbParsed, p.reasoningCache, IsKimiURL(fb.BaseURL))
			fbParsed = nil
			if len(fbBody) > 0 {
				_ = json.Unmarshal(fbBody, &fbParsed)
			}

			// 构建 fallback 请求
			fbURL := fb.BaseURL + r.URL.Path
			if r.URL.RawQuery != "" {
				fbURL += "?" + r.URL.RawQuery
			}
			logger.Info("proxy", "[Turn %d] -> %s %s (fallback[%d])", turn, r.Method, fbURL, idx)

			fbReq, err := http.NewRequest(r.Method, fbURL, bytes.NewReader(fbBody))
			if err != nil {
				p.advanceFallback()
				continue
			}

			for k, vals := range r.Header {
				lower := strings.ToLower(k)
				if hopByHopHeaders[lower] {
					continue
				}
				for _, v := range vals {
					fbReq.Header.Add(k, v)
				}
			}
			fbReq.Header.Del("Host")

			// 替换认证头
			if fb.APIKey != "" {
				fbReq.Header.Set("x-api-key", fb.APIKey)
			}
			if fb.AuthToken != "" {
				fbReq.Header.Set("Authorization", "Bearer "+fb.AuthToken)
			}

			// 发送请求
			fbResp, fbErr := p.client.Do(fbReq)
			if fbErr != nil {
				logger.Warn("proxy", "[Turn %d] fallback[%d] connection error: %v", turn, idx, fbErr)
				p.advanceFallback()
				continue
			}

			if shouldFallback(fbResp.StatusCode) {
				fbResp.Body.Close()
				logger.Warn("proxy", "[Turn %d] fallback[%d] returned %d, trying next", turn, idx, fbResp.StatusCode)
				p.advanceFallback()
				continue
			}

			// Fallback 成功！更新索引
			p.availableMu.Lock()
			p.fallbackIndex = idx
			p.availableMu.Unlock()
			logger.Info("proxy", "[Turn %d] fallback[%d] succeeded (status %d)", turn, idx, fbResp.StatusCode)

			defer fbResp.Body.Close()
			p.dispatchResponse(w, fbResp, reqID, turn, r, fbParsed, t0, fb.BaseURL)
			return
		}
	}

	// ========== Phase 3: 所有目标失败 → 日志错误 ==========
	logProxyError(w, turn, fmt.Errorf("all targets failed"), 0)
}

// handleStreaming 处理 SSE 流式响应：
//   - 复制响应头并写入客户端
//   - 逐块读取上游响应体并实时转发给客户端
//   - 使用 SSE 重组器收集完整事件
//   - 将重组后的结果写入 Trace
func (p *ReverseProxy) handleStreaming(
	w http.ResponseWriter,
	upstreamResp *http.Response,
	reqID string, turn, durationMs int,
	r *http.Request, reqBody any,
	t0 time.Time, model, upstreamBaseURL string,
) {
	// 复制响应头
	respHeaders := FilterHeaders(upstreamResp.Header, false)
	for k, vals := range respHeaders {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(upstreamResp.StatusCode)

	flusher, canFlush := w.(http.Flusher)
	reassembler := sse.NewSSEReassembler()

	// 逐块读取并转发流式数据
	buf := make([]byte, 32*1024)
	for {
		n, err := upstreamResp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			if canFlush {
				flusher.Flush()
			}
			reassembler.FeedBytes(buf[:n])
		}
		if err != nil {
			break
		}
	}

	durationMs = int(time.Since(t0).Milliseconds())
	reconstructed := reassembler.Reconstruct()
	logger.Debug("proxy", "[Turn %d] stream done: %d SSE events, %dms", turn, len(reassembler.Events), durationMs)

	// kimi 模式下缓存 SSE 响应中的 reasoning_content
	if p.kimiMode && reconstructed != nil {
		p.cacheFromResponse(reconstructed)
	}

	// 构建 Trace 记录
	record := buildRecord(
		reqID, turn, durationMs,
		r.Method, r.URL.RequestURI(),
		r.Header, reqBody,
		upstreamResp.StatusCode, upstreamResp.Header,
		reconstructed,
		reassembler.Events,
		upstreamBaseURL,
	)

	if w := p.getWriter(); w != nil {
		if err := w.Write(record); err != nil {
			logger.Error("proxy", "[Turn %d] trace write error: %v", turn, err)
		}
	} else {
		logger.Warn("proxy", "[Turn %d] trace skipped: writer not initialized", turn)
	}

	logger.Info("proxy", "[Turn %d] <- %d stream done (%dms, model=%s)", turn, upstreamResp.StatusCode, durationMs, model)
}

// handleNonStreaming 处理普通（非流式）响应：
//   - 读取完整响应体
//   - 按需解压（gzip / deflate）
//   - 解析 JSON 响应体
//   - 写入 Trace 并返回给客户端
func (p *ReverseProxy) handleNonStreaming(
	w http.ResponseWriter,
	upstreamResp *http.Response,
	reqID string, turn, durationMs int,
	r *http.Request, reqBody any, model, upstreamBaseURL string,
) {
	respBytes, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		http.Error(w, "read upstream: "+err.Error(), http.StatusBadGateway)
		return
	}

	// 按需解压响应体
	decodeBytes := respBytes
	encoding := strings.ToLower(upstreamResp.Header.Get("Content-Encoding"))
	if len(respBytes) > 0 && (encoding == "gzip" || encoding == "deflate") {
		var reader io.ReadCloser
		if encoding == "gzip" {
			reader, _ = gzip.NewReader(bytes.NewReader(respBytes))
		} else {
			reader, _ = zlib.NewReader(bytes.NewReader(respBytes))
		}
		if reader != nil {
			if decoded, err := io.ReadAll(reader); err == nil {
				logger.Debug("proxy", "[Turn %d] decompressed: %d -> %d bytes (%s)", turn, len(respBytes), len(decoded), encoding)
				decodeBytes = decoded
			}
			_ = reader.Close()
		}
	}

	// 解析解压后的 JSON 响应体
	var respBody any
	if len(decodeBytes) > 0 {
		_ = json.Unmarshal(decodeBytes, &respBody)
	}

	// kimi 模式下缓存响应中的 reasoning_content
	if p.kimiMode {
		if respMap, ok := respBody.(map[string]any); ok {
			p.cacheFromResponse(respMap)
		}
	}

	// 构建 Trace 记录
	record := buildRecord(
		reqID, turn, durationMs,
		r.Method, r.URL.RequestURI(),
		r.Header, reqBody,
		upstreamResp.StatusCode, upstreamResp.Header,
		respBody,
		nil,
		upstreamBaseURL,
	)

	if w := p.getWriter(); w != nil {
		if err := w.Write(record); err != nil {
			logger.Error("proxy", "[Turn %d] trace write error: %v", turn, err)
		}
	} else {
		logger.Warn("proxy", "[Turn %d] trace skipped: writer not initialized", turn)
	}

	// 将响应返回给客户端
	respHeaders := FilterHeaders(upstreamResp.Header, false)
	for k, vals := range respHeaders {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(upstreamResp.StatusCode)
	_, _ = w.Write(respBytes)

	logger.Info("proxy", "[Turn %d] <- %d (%dms, %d bytes, model=%s)", turn, upstreamResp.StatusCode, durationMs, len(respBytes), model)
}

// buildRecord 构建一条 Trace 记录，包含请求/响应的完整信息。
func buildRecord(
	reqID string, turn, durationMs int,
	method, path string,
	reqHeaders http.Header, reqBody any,
	status int, respHeaders http.Header,
	respBody any,
	sseEvents []map[string]any,
	upstreamBaseURL string,
) map[string]any {
	record := map[string]any{
		"timestamp":   time.Now().UTC().Format("2006-01-02T15:04:05.000000Z"),
		"request_id":  reqID,
		"session_id":  extractSessionID(reqBody),
		"turn":        turn,
		"duration_ms": durationMs,
		"request": map[string]any{
			"method":  method,
			"path":    path,
			"headers": HeadersToMap(FilterHeaders(reqHeaders, true)),
			"body":    reqBody,
		},
		"response": map[string]any{
			"status":  status,
			"headers": HeadersToMap(FilterHeaders(respHeaders, true)),
			"body":    respBody,
		},
	}
	if len(sseEvents) > 0 {
		record["response"].(map[string]any)["sse_events"] = sseEvents
	}
	if upstreamBaseURL != "" {
		record["upstream_base_url"] = upstreamBaseURL
	}
	return record
}

// extractSessionID 从请求体的 metadata.user_id 字段中提取 session_id。
// Claude Code 发送的格式为：metadata.user_id = "{\"session_id\": \"uuid\", ...}"
func extractSessionID(reqBody any) string {
	body, ok := reqBody.(map[string]any)
	if !ok {
		return ""
	}
	metadata, _ := body["metadata"].(map[string]any)
	if metadata == nil {
		return ""
	}
	userIDRaw, _ := metadata["user_id"].(string)
	if userIDRaw == "" {
		return ""
	}
	var userData map[string]string
	if json.Unmarshal([]byte(userIDRaw), &userData) != nil {
		return ""
	}
	return userData["session_id"]
}

// handleInternal 分发内部代理端点请求。
func (p *ReverseProxy) handleInternal(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/_internal/trace-init":
		p.handleTraceInit(w, r)
	case "/_internal/session-close":
		p.handleSessionClose(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleSessionClose 注销 proxy.json 中的会话条目（由 29-session-end 钩子触发）。
// 与 trace-init 对称：注册走钩子→trace-init，注销走钩子→session-close，
// 不再单靠 proxy 进程的 defer（进程被强杀时 defer 不执行，会残留条目）。
func (p *ReverseProxy) handleSessionClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	if p.OnSessionClose != nil {
		p.OnSessionClose()
	}
	logger.Info("proxy", "session-close: proxy.json sessions unregistered")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleTraceInit 接收 SessionStart 钩子传来的会话信息，创建 Trace 写入器。
func (p *ReverseProxy) handleTraceInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID      string `json:"session_id"`
		MachineID      string `json:"machine_id"`
		ProjectSlug    string `json:"project_slug"`
		TranscriptPath string `json:"transcript_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("proxy", "trace-init: invalid JSON: %v", err)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	// 允许 session 重新绑定（/clear 后新 session 取代旧 session）
	if p.sessionID != "" && p.sessionID != req.SessionID {
		logger.Debug("proxy", "trace-init: rebinding session %s -> %s", p.sessionID, req.SessionID)
	}

	logger.Debug("proxy", "trace-init: session=%s machine=%s slug=%s", req.SessionID, req.MachineID, req.ProjectSlug)

	// 若请求中未提供，则通过 trace 包自动检测
	machineID := req.MachineID
	if machineID == "" {
		machineID = trace.MachineID()
	}

	projectSlug := req.ProjectSlug
	if projectSlug == "" {
		projectSlug = trace.ExtractProjectSlug(req.TranscriptPath)
		if projectSlug == "" {
			projectSlug = trace.DetectProjectName()
		}
	}

	// 创建会话专属的 Trace 写入器
	tracePath := trace.NewSessionTracePath(p.baseDir, machineID, projectSlug, req.SessionID)
	writer, err := trace.NewTraceWriter(tracePath)
	p.sessionID = req.SessionID   // 记录当前会话 ID
	p.projectSlug = projectSlug   // 记录当前项目标识
	if err != nil {
		logger.Error("proxy", "trace-init: failed to create writer: %v", err)
		http.Error(w, "failed to create trace file", http.StatusInternalServerError)
		return
	}

		// 关闭之前的写入器（如果存在，如 session resume 场景）
		if p.writer != nil {
			p.writer.Close()
			logger.Debug("proxy", "previous writer closed")
		}
	p.writer = writer
	logger.Info("proxy", "trace initialized: %s", tracePath)

	// 通知外部注册 proxy.json
	if p.OnSessionInit != nil {
		p.OnSessionInit(req.SessionID, projectSlug)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "ok",
		"trace_path": tracePath,
	})
}

// getWriter 返回当前会话的 Trace 写入器。
// trace-init 调用前返回 nil（由 SessionStart hook 触发，一定在首个 API 请求之前）。
func (p *ReverseProxy) getWriter() *trace.TraceWriter {
	return p.writer
}

// Summary 返回当前 Trace 写入器的聚合统计信息。
// 若无有效写入器，返回零值统计。
func (p *ReverseProxy) Summary() map[string]any {
	if w := p.getWriter(); w != nil {
		return w.Summary()
	}
	return map[string]any{
		"api_calls":           0,
		"input_tokens":        int64(0),
		"output_tokens":       int64(0),
		"cache_read_tokens":   int64(0),
		"cache_create_tokens": int64(0),
	}
}

// SessionID 返回当前会话的 session_id。
// 优先返回 trace-init 设置的值，其次回退到写入器中提取的值。
func (p *ReverseProxy) SessionID() string {
	if p.sessionID != "" {
		return p.sessionID
	}
	if w := p.getWriter(); w != nil {
		return w.SessionID()
	}
	return ""
}

// ProjectSlug 返回当前项目的 project_slug（由 trace-init 设置）。
func (p *ReverseProxy) ProjectSlug() string {
	return p.projectSlug
}

// TracePath 返回当前追踪文件的完整路径。
func (p *ReverseProxy) TracePath() string {
	if w := p.getWriter(); w != nil {
		return w.Path()
	}
	return ""
}

// doWithRetry 向上游发送请求，连接错误或 5xx 时进行重试。
// 如果 hasFallback 为 true（有兜底配置），不重试直接返回错误，由调用方切换兜底。
// 如果 hasFallback 为 false（无兜底配置），重试 maxUpstreamRetries 次，间隔 retryInterval。
func (p *ReverseProxy) doWithRetry(req *http.Request, turn int, hasFallback bool) (*http.Response, error) {
	var lastErr error
	var savedBody []byte

	maxRetries := maxUpstreamRetries
	if hasFallback {
		maxRetries = 0 // 有兜底时不重试，让 serveHTTP 立即切换
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			interval := p.retryInterval
			if interval == 0 {
				interval = retryInterval
			}
			logger.Warn("proxy", "[Turn %d] retry %d/%d after %v", turn, attempt, maxUpstreamRetries, interval)
			time.Sleep(interval)
		}

		// 第一次读取时保存 body，后续复用
		if savedBody == nil && req.Body != nil {
			savedBody, _ = io.ReadAll(req.Body)
			req.Body.Close()
		}

		retryReq, err := http.NewRequest(req.Method, req.URL.String(), bytes.NewReader(savedBody))
		if err != nil {
			return nil, err
		}
		retryReq.Header = req.Header.Clone()

		resp, err := p.client.Do(retryReq)
		if err != nil {
			lastErr = err
			logger.Warn("proxy", "[Turn %d] attempt %d failed: %v", turn, attempt+1, err)
			if attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("all %d retries exhausted: %w", maxRetries, lastErr)
		}

		// 5xx 错误触发重试，4xx 不重试；最后一次直接返回（让 serveHTTP 透传）
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("upstream returned %d", resp.StatusCode)
			if attempt < maxRetries {
				logger.Warn("proxy", "[Turn %d] attempt %d: status %d, will retry", turn, attempt+1, resp.StatusCode)
				resp.Body.Close()
				continue
			}
			// 最后一次尝试，直接返回响应（serveHTTP 会标记不可用并透传）
			return resp, nil
		}

		return resp, nil
	}

	return nil, fmt.Errorf("all %d retries exhausted: %w", maxRetries, lastErr)
}

// shouldFallback reports whether the HTTP status code should trigger fallback to the next profile.
// 401 (认证失败)、403 (访问被拒)、429 (频率超限)、5xx (服务器错误) 均触发降级。
func shouldFallback(statusCode int) bool {
	return statusCode == http.StatusUnauthorized ||
		statusCode == http.StatusForbidden ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}

// logProxyError 将友好中文提示输出到日志，同时返回 Anthropic API 格式的 JSON 错误响应。
// 必须返回 JSON 而非纯文本，否则 Claude Code 解析会崩溃。
func logProxyError(w http.ResponseWriter, turn int, lastErr error, lastStatusCode int) {
	msg := "上游服务暂时不可用，请稍后重试"
	switch {
	case lastStatusCode == http.StatusForbidden:
		msg = "API 访问被拒绝 (403)，可能是认证信息过期或无权限，已尝试所有可用配置"
	case lastStatusCode == http.StatusUnauthorized:
		msg = "API 认证失败 (401)，请检查 API Key 或 Token 配置，已尝试所有可用配置"
	case lastStatusCode == http.StatusTooManyRequests:
		msg = "API 请求频率超限 (429)，所有配置均已限流，请稍后重试"
	case lastErr != nil:
		msg = "上游连接失败: " + lastErr.Error()
	}

	logger.Error("proxy", "[Turn %d] %s", turn, msg)

	statusCode := lastStatusCode
	if statusCode == 0 {
		statusCode = http.StatusBadGateway
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    "api_error",
			"message": msg,
		},
	})
}

// dispatchResponse extracts model/stream info from reqBody and dispatches to the appropriate handler.
// upstreamBaseURL 为本次实际命中的上游地址，写入 trace 的 upstream_base_url 字段。
func (p *ReverseProxy) dispatchResponse(w http.ResponseWriter, resp *http.Response, reqID string, turn int, r *http.Request, reqBody any, t0 time.Time, upstreamBaseURL string) {
	durationMs := int(time.Since(t0).Milliseconds())

	model := ""
	isStreaming := false
	if m, ok := reqBody.(map[string]any); ok {
		if v, ok := m["model"].(string); ok {
			model = v
		}
		if s, ok := m["stream"].(bool); ok {
			isStreaming = s
		}
	}

	if isStreaming {
		p.handleStreaming(w, resp, reqID, turn, durationMs, r, reqBody, t0, model, upstreamBaseURL)
	} else {
		p.handleNonStreaming(w, resp, reqID, turn, durationMs, r, reqBody, model, upstreamBaseURL)
	}
}

// markUnavailable 标记上游不可用（契约 4）。
// 首次失败后置 upstreamAvailable = false，进程生命周期内不再恢复。
func (p *ReverseProxy) markUnavailable() {
	p.availableMu.Lock()
	defer p.availableMu.Unlock()
	if p.upstreamAvailable {
		p.upstreamAvailable = false
		logger.Warn("proxy", "upstream marked unavailable, switching to fallback")
	}
}

// SetModel 设置强制替换的模型名。
func (p *ReverseProxy) SetModel(model string) {
	p.model = model
}

// SetFallbackConfigs 设置兜底配置列表，支持多 profile 轮询。
func (p *ReverseProxy) SetFallbackConfigs(cfgs []*FallbackConfig) {
	p.availableMu.Lock()
	defer p.availableMu.Unlock()
	p.fallbackConfigs = cfgs
	p.fallbackIndex = 0
}

// currentFallbackConfig 返回当前轮询的 fallback 配置。
func (p *ReverseProxy) currentFallbackConfig() *FallbackConfig {
	if len(p.fallbackConfigs) == 0 {
		return nil
	}
	idx := p.fallbackIndex % len(p.fallbackConfigs)
	return p.fallbackConfigs[idx]
}

// advanceFallback 切换到下一个 fallback 配置（轮询）。
func (p *ReverseProxy) advanceFallback() {
	p.availableMu.Lock()
	defer p.availableMu.Unlock()
	p.fallbackIndex++
	logger.Info("proxy", "fallback advanced to index %d", p.fallbackIndex)
}

// SetAliases 装配别名表，启用别名路由模式。
// 别名表为启动时一次性加载（本期不支持热重载）；重复 name 由调用方去重（后者覆盖）。
// 若任一别名为 kimi（显式或按 base_url 自动判断），预先创建 reasoning_content 缓存。
func (p *ReverseProxy) SetAliases(aliases []*Alias, defaultAlias string) {
	p.aliases = make(map[string]*Alias, len(aliases))
	p.aliasOrder = make([]string, 0, len(aliases))
	needKimi := false
	for _, a := range aliases {
		if a == nil {
			continue
		}
		p.aliases[a.Name] = a
		p.aliasOrder = append(p.aliasOrder, a.Name)
		if aliasKimiMode(a) {
			needKimi = true
		}
	}
	p.defaultAlias = defaultAlias
	if needKimi && p.reasoningCache == nil {
		p.reasoningCache = NewReasoningCache()
		logger.Info("proxy", "reasoning_content cache enabled (alias mode)")
	}
	logger.Info("proxy", "alias routing enabled: %d aliases, default_alias=%q", len(p.aliasOrder), defaultAlias)
}

// aliasKimiMode 计算别名的 kimi 注入开关：显式值优先，缺省按 base_url 自动判断（复用 IsKimiURL）。
func aliasKimiMode(a *Alias) bool {
	if a.KimiMode != nil {
		return *a.KimiMode
	}
	return IsKimiURL(a.BaseURL)
}

// serveAlias 是别名路由模式的核心处理：解析别名 → 主别名 + 同真实 model 候选链 → 逐个转发。
func (p *ReverseProxy) serveAlias(w http.ResponseWriter, r *http.Request, originalBody []byte, reqBody any, reqID string, turn int, t0 time.Time) {
	reqModel := extractModel(reqBody)

	primary, hit := p.aliases[reqModel]
	if !hit {
		if p.defaultAlias != "" {
			primary = p.aliases[p.defaultAlias]
			logger.Warn("proxy", "[Turn %d] model %q 未命中别名，兜底 default_alias=%q", turn, reqModel, p.defaultAlias)
		}
	}
	if primary == nil {
		// F6.2：打印原始 model 与可用别名列表，便于定位（如 Claude Code 是否剥离了 [1m]）
		logger.Warn("proxy", "[Turn %d] model %q 未命中任何别名且无 default_alias，可用别名: %v", turn, reqModel, p.aliasOrder)
		logProxyError(w, turn, fmt.Errorf("unknown model %q: no matching alias and no default_alias", reqModel), 0)
		return
	}
	if hit {
		logger.Info("proxy", "[Turn %d] alias=%s real_model=%s base_url=%s key=%s",
			turn, primary.Name, primary.Model, primary.BaseURL, maskKey(primary.AuthToken, primary.APIKey))
	}

	// 候选链：主别名在前，其后是同真实 model 的其他别名（按数组顺序，F2.2）。
	candidates := []*Alias{primary}
	candidates = append(candidates, p.fallbackAliasesFor(primary)...)

	var lastStatus int
	for _, a := range candidates {
		dispatched, status := p.forwardAlias(w, r, a, originalBody, reqID, turn, t0)
		if dispatched {
			return
		}
		if status > lastStatus {
			lastStatus = status
		}
	}

	logger.Error("proxy", "[Turn %d] model %q 所有别名（含候选）均失败", turn, reqModel)
	logProxyError(w, turn, fmt.Errorf("all aliases failed for model %q", reqModel), lastStatus)
}

// fallbackAliasesFor 返回真实 model 与 primary 相同、排除 primary 自身的别名，按数组顺序。
func (p *ReverseProxy) fallbackAliasesFor(primary *Alias) []*Alias {
	var out []*Alias
	for _, name := range p.aliasOrder {
		a := p.aliases[name]
		if a == nil || a.Name == primary.Name || primary.Model == "" {
			continue
		}
		if a.Model == primary.Model {
			out = append(out, a)
		}
	}
	return out
}

// forwardAlias 按单个别名转发一次请求。
// 返回 (dispatched, lastStatus)：dispatched=true 表示已向客户端写出响应（成功或非降级状态如 400），调用方应停止重试；
// dispatched=false 表示命中降级条件（连接错误或 401/403/429/5xx），调用方应尝试下一个候选。
func (p *ReverseProxy) forwardAlias(w http.ResponseWriter, r *http.Request, a *Alias, originalBody []byte, reqID string, turn int, t0 time.Time) (bool, int) {
	// 改写 model 为真实 model
	body := originalBody
	var parsed any
	if len(body) > 0 {
		_ = json.Unmarshal(body, &parsed)
	}
	body = rewriteModel(body, parsed, a.Model)
	parsed = nil
	if len(body) > 0 {
		_ = json.Unmarshal(body, &parsed)
	}

	// 按该别名注入 reasoning_content（kimi 系）
	body = injectReasoningContentCached(body, parsed, p.reasoningCache, aliasKimiMode(a))
	parsed = nil
	if len(body) > 0 {
		_ = json.Unmarshal(body, &parsed)
	}

	// 构建上游请求
	upstreamURL := strings.TrimRight(a.BaseURL, "/") + r.URL.Path
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}
	logger.Info("proxy", "[Turn %d] -> %s %s (alias=%s real_model=%s)",
		turn, r.Method, upstreamURL, a.Name, a.Model)
	upstreamReq, err := http.NewRequest(r.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		logger.Warn("proxy", "[Turn %d] alias %s build request error: %v", turn, a.Name, err)
		return false, 0
	}
	for k, vals := range r.Header {
		if hopByHopHeaders[strings.ToLower(k)] {
			continue
		}
		for _, v := range vals {
			upstreamReq.Header.Add(k, v)
		}
	}
	upstreamReq.Header.Del("Host")
	applyAliasAuth(upstreamReq.Header, a)

	resp, err := p.client.Do(upstreamReq)
	if err != nil {
		logger.Warn("proxy", "[Turn %d] alias %s connection error: %v", turn, a.Name, err)
		return false, 0
	}
	if shouldFallback(resp.StatusCode) {
		resp.Body.Close()
		logger.Warn("proxy", "[Turn %d] alias %s returned %d, trying next", turn, a.Name, resp.StatusCode)
		return false, resp.StatusCode
	}

	logger.Info("proxy", "[Turn %d] alias %s <- %d", turn, a.Name, resp.StatusCode)
	defer resp.Body.Close()
	p.dispatchResponse(w, resp, reqID, turn, r, parsed, t0, a.BaseURL)
	return true, resp.StatusCode
}

// applyAliasAuth 按别名凭证与 provider 设置鉴权头（覆盖 Claude Code 自带的占位凭证）。
// 决策 5：auth_token 优先于 api_key；provider 决定 api_key 走 x-api-key 还是 Bearer。
func applyAliasAuth(h http.Header, a *Alias) {
	h.Del("Authorization")
	h.Del("x-api-key")
	switch {
	case a.AuthToken != "":
		h.Set("Authorization", "Bearer "+a.AuthToken)
	case a.APIKey != "":
		if strings.EqualFold(a.Provider, "openai") || strings.EqualFold(a.Provider, "gemini") {
			h.Set("Authorization", "Bearer "+a.APIKey)
		} else {
			h.Set("x-api-key", a.APIKey)
		}
	}
}

// extractModel 从请求体解析 model 字段（即 Claude Code 发来的别名 name）。
func extractModel(reqBody any) string {
	m, ok := reqBody.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := m["model"].(string)
	return s
}

// maskKey 脱敏凭证用于日志：保留前 4 后 4，过短则全掩。
func maskKey(token, key string) string {
	s := token
	if s == "" {
		s = key
	}
	if s == "" {
		return "<none>"
	}
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + "..." + s[len(s)-4:]
}

// SetKimiMode 启用或禁用 kimi 模式。
// 启用时会自动创建 reasoning_content 缓存。
func (p *ReverseProxy) SetKimiMode(kimi bool) {
	p.kimiMode = kimi
	if kimi && p.reasoningCache == nil {
		p.reasoningCache = NewReasoningCache()
		logger.Info("proxy", "reasoning_content mode enabled (kimi/deepseek), caching active")
	}
}

// reasoningPrefixes 定义需要 reasoning_content 注入的上游 URL 前缀（scheme + host）。
var reasoningPrefixes = []string{
	"https://api.kimi.com",
	"https://api.moonshot.cn",
	"https://api.deepseek.com",
}

// IsKimiURL 判断 URL 是否为需要 reasoning_content 注入的上游（kimi/moonshot/deepseek）。
// 基于 URL 前缀匹配，避免子串误判。
func IsKimiURL(url string) bool {
	lower := strings.ToLower(url)
	for _, prefix := range reasoningPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// cacheFromResponse 从 API 响应快照中提取 reasoning_content 并存入缓存。
//
// 同时处理两种响应格式：
//   - OpenAI Chat Completions: snap["choices"][0]["message"] 包含 reasoning_content/content/tool_calls
//   - Anthropic: snap["content"] 包含 thinking 块和 tool_use 块
//
// 仅在 kimi 模式下调用。
func (p *ReverseProxy) cacheFromResponse(snap map[string]any) {
	if p.reasoningCache == nil || snap == nil {
		return
	}

	// 提取 reasoning_content 和消息内容
	var reasoningContent string
	var msgContent string    // 文本内容（用于 full key）
	var toolCallsJSON string // tool_calls JSON（用于 tc key 和 full key）

	// 路径 1: OpenAI Chat Completions 格式
	choices, _ := snap["choices"].([]any)
	if len(choices) > 0 {
		choice, _ := choices[0].(map[string]any)
		if choice != nil {
			msg, _ := choice["message"].(map[string]any)
			if msg != nil {
				reasoningContent, _ = msg["reasoning_content"].(string)
				msgContent, _ = msg["content"].(string)
				if tc, ok := msg["tool_calls"]; ok && tc != nil {
					if tcBytes, err := json.Marshal(tc); err == nil {
						toolCallsJSON = string(tcBytes)
					}
				}
			}
		}
	}

	// 路径 2: Anthropic 格式（从 content 数组中的 thinking 块提取）
	if reasoningContent == "" {
		contentArr, _ := snap["content"].([]any)
		for _, blockRaw := range contentArr {
			block, ok := blockRaw.(map[string]any)
			if !ok {
				continue
			}
			if blockType, _ := block["type"].(string); blockType == "thinking" {
				if thinking, ok := block["thinking"].(string); ok && thinking != "" {
					reasoningContent = thinking
					break
				}
			}
		}
		// 同时从 Anthropic 格式提取 tool_use 块用于 key 计算
		if toolCallsJSON == "" {
			var toolUseBlocks []any
			for _, blockRaw := range contentArr {
				block, ok := blockRaw.(map[string]any)
				if !ok {
					continue
				}
				if blockType, _ := block["type"].(string); blockType == "tool_use" {
					toolUseBlocks = append(toolUseBlocks, block)
				}
			}
			if len(toolUseBlocks) > 0 {
				if tcBytes, err := json.Marshal(toolUseBlocks); err == nil {
					toolCallsJSON = string(tcBytes)
				}
			}
		}
	}

	if reasoningContent == "" {
		return // 无 reasoning_content 可缓存
	}

	fullKey := MakeFullKey(msgContent, toolCallsJSON)
	tcKey := MakeToolcallKey(toolCallsJSON)
	p.reasoningCache.Store(fullKey, tcKey, reasoningContent)

	f, t := p.reasoningCache.Len()
	logger.Debug("proxy", "cached reasoning_content: rc_len=%d full=%d tc=%d", len(reasoningContent), f, t)
}

// rewriteModel 改写请求体 JSON 中的 model 字段（契约 3）。
// 如果 model 为空、body 为空、body 非 JSON 或 body 为 JSON 非对象，原样返回。
func rewriteModel(body []byte, parsed any, model string) []byte {
	if model == "" || len(body) == 0 {
		return body
	}

	m, ok := parsed.(map[string]any)
	if !ok {
		return body // 非 JSON 或非对象
	}

	m["model"] = model
	rewritten, err := json.Marshal(m)
	if err != nil {
		return body // 改写失败，原样返回
	}
	return rewritten
}

// injectReasoningContentCached 为 assistant tool_call 消息注入 reasoning_content 补丁。
//
// Kimi 等兼容 Anthropic 的端点要求：assistant tool_call 消息必须携带顶层字段 reasoning_content，
// 以及 content 数组中的 thinking 块。Claude Code 发送 tool_call 消息时不包含这些字段，导致上游 400 错误：
//
//	400 thinking is enabled but reasoning_content is missing in assistant tool call message at index N
//
// 行为随 isKimi 参数变化：
//   - isKimi=false: 不做任何修改，直接透传
//   - isKimi=true + cache!=nil: 使用缓存的三级查找回填真实 reasoning_content
//   - isKimi=true + cache==nil: 注入空字符串（兜底行为，空字符串无害但能防止报错）
func injectReasoningContentCached(body []byte, parsed any, cache *ReasoningCache, isKimi bool) []byte {
	if !isKimi {
		return body // 非 kimi 上游不做任何注入
	}

	if len(body) == 0 {
		return body
	}

	m, ok := parsed.(map[string]any)
	if !ok {
		return body // 非 JSON 或非对象
	}

	// 获取 messages 数组（无 messages 则无需处理）
	messages, ok := m["messages"].([]any)
	if !ok || len(messages) == 0 {
		return body
	}

	modified := false
	for i, msgRaw := range messages {
		msg, ok := msgRaw.(map[string]any)
		if !ok {
			continue
		}

		// 只处理 assistant 消息
		role, _ := msg["role"].(string)
		if role != "assistant" {
			continue
		}

		// 检查 content 是否为数组
		content, ok := msg["content"].([]any)
		if !ok || len(content) == 0 {
			continue
		}

		// 检查是否有 tool_use 块、thinking 块、reasoning_content 字段
		hasToolUse := false
		hasThinking := false
		for _, blockRaw := range content {
			block, ok := blockRaw.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := block["type"].(string)
			if blockType == "tool_use" {
				hasToolUse = true
			}
			if blockType == "thinking" {
				hasThinking = true
			}
		}

		// 检查已有的 reasoning_content
		existingRC, hasReasoningContent := msg["reasoning_content"].(string)

		// 只处理含 tool_use 的 assistant 消息
		if !hasToolUse {
			continue
		}

		needPatch := false

		// 如果已有非空的 reasoning_content，用它刷新缓存
		if hasReasoningContent && existingRC != "" && cache != nil {
			fullKey := MakeFullKeyFromMsg(msg)
			tcKey := MakeToolcallKeyFromMsg(msg)
			cache.Store(fullKey, tcKey, existingRC)
			// 已有完整 reasoning_content，不需要补丁
			// 但仍需确保 thinking 块存在
		}

		// 补丁 1: content 数组缺少 thinking 块 → 在开头插入
		if !hasThinking {
			thinkingBlock := map[string]any{
				"type":     "thinking",
				"thinking": "",
			}
			newContent := make([]any, 0, len(content)+1)
			newContent = append(newContent, thinkingBlock)
			newContent = append(newContent, content...)
			msg["content"] = newContent
			needPatch = true
		}

		// 补丁 2: 消息缺少顶层 reasoning_content 字段 → 尝试从缓存回填
		if !hasReasoningContent {
			rcValue := "" // 兜底空字符串
			if cache != nil {
				fullKey := MakeFullKeyFromMsg(msg)
				tcKey := MakeToolcallKeyFromMsg(msg)
				if cached, found := cache.Lookup(fullKey, tcKey, true); found {
					rcValue = cached
				}
			}
			msg["reasoning_content"] = rcValue
			needPatch = true
		}

		if needPatch {
			messages[i] = msg
			modified = true
		}
	}

	if !modified {
		return body
	}

	m["messages"] = messages
	rewritten, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return rewritten
}
