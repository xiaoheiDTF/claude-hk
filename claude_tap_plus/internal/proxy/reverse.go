package proxy

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/sse"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
)

// ReverseProxy intercepts HTTP requests, forwards them to an upstream API,
// records the request/response to a JSONL trace, and handles SSE streaming.
type ReverseProxy struct {
	target    string
	writer    *trace.TraceWriter
	client    *http.Client
	turn      atomic.Int64
	server    *http.Server
	startOnce sync.Once
}

// NewReverseProxy creates a proxy that forwards to the given target URL.
func NewReverseProxy(target string, writer *trace.TraceWriter) *ReverseProxy {
	return &ReverseProxy{
		target: strings.TrimRight(target, "/"),
		writer: writer,
		client: &http.Client{
			Timeout: 0, // no timeout for streaming
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Start begins the proxy HTTP server on the given host:port.
// If port is 0, a random available port is chosen. Returns the actual port.
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

	go func() {
		if err := p.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("proxy server error: %v", err)
		}
	}()

	actualPort := listener.Addr().(*netTCPAddr).Port
	return actualPort, nil
}

// Stop gracefully shuts down the proxy server.
func (p *ReverseProxy) Stop() {
	if p.server != nil {
		_ = p.server.Close()
	}
}

func (p *ReverseProxy) serveHTTP(w http.ResponseWriter, r *http.Request) {
	// Reject unknown paths.
	if !IsAllowedPath(r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	// Read request body.
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	// Parse request body as JSON.
	var reqBody any
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &reqBody)
	}

	reqID := fmt.Sprintf("req_%d", time.Now().UnixNano()%1e12)
	turn := int(p.turn.Add(1))
	t0 := time.Now()

	// Build upstream request URL.
	upstreamURL := p.target + r.URL.Path
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	upstreamReq, err := http.NewRequest(r.Method, upstreamURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, "create upstream request: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Copy headers, filter hop-by-hop.
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

	upstreamResp, err := p.client.Do(upstreamReq)
	if err != nil {
		log.Printf("[Turn %d] upstream error: %v", turn, err)
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer upstreamResp.Body.Close()

	duration := time.Since(t0).Milliseconds()
	model := ""
	if m, ok := reqBody.(map[string]any); ok {
		if v, ok := m["model"].(string); ok {
			model = v
		}
	}

	// Detect streaming.
	isStreaming := false
	if m, ok := reqBody.(map[string]any); ok {
		if s, ok := m["stream"].(bool); ok {
			isStreaming = s
		}
	}

	if isStreaming {
		p.handleStreaming(w, upstreamResp, reqID, turn, int(duration), r, reqBody, t0, model)
	} else {
		p.handleNonStreaming(w, upstreamResp, reqID, turn, int(duration), r, reqBody, model)
	}
}

func (p *ReverseProxy) handleStreaming(
	w http.ResponseWriter,
	upstreamResp *http.Response,
	reqID string, turn, durationMs int,
	r *http.Request, reqBody any,
	t0 time.Time, model string,
) {
	// Copy response headers.
	respHeaders := FilterHeaders(upstreamResp.Header, false)
	for k, vals := range respHeaders {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(upstreamResp.StatusCode)

	flusher, canFlush := w.(http.Flusher)
	reassembler := sse.NewSSEReassembler()

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

	record := buildRecord(
		reqID, turn, durationMs,
		r.Method, r.URL.RequestURI(),
		r.Header, reqBody,
		upstreamResp.StatusCode, upstreamResp.Header,
		reconstructed,
		reassembler.Events,
		p.target,
	)

	if err := p.writer.Write(record); err != nil {
		log.Printf("[Turn %d] trace write error: %v", turn, err)
	}

	log.Printf("[Turn %d] ← %d stream done (%dms, model=%s)", turn, upstreamResp.StatusCode, durationMs, model)
}

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

	// Decompress if needed.
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
				decodeBytes = decoded
			}
			_ = reader.Close()
		}
	}

	var respBody any
	if len(decodeBytes) > 0 {
		_ = json.Unmarshal(decodeBytes, &respBody)
	}

	record := buildRecord(
		reqID, turn, durationMs,
		r.Method, r.URL.RequestURI(),
		r.Header, reqBody,
		upstreamResp.StatusCode, upstreamResp.Header,
		respBody,
		nil,
		p.target,
	)

	if err := p.writer.Write(record); err != nil {
		log.Printf("[Turn %d] trace write error: %v", turn, err)
	}

	// Write response to client.
	respHeaders := FilterHeaders(upstreamResp.Header, false)
	for k, vals := range respHeaders {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(upstreamResp.StatusCode)
	_, _ = w.Write(respBytes)

	log.Printf("[Turn %d] ← %d (%dms, %d bytes, model=%s)", turn, upstreamResp.StatusCode, durationMs, len(respBytes), model)
}

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
		"timestamp":    time.Now().UTC().Format("2006-01-02T15:04:05.000000Z"),
		"request_id":   reqID,
		"session_id":   extractSessionID(reqBody),
		"turn":         turn,
		"duration_ms":  durationMs,
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

// extractSessionID extracts session_id from request.body.metadata.user_id (JSON string).
// Claude Code sends: metadata.user_id = "{\"session_id\": \"uuid\", ...}"
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
