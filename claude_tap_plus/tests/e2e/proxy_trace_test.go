package e2e_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/proxy"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/testutil"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
)

// TestNonStreamingRequest tests the full proxy chain with a non-streaming API call.
func TestNonStreamingRequest(t *testing.T) {
	fx := testutil.LoadFixture(t, "nonstreaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	tracePath := testutil.TempTraceFile(t)
	writer, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}
	defer writer.Close()

	rp := proxy.NewReverseProxy(mockServer.URL, writer)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)
	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	testutil.AssertResponseStatus(t, resp, 200)

	body := testutil.DrainAndClose(t, resp)
	var respBody map[string]any
	if err := json.Unmarshal(body, &respBody); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	// Verify response content.
	if respBody["type"] != "message" {
		t.Errorf("response type: got %v, want message", respBody["type"])
	}
	if respBody["stop_reason"] != "tool_use" {
		t.Errorf("stop_reason: got %v, want tool_use", respBody["stop_reason"])
	}

	// Verify trace.
	writer.Close()
	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) != 1 {
		t.Fatalf("expected 1 trace record, got %d", len(records))
	}
	testutil.AssertTrace(t, records[0], fx.ExpectedTrace)

	// Verify sensitive headers are redacted in trace.
	reqHeaders, _ := records[0]["request"].(map[string]any)["headers"].(map[string]any)
	if auth, ok := reqHeaders["Authorization"].(string); ok {
		if auth == "Bearer sk-ant-test-key-12345678" {
			t.Error("Authorization header not redacted in trace")
		}
	}
	if apiKey, ok := reqHeaders["X-Api-Key"].(string); ok {
		if apiKey == "sk-ant-test-key-12345678" {
			t.Error("X-Api-Key header not redacted in trace")
		}
	}
}

// TestStreamingSSE tests the full proxy chain with SSE streaming.
func TestStreamingSSE(t *testing.T) {
	fx := testutil.LoadFixture(t, "streaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	tracePath := testutil.TempTraceFile(t)
	writer, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}
	defer writer.Close()

	rp := proxy.NewReverseProxy(mockServer.URL, writer)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)
	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	testutil.AssertResponseStatus(t, resp, 200)

	// Read streaming response — should contain SSE events.
	body := testutil.DrainAndClose(t, resp)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "message_start") {
		t.Error("response missing message_start event")
	}
	if !strings.Contains(bodyStr, "content_block_delta") {
		t.Error("response missing content_block_delta event")
	}
	if !strings.Contains(bodyStr, "message_stop") {
		t.Error("response missing message_stop event")
	}

	// Verify trace.
	writer.Close()
	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) != 1 {
		t.Fatalf("expected 1 trace record, got %d", len(records))
	}
	testutil.AssertTrace(t, records[0], fx.ExpectedTrace)

	// Verify trace contains SSE events.
	respMap, _ := records[0]["response"].(map[string]any)
	if sseEvents, ok := respMap["sse_events"].([]any); ok {
		if len(sseEvents) == 0 {
			t.Error("trace has no sse_events for streaming response")
		}
	}
}

// TestMultiTurn tests multiple sequential requests through the same proxy.
func TestMultiTurn(t *testing.T) {
	fx := testutil.LoadFixture(t, "nonstreaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	tracePath := testutil.TempTraceFile(t)
	writer, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}
	defer writer.Close()

	rp := proxy.NewReverseProxy(mockServer.URL, writer)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)

	// Send 3 requests.
	for i := 0; i < 3; i++ {
		resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
		testutil.AssertResponseStatus(t, resp, 200)
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	// Verify trace has 3 records with incrementing turn numbers.
	writer.Close()
	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) != 3 {
		t.Fatalf("expected 3 trace records, got %d", len(records))
	}

	for i, rec := range records {
		turn, _ := rec["turn"].(float64)
		if int(turn) != i+1 {
			t.Errorf("record %d: turn = %v, want %d", i, turn, i+1)
		}
		// All should have same session_id.
		sid, _ := rec["session_id"].(string)
		if sid != "test-session-001" {
			t.Errorf("record %d: session_id = %q, want test-session-001", i, sid)
		}
	}
}

