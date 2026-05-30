package trace

import (
	"fmt"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// AnthropicTraceRecord 表示 JSONL 追踪文件中的单条 API 调用记录，
// 遵循 Anthropic Messages API 协议的数据结构。
type AnthropicTraceRecord struct {
	Timestamp       string              `json:"timestamp"`                     // 请求时间戳（UTC，RFC3339 格式）
	RequestID       string              `json:"request_id"`                    // 自动生成的请求唯一标识
	Turn            int                 `json:"turn"`                          // 当前对话轮次序号
	DurationMs      int                 `json:"duration_ms"`                   // 请求总耗时（毫秒）
	Request         AnthropicRequest    `json:"request"`                       // 请求详情
	Response        AnthropicResponse   `json:"response"`                      // 响应详情
	UpstreamBaseURL string              `json:"upstream_base_url,omitempty"`   // 上游 API 基础地址（可选）
}

// AnthropicRequest 记录被拦截的 HTTP 请求详情。
type AnthropicRequest struct {
	Method  string            `json:"method"`  // HTTP 方法（如 POST）
	Path    string            `json:"path"`    // 请求路径（如 /v1/messages）
	Headers map[string]string `json:"headers"` // 请求头（已脱敏处理）
	Body    any               `json:"body"`    // 请求体（JSON 对象）
}

// AnthropicResponse 记录上游 API 返回的响应详情。
type AnthropicResponse struct {
	Status    int                 `json:"status"`              // HTTP 状态码
	Headers   map[string]string   `json:"headers"`             // 响应头
	Body      any                 `json:"body"`                // 响应体（JSON 对象或 SSE 解析结果）
	SSEEvents []AnthropicSSEEvent `json:"sse_events,omitempty"` // SSE 事件流帧列表（流式响应时使用）
}

// AnthropicSSEEvent 表示单个 Server-Sent Events（SSE）帧。
type AnthropicSSEEvent struct {
	Event string `json:"event"` // SSE 事件类型（如 content_block_delta）
	Data  any    `json:"data"`  // 事件携带的数据负载
}

// AnthropicTraceSummary 保存一次追踪会话的聚合统计信息。
type AnthropicTraceSummary struct {
	APICalls          int            `json:"api_calls"`            // API 调用总次数
	InputTokens       int64          `json:"input_tokens"`         // 累计输入 Token 数量
	OutputTokens      int64          `json:"output_tokens"`        // 累计输出 Token 数量
	CacheReadTokens   int64          `json:"cache_read_tokens"`    // 累计缓存读取 Token 数量
	CacheCreateTokens int64          `json:"cache_create_tokens"`  // 累计缓存创建 Token 数量
	ModelsUsed        map[string]int `json:"models_used"`          // 各模型调用次数统计
}

// NewAnthropicTraceRecord 创建一个带有自动生成字段的追踪记录。
//
// 自动填充的字段：
//   - Timestamp: 当前 UTC 时间（RFC3339 微秒精度）
//   - RequestID: 基于纳秒时间戳生成的请求标识
//
// 参数说明：
//   - turn:      当前对话轮次
//   - durationMs: 请求耗时（毫秒）
//   - method:    HTTP 方法
//   - path:      请求路径
func NewAnthropicTraceRecord(turn, durationMs int, method, path string) *AnthropicTraceRecord {
	logger.Debug("trace", "new trace record: turn=%d method=%s path=%s", turn, method, path)
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
