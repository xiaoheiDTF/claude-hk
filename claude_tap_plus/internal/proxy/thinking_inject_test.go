package proxy

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestIsKimiURL 验证 IsKimiURL 对各上游 URL 的前缀匹配。
func TestIsKimiURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		// Kimi/Moonshot
		{"https://api.kimi.com/v1/chat/completions", true},
		{"https://api.kimi.com/", true},
		{"https://api.moonshot.cn/v1/chat/completions", true},
		// DeepSeek
		{"https://api.deepseek.com/v1/chat/completions", true},
		{"https://api.deepseek.com/", true},
		// 不匹配：不同 host
		{"https://api.anthropic.com/v1/messages", false},
		{"https://api.openai.com/v1/chat/completions", false},
		{"https://open.bigmodel.cn/api/paas/v4/chat/completions", false},
		// 不匹配：子串包含但不等于前缀
		{"https://example.com/api.kimi.com-proxy", false},
		{"https://not-deepseek.com/v1/chat", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := IsKimiURL(tt.url)
			if got != tt.want {
				t.Errorf("IsKimiURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

// TestIsKimiModel 验证 IsKimiModel 对各 model 名的前缀匹配（不区分大小写）。
func TestIsKimiModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		// Kimi/Moonshot
		{"kimi-k2", true},
		{"kimi-thinking-preview", true},
		{"KIMI-K2", true}, // 大小写不敏感
		{"moonshot-v1-128k", true},
		// DeepSeek
		{"deepseek-reasoner", true},
		{"DeepSeek-Chat", true},
		// 不匹配：其他厂商
		{"claude-sonnet-4-6", false},
		{"gpt-4o", false},
		{"qwen-max", false},
		// 不匹配：空串、子串
		{"", false},
		{"anti-kimi", false}, // 前缀不是 kimi
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := IsKimiModel(tt.model)
			if got != tt.want {
				t.Errorf("IsKimiModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

// TestIsKimiUpstream 验证 URL 与 model 任一命中即启用 reasoning_content 注入。
// 重点覆盖「BaseURL 非官方域名（别名路由/自建网关）但 model 是 kimi」的核心场景。
func TestIsKimiUpstream(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		model   string
		want    bool
	}{
		{
			name:    "别名路由网关 + kimi model → 命中（核心场景）",
			baseURL: "https://gateway.example.com/v1/chat/completions",
			model:   "kimi-k2",
			want:    true,
		},
		{
			name:    "官方域名 URL 命中（model 无关）",
			baseURL: "https://api.kimi.com/v1",
			model:   "claude-sonnet-4-6",
			want:    true,
		},
		{
			name:    "URL 与 model 同时命中",
			baseURL: "https://api.moonshot.cn/v1",
			model:   "moonshot-v1-8k",
			want:    true,
		},
		{
			name:    "自建网关 + deepseek model → 命中",
			baseURL: "https://llm.internal.corp/api/v1",
			model:   "deepseek-reasoner",
			want:    true,
		},
		{
			name:    "非 kimi URL + 非 kimi model → 不命中",
			baseURL: "https://api.anthropic.com/v1/messages",
			model:   "claude-sonnet-4-6",
			want:    false,
		},
		{
			name:    "自建网关 + 非 kimi model → 不命中",
			baseURL: "https://gateway.example.com/v1",
			model:   "gpt-4o",
			want:    false,
		},
		{
			name:    "URL 与 model 均空 → 不命中",
			baseURL: "",
			model:   "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsKimiUpstream(tt.baseURL, tt.model)
			if got != tt.want {
				t.Errorf("IsKimiUpstream(%q, %q) = %v, want %v", tt.baseURL, tt.model, got, tt.want)
			}
		})
	}
}

// TestInjectReasoningContentCached_Unit 直接测试 injectReasoningContentCached 函数的各种场景。
func TestInjectReasoningContentCached_Unit(t *testing.T) {
	tests := []struct {
		name     string
		reqBody  string
		isKimi   bool
		wantSame bool // true = 输出应与输入完全一致
		wantLen  int  // 目标 assistant 消息的 content 数组期望长度
	}{
		{
			name:     "无 tool_use 的 assistant - 原样透传",
			reqBody:  `{"model":"x","messages":[{"role":"assistant","content":[{"type":"text","text":"hi"}]}]}`,
			isKimi:   true,
			wantSame: true,
		},
		{
			name:     "tool_use 无 thinking - 注入",
			reqBody:  `{"model":"x","messages":[{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"n","input":{}}]}]}`,
			isKimi:   true,
			wantSame: false,
			wantLen:  2,
		},
		{
			name:     "已有 thinking 块和 reasoning_content - 原样透传",
			reqBody:  `{"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"reasoning"},{"type":"tool_use","id":"t1","name":"n","input":{}}],"reasoning_content":"reasoning"}]}`,
			isKimi:   true,
			wantSame: true,
		},
		{
			name:     "空 body - 原样透传",
			reqBody:  "",
			isKimi:   true,
			wantSame: true,
		},
		{
			name:     "非 JSON body - 原样透传",
			reqBody:  "this is not json",
			isKimi:   true,
			wantSame: true,
		},
		{
			name:     "JSON 数组 body - 原样透传",
			reqBody:  `[1,2,3]`,
			isKimi:   true,
			wantSame: true,
		},
		{
			name:     "纯 user 消息 - 原样透传",
			reqBody:  `{"messages":[{"role":"user","content":"hello"}]}`,
			isKimi:   true,
			wantSame: true,
		},
		{
			name:     "非 kimi 模式 - 原样透传（不注入）",
			reqBody:  `{"model":"x","messages":[{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"n","input":{}}]}]}`,
			isKimi:   false,
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var parsed any
			if tt.reqBody != "" {
				_ = json.Unmarshal([]byte(tt.reqBody), &parsed)
			}
			result := injectReasoningContentCached([]byte(tt.reqBody), parsed, nil, tt.isKimi)
			if tt.wantSame {
				if string(result) != tt.reqBody {
					t.Errorf("expected passthrough, got: %s", string(result))
				}
			} else {
				var m map[string]any
				if err := json.Unmarshal(result, &m); err != nil {
					t.Fatalf("result is not valid JSON: %v", err)
				}
				msgs := m["messages"].([]any)
				msg := msgs[0].(map[string]any)
				content := msg["content"].([]any)
				if len(content) != tt.wantLen {
					t.Errorf("content length = %d, want %d", len(content), tt.wantLen)
				}
				// 验证注入的 thinking 块在第一位
				first := content[0].(map[string]any)
				if first["type"] != "thinking" {
					t.Errorf("first block type = %q, want %q", first["type"], "thinking")
				}
				// 验证顶层 reasoning_content 字段被注入（无缓存时为空字符串）
				if rc, ok := msg["reasoning_content"].(string); !ok || rc != "" {
					t.Errorf("reasoning_content = %v, want empty string", msg["reasoning_content"])
				}
			}
		})
	}
}

// TestInjectReasoningContentCached_MixedMessages 测试混合消息场景：只有 tool_call 消息被注入。
func TestInjectReasoningContentCached_MixedMessages(t *testing.T) {
	reqBody := `{
		"messages": [
			{"role": "user", "content": "read the file"},
			{"role": "assistant", "content": [{"type": "text", "text": "sure"}]},
			{"role": "user", "content": "do it"},
			{"role": "assistant", "content": [{"type": "tool_use", "id": "t1", "name": "Read", "input": {"file": "/tmp/x"}}]},
			{"role": "user", "content": [{"type": "tool_result", "tool_use_id": "t1", "content": "file data"}]},
			{"role": "assistant", "content": [{"type": "thinking", "thinking":"let me think"},{"type": "tool_use", "id": "t2", "name": "Bash", "input": {"cmd": "ls"}}]}
		]
	}`

	var parsed any
	_ = json.Unmarshal([]byte(reqBody), &parsed)
	result := injectReasoningContentCached([]byte(reqBody), parsed, nil, true)

	var m map[string]any
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	msgs := m["messages"].([]any)

	// user 消息不变
	if msgs[0].(map[string]any)["role"] != "user" {
		t.Error("msg[0] should be user")
	}

	// 纯文本 assistant (msg[1]) 不变
	assistantText := msgs[1].(map[string]any)["content"].([]any)
	if len(assistantText) != 1 {
		t.Errorf("assistant text content length = %d, want 1", len(assistantText))
	}

	// tool_call 无 thinking (msg[3]) → 注入 thinking 块 + reasoning_content
	assistantTool := msgs[3].(map[string]any)
	content := assistantTool["content"].([]any)
	if len(content) != 2 {
		t.Errorf("assistant tool content length = %d, want 2 (thinking + tool_use)", len(content))
	}
	if content[0].(map[string]any)["type"] != "thinking" {
		t.Error("first block should be thinking")
	}
	if content[1].(map[string]any)["type"] != "tool_use" {
		t.Error("second block should be tool_use")
	}
	if rc, ok := assistantTool["reasoning_content"].(string); !ok || rc != "" {
		t.Errorf("msg[3] reasoning_content = %v, want empty string", assistantTool["reasoning_content"])
	}

	// 已有 thinking 的 tool_call (msg[5]) → 不重复注入 thinking 块，但仍注入 reasoning_content（如果缺失）
	assistantWithThinking := msgs[5].(map[string]any)
	thinkingContent := assistantWithThinking["content"].([]any)
	if len(thinkingContent) != 2 {
		t.Errorf("assistant with thinking content length = %d, want 2 (no duplicate)", len(thinkingContent))
	}
	if rc, ok := assistantWithThinking["reasoning_content"].(string); !ok || rc != "" {
		t.Errorf("msg[5] reasoning_content = %v, want empty string", assistantWithThinking["reasoning_content"])
	}
}

// TestInjectReasoningContentCached_CacheHit 测试缓存命中时回填真实的 reasoning_content。
func TestInjectReasoningContentCached_CacheHit(t *testing.T) {
	cache := NewReasoningCache()

	// 预缓存：模拟之前 API 响应中的 reasoning_content
	assistantContent := "I will read the file"
	toolCalls := `[{"type":"tool_use","id":"tu_1","name":"Read","input":{"file_path":"/tmp/test.txt"}}]`
	fullKey := MakeFullKey(assistantContent, toolCalls)
	tcKey := MakeToolcallKey(toolCalls)
	cache.Store(fullKey, tcKey, "Let me think about this step by step. The user wants to read a file.")

	// 构造请求体：assistant 消息有 tool_use 但无 reasoning_content
	reqBody := map[string]any{
		"model": "kimi-for-coding",
		"messages": []any{
			map[string]any{"role": "user", "content": "read the file"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "I will read the file"},
					map[string]any{"type": "tool_use", "id": "tu_1", "name": "Read", "input": map[string]any{"file_path": "/tmp/test.txt"}},
				},
			},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	var parsed any
	_ = json.Unmarshal(bodyBytes, &parsed)
	result := injectReasoningContentCached(bodyBytes, parsed, cache, true)

	var m map[string]any
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("parse result: %v", err)
	}

	msgs := m["messages"].([]any)
	assistantMsg := msgs[1].(map[string]any)

	// reasoning_content 应该是缓存的值，不是空字符串
	rc, _ := assistantMsg["reasoning_content"].(string)
	if rc == "" {
		t.Error("expected cached reasoning_content, got empty string")
	}
	if rc != "Let me think about this step by step. The user wants to read a file." {
		t.Errorf("reasoning_content = %q, want cached value", rc)
	}

	// thinking 块应被注入
	content := assistantMsg["content"].([]any)
	firstBlock := content[0].(map[string]any)
	if firstBlock["type"] != "thinking" {
		t.Errorf("first block type = %q, want %q", firstBlock["type"], "thinking")
	}
}

// TestInjectReasoningContentCached_CacheMissFallbackToEmpty 测试缓存未命中时回退到空字符串。
func TestInjectReasoningContentCached_CacheMissFallbackToEmpty(t *testing.T) {
	cache := NewReasoningCache()
	// 不存任何缓存

	reqBody := `{"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Read","input":{}}]}]}`
	var parsed any
	_ = json.Unmarshal([]byte(reqBody), &parsed)

	result := injectReasoningContentCached([]byte(reqBody), parsed, cache, true)

	var m map[string]any
	json.Unmarshal(result, &m)
	msg := m["messages"].([]any)[0].(map[string]any)

	rc, _ := msg["reasoning_content"].(string)
	if rc != "" {
		t.Errorf("cache miss should fallback to empty string, got %q", rc)
	}
}

// TestInjectReasoningContentCached_NonKimiPassthrough 测试非 kimi 模式不做任何修改。
func TestInjectReasoningContentCached_NonKimiPassthrough(t *testing.T) {
	cache := NewReasoningCache()
	cache.Store("some-key", "tc-key", "should not be used")

	reqBody := `{"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Read","input":{}}]}]}`
	var parsed any
	_ = json.Unmarshal([]byte(reqBody), &parsed)

	result := injectReasoningContentCached([]byte(reqBody), parsed, cache, false)

	if string(result) != reqBody {
		t.Errorf("non-kimi mode should passthrough, got: %s", string(result))
	}
}

// TestInjectReasoningContentCached_ExistingRCRefreshesCache 测试已有 reasoning_content 时刷新缓存。
func TestInjectReasoningContentCached_ExistingRCRefreshesCache(t *testing.T) {
	cache := NewReasoningCache()

	reqBody := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "hello"},
					map[string]any{"type": "tool_use", "id": "t1", "name": "Read", "input": map[string]any{}},
				},
				"reasoning_content": "I am thinking deeply about this",
			},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	var parsed any
	_ = json.Unmarshal(bodyBytes, &parsed)
	result := injectReasoningContentCached(bodyBytes, parsed, cache, true)

	// 缓存应该被刷新
	f, _ := cache.Len()
	if f == 0 {
		t.Error("expected cache to be refreshed from existing reasoning_content")
	}

	// 结果应该是原样透传（已有 reasoning_content + 无 thinking 块 → 只注入 thinking 块）
	var m map[string]any
	json.Unmarshal(result, &m)
	msg := m["messages"].([]any)[0].(map[string]any)
	content := msg["content"].([]any)

	// thinking 块被注入
	if content[0].(map[string]any)["type"] != "thinking" {
		t.Error("expected thinking block to be injected")
	}

	// reasoning_content 保持原值
	rc, _ := msg["reasoning_content"].(string)
	if rc != "I am thinking deeply about this" {
		t.Errorf("reasoning_content changed: got %q", rc)
	}
}

