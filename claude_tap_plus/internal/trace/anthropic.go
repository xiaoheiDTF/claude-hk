package trace

import (
	"fmt"
	"time"
)

// AnthropicTraceRecord represents a single API call in the JSONL trace,
// following the Anthropic Messages API protocol shape.
type AnthropicTraceRecord struct {
	Timestamp       string               `json:"timestamp"`
	RequestID       string               `json:"request_id"`
	Turn            int                  `json:"turn"`
	DurationMs      int                  `json:"duration_ms"`
	Request         AnthropicRequest     `json:"request"`
	Response        AnthropicResponse    `json:"response"`
	UpstreamBaseURL string               `json:"upstream_base_url,omitempty"`
}

// AnthropicRequest captures the intercepted HTTP request details.
type AnthropicRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

// AnthropicResponse captures the upstream API response details.
type AnthropicResponse struct {
	Status    int              `json:"status"`
	Headers   map[string]string `json:"headers"`
	Body      any              `json:"body"`
	SSEEvents []AnthropicSSEEvent `json:"sse_events,omitempty"`
}

// AnthropicSSEEvent represents a single Server-Sent Events frame.
type AnthropicSSEEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// AnthropicTraceSummary holds aggregate statistics for a trace session.
type AnthropicTraceSummary struct {
	APICalls          int            `json:"api_calls"`
	InputTokens       int64          `json:"input_tokens"`
	OutputTokens      int64          `json:"output_tokens"`
	CacheReadTokens   int64          `json:"cache_read_tokens"`
	CacheCreateTokens int64          `json:"cache_create_tokens"`
	ModelsUsed        map[string]int `json:"models_used"`
}

// NewAnthropicTraceRecord creates a TraceRecord with auto-generated fields.
func NewAnthropicTraceRecord(turn, durationMs int, method, path string) *AnthropicTraceRecord {
	return &AnthropicTraceRecord{
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000000Z"),
		RequestID: fmt.Sprintf("req_%d", time.Now().UnixNano()%1e12),
		Turn:       turn,
		DurationMs: durationMs,
		Request: AnthropicRequest{
			Method:  method,
			Path:    path,
			Headers: make(map[string]string),
		},
		Response: AnthropicResponse{
			Headers: make(map[string]string),
		},
	}
}
