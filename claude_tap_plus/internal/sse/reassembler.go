package sse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/usage"
)

// SSEReassembler 解析原始 SSE（Server-Sent Events）字节流，并将流式片段重组成完整的 API 响应。
//
// 支持的协议：
//   - Anthropic Messages API Streaming
//   - OpenAI Chat Completions Streaming
//   - OpenAI Responses API Streaming
//
// 使用方式：
//
//	r := sse.NewSSEReassembler()
//	r.FeedBytes(chunk)       // 多次调用，分块送入 SSE 数据
//	snapshot := r.Reconstruct() // 获取当前累积的完整响应快照
//	events := r.Events       // 获取所有解析出的原始事件列表
type SSEReassembler struct {
	Events   []map[string]any // 解析出的所有原始 SSE 事件列表
	buf      []byte           // 未处理完的 SSE 字节缓冲区
	curEv    *string          // 当前正在解析的事件类型（event: xxx）
	curData  []string         // 当前事件的 data 行累积（支持多行 data）
	snap     map[string]any   // 重建后的完整响应快照
	fedOnce  bool             // 是否已接收过首块数据（用于首次日志）
}

// NewSSEReassembler 创建一个新的 SSE 重组器实例。
func NewSSEReassembler() *SSEReassembler {
	return &SSEReassembler{}
}