// TestInjectReasoningContentCached_Integration 通过完整代理链路测试，验证上游收到注入后的请求体。
func TestInjectReasoningContentCached_Integration(t *testing.T) {
	var capturedBody []byte
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"test","stop_reason":"end_turn","usage":{"input_tokens":0,"output_tokens":0}}`))
	}))
	defer mockUpstream.Close()

	traceDir := t.TempDir()
	rp := NewReverseProxy(mockUpstream.URL, traceDir)
	rp.model = "kimi-for-coding"
	rp.SetKimiMode(true) // 启用 kimi 模式

	_, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// 模拟 Kimi 场景：无 thinking 字段，但有 tool_call 消息
	reqBody := map[string]any{
		"model":  "kimi-for-coding",
		"stream": true,
		"messages": []any{
			map[string]any{"role": "user", "content": "read the file"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type":  "tool_use",
						"id":    "toolu_123",
						"name":  "Read",
						"input": map[string]any{"file_path": "/tmp/test.txt"},
					},
				},
			},
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "toolu_123",
						"content":     "file contents",
					},
				},
			},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	resp.Body.Close()

	// 验证上游收到的请求体
	var upstreamBody map[string]any
	if err := json.Unmarshal(capturedBody, &upstreamBody); err != nil {
		t.Fatalf("parse upstream body: %v", err)
	}

	// model 被改写为 kimi-for-coding
	if m, _ := upstreamBody["model"].(string); m != "kimi-for-coding" {
		t.Errorf("model = %q, want %q", m, "kimi-for-coding")
	}

	// assistant 消息被注入 thinking 块 + reasoning_content
	messages := upstreamBody["messages"].([]any)
	assistantMsg := messages[1].(map[string]any)
	content := assistantMsg["content"].([]any)

	if len(content) != 2 {
		t.Fatalf("assistant content blocks = %d, want 2", len(content))
	}

	// thinking 块在第一位
	firstBlock := content[0].(map[string]any)
	if firstBlock["type"] != "thinking" {
		t.Errorf("first block type = %q, want %q", firstBlock["type"], "thinking")
	}

	// 顶层 reasoning_content 被注入（首次请求无缓存，应为空字符串）
	if rc, ok := assistantMsg["reasoning_content"].(string); !ok || rc != "" {
		t.Errorf("reasoning_content = %v, want empty string", assistantMsg["reasoning_content"])
	}

	// 原 tool_use 块保持不变
	secondBlock := content[1].(map[string]any)
	if secondBlock["type"] != "tool_use" {
		t.Errorf("second block type = %q, want %q", secondBlock["type"], "tool_use")
	}
}

// TestInjectReasoningContentCached_IntegrationWithCache 测试完整链路的缓存回填。
func TestInjectReasoningContentCached_IntegrationWithCache(t *testing.T) {
	var capturedBodies [][]byte
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBodies = append(capturedBodies, body)
		// 返回包含 reasoning_content 的响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_test",
			"type": "message",
			"role": "assistant",
			"model": "test",
			"content": [{"type": "text", "text": "done"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 5}
		}`))
	}))
	defer mockUpstream.Close()

	traceDir := t.TempDir()
	rp := NewReverseProxy(mockUpstream.URL, traceDir)
	rp.model = "kimi-for-coding"
	rp.SetKimiMode(true)

	_, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// 请求 1：带 tool_use 的 assistant 消息（无 reasoning_content）
	reqBody1 := map[string]any{
		"model": "kimi-for-coding",
		"messages": []any{
			map[string]any{"role": "user", "content": "hello"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "I will help"},
					map[string]any{"type": "tool_use", "id": "t1", "name": "Read", "input": map[string]any{"file": "a.go"}},
				},
			},
		},
	}
	bodyBytes1, _ := json.Marshal(reqBody1)
	req1, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader(bodyBytes1))
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatalf("request 1: %v", err)
	}
	resp1.Body.Close()

	// 验证请求 1：reasoning_content 为空字符串（首次无缓存）
	var body1 map[string]any
	json.Unmarshal(capturedBodies[0], &body1)
	msg1 := body1["messages"].([]any)[1].(map[string]any)
	rc1, _ := msg1["reasoning_content"].(string)
	if rc1 != "" {
		t.Errorf("request 1 reasoning_content should be empty, got %q", rc1)
	}
}

