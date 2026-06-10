// Package proxy 提供 HTTP 代理功能，包括请求转发、SSE 流式处理与 Trace 记录。
package proxy

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/sse"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
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
	model              string              // 强制替换的模型名（空=不改写）
	upstreamAvailable  bool                // 上游可用性标记，初始 true
	fallbackConfig     *FallbackConfig      // 兜底配置（上游不可用时切换）
	availableMu        sync.Mutex          // 保护 upstreamAvailable 和 fallbackConfig 的并发安全
	actualAddr         string              // 实际监听地址（启动后设置）
	reasoningCache     *ReasoningCache     // reasoning_content 缓存（仅 kimi 模式）
	kimiMode           bool                // 是否为 kimi 上游
}

// FallbackConfig 存储上游不可用时的兜底配置。
type FallbackConfig struct {
	BaseURL   string // 兜底上游 API 地址
	Model     string // 兜底模型名
	AuthToken string // 兜底认证 Token
	APIKey    string // 兜底 API Key
}

// NewReverseProxy 创建一个代理实例，将请求转发到指定的 target URL。
// traceDir 为 Trace 文件存放的基础目录。
func NewReverseProxy(target, traceDir string) *ReverseProxy {
	logger.Debug("proxy", "new proxy: target=%s baseDir=%s", target, traceDir)
	return &ReverseProxy{
		target:            strings.TrimRight(target, "/"),
		baseDir:           traceDir,
		upstreamAvailable: true,
		client: &http.Client{
			Timeout: 0, // 流式场景不设超时
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // 不自动跟随重定向
			},
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
// 处理流程：内部端点 → 路径白名单检查 → 读取请求体 → 构建上游请求 → 转发 → 流式/非流式处理。
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

	// 确定上游目标：检查是否需要走兜底配置（契约 4）
	p.availableMu.Lock()
	useFallback := !p.upstreamAvailable && p.fallbackConfig != nil
	activeTarget := p.target
	var fallbackModel string
	if useFallback {
		activeTarget = p.fallbackConfig.BaseURL
		fallbackModel = p.fallbackConfig.Model
	}
	p.availableMu.Unlock()

	// 兜底模式下重新改写 model
	if useFallback && fallbackModel != "" {
		bodyBytes = rewriteModel(bodyBytes, reqBody, fallbackModel)
		reqBody = nil
		if len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &reqBody)
		}

		// 为 thinking 模式下的 assistant tool_call 消息注入 reasoning_content
		bodyBytes = injectReasoningContentCached(bodyBytes, reqBody, p.reasoningCache, IsKimiURL(activeTarget))
		reqBody = nil
		if len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &reqBody)
		}
	}

	// 构建上游请求 URL
	upstreamURL := activeTarget + r.URL.Path
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	upstreamReq, err := http.NewRequest(r.Method, upstreamURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, "create upstream request: "+err.Error(), http.StatusBadGateway)
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

	// 兜底模式下替换认证头
	if useFallback && p.fallbackConfig != nil {
		if p.fallbackConfig.APIKey != "" {
			upstreamReq.Header.Set("x-api-key", p.fallbackConfig.APIKey)
		}
		if p.fallbackConfig.AuthToken != "" {
			upstreamReq.Header.Set("Authorization", "Bearer "+p.fallbackConfig.AuthToken)
		}
	}

	// 向上游发送请求
	upstreamResp, err := p.client.Do(upstreamReq)
	if err != nil {
		logger.Warn("proxy", "[Turn %d] upstream error: %v", turn, err)
		p.markUnavailable()
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer upstreamResp.Body.Close()

	// 响应错误（4xx/5xx）→ 标记上游不可用（契约 4）
	if upstreamResp.StatusCode >= 400 {
		logger.Warn("proxy", "[Turn %d] upstream error response: %d", turn, upstreamResp.StatusCode)
		p.markUnavailable()
	}

	duration := time.Since(t0).Milliseconds()

	// 从请求体中提取 model 字段
	model := ""
	if m, ok := reqBody.(map[string]any); ok {
		if v, ok := m["model"].(string); ok {
			model = v
		}
	}

	// 检测是否为流式请求
	isStreaming := false
	if m, ok := reqBody.(map[string]any); ok {
		if s, ok := m["stream"].(bool); ok {
			isStreaming = s
		}
	}

	// 根据是否流式分发到对应处理器
	if isStreaming {
		p.handleStreaming(w, upstreamResp, reqID, turn, int(duration), r, reqBody, t0, model)
	} else {
		p.handleNonStreaming(w, upstreamResp, reqID, turn, int(duration), r, reqBody, model)
	}
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
	t0 time.Time, model string,
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
		p.target,
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
	r *http.Request, reqBody any, model string,
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
		p.target,
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
	default:
		http.NotFound(w, r)
	}
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

// SetFallbackConfig 设置兜底配置。
func (p *ReverseProxy) SetFallbackConfig(cfg *FallbackConfig) {
	p.fallbackConfig = cfg
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

// IsKimiURL 判断 URL 是否为需要 reasoning_content 注入的上游（kimi/moonshot/deepseek）。
func IsKimiURL(url string) bool {
	lower := strings.ToLower(url)
	return strings.Contains(lower, "kimi") || strings.Contains(lower, "moonshot") || strings.Contains(lower, "deepseek")
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
