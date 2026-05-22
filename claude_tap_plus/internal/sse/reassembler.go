package sse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/usage"
)

// SSEReassembler parses raw SSE byte streams and reconstructs complete API responses.
// It supports Anthropic streaming, OpenAI Chat Completions, and OpenAI Responses protocols.
type SSEReassembler struct {
	Events  []map[string]any
	buf     []byte
	curEv   *string
	curData []string
	snap    map[string]any
}

// NewSSEReassembler creates a new reassembler.
func NewSSEReassembler() *SSEReassembler {
	return &SSEReassembler{}
}

// FeedBytes appends raw SSE bytes and processes complete lines.
func (r *SSEReassembler) FeedBytes(chunk []byte) {
	r.buf = append(r.buf, chunk...)
	for {
		idx := bytes.IndexByte(r.buf, '\n')
		if idx < 0 {
			break
		}
		line := r.buf[:idx]
		r.buf = r.buf[idx+1:]
		r.feedLine(string(line))
	}
}

// Reconstruct returns the accumulated message snapshot, or nil if no snapshot yet.
func (r *SSEReassembler) Reconstruct() map[string]any {
	return r.snap
}

func (r *SSEReassembler) feedLine(line string) {
	line = strings.TrimRight(line, "\r")

	if strings.HasPrefix(line, "event:") {
		ev := strings.TrimSpace(line[len("event:"):])
		r.curEv = &ev
		r.curData = nil
	} else if strings.HasPrefix(line, "data:") {
		r.curData = append(r.curData, strings.TrimSpace(line[len("data:"):]))
	} else if line == "" {
		// Emit on blank line if we have event or data lines.
		if r.curEv == nil && len(r.curData) == 0 {
			return
		}
		raw := strings.Join(r.curData, "\n")

		// Skip [DONE] sentinel.
		if raw == "[DONE]" && r.curEv == nil {
			r.curEv = nil
			r.curData = nil
			return
		}

		var data any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			data = raw
		}

		evType := "message"
		if r.curEv != nil {
			evType = *r.curEv
		}
		r.addEvent(evType, data)
		r.curEv = nil
		r.curData = nil
	}
}

func (r *SSEReassembler) addEvent(eventType string, data any) {
	r.Events = append(r.Events, map[string]any{"event": eventType, "data": data})
	r.accumulate(eventType, data)
}

func (r *SSEReassembler) accumulate(eventType string, data any) {
	m, ok := data.(map[string]any)
	if !ok {
		return
	}

	switch eventType {
	case "message_start":
		if msg, ok := m["message"].(map[string]any); ok {
			r.snap = deepCopy(msg)
		}

	case "response.created", "response.completed", "response.done":
		if resp, ok := m["response"].(map[string]any); ok {
			r.snap = deepCopy(resp)
		} else if eventType == "response.completed" || eventType == "response.done" {
			r.snap = deepCopy(m)
		}

	case "message":
		if _, hasChoices := m["choices"]; hasChoices {
			r.accumulateChatCompletionChunk(m)
			return
		}

	case "content_block_start":
		if r.snap == nil {
			return
		}
		block := deepCopy(m["content_block"].(map[string]any))
		if _, ok := r.snap["content"]; !ok {
			r.snap["content"] = []any{}
		}
		content := r.snap["content"].([]any)
		idx := int(float64Val(m["index"]))
		for len(content) <= idx {
			content = append(content, map[string]any{})
		}
		content[idx] = block
		r.snap["content"] = content

	case "content_block_delta":
		if r.snap == nil {
			return
		}
		idx := int(float64Val(m["index"]))
		delta, _ := m["delta"].(map[string]any)
		if delta == nil {
			return
		}
		content, _ := r.snap["content"].([]any)
		if idx >= len(content) {
			return
		}
		block, _ := content[idx].(map[string]any)
		if block == nil {
			return
		}
		switch delta["type"] {
		case "text_delta":
			block["text"] = strVal(block["text"]) + strVal(delta["text"])
		case "thinking_delta":
			block["thinking"] = strVal(block["thinking"]) + strVal(delta["thinking"])
		case "input_json_delta":
			block["_partial_json"] = strVal(block["_partial_json"]) + strVal(delta["partial_json"])
		}

	case "content_block_stop":
		if r.snap == nil {
			return
		}
		idx := int(float64Val(m["index"]))
		content, _ := r.snap["content"].([]any)
		if idx >= len(content) {
			return
		}
		block, _ := content[idx].(map[string]any)
		if block == nil {
			return
		}
		if pj, ok := block["_partial_json"]; ok {
			var parsed any
			if json.Unmarshal([]byte(strVal(pj)), &parsed) == nil {
				block["input"] = parsed
			}
			delete(block, "_partial_json")
		}

	case "message_delta":
		if r.snap == nil {
			return
		}
		delta, _ := m["delta"].(map[string]any)
		if delta != nil {
			for k, v := range delta {
				r.snap[k] = v
			}
		}
		if u, ok := m["usage"].(map[string]any); ok {
			if _, exists := r.snap["usage"]; !exists {
				r.snap["usage"] = map[string]any{}
			}
			existing := r.snap["usage"].(map[string]any)
			for k, v := range u {
				existing[k] = v
			}
		}
	}
}

// --- OpenAI Chat Completions accumulation ---