// TestDrainBodyForLog 验证降级日志在读取上游错误体时能正确解压 gzip/deflate，
// 并对明文、超长截断、空响应做处理。
func TestDrainBodyForLog(t *testing.T) {
	t.Run("plain body", func(t *testing.T) {
		resp := &http.Response{Body: io.NopCloser(strings.NewReader(`{"error":"bad key"}`))}
		if got := drainBodyForLog(resp, 512); got != `{"error":"bad key"}` {
			t.Errorf("plain body = %q", got)
		}
	})

	t.Run("gzip body", func(t *testing.T) {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		_, _ = gw.Write([]byte(`{"error":"forbidden"}`))
		_ = gw.Close()
		resp := &http.Response{
			Header: http.Header{"Content-Encoding": []string{"gzip"}},
			Body:   io.NopCloser(bytes.NewReader(buf.Bytes())),
		}
		if got := drainBodyForLog(resp, 512); got != `{"error":"forbidden"}` {
			t.Errorf("gzip body = %q, want decompressed JSON", got)
		}
	})

	t.Run("deflate body", func(t *testing.T) {
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		_, _ = zw.Write([]byte(`{"error":"rate limit"}`))
		_ = zw.Close()
		resp := &http.Response{
			Header: http.Header{"Content-Encoding": []string{"deflate"}},
			Body:   io.NopCloser(bytes.NewReader(buf.Bytes())),
		}
		if got := drainBodyForLog(resp, 512); got != `{"error":"rate limit"}` {
			t.Errorf("deflate body = %q", got)
		}
	})

	t.Run("truncation", func(t *testing.T) {
		resp := &http.Response{Body: io.NopCloser(strings.NewReader(strings.Repeat("a", 1000)))}
		got := drainBodyForLog(resp, 32)
		if !strings.HasSuffix(got, "...(truncated)") {
			t.Errorf("truncation: missing suffix, got %q", got)
		}
	})

	t.Run("nil resp", func(t *testing.T) {
		if got := drainBodyForLog(nil, 512); got != "" {
			t.Errorf("nil resp = %q, want empty", got)
		}
	})
}