// TestSessionIDExtraction tests that session_id is correctly extracted from metadata.user_id.
func TestSessionIDExtraction(t *testing.T) {
	fx := testutil.LoadFixture(t, "nonstreaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	tracePath := testutil.TempTraceFile(t)
	writer, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}
	defer writer.Close()

	rp := proxy.NewReverseProxy(mockServer.URL, writer)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)
	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	io.ReadAll(resp.Body)
	resp.Body.Close()

	writer.Close()
	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) == 0 {
		t.Fatal("no trace records")
	}

	sid, _ := records[0]["session_id"].(string)
	if sid != "test-session-001" {
		t.Errorf("session_id: got %q, want %q", sid, "test-session-001")
	}
}

// TestTokenStats verifies that TraceWriter.Summary() accumulates token counts correctly.
func TestTokenStats(t *testing.T) {
	fx := testutil.LoadFixture(t, "nonstreaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	tracePath := testutil.TempTraceFile(t)
	writer, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}
	defer writer.Close()

	rp := proxy.NewReverseProxy(mockServer.URL, writer)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)
	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	io.ReadAll(resp.Body)
	resp.Body.Close()

	summary := writer.Summary()

	apiCalls, _ := summary["api_calls"].(int)
	if apiCalls != 1 {
		t.Errorf("api_calls: got %d, want 1", apiCalls)
	}

	inputTokens, _ := summary["input_tokens"].(int64)
	if inputTokens != 1234 {
		t.Errorf("input_tokens: got %d, want 1234", inputTokens)
	}

	outputTokens, _ := summary["output_tokens"].(int64)
	if outputTokens != 56 {
		t.Errorf("output_tokens: got %d, want 56", outputTokens)
	}
}

// TestHeaderRedaction verifies that sensitive headers are redacted in trace output.
func TestHeaderRedaction(t *testing.T) {
	fx := testutil.LoadFixture(t, "nonstreaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	tracePath := testutil.TempTraceFile(t)
	writer, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}
	defer writer.Close()

	rp := proxy.NewReverseProxy(mockServer.URL, writer)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)
	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	io.ReadAll(resp.Body)
	resp.Body.Close()

	writer.Close()
	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) == 0 {
		t.Fatal("no trace records")
	}

	reqMap, _ := records[0]["request"].(map[string]any)
	headers, _ := reqMap["headers"].(map[string]any)

	// X-Api-Key should be truncated.
	if apiKey, ok := headers["X-Api-Key"].(string); ok {
		if apiKey == "sk-ant-test-key-12345678" {
			t.Error("X-Api-Key not redacted in trace")
		}
	}

	// Authorization should be masked.
	if auth, ok := headers["Authorization"].(string); ok {
		if auth == "Bearer sk-ant-test-key-12345678" {
			t.Error("Authorization not redacted in trace")
		}
	}
}

// TestBlockedPath verifies that non-whitelisted paths are rejected.
func TestBlockedPath(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("should not reach"))
	})
	mockServer := httptest.NewServer(mockHandler)
	defer mockServer.Close()

	tracePath := testutil.TempTraceFile(t)
	writer, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}
	defer writer.Close()

	rp := proxy.NewReverseProxy(mockServer.URL, writer)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// Request to a blocked path.
	resp, err := http.Get("http://127.0.0.1:" + fmt.Sprint(port) + "/admin/config")
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("blocked path status: got %d, want 404", resp.StatusCode)
	}

	// Trace should be empty (no record written for blocked path).
	writer.Close()
	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) != 0 {
		t.Errorf("expected 0 trace records for blocked path, got %d", len(records))
	}
}

// TestGzipResponse verifies that gzip-compressed responses are correctly decompressed in trace.
func TestGzipResponse(t *testing.T) {
	fx := testutil.LoadFixture(t, "gzip_response.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	tracePath := testutil.TempTraceFile(t)
	writer, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		t.Fatalf("create writer: %v", err)
	}
	defer writer.Close()

	rp := proxy.NewReverseProxy(mockServer.URL, writer)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)
	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	testutil.AssertResponseStatus(t, resp, 200)

	// Client receives decompressed response.
	body := testutil.DrainAndClose(t, resp)
	var respBody map[string]any
	if err := json.Unmarshal(body, &respBody); err != nil {
		t.Fatalf("parse response (should be decompressed): %v", err)
	}
	if respBody["type"] != "message" {
		t.Errorf("response type: got %v, want message", respBody["type"])
	}

	// Trace should have decompressed body.
	writer.Close()
	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) == 0 {
		t.Fatal("no trace records")
	}
	testutil.AssertTrace(t, records[0], fx.ExpectedTrace)
}
