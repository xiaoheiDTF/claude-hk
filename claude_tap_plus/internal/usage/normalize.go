package usage

// NormalizeUsage maps provider-specific token field names to canonical names.
// Supported providers: Anthropic, OpenAI, Google Gemini.
func NormalizeUsage(usage map[string]any) map[string]int64 {
	if usage == nil {
		return nil
	}

	result := make(map[string]int64)

	// input_tokens
	result["input_tokens"] = firstInt64(usage,
		"input_tokens",
		"prompt_tokens",
		"promptTokenCount",
	)

	// output_tokens
	result["output_tokens"] = firstInt64(usage,
		"output_tokens",
		"completion_tokens",
		"candidatesTokenCount",
	)

	// cache_read_input_tokens
	result["cache_read_input_tokens"] = firstInt64(usage,
		"cache_read_input_tokens",
		"cached_tokens",
		"cachedContentTokenCount",
	)

	// cache_creation_input_tokens
	result["cache_creation_input_tokens"] = firstInt64(usage,
		"cache_creation_input_tokens",
	)

	// Nested: usage.input_tokens_details.cached_tokens
	if details, ok := usage["input_tokens_details"].(map[string]any); ok {
		if v := intFromMap(details, "cached_tokens"); v > 0 {
			result["cache_read_input_tokens"] += v
		}
	}
	// Nested: usage.prompt_tokens_details.cached_tokens
	if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
		if v := intFromMap(details, "cached_tokens"); v > 0 {
			result["cache_read_input_tokens"] += v
		}
	}

	return result
}

// firstInt64 returns the first non-zero int64 value found for any of the given keys.
func firstInt64(m map[string]any, keys ...string) int64 {
	for _, k := range keys {
		if v := intFromMap(m, k); v != 0 {
			return v
		}
	}
	return 0
}

// intFromMap extracts an int64 from a map, handling float64 (JSON numbers) and int variants.
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

type json_number interface{ Int64() (int64, error) }
