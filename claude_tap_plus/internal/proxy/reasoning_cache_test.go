package proxy

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestReasoningCache_StoreAndLookup(t *testing.T) {
	cache := NewReasoningCache()

	// 存入缓存
	cache.Store("full-1", "tc-1", "thinking step by step")

	// 精确查找
	got, found := cache.Lookup("full-1", "tc-1", true)
	if !found {
		t.Fatal("expected to find cached value")
	}
	if got != "thinking step by step" {
		t.Errorf("got %q, want %q", got, "thinking step by step")
	}

	// 不存在的 key - 由于 Level 3 兜底（lastValue），对 hasToolCalls=true 仍会命中
	// 所以用 hasToolCalls=false 来测试真正未命中的情况
	_, found = cache.Lookup("nonexistent", "nonexistent-tc", false)
	if found {
		t.Error("should not find nonexistent key with hasToolCalls=false")
	}
}

func TestReasoningCache_ThreeLevelLookup(t *testing.T) {
	cache := NewReasoningCache()
	cache.Store("full-A", "tc-A", "first reasoning")
	cache.Store("full-B", "tc-B", "second reasoning")

	// Level 1: full key 精确匹配
	got, found := cache.Lookup("full-A", "tc-A", true)
	if !found || got != "first reasoning" {
		t.Errorf("level 1 lookup failed: got=%q found=%v", got, found)
	}

	// Level 2: toolcall key 匹配（full key 不匹配）
	got, found = cache.Lookup("nonexistent-full", "tc-B", true)
	if !found || got != "second reasoning" {
		t.Errorf("level 2 lookup failed: got=%q found=%v", got, found)
	}

	// Level 3: 兜底 - 最近缓存值（仅对 hasToolCalls=true）
	got, found = cache.Lookup("nonexistent-full", "nonexistent-tc", true)
	if !found || got != "second reasoning" {
		t.Errorf("level 3 fallback failed: got=%q found=%v", got, found)
	}

	// Level 3 不应用于 hasToolCalls=false
	_, found = cache.Lookup("nonexistent-full", "nonexistent-tc", false)
	if found {
		t.Error("level 3 should not apply when hasToolCalls=false")
	}
}

func TestReasoningCache_EmptyValues(t *testing.T) {
	cache := NewReasoningCache()

	// 空 reasoning_content 不应缓存
	cache.Store("full-1", "tc-1", "")
	_, found := cache.Lookup("full-1", "tc-1", true)
	if found {
		t.Error("empty reasoning_content should not be cached")
	}
}

func TestReasoningCache_NoToolcallKey(t *testing.T) {
	cache := NewReasoningCache()
	cache.Store("full-1", "", "reasoning without toolcalls")

	// full key 应该能找到
	got, found := cache.Lookup("full-1", "", true)
	if !found || got != "reasoning without toolcalls" {
		t.Errorf("lookup failed: got=%q found=%v", got, found)
	}
}

func TestReasoningCache_Concurrent(t *testing.T) {
	cache := NewReasoningCache()
	var wg sync.WaitGroup

	// 并发写入
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			cache.Store("full-"+key, "tc-"+key, "reasoning-"+key)
		}(i)
	}

	// 并发读取
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			cache.Lookup("full-"+key, "tc-"+key, true)
		}(i)
	}

	wg.Wait()

	f, tc := cache.Len()
	if f == 0 {
		t.Error("expected some cached entries after concurrent writes")
	}
	_ = tc // toolcall count may vary
}

func TestReasoningCache_Len(t *testing.T) {
	cache := NewReasoningCache()
	cache.Store("full-1", "tc-1", "r1")
	cache.Store("full-2", "tc-2", "r2")
	cache.Store("full-3", "", "r3") // no tc key

	f, tc := cache.Len()
	if f != 3 {
		t.Errorf("full count: got %d, want 3", f)
	}
	if tc != 2 {
		t.Errorf("toolcall count: got %d, want 2", tc)
	}
}

func TestMakeFullKey(t *testing.T) {
	// 相同输入应产生相同 key
	key1 := MakeFullKey("hello", `[{"id":"t1"}]`)
	key2 := MakeFullKey("hello", `[{"id":"t1"}]`)
	if key1 != key2 {
		t.Error("same input should produce same key")
	}

	// 不同输入应产生不同 key
	key3 := MakeFullKey("world", `[{"id":"t1"}]`)
	if key1 == key3 {
		t.Error("different input should produce different key")
	}
}

func TestMakeToolcallKey(t *testing.T) {
	// 空 JSON 返回空字符串
	key := MakeToolcallKey("")
	if key != "" {
		t.Errorf("empty JSON should return empty key, got %q", key)
	}

	// 非空 JSON 返回非空 key
	key = MakeToolcallKey(`[{"id":"t1"}]`)
	if key == "" {
		t.Error("non-empty JSON should return non-empty key")
	}
}