func (r *SSEReassembler) accumulateChatCompletionChunk(data map[string]any) {
	choices, _ := data["choices"].([]any)
	rawUsage, _ := data["usage"]

	if len(choices) == 0 {
		if u, ok := rawUsage.(map[string]any); ok && r.snap != nil {
			r.mergeChatUsage(u)
		}
		return
	}

	choice, _ := choices[0].(map[string]any)
	if choice == nil {
		return
	}
	delta, _ := choice["delta"].(map[string]any)
	finishReason := choice["finish_reason"]
	choiceUsage, _ := choice["usage"].(map[string]any)

	// Initialize snapshot on first chunk.
	if r.snap == nil {
		r.snap = map[string]any{
			"id":     strVal(data["id"]),
			"object": "chat.completion",
			"model":  strVal(data["model"]),
			"choices": []any{
				map[string]any{
					"index":         0,
					"message":       map[string]any{"role": strVal(fieldStr(delta, "role")), "content": ""},
					"finish_reason": nil,
				},
			},
			"content": []any{map[string]any{"type": "text", "text": ""}},
		}
		if role := strVal(fieldStr(delta, "role")); role == "" {
			r.snap["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["role"] = "assistant"
		}
	}

	msg := r.snap["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)
	textBlock := r.chatTextBlock()

	if role := strVal(fieldStr(delta, "role")); role != "" {
		msg["role"] = role
	}
	if rc := strVal(fieldStr(delta, "reasoning_content")); rc != "" {
		msg["reasoning_content"] = strVal(msg["reasoning_content"]) + rc
		r.mirrorReasoning(strVal(msg["reasoning_content"]))
	}
	if c := strVal(fieldStr(delta, "content")); c != "" {
		msg["content"] = strVal(msg["content"]) + c
		textBlock["text"] = strVal(textBlock["text"]) + c
	}

	// Tool calls.
	for _, tcRaw := range sliceField(delta, "tool_calls") {
		tcDelta, _ := tcRaw.(map[string]any)
		if tcDelta == nil {
			continue
		}
		idx := int(float64Val(tcDelta["index"]))
		toolCalls, _ := msg["tool_calls"].([]any)
		if toolCalls == nil {
			toolCalls = []any{}
		}
		for len(toolCalls) <= idx {
			toolCalls = append(toolCalls, map[string]any{
				"id":   "",
				"type": "function",
				"function": map[string]any{
					"name":      "",
					"arguments": "",
				},
			})
		}
		existing := toolCalls[idx].(map[string]any)
		if id := strVal(tcDelta["id"]); id != "" {
			existing["id"] = id
		}
		if t := strVal(tcDelta["type"]); t != "" {
			existing["type"] = t
		}
		if fnDelta, ok := tcDelta["function"].(map[string]any); ok {
			fn, _ := existing["function"].(map[string]any)
			if fn == nil {
				fn = map[string]any{"name": "", "arguments": ""}
			}
			if n := strVal(fnDelta["name"]); n != "" {
				fn["name"] = strVal(fn["name"]) + n
			}
			if a := strVal(fnDelta["arguments"]); a != "" {
				fn["arguments"] = strVal(fn["arguments"]) + a
			}
			existing["function"] = fn
		}
		toolCalls[idx] = existing
		msg["tool_calls"] = toolCalls
		r.mirrorToolCall(idx, existing)
	}

	if finishReason != nil && finishReason != "" {
		r.snap["choices"].([]any)[0].(map[string]any)["finish_reason"] = finishReason
	}

	if u, ok := rawUsage.(map[string]any); ok {
		r.mergeChatUsage(u)
	}
	if choiceUsage != nil {
		r.mergeChatUsage(choiceUsage)
	}
}

func (r *SSEReassembler) chatTextBlock() map[string]any {
	content := r.snap["content"].([]any)
	for _, b := range content {
		if block, ok := b.(map[string]any); ok && block["type"] == "text" {
			return block
		}
	}
	block := map[string]any{"type": "text", "text": ""}
	r.snap["content"] = append(content, block)
	return block
}

func (r *SSEReassembler) mirrorReasoning(reasoning string) {
	content := r.snap["content"].([]any)
	for _, b := range content {
		if block, ok := b.(map[string]any); ok && block["type"] == "thinking" {
			block["thinking"] = reasoning
			return
		}
	}
	block := map[string]any{"type": "thinking", "thinking": reasoning}
	r.snap["content"] = append([]any{block}, content...)
}

func (r *SSEReassembler) mirrorToolCall(idx int, tc map[string]any) {
	content := r.snap["content"].([]any)
	// Count thinking block for offset.
	offset := 1
	for _, b := range content {
		if block, ok := b.(map[string]any); ok && block["type"] == "thinking" {
			offset = 2
			break
		}
	}
	target := idx + offset
	for len(content) <= target {
		content = append(content, map[string]any{"type": "tool_use", "id": "", "name": "", "input": map[string]any{}})
	}
	block := content[target].(map[string]any)
	if id := strVal(tc["id"]); id != "" {
		block["id"] = id
	}
	if fn, ok := tc["function"].(map[string]any); ok {
		if n := strVal(fn["name"]); n != "" {
			block["name"] = n
		}
		if args := strVal(fn["arguments"]); args != "" {
			var parsed any
			if json.Unmarshal([]byte(args), &parsed) == nil {
				block["input"] = parsed
			}
		}
	}
	r.snap["content"] = content
}

func (r *SSEReassembler) mergeChatUsage(u map[string]any) {
	r.snap["usage"] = usage.NormalizeUsage(u)
}

// --- helpers ---

func deepCopy(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	data, _ := json.Marshal(m)
	result := make(map[string]any)
	_ = json.Unmarshal(data, &result)
	return result
}

func float64Val(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func fieldStr(m map[string]any, key string) any {
	v, _ := m[key]
	return v
}

func sliceField(m map[string]any, key string) []any {
	v, _ := m[key].([]any)
	if v == nil {
		return nil
	}
	return v
}
