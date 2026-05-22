package usage

// AnthropicNormalizedUsage maps provider-specific token field names
// to Anthropic canonical names.
type AnthropicNormalizedUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,omitempty"`
}
