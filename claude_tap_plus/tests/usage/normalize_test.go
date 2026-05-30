// Package usage_test 包含 token 用量标准化函数的单元测试。
// 测试覆盖 Anthropic、OpenAI、Google 等不同厂商的用量字段映射。
package usage_test

import (
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/usage"
)

// TestNormalizeUsageAnthropicFields 验证：Anthropic 字段（input_tokens/output_tokens/cache_read_input_tokens/cache_creation_input_tokens）被正确标准化。
func TestNormalizeUsageAnthropicFields(t *testing.T) {
	raw := map[string]any{
		"input_tokens":                float64(100),
		"output_tokens":               float64(50),
		"cache_read_input_tokens":     float64(30),
		"cache_creation_input_tokens": float64(10),
	}
	result := usage.NormalizeUsage(raw)

	if result["input_tokens"] != 100 {
		t.Errorf("input_tokens: got %d, want 100", result["input_tokens"])
	}
	if result["output_tokens"] != 50 {
		t.Errorf("output_tokens: got %d, want 50", result["output_tokens"])
	}
	if result["cache_read_input_tokens"] != 30 {
		t.Errorf("cache_read_input_tokens: got %d, want 30", result["cache_read_input_tokens"])
	}
	if result["cache_creation_input_tokens"] != 10 {
		t.Errorf("cache_creation_input_tokens: got %d, want 10", result["cache_creation_input_tokens"])
	}
}

// TestNormalizeUsageOpenAIFields 验证：OpenAI 字段（prompt_tokens/completion_tokens/cached_tokens）被映射为标准字段。
func TestNormalizeUsageOpenAIFields(t *testing.T) {
	raw := map[string]any{
		"prompt_tokens":     float64(200),
		"completion_tokens": float64(80),
		"cached_tokens":     float64(40),
	}
	result := usage.NormalizeUsage(raw)

	if result["input_tokens"] != 200 {
		t.Errorf("input_tokens: got %d, want 200", result["input_tokens"])
	}
	if result["output_tokens"] != 80 {
		t.Errorf("output_tokens: got %d, want 80", result["output_tokens"])
	}
	if result["cache_read_input_tokens"] != 40 {
		t.Errorf("cache_read_input_tokens: got %d, want 40", result["cache_read_input_tokens"])
	}
}

// TestNormalizeUsageGoogleFields 验证：Google Gemini 字段（promptTokenCount/candidatesTokenCount/cachedContentTokenCount）被映射为标准字段。
func TestNormalizeUsageGoogleFields(t *testing.T) {
	raw := map[string]any{
		"promptTokenCount":        float64(300),
		"candidatesTokenCount":    float64(120),
		"cachedContentTokenCount": float64(60),
	}
	result := usage.NormalizeUsage(raw)

	if result["input_tokens"] != 300 {
		t.Errorf("input_tokens: got %d, want 300", result["input_tokens"])
	}
	if result["output_tokens"] != 120 {
		t.Errorf("output_tokens: got %d, want 120", result["output_tokens"])
	}
	if result["cache_read_input_tokens"] != 60 {
		t.Errorf("cache_read_input_tokens: got %d, want 60", result["cache_read_input_tokens"])
	}
}

// TestNormalizeUsageNil 验证：nil 输入返回 nil。
func TestNormalizeUsageNil(t *testing.T) {
	result := usage.NormalizeUsage(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

// TestNormalizeUsageEmpty 验证：空 map 输入返回包含默认值 0 的结果。
func TestNormalizeUsageEmpty(t *testing.T) {
	result := usage.NormalizeUsage(map[string]any{})
	if len(result) == 0 {
		t.Error("expected non-empty result even for empty input")
	}
	for k, v := range result {
		if v != 0 {
			t.Errorf("expected 0 for %s, got %d", k, v)
		}
	}
}

// TestNormalizeUsageAnthropicInputTokensDetails 验证：Anthropic input_tokens_details.cached_tokens 被提取为 cache_read_input_tokens。
func TestNormalizeUsageAnthropicInputTokensDetails(t *testing.T) {
	raw := map[string]any{
		"input_tokens":  float64(100),
		"output_tokens": float64(50),
		"input_tokens_details": map[string]any{
			"cached_tokens": float64(25),
		},
	}
	result := usage.NormalizeUsage(raw)

	if result["input_tokens"] != 100 {
		t.Errorf("input_tokens: got %d, want 100", result["input_tokens"])
	}
	if result["cache_read_input_tokens"] != 25 {
		t.Errorf("cache_read_input_tokens from details: got %d, want 25", result["cache_read_input_tokens"])
	}
}

// TestNormalizeUsageOpenAIPromptTokensDetails 验证：OpenAI prompt_tokens_details.cached_tokens 被提取为 cache_read_input_tokens。
func TestNormalizeUsageOpenAIPromptTokensDetails(t *testing.T) {
	raw := map[string]any{
		"prompt_tokens":     float64(200),
		"completion_tokens": float64(80),
		"prompt_tokens_details": map[string]any{
			"cached_tokens": float64(50),
		},
	}
	result := usage.NormalizeUsage(raw)

	if result["cache_read_input_tokens"] != 50 {
		t.Errorf("cache_read_input_tokens from prompt_tokens_details: got %d, want 50", result["cache_read_input_tokens"])
	}
}

// TestNormalizeUsagePriorityFirstWins 验证：当同时存在多个同义字段时，先出现的字段优先。
func TestNormalizeUsagePriorityFirstWins(t *testing.T) {
	raw := map[string]any{
		"input_tokens":  float64(100),
		"prompt_tokens": float64(999),
	}
	result := usage.NormalizeUsage(raw)

	if result["input_tokens"] != 100 {
		t.Errorf("first key should win: got %d, want 100", result["input_tokens"])
	}
}

// TestNormalizeUsageIntTypes 验证：不同整数类型（float64/int/int64）都能被正确转换。
func TestNormalizeUsageIntTypes(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want int64
	}{
		{"float64", float64(42), 42},
		{"int", int(42), 42},
		{"int64", int64(42), 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := map[string]any{"input_tokens": tt.val}
			result := usage.NormalizeUsage(raw)
			if result["input_tokens"] != tt.want {
				t.Errorf("got %d, want %d", result["input_tokens"], tt.want)
			}
		})
	}
}
