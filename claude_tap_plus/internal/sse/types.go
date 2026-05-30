package sse

// AnthropicContentBlock 表示 Anthropic API 响应中的内容块。
//
// 支持的类型（Type 字段）：
//   - "text":        普通文本内容
//   - "thinking":    模型推理/思考内容
//   - "tool_use":    工具调用请求
//   - "tool_result": 工具调用结果
type AnthropicContentBlock struct {
	Type      string `json:"type"`                  // 内容块类型
	Text      string `json:"text,omitempty"`        // 文本内容（type="text" 时使用）
	Thinking  string `json:"thinking,omitempty"`    // 推理内容（type="thinking" 时使用）
	ID        string `json:"id,omitempty"`          // 工具调用标识（type="tool_use" / "tool_result" 时使用）
	Name      string `json:"name,omitempty"`        // 工具名称（type="tool_use" 时使用）
	Input     any    `json:"input,omitempty"`       // 工具输入参数（type="tool_use" 时使用）
	Content   any    `json:"content,omitempty"`     // 工具结果内容（type="tool_result" 时使用）
	ToolUseID string `json:"tool_use_id,omitempty"` // 关联的工具调用 ID（type="tool_result" 时使用）
	Signature string `json:"signature,omitempty"`   // 推理块的签名（用于 thinking 块验证）
}

// AnthropicUsage 表示 Anthropic API 响应中的 Token 用量统计。
type AnthropicUsage struct {
	InputTokens              int64 `json:"input_tokens"`                          // 输入 Token 数量
	OutputTokens             int64 `json:"output_tokens"`                         // 输出 Token 数量
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens,omitempty"`     // 缓存读取的输入 Token（可选）
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,omitempty"` // 缓存创建的输入 Token（可选）
}

// AnthropicMessage 是重建后的 Anthropic Messages API 完整响应结构。
//
// 该结构体由 SSEReassembler 将流式 SSE 事件逐块累积后组装而成，
// 其数据形状与非流式的 Messages API 响应保持一致。
type AnthropicMessage struct {
	ID           string                  `json:"id"`            // 消息唯一标识
	Type         string                  `json:"type"`          // 对象类型（通常为 "message"）
	Role         string                  `json:"role"`          // 角色（通常为 "assistant"）
	Model        string                  `json:"model"`         // 使用的模型名称
	Content      []AnthropicContentBlock `json:"content"`       // 内容块列表
	StopReason   string                  `json:"stop_reason,omitempty"` // 停止原因（如 "end_turn"）
	StopSequence *string                 `json:"stop_sequence"` // 触发的停止序列（如有）
	Usage        AnthropicUsage          `json:"usage"`         // Token 用量统计
}