// FeedBytes 追加原始 SSE 字节数据并处理已完整的行。
//
// 该方法按 \n 分割输入数据，逐行解析 SSE 协议中的 event、data 字段，
// 遇到空行时触发事件生成并调用累积逻辑更新快照。
// 可以多次调用以处理分块传输的数据。
func (r *SSEReassembler) FeedBytes(chunk []byte) {
	if !r.fedOnce {
		logger.Debug("sse", "FeedBytes: first chunk size=%d", len(chunk))
		r.fedOnce = true
	}
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

// Reconstruct 返回当前累积的消息快照。
//
// 如果尚未收到足够的事件来构建快照（如仅收到部分 SSE 帧），则返回 nil。
func (r *SSEReassembler) Reconstruct() map[string]any {
	return r.snap
}

// feedLine 处理单条 SSE 文本行，解析 event、data 和空行分隔符。
//
// SSE 协议规则：
//   - 以 "event:" 开头的行设置当前事件类型
//   - 以 "data:" 开头的行追加到当前事件的数据 payload
//   - 空行表示当前事件结束，触发事件生成
//   - "[DONE]" 是 OpenAI 的流结束标记，直接忽略
func (r *SSEReassembler) feedLine(line string) {
	line = strings.TrimRight(line, "\r")

	if strings.HasPrefix(line, "event:") {
		ev := strings.TrimSpace(line[len("event:"):])
		r.curEv = &ev
		r.curData = nil
	} else if strings.HasPrefix(line, "data:") {
		r.curData = append(r.curData, strings.TrimSpace(line[len("data:"):]))
	} else if line == "" {
		// 空行触发事件：如果既没有 event 也没有 data 则跳过
		if r.curEv == nil && len(r.curData) == 0 {
			return
		}
		raw := strings.Join(r.curData, "\n")

		// 跳过 OpenAI 的 [DONE] 哨兵标记
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

// addEvent 将解析出的单个 SSE 事件追加到事件列表，并触发累积更新。
func (r *SSEReassembler) addEvent(eventType string, data any) {
	r.Events = append(r.Events, map[string]any{"event": eventType, "data": data})
	r.accumulate(eventType, data)
}

// accumulate 按事件类型将流式片段累积到响应快照中。
//
// 支持的事件类型及处理逻辑：
//   - message_start:      初始化快照为 message 对象
//   - response.created/completed/done: OpenAI Responses API，初始化或更新快照
//   - message:            OpenAI Chat Completions 流，调用 accumulateChatCompletionChunk
//   - content_block_start: 向快照 content 数组添加新的内容块
//   - content_block_delta: 更新指定索引的内容块（文本、推理、JSON 片段）
//   - content_block_stop:  将累积的 partial_json 解析为 input 字段
//   - message_delta:       更新 message 级别的 delta（如 stop_reason）和 usage
func (r *SSEReassembler) accumulate(eventType string, data any) {
	m, ok := data.(map[string]any)
	if !ok {
		return
	}

	switch eventType {
	case "message_start":
		if msg, ok := m["message"].(map[string]any); ok {
			r.snap = deepCopy(msg)
			logger.Debug("sse", "accumulate: message_start, model=%v", msg["model"])
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
		idx := int(float64Val(m["index"]))
		logger.Debug("sse", "accumulate: content_block_start idx=%d type=%v", idx, block["type"])
		if _, ok := r.snap["content"]; !ok {
			r.snap["content"] = []any{}
		}
		content := r.snap["content"].([]any)
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
		// 将累积的 JSON 字符串片段解析为结构化的 input 字段
		if pj, ok := block["_partial_json"]; ok {
			var parsed any
			if json.Unmarshal([]byte(strVal(pj)), &parsed) == nil {
				block["input"] = parsed
			}
			delete(block, "_partial_json")
		}
		logger.Debug("sse", "accumulate: content_block_stop idx=%d", idx)

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
		if delta != nil {
			if sr, ok := delta["stop_reason"]; ok && sr != nil {
				logger.Debug("sse", "accumulate: message_delta stop_reason=%v", sr)
			}
		}
	}
}

// --- OpenAI Chat Completions 流式累积 ---

// accumulateChatCompletionChunk 处理 OpenAI Chat Completions Streaming 的单个 chunk。
//
// 该函数将 choices[0].delta 中的内容逐块累积，同时处理：
//   - role、content、reasoning_content 的拼接
//   - tool_calls 的渐进式构建（id、name、arguments 分块到达）
//   - usage 信息的合并
//   - finish_reason 的更新
//
// 快照结构兼容 Anthropic 格式，content 数组中会同时维护文本块和工具调用块。
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

	// 首个 chunk 时初始化快照
	if r.snap == nil {
		logger.Debug("sse", "accumulate: chat chunk init, model=%s", strVal(data["model"]))
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

	// 工具调用增量处理
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
		logger.Debug("sse", "accumulate: chat chunk finish_reason=%v", finishReason)
	}

	if u, ok := rawUsage.(map[string]any); ok {
		r.mergeChatUsage(u)
	}
	if choiceUsage != nil {
		r.mergeChatUsage(choiceUsage)
	}
}

// chatTextBlock 获取或创建快照中的文本内容块。
// 返回 content 数组中第一个 type="text" 的块；如果不存在则新建一个。
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

// mirrorReasoning 将 reasoning_content 同步到 content 数组中的 thinking 块。
// 如果 content 中不存在 thinking 块，则在开头插入一个。
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

// mirrorToolCall 将 Chat Completions 的 tool_call 同步到 Anthropic 格式的 content 数组中。
//
// 同步规则：
//   - 在 content 数组中查找或创建对应的 tool_use 块
//   - 将 function name 映射为 tool_use 的 name
//   - 将 arguments JSON 解析后映射为 tool_use 的 input
//   - thinking 块的存在会影响工具调用块在 content 中的索引偏移
func (r *SSEReassembler) mirrorToolCall(idx int, tc map[string]any) {
	content := r.snap["content"].([]any)
	// 如果存在 thinking 块，工具调用的位置需要偏移 1
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

// mergeChatUsage 将 OpenAI Chat Completions 的 usage 数据标准化后合并到快照。
func (r *SSEReassembler) mergeChatUsage(u map[string]any) {
	r.snap["usage"] = usage.NormalizeUsage(u)
}

// --- 辅助函数 ---

// deepCopy 通过 JSON 序列化/反序列化实现 map 的深拷贝。
func deepCopy(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	data, _ := json.Marshal(m)
	result := make(map[string]any)
	_ = json.Unmarshal(data, &result)
	return result
}

// float64Val 将任意数值类型转换为 float64。
// 支持 float64、int、int64；不支持的类型返回 0。
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

// strVal 将任意值转换为字符串。
// nil 返回空字符串，string 类型直接返回，其他类型使用 fmt.Sprintf 格式化。
func strVal(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// fieldStr 从 map 中安全获取字段值。
func fieldStr(m map[string]any, key string) any {
	v, _ := m[key]
	return v
}

// sliceField 从 map 中获取 []any 类型的字段，如果不存在或类型不匹配则返回 nil。
func sliceField(m map[string]any, key string) []any {
	v, _ := m[key].([]any)
	if v == nil {
		return nil
	}
	return v
}
