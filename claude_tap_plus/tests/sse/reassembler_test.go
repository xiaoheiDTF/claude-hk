package sse_test

import (
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/sse"
)

func TestAnthropicStreamReconstruction(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte(
		"event: message_start\n"+
			"data: {\"type\":\"message_start\",\"message\":{\"id\":\"m1\",\"role\":\"assistant\","+
			"\"content\":[],\"model\":\"claude-x\",\"usage\":{\"input_tokens\":3,\"output_tokens\":0}}}\n\n"+
			"event: content_block_start\n"+
			"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"+
			"event: content_block_delta\n"+
			"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hi\"}}\n\n"+
			"event: content_block_stop\n"+
			"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n"+
			"event: message_delta\n"+
			"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":1}}\n\n"+
			"event: message_stop\n"+
			"data: {\"type\":\"message_stop\"}\n\n",
	))

	snap := r.Reconstruct()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	content := snap["content"].([]any)
	block := content[0].(map[string]any)
	if block["text"] != "hi" {
		t.Errorf("expected text 'hi', got %v", block["text"])
	}
	usage := snap["usage"].(map[string]any)
	got := usage["output_tokens"]
	if got != float64(1) {
		t.Errorf("expected output_tokens=1, got %v", got)
	}
}

func TestChatCompletionsStream(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte(
		"data: {\"id\":\"c1\",\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n"+
			"data: {\"id\":\"c1\",\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\n"+
			"data: {\"id\":\"c1\",\"choices\":[{\"finish_reason\":\"stop\",\"delta\":{}}]}\n\n"+
			"data: [DONE]\n\n",
	))

	if len(r.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(r.Events))
	}
	for _, ev := range r.Events {
		if ev["event"] != "message" {
			t.Errorf("expected event type 'message', got %v", ev["event"])
		}
	}

	snap := r.Reconstruct()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	choices := snap["choices"].([]any)
	msg := choices[0].(map[string]any)["message"].(map[string]any)
	if msg["role"] != "assistant" {
		t.Errorf("expected role 'assistant', got %v", msg["role"])
	}
	if msg["content"] != "OK" {
		t.Errorf("expected content 'OK', got %v", msg["content"])
	}

	content := snap["content"].([]any)
	textBlock := content[0].(map[string]any)
	if textBlock["type"] != "text" || textBlock["text"] != "OK" {
		t.Errorf("expected text mirror block, got %v", textBlock)
	}
}

func TestChatCompletionsUsageNormalization(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte(
		"data: {\"id\":\"c1\",\"model\":\"hy3\",\"choices\":[{\"delta\":{\"role\":\"assistant\",\"content\":\"hi\"}}]}\n\n"+
			"data: {\"id\":\"c1\",\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],"+
			"\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":3,\"total_tokens\":15}}\n\n"+
			"data: [DONE]\n\n",
	))

	snap := r.Reconstruct()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	usageRaw := snap["usage"]
	usage, ok := usageRaw.(map[string]int64)
	if !ok {
		t.Fatalf("expected usage to be map[string]int64, got %T", usageRaw)
	}
	if usage["input_tokens"] != 12 {
		t.Errorf("expected input_tokens=12, got %d", usage["input_tokens"])
	}
	if usage["output_tokens"] != 3 {
		t.Errorf("expected output_tokens=3, got %d", usage["output_tokens"])
	}
	if snap["model"] != "hy3" {
		t.Errorf("expected model 'hy3', got %v", snap["model"])
	}
}

func TestChatCompletionsCachedTokens(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte(
		"data: {\"id\":\"c_kimi\",\"model\":\"kimi-k2\",\"choices\":[{\"delta\":{\"role\":\"assistant\",\"content\":\"hi\"}}]}\n\n"+
			"data: {\"id\":\"c_kimi\",\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\","+
			"\"usage\":{\"prompt_tokens\":8,\"completion_tokens\":5,\"total_tokens\":13,\"cached_tokens\":3}}]}\n\n"+
			"data: [DONE]\n\n",
	))

	snap := r.Reconstruct()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	usageRaw := snap["usage"]
	usage, ok := usageRaw.(map[string]int64)
	if !ok {
		t.Fatalf("expected usage to be map[string]int64, got %T", usageRaw)
	}
	if usage["input_tokens"] != 8 {
		t.Errorf("expected input_tokens=8, got %d", usage["input_tokens"])
	}
	if usage["output_tokens"] != 5 {
		t.Errorf("expected output_tokens=5, got %d", usage["output_tokens"])
	}
	if usage["cache_read_input_tokens"] != 3 {
		t.Errorf("expected cache_read_input_tokens=3, got %d", usage["cache_read_input_tokens"])
	}
}