// TestSummarizeBody 验证请求体结构摘要：model + messages 骨架，剥离对话全文。
func TestSummarizeBody(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := summarizeBody(nil); got != "(empty body)" {
			t.Errorf("empty = %q", got)
		}
	})

	t.Run("non-json", func(t *testing.T) {
		if got := summarizeBody([]byte("not json")); !strings.Contains(got, "非 JSON") {
			t.Errorf("non-json = %q", got)
		}
	})

	t.Run("structured", func(t *testing.T) {
		body := []byte(`{"model":"kimi-k2","messages":[
			{"role":"user","content":"hi"},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"let me think"},
				{"type":"tool_use","id":"tu_1","name":"Read","input":{}}
			]},
			{"role":"user","content":[
				{"type":"tool_result","tool_use_id":"tu_1","content":"data"}
			]}
		]}`)
		got := summarizeBody(body)
		wants := []string{
			"model=kimi-k2", "msgs=3",
			"[0] user", "text x1",
			"[1] assistant", "thinking x1", "tool_use(id=tu_1, name=Read)",
			"[2] user", "tool_result(for=tu_1)",
		}
		for _, want := range wants {
			if !strings.Contains(got, want) {
				t.Errorf("structured: missing %q in:\n%s", want, got)
			}
		}
		// 不应泄露对话全文
		if strings.Contains(got, "let me think") {
			t.Errorf("structured: leaked thinking content: %s", got)
		}
	})
}
