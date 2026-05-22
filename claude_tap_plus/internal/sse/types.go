package sse

// AnthropicContentBlock represents a content block in an Anthropic API response.
type AnthropicContentBlock struct {
	Type      string `json:"type"`                // "text", "thinking", "tool_use", "tool_result"
	Text      string `json:"text,omitempty"`       // for type="text"
	Thinking  string `json:"thinking,omitempty"`   // for type="thinking"
	ID        string `json:"id,omitempty"`         // for type="tool_use" / "tool_result"
	Name      string `json:"name,omitempty"`       // for type="tool_use"
	Input     any    `json:"input,omitempty"`      // for type="tool_use"
	Content   any    `json:"content,omitempty"`    // for type="tool_result"
	ToolUseID string `json:"tool_use_id,omitempty"` // for type="tool_result"
	Signature string `json:"signature,omitempty"`  // thinking block signature
}

// AnthropicUsage represents token usage in an Anthropic API response.
type AnthropicUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,omitempty"`
}

// AnthropicMessage is the reconstructed Anthropic Messages API response.
type AnthropicMessage struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Role         string                 `json:"role"`
	Model        string                 `json:"model"`
	Content      []AnthropicContentBlock `json:"content"`
	StopReason   string                 `json:"stop_reason,omitempty"`
	StopSequence *string                `json:"stop_sequence"`
	Usage        AnthropicUsage         `json:"usage"`
}