func TestChatCompletionsReasoningContent(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte(
		"data: {\"id\":\"c_kimi\",\"choices\":[{\"delta\":{\"role\":\"assistant\",\"reasoning_content\":\"Think \"}}]}\n\n"+
			"data: {\"id\":\"c_kimi\",\"choices\":[{\"delta\":{\"reasoning_content\":\"carefully.\"}}]}\n\n"+
			"data: {\"id\":\"c_kimi\",\"choices\":[{\"delta\":{\"content\":\"Done.\"}}]}\n\n"+
			"data: [DONE]\n\n",
	))

	snap := r.Reconstruct()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	msg := snap["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)
	if msg["reasoning_content"] != "Think carefully." {
		t.Errorf("expected reasoning 'Think carefully.', got %v", msg["reasoning_content"])
	}
	if msg["content"] != "Done." {
		t.Errorf("expected content 'Done.', got %v", msg["content"])
	}

	content := snap["content"].([]any)
	thinking := content[0].(map[string]any)
	if thinking["type"] != "thinking" || thinking["thinking"] != "Think carefully." {
		t.Errorf("expected thinking mirror block, got %v", thinking)
	}
	text := content[1].(map[string]any)
	if text["type"] != "text" || text["text"] != "Done." {
		t.Errorf("expected text mirror block, got %v", text)
	}
}

func TestChatCompletionsToolCallAccumulation(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte(
		"data: {\"id\":\"c1\",\"choices\":[{\"delta\":{\"role\":\"assistant\",\"tool_calls\":"+
			"[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"\"}}]}}]}\n\n"+
			"data: {\"id\":\"c1\",\"choices\":[{\"delta\":{\"tool_calls\":"+
			"[{\"index\":0,\"function\":{\"arguments\":\"{\\\"city\\\":\"}}]}}]}\n\n"+
			"data: {\"id\":\"c1\",\"choices\":[{\"delta\":{\"tool_calls\":"+
			"[{\"index\":0,\"function\":{\"arguments\":\"\\\"SF\\\"}\"}}]}}]}\n\n"+
			"data: {\"id\":\"c1\",\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n"+
			"data: [DONE]\n\n",
	))

	snap := r.Reconstruct()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	msg := snap["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)
	tc := msg["tool_calls"].([]any)[0].(map[string]any)
	fn := tc["function"].(map[string]any)
	if tc["id"] != "call_1" {
		t.Errorf("expected id 'call_1', got %v", tc["id"])
	}
	if fn["name"] != "get_weather" {
		t.Errorf("expected name 'get_weather', got %v", fn["name"])
	}
	if fn["arguments"] != `{"city":"SF"}` {
		t.Errorf("expected arguments '{\"city\":\"SF\"}', got %v", fn["arguments"])
	}

	toolUseBlocks := []map[string]any{}
	for _, b := range snap["content"].([]any) {
		block := b.(map[string]any)
		if block["type"] == "tool_use" {
			toolUseBlocks = append(toolUseBlocks, block)
		}
	}
	if len(toolUseBlocks) != 1 {
		t.Fatalf("expected 1 tool_use block, got %d", len(toolUseBlocks))
	}
	if toolUseBlocks[0]["id"] != "call_1" {
		t.Errorf("expected mirrored id 'call_1', got %v", toolUseBlocks[0]["id"])
	}
	if toolUseBlocks[0]["name"] != "get_weather" {
		t.Errorf("expected mirrored name 'get_weather', got %v", toolUseBlocks[0]["name"])
	}
}

func TestChatCompletionsParallelToolCalls(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte(
		"data: {\"id\":\"c3\",\"choices\":[{\"delta\":{\"role\":\"assistant\",\"tool_calls\":["+
			"{\"index\":0,\"id\":\"a\",\"type\":\"function\",\"function\":{\"name\":\"f1\",\"arguments\":\"{}\"}},"+
			"{\"index\":1,\"id\":\"b\",\"type\":\"function\",\"function\":{\"name\":\"f2\",\"arguments\":\"{}\"}}"+
			"]}}]}\n\n"+
			"data: {\"id\":\"c3\",\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n"+
			"data: [DONE]\n\n",
	))

	snap := r.Reconstruct()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	toolUseBlocks := []map[string]any{}
	for _, b := range snap["content"].([]any) {
		block := b.(map[string]any)
		if block["type"] == "tool_use" {
			toolUseBlocks = append(toolUseBlocks, block)
		}
	}
	names := []string{}
	for _, b := range toolUseBlocks {
		names = append(names, b["name"].(string))
	}
	if len(names) != 2 || names[0] != "f1" || names[1] != "f2" {
		t.Errorf("expected tool names [f1, f2], got %v", names)
	}
}