func TestMakeFullKeyFromMsg(t *testing.T) {
	msg := map[string]any{
		"role": "assistant",
		"content": []any{
			map[string]any{"type": "text", "text": "I will read the file"},
			map[string]any{"type": "tool_use", "id": "tu_1", "name": "Read", "input": map[string]any{"file": "a.go"}},
		},
	}

	key := MakeFullKeyFromMsg(msg)
	if key == "" {
		t.Error("expected non-empty key from message with tool_use")
	}

	// 同一消息应产生相同 key
	key2 := MakeFullKeyFromMsg(msg)
	if key != key2 {
		t.Error("same message should produce same key")
	}
}

func TestMakeToolcallKeyFromMsg(t *testing.T) {
	// 有 tool_use 的消息
	msg := map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": "hello"},
			map[string]any{"type": "tool_use", "id": "tu_1", "name": "Read", "input": map[string]any{}},
		},
	}
	key := MakeToolcallKeyFromMsg(msg)
	if key == "" {
		t.Error("expected non-empty toolcall key")
	}

	// 没有 tool_use 的消息
	msg2 := map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": "just text"},
		},
	}
	key2 := MakeToolcallKeyFromMsg(msg2)
	if key2 != "" {
		t.Error("expected empty toolcall key for message without tool_use")
	}

	// content 是字符串
	msg3 := map[string]any{
		"content": "plain text content",
	}
	key3 := MakeToolcallKeyFromMsg(msg3)
	if key3 != "" {
		t.Error("expected empty toolcall key for string content")
	}
}

func TestExtractMsgParts(t *testing.T) {
	tests := []struct {
		name       string
		msg        map[string]any
		wantText   string
		wantTCName string // first tool_use name, empty if none
	}{
		{
			name: "text and tool_use",
			msg: map[string]any{
				"content": []any{
					map[string]any{"type": "text", "text": "I will read"},
					map[string]any{"type": "tool_use", "id": "tu_1", "name": "Read", "input": map[string]any{}},
				},
			},
			wantText:   "I will read",
			wantTCName: "Read",
		},
		{
			name: "thinking blocks excluded",
			msg: map[string]any{
				"content": []any{
					map[string]any{"type": "thinking", "thinking": "let me think"},
					map[string]any{"type": "text", "text": "result"},
					map[string]any{"type": "tool_use", "id": "tu_1", "name": "Bash", "input": map[string]any{}},
				},
			},
			wantText:   "result",
			wantTCName: "Bash",
		},
		{
			name: "string content",
			msg: map[string]any{
				"content": "just a string",
			},
			wantText:   "just a string",
			wantTCName: "",
		},
		{
			name: "multiple text blocks concatenated",
			msg: map[string]any{
				"content": []any{
					map[string]any{"type": "text", "text": "part1"},
					map[string]any{"type": "text", "text": "part2"},
				},
			},
			wantText:   "part1part2",
			wantTCName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, toolCalls := extractMsgParts(tt.msg)
			if text != tt.wantText {
				t.Errorf("text: got %q, want %q", text, tt.wantText)
			}
			if tt.wantTCName == "" {
				if len(toolCalls) != 0 {
					t.Errorf("expected no tool_calls, got %d", len(toolCalls))
				}
			} else {
				if len(toolCalls) == 0 {
					t.Fatal("expected tool_calls but got none")
				}
				tc, _ := toolCalls[0].(map[string]any)
				name, _ := tc["name"].(string)
				if name != tt.wantTCName {
					t.Errorf("tool name: got %q, want %q", name, tt.wantTCName)
				}
			}
		})
	}
}

// TestKeyDeterminism 验证 JSON 序列化顺序确定性
func TestKeyDeterminism(t *testing.T) {
	// Go 的 json.Marshal 对 map key 排序，因此序列化应是确定性的
	msg := map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": "hello"},
			map[string]any{"type": "tool_use", "id": "tu_1", "name": "Read", "input": map[string]any{"z": "1", "a": "2"}},
		},
	}

	key1 := MakeFullKeyFromMsg(msg)
	key2 := MakeFullKeyFromMsg(msg)

	if key1 != key2 {
		t.Errorf("keys not deterministic: %s != %s", key1, key2)
	}

	// 验证 JSON 序列化确定性
	tc, _ := json.Marshal([]any{
		map[string]any{"z": "1", "a": "2"},
	})
	// Go 的 json.Marshal 会排序 map keys
	if string(tc) != `[{"a":"2","z":"1"}]` {
		t.Logf("note: json.Marshal output = %s", string(tc))
	}
}
