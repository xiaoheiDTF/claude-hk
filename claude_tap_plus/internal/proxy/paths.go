package proxy

import "strings"

// AllowedPathPrefixes defines the API paths that the proxy will forward.
var AllowedPathPrefixes = []string{
	// Anthropic API
	"/v1/messages",
	"/v1/complete",
	// OpenAI API
	"/v1/responses",
	"/v1/chat/completions",
	"/v1/completions",
	"/v1/models",
	"/v1/embeddings",
	"/v1/files",
	// OpenAI Responses API (after strip_path_prefix removes /v1)
	"/responses",
	"/chat/completions",
	"/completions",
	"/models",
	// Google Gemini API
	"/v1beta/",
	"/v1/",
	// Kimi API
	"/coding/v1/",
	// OpenAI compatible relay endpoints
	"/anthropic",
}

// IsAllowedPath checks whether a request path should be forwarded.
// Unknown paths (scanners, crawlers) are rejected with 404.
func IsAllowedPath(path string) bool {
	for _, prefix := range AllowedPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