func TestDoneSentinelFiltered(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte("data: [DONE]\n\n"))
	if len(r.Events) != 0 {
		t.Errorf("expected 0 events for [DONE], got %d", len(r.Events))
	}
}

func TestChunkedAcrossFeeds(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte("data: {\"id\":\"c1\",\"choices\":[{\"de"))
	r.FeedBytes([]byte("lta\":{\"content\":\"hel"))
	r.FeedBytes([]byte("lo\"}}]}\n\n"))
	if len(r.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(r.Events))
	}
	delta := r.Events[0]["data"].(map[string]any)["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
	if delta["content"] != "hello" {
		t.Errorf("expected content 'hello', got %v", delta["content"])
	}
}

func TestUsageOnlyFinalChunk(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte(
		"data: {\"id\":\"c4\",\"model\":\"hy3\",\"choices\":[{\"delta\":{\"role\":\"assistant\",\"content\":\"hi\"}}]}\n\n"+
			"data: {\"id\":\"c4\",\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"+
			"data: {\"id\":\"c4\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":2,\"total_tokens\":12}}\n\n"+
			"data: [DONE]\n\n",
	))

	snap := r.Reconstruct()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	usageRaw := snap["usage"]
	usage, ok := usageRaw.(map[string]int64)
	if !ok {
		t.Fatalf("expected usage to be map[string]int64, got %T", usageRaw)
	}
	if usage["input_tokens"] != 10 {
		t.Errorf("expected input_tokens=10, got %d", usage["input_tokens"])
	}
	if usage["output_tokens"] != 2 {
		t.Errorf("expected output_tokens=2, got %d", usage["output_tokens"])
	}
}

func TestUsageOnlyChunkWithoutPriorSnapshotIsSkipped(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":1}}\n\n"))
	if r.Reconstruct() != nil {
		t.Error("expected nil snapshot when no prior choices set up")
	}
}

func TestMixedEventAndBareData(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte("data: {\"bare\":1}\n\nevent: ping\ndata: {\"named\":2}\n\ndata: {\"bare\":3}\n\n"))

	expectedTypes := []string{"message", "ping", "message"}
	if len(r.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(r.Events))
	}
	for i, ev := range r.Events {
		if ev["event"] != expectedTypes[i] {
			t.Errorf("event %d: expected type %q, got %v", i, expectedTypes[i], ev["event"])
		}
	}
}

func TestAnthropicThinkingBlock(t *testing.T) {
	r := sse.NewSSEReassembler()
	r.FeedBytes([]byte(
		"event: message_start\n"+
			"data: {\"type\":\"message_start\",\"message\":{\"id\":\"m2\",\"role\":\"assistant\","+
			"\"content\":[],\"model\":\"claude-x\",\"usage\":{\"input_tokens\":5,\"output_tokens\":0}}}\n\n"+
			"event: content_block_start\n"+
			"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"thinking\",\"thinking\":\"\"}}\n\n"+
			"event: content_block_delta\n"+
			"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"Let me think\"}}\n\n"+
			"event: content_block_stop\n"+
			"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n"+
			"event: content_block_start\n"+
			"data: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"+
			"event: content_block_delta\n"+
			"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"text_delta\",\"text\":\"Answer\"}}\n\n"+
			"event: content_block_stop\n"+
			"data: {\"type\":\"content_block_stop\",\"index\":1}\n\n"+
			"event: message_delta\n"+
			"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":10}}\n\n"+
			"event: message_stop\n"+
			"data: {\"type\":\"message_stop\"}\n\n",
	))

	snap := r.Reconstruct()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	content := snap["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(content))
	}
	thinking := content[0].(map[string]any)
	if thinking["type"] != "thinking" || thinking["thinking"] != "Let me think" {
		t.Errorf("expected thinking block, got %v", thinking)
	}
	text := content[1].(map[string]any)
	if text["type"] != "text" || text["text"] != "Answer" {
		t.Errorf("expected text block 'Answer', got %v", text)
	}
}
