package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRequestBodyRewrite_Model 测试请求体 model 字段改写规则
// 来源：契约 3 — 请求体改写规则
func TestRequestBodyRewrite_Model(t *testing.T) {
	tests := []struct {
		name        string
		model       string // proxy 配置的 model（空=不改写）
		reqBody     string // 原始请求体
		wantModel   string // 期望上游收到的 model 值（空=不改写，用原始值）
		reqMethod   string
		reqPath     string
	}{
		// 1.1-P1: 强制覆盖请求中的模型
		{
			name:      "1.1-P1 force override existing model",
			model:     "glm-5.1",
			reqBody:   `{"model":"claude-sonnet-4-6","stream":true}`,
			wantModel: "glm-5.1",
			reqMethod: "POST",
			reqPath:   "/v1/messages",
		},
		// 1.2-P1: 注入不存在的 model 字段
		{
			name:      "1.2-P1 inject model when missing",
			model:     "glm-5.1",
			reqBody:   `{"stream":true,"messages":[{"role":"user","content":"hi"}]}`,
			wantModel: "glm-5.1",
			reqMethod: "POST",
			reqPath:   "/v1/messages",
		},
		// 1.3-E1: model 相同，幂等替换
		{
			name:      "1.3-E1 idempotent when model same",
			model:     "glm-5.1",
			reqBody:   `{"model":"glm-5.1","stream":false}`,
			wantModel: "glm-5.1",
			reqMethod: "POST",
			reqPath:   "/v1/messages",
		},
		// 3.1-P1: model 为空，原样透传
		{
			name:      "3.1-P1 no model config - pass through",
			model:     "",
			reqBody:   `{"model":"claude-opus-4-8","stream":true}`,
			wantModel: "claude-opus-4-8",
			reqMethod: "POST",
			reqPath:   "/v1/messages",
		},
		// 4.1-E1: 空请求体
		{
			name:      "4.1-E1 empty body - skip rewrite",
			model:     "glm-5.1",
			reqBody:   "",
			wantModel: "",
			reqMethod: "GET",
			reqPath:   "/v1/models",
		},
		// 4.2-E1: 非 JSON 请求体
		{
			name:      "4.2-E1 non-JSON body - skip rewrite",
			model:     "glm-5.1",
			reqBody:   "this is not json",
			wantModel: "",
			reqMethod: "POST",
			reqPath:   "/v1/messages",
		},
		// X2: JSON 数组请求体
		{
			name:      "X2 JSON array body - skip rewrite",
			model:     "glm-5.1",
			reqBody:   `[1,2,3]`,
			wantModel: "",
			reqMethod: "POST",
			reqPath:   "/v1/messages",
		},
		// X3: model 字段非字符串
		{
			name:      "X3 model is number - force override to string",
			model:     "glm-5.1",
			reqBody:   `{"model":123,"stream":true}`,
			wantModel: "glm-5.1",
			reqMethod: "POST",
			reqPath:   "/v1/messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 上游服务器，捕获收到的请求体
			var capturedBody []byte
			mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedBody, _ = io.ReadAll(r.Body)
				// 返回简单响应
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"test","stop_reason":"end_turn","usage":{"input_tokens":0,"output_tokens":0}}`))
			}))
			defer mockUpstream.Close()

			// 创建代理，传入 model 配置
			traceDir := t.TempDir()
			rp := NewReverseProxy(mockUpstream.URL, traceDir)
			rp.model = tt.model

			// 启动代理
			_, err := rp.Start("127.0.0.1", 0)
			if err != nil {
				t.Fatalf("start proxy: %v", err)
			}
			defer rp.Stop()

			// 通过代理发送请求
			proxyURL := rp.URL()
			var body io.Reader
			if tt.reqBody != "" {
				body = bytes.NewReader([]byte(tt.reqBody))
			}
			req, err := http.NewRequest(tt.reqMethod, proxyURL+tt.reqPath, body)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			if tt.reqBody != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("send request: %v", err)
			}
			resp.Body.Close()

			// 验证上游收到的 model
			if tt.wantModel == "" && len(capturedBody) == 0 {
				return // 空 body 场景，无需验证 model
			}
			if len(capturedBody) == 0 {
				t.Fatal("upstream received empty body")
			}

			var upstreamBody map[string]any
			if err := json.Unmarshal(capturedBody, &upstreamBody); err != nil {
				// 非 JSON body（如 4.2-E1），验证原样透传
				if string(capturedBody) != tt.reqBody {
					t.Errorf("non-JSON body changed: got %q, want %q", string(capturedBody), tt.reqBody)
				}
				return
			}

			if tt.wantModel != "" {
				gotModel, _ := upstreamBody["model"].(string)
				if gotModel != tt.wantModel {
					t.Errorf("upstream model = %q, want %q", gotModel, tt.wantModel)
				}
			}
		})
	}
}

// TestRequestBodyRewrite_OtherFieldsUnchanged 测试改写 model 时其他字段不变
// 来源：契约 3 — 只改 model，不动其他字段
func TestRequestBodyRewrite_OtherFieldsUnchanged(t *testing.T) {
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
	rp.model = "glm-5.1"

	_, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// 发送包含多个字段的请求
	origBody := map[string]any{
		"model":      "claude-sonnet-4-6",
		"stream":     true,
		"max_tokens": 4096,
		"messages":   []map[string]string{{"role": "user", "content": "hello"}},
	}
	bodyBytes, _ := json.Marshal(origBody)

	req, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	resp.Body.Close()

	// 验证上游收到的完整请求体
	var upstreamBody map[string]any
	if err := json.Unmarshal(capturedBody, &upstreamBody); err != nil {
		t.Fatalf("parse upstream body: %v", err)
	}

	// model 被替换
	if m, _ := upstreamBody["model"].(string); m != "glm-5.1" {
		t.Errorf("model = %q, want %q", m, "glm-5.1")
	}
	// stream 不变
	if s, _ := upstreamBody["stream"].(bool); s != true {
		t.Errorf("stream = %v, want true", s)
	}
	// max_tokens 不变
	if mt, ok := upstreamBody["max_tokens"].(float64); !ok || mt != 4096 {
		t.Errorf("max_tokens = %v, want 4096", upstreamBody["max_tokens"])
	}
	// messages 不变
	msgs, ok := upstreamBody["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Errorf("messages length = %v, want 1", len(msgs))
	}
}
