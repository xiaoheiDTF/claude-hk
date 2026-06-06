// Package main 提供 Kimi API 轻量代理，解决 Claude Code 使用 Kimi 时
// thinking 模式下 tool_call 消息缺少 reasoning_content 导致的 400 错误。
//
// 用法：
//
//	kimi-proxy [--listen :9090] --target https://api.kimi.com/coding
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	listenAddr = flag.String("listen", ":9090", "本地监听地址")
	targetURL  = flag.String("target", "https://api.kimi.com/coding", "Kimi API 上游地址")
)

func main() {
	flag.Parse()
	target := strings.TrimRight(*targetURL, "/")

	log.Printf("kimi-proxy: listening on %s, forwarding to %s", *listenAddr, target)

	proxy := &kimiProxy{target: target}
	if err := http.ListenAndServe(*listenAddr, proxy); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

// kimiProxy 是一个反向代理，在转发前为 thinking 模式下的 assistant tool_call 消息注入 reasoning_content。
type kimiProxy struct {
	target string
}

func (p *kimiProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 读取请求体
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// 注入 reasoning_content
	bodyBytes = injectReasoningContent(bodyBytes)

	// 构建上游请求
	upstreamURL := p.target + r.URL.Path
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	upstreamReq, err := http.NewRequest(r.Method, upstreamURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, "create request: "+err.Error(), http.StatusBadGateway)
		return
	}

	// 复制请求头
	for k, vals := range r.Header {
		lower := strings.ToLower(k)
		if lower == "host" {
			continue
		}
		for _, v := range vals {
			upstreamReq.Header.Add(k, v)
		}
	}

	// 转发
	client := &http.Client{}
	resp, err := client.Do(upstreamReq)
	if err != nil {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// 流式转发响应体
	flusher, canFlush := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			if canFlush {
				flusher.Flush()
			}
		}
		if err != nil {
			break
		}
	}

	_ = fmt.Sprintf("") // avoid unused import
}

// injectReasoningContent 为 thinking 模式下的 assistant tool_call 消息注入 reasoning_content 字段。
//
// Kimi Code 要求：当 thinking 启用时，所有 assistant tool_call 消息必须携带顶层 reasoning_content 字段。
// Claude Code 发送 tool_call 消息时不包含此字段，导致 400 错误：
//
//	thinking is enabled but reasoning_content is missing in assistant tool call message at index N
func injectReasoningContent(body []byte) []byte {
	if len(body) == 0 {
		return body
	}

	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return body // 非 JSON，原样返回
	}

	// 检测 thinking 是否启用
	thinking, ok := m["thinking"].(map[string]any)
	if !ok {
		return body
	}
	thinkingType, _ := thinking["type"].(string)
	if thinkingType != "enabled" {
		return body
	}

	// 获取 messages 数组
	messages, ok := m["messages"].([]any)
	if !ok || len(messages) == 0 {
		return body
	}

	modified := false
	for i, msgRaw := range messages {
		msg, ok := msgRaw.(map[string]any)
		if !ok {
			continue
		}

		// 只处理 assistant 消息
		role, _ := msg["role"].(string)
		if role != "assistant" {
			continue
		}

		// 检查 content 是否为数组
		content, ok := msg["content"].([]any)
		if !ok || len(content) == 0 {
			continue
		}

		// 检查是否有 tool_use 块
		hasToolUse := false
		hasThinking := false
		for _, blockRaw := range content {
			block, ok := blockRaw.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := block["type"].(string)
			if blockType == "tool_use" {
				hasToolUse = true
			}
			if blockType == "thinking" {
				hasThinking = true
			}
		}

		if !hasToolUse {
			continue
		}

		needPatch := false

		// 补丁 1: content 数组缺少 thinking 块 → 在开头插入
		if !hasThinking {
			thinkingBlock := map[string]any{
				"type":     "thinking",
				"thinking": "",
			}
			newContent := make([]any, 0, len(content)+1)
			newContent = append(newContent, thinkingBlock)
			newContent = append(newContent, content...)
			msg["content"] = newContent
			needPatch = true
		}

		// 补丁 2: 消息缺少顶层 reasoning_content 字段 → 注入
		if _, exists := msg["reasoning_content"]; !exists {
			msg["reasoning_content"] = ""
			needPatch = true
		}

		if needPatch {
			messages[i] = msg
			modified = true
		}
	}

	if !modified {
		return body
	}

	m["messages"] = messages
	rewritten, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return rewritten
}

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime | log.Lshortfile)
}
