package proxy

import (
	"net/http"
	"strings"
)

var sensitiveHeaderKeys = map[string]bool{
	"authorization":       true,
	"cookie":              true,
	"set-cookie":          true,
	"set-cookie2":         true,
	"x-api-key":           true,
	"cosy-key":            true,
	"cosy-machinetoken":   true,
	"cosy-machine-token":  true,
	"cosy-machineid":      true,
	"cosy-machine-id":     true,
	"cosy-machinetype":    true,
	"cosy-machine-type":   true,
	"cosy-user":           true,
}

var prefixRedactedKeys = map[string]bool{
	"authorization": true,
	"x-api-key":     true,
}

var hopByHopHeaders = map[string]bool{
	"connection":          true,
	"keep-alive":          true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailers":            true,
	"transfer-encoding":   true,
	"upgrade":             true,
}

// FilterHeaders returns a copy of headers with hop-by-hop headers removed.
// If redact is true, sensitive headers are partially masked.
func FilterHeaders(headers http.Header, redact bool) http.Header {
	out := make(http.Header)
	for k, vals := range headers {
		lower := strings.ToLower(k)
		if hopByHopHeaders[lower] {
			continue
		}
		if redact && sensitiveHeaderKeys[lower] {
			for _, v := range vals {
				if prefixRedactedKeys[lower] && len(v) > 12 {
					out.Set(k, v[:12]+"...")
				} else {
					out.Set(k, "***")
				}
			}
			continue
		}
		for _, v := range vals {
			out.Add(k, v)
		}
	}
	return out
}

// HeadersToMap converts http.Header to map[string]string for trace recording.
func HeadersToMap(h http.Header) map[string]string {
	m := make(map[string]string, len(h))
	for k, vals := range h {
		if len(vals) > 0 {
			m[k] = vals[0]
		}
	}
	return m
}
