package usage

// AnthropicNormalizedUsage 定义了 Anthropic 风格的标准化 Token 用量结构体。
//
// 该结构体将不同 Provider（如 Anthropic、OpenAI、Google Gemini）各自的 Token 字段名
// 统一映射到 Anthropic 规范下的标准名称，便于后续统计和展示。
type AnthropicNormalizedUsage struct {
	InputTokens              int64 `json:"input_tokens"`                            // 输入 Token 数量（prompt tokens）
	OutputTokens             int64 `json:"output_tokens"`                           // 输出 Token 数量（completion tokens）
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens,omitempty"`       // 缓存命中读取的输入 Token 数量（可选）
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,omitempty"`   // 缓存创建时写入的输入 Token 数量（可选）
}
