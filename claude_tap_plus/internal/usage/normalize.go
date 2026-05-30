package usage

import (
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// NormalizeUsage 将不同 Provider 特定的 Token 字段名统一映射到 Anthropic 规范的标准名称。
//
// 支持的 Provider 及其字段映射关系：
//   - Anthropic: input_tokens / output_tokens / cache_read_input_tokens / cache_creation_input_tokens
//   - OpenAI:    prompt_tokens / completion_tokens / cached_tokens
//   - Google Gemini: promptTokenCount / candidatesTokenCount / cachedContentTokenCount
//
// 此外，该函数还会检查嵌套结构（如 input_tokens_details.cached_tokens、
// prompt_tokens_details.cached_tokens），将其累加到 cache_read_input_tokens 中。
//
// 参数 usage 是原始 API 响应中的 usage 对象（map[string]any）。
// 返回按 Anthropic 规范标准化后的 Token 统计 map。
func NormalizeUsage(usage map[string]any) map[string]int64 {
	if usage == nil {
		return nil
	}

	logger.Debug("usage", "normalize usage: keys=%v", mapKeys(usage))

	result := make(map[string]int64)

	// 映射输入 Token 数量：优先查找 Anthropic 字段名，其次 OpenAI、Gemini
	result["input_tokens"] = firstInt64(usage,
		"input_tokens",
		"prompt_tokens",
		"promptTokenCount",
	)

	// 映射输出 Token 数量：优先查找 Anthropic 字段名，其次 OpenAI、Gemini
	result["output_tokens"] = firstInt64(usage,
		"output_tokens",
		"completion_tokens",
		"candidatesTokenCount",
	)

	// 映射缓存读取的输入 Token 数量
	result["cache_read_input_tokens"] = firstInt64(usage,
		"cache_read_input_tokens",
		"cached_tokens",
		"cachedContentTokenCount",
	)

	// 映射缓存创建的输入 Token 数量（目前仅 Anthropic 使用）
	result["cache_creation_input_tokens"] = firstInt64(usage,
		"cache_creation_input_tokens",
	)

	// 处理嵌套结构：OpenAI 风格的 usage.input_tokens_details.cached_tokens
	if details, ok := usage["input_tokens_details"].(map[string]any); ok {
		if v := intFromMap(details, "cached_tokens"); v > 0 {
			result["cache_read_input_tokens"] += v
		}
	}
	// 处理嵌套结构：OpenAI 风格的 usage.prompt_tokens_details.cached_tokens
	if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
		if v := intFromMap(details, "cached_tokens"); v > 0 {
			result["cache_read_input_tokens"] += v
		}
	}

	logger.Debug("usage", "normalized: in=%d out=%d cache_read=%d cache_create=%d",
		result["input_tokens"], result["output_tokens"],
		result["cache_read_input_tokens"], result["cache_creation_input_tokens"])

	return result
}

// firstInt64 在 map 中按给定 keys 的优先顺序查找第一个非零的 int64 值。
// 如果所有 key 都不存在或值为零，则返回 0。
func firstInt64(m map[string]any, keys ...string) int64 {
	for _, k := range keys {
		if v := intFromMap(m, k); v != 0 {
			return v
		}
	}
	return 0
}

// intFromMap 从 map 中按 key 提取 int64 值，兼容多种数值类型。
//
// 支持的类型：
//   - float64（JSON 反序列化后的数字默认类型）
//   - int
//   - int64
//   - json.Number（encoding/json 的数字类型，通过接口匹配避免引入包依赖）
func intFromMap(m map[string]any, key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case json_number:
		if i, err := n.Int64(); err == nil {
			return i
		}
	}
	return 0
}

// json_number 是一个本地接口，用于匹配 encoding/json.Number 类型，
// 从而在不直接导入 encoding/json 包的情况下，安全地处理 JSON 数字值。
type json_number interface{ Int64() (int64, error) }

// mapKeys 返回 map 的 key 列表，用于调试日志。
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
