// Package proxy 提供 HTTP 代理功能，包括请求转发、SSE 流式处理与 Trace 记录。
package proxy

import (
	"net/http"
	"strings"
)

// sensitiveHeaderKeys 定义需要在日志中脱敏的敏感请求头。
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

// prefixRedactedKeys 定义需要保留前 12 个字符后截断的敏感头（如 Authorization）。
var prefixRedactedKeys = map[string]bool{
	"authorization": true,
	"x-api-key":     true,
}

// hopByHopHeaders 定义 HTTP/1.1 逐跳（hop-by-hop）头，这些头不应被转发。
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

// FilterHeaders 返回过滤掉逐跳头后的 Header 副本。
// 若 redact 为 true，敏感头将被部分遮蔽。
func FilterHeaders(headers http.Header, redact bool) http.Header {
	out := make(http.Header)
	for k, vals := range headers {
		lower := strings.ToLower(k)
		// 跳过逐跳头
		if hopByHopHeaders[lower] {
			continue
		}
		// 敏感头脱敏处理
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

// HeadersToMap 将 http.Header 转换为 map[string]string，用于 Trace 记录。
// 每个键只保留第一个值。
func HeadersToMap(h http.Header) map[string]string {
	m := make(map[string]string, len(h))
	for k, vals := range h {
		if len(vals) > 0 {
			m[k] = vals[0]
		}
	}
	return m
}
