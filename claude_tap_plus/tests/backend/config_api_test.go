// Package backend_test 包含后端 Config API 的验收测试，覆盖 GET/PUT /api/config 接口。
package backend_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/api"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

// --- helpers ---

// configTestEnv 是 Config API 测试环境。
type configTestEnv struct {
	srv *httptest.Server
}

// setupConfigTest 创建 Config API 测试环境。
func setupConfigTest(t *testing.T) *configTestEnv {
	t.Helper()

	f, err := os.CreateTemp("", "test-config-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()

	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatal(err)
	}

	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	issueSvc := service.NewIssueService(s.Issues())
	sessionSvc := service.NewSessionService(s.Sessions())
	configSvc := service.NewConfigService(s.Configs())

	router := api.NewRouter(api.Handlers{
		Issue:  api.NewIssueHandler(issueSvc),
		Session: api.NewSessionHandler(sessionSvc, issueSvc, service.NewTokenService(s.Sessions()), service.NewTraceService(s.Sessions())),
		Config: api.NewConfigHandler(configSvc),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &configTestEnv{srv: srv}
}

// getConfig 发送 GET /api/config 请求。
func (e *configTestEnv) getConfig(t *testing.T) *http.Response {
	t.Helper()
	resp, err := http.Get(e.srv.URL + "/api/config")
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// putConfig 发送 PUT /api/config 请求。
func (e *configTestEnv) putConfig(t *testing.T, body string) *http.Response {
	t.Helper()
	resp, err := http.NewRequest(http.MethodPut, e.srv.URL+"/api/config", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Header.Set("Content-Type", "application/json")

	// 用 http.Client 发送
	client := &http.Client{}
	r, err := client.Do(resp)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// readConfigJSON 读取 HTTP 响应并解析为 JSON。
func readConfigJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
}

// --- tests ---

// TestConfig_GetDefaults 验证：GET 返回 5 个默认配置项。
func TestConfig_GetDefaults(t *testing.T) {
	env := setupConfigTest(t)

	resp := env.getConfig(t)

	var result struct {
		Config map[string]interface{} `json:"config"`
	}
	readConfigJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// 验证 5 个默认配置项都存在
	expectedKeys := []string{"log_level", "session_timeout", "max_concurrent", "auto_claim", "default_model"}
	for _, key := range expectedKeys {
		if _, ok := result.Config[key]; !ok {
			t.Errorf("expected config key %s to exist", key)
		}
	}

	// 验证默认值
	if result.Config["log_level"] != "info" {
		t.Errorf("expected log_level=info, got %v", result.Config["log_level"])
	}
	if result.Config["session_timeout"].(float64) != 3600 {
		t.Errorf("expected session_timeout=3600, got %v", result.Config["session_timeout"])
	}
	if result.Config["max_concurrent"].(float64) != 10 {
		t.Errorf("expected max_concurrent=10, got %v", result.Config["max_concurrent"])
	}
	if result.Config["auto_claim"] != true {
		t.Errorf("expected auto_claim=true, got %v", result.Config["auto_claim"])
	}
	if result.Config["default_model"] != "claude-3-opus" {
		t.Errorf("expected default_model=claude-3-opus, got %v", result.Config["default_model"])
	}
}

// TestConfig_UpdateSingleField 验证：PUT 更新单个配置项。
func TestConfig_UpdateSingleField(t *testing.T) {
	env := setupConfigTest(t)

	resp := env.putConfig(t, `{"log_level":"debug"}`)

	var result struct {
		Config map[string]interface{} `json:"config"`
	}
	readConfigJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if result.Config["log_level"] != "debug" {
		t.Errorf("expected log_level=debug, got %v", result.Config["log_level"])
	}

	// 其他配置项保持不变
	if result.Config["session_timeout"].(float64) != 3600 {
		t.Errorf("session_timeout should be unchanged, got %v", result.Config["session_timeout"])
	}
	if result.Config["auto_claim"] != true {
		t.Errorf("auto_claim should be unchanged, got %v", result.Config["auto_claim"])
	}
}

// TestConfig_UpdateMultipleFields 验证：PUT 同时更新多个配置项。
func TestConfig_UpdateMultipleFields(t *testing.T) {
	env := setupConfigTest(t)

	resp := env.putConfig(t, `{"log_level":"warn","session_timeout":7200}`)

	var result struct {
		Config map[string]interface{} `json:"config"`
	}
	readConfigJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if result.Config["log_level"] != "warn" {
		t.Errorf("expected log_level=warn, got %v", result.Config["log_level"])
	}
	if result.Config["session_timeout"].(float64) != 7200 {
		t.Errorf("expected session_timeout=7200, got %v", result.Config["session_timeout"])
	}

	// 未更新的配置项保持不变
	if result.Config["max_concurrent"].(float64) != 10 {
		t.Errorf("max_concurrent should be unchanged, got %v", result.Config["max_concurrent"])
	}
}

// TestConfig_InvalidValue 验证：无效配置值返回 400。
func TestConfig_InvalidValue(t *testing.T) {
	env := setupConfigTest(t)

	resp := env.putConfig(t, `{"session_timeout":-1}`)

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readConfigJSON(t, resp, &errResp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if errResp.Code != "invalid_config" {
		t.Errorf("expected error=invalid_config, got %s", errResp.Code)
	}
}

// TestConfig_InvalidLogLevel 验证：无效的 log_level 值返回 400。
func TestConfig_InvalidLogLevel(t *testing.T) {
	env := setupConfigTest(t)

	resp := env.putConfig(t, `{"log_level":"verbose"}`)

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readConfigJSON(t, resp, &errResp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if errResp.Code != "invalid_config" {
		t.Errorf("expected error=invalid_config, got %s", errResp.Code)
	}
}

// TestConfig_UnknownKey 验证：不存在的配置键返回 400。
func TestConfig_UnknownKey(t *testing.T) {
	env := setupConfigTest(t)

	resp := env.putConfig(t, `{"unknown_key":"value"}`)

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readConfigJSON(t, resp, &errResp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if errResp.Code != "unknown_config_key" {
		t.Errorf("expected error=unknown_config_key, got %s", errResp.Code)
	}
}

// TestConfig_MethodNotAllowed 验证：POST 返回 405。
func TestConfig_MethodNotAllowed(t *testing.T) {
	env := setupConfigTest(t)

	resp, err := http.Post(env.srv.URL+"/api/config", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readConfigJSON(t, resp, &errResp)

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
	if errResp.Code != "method_not_allowed" {
		t.Errorf("expected error=method_not_allowed, got %s", errResp.Code)
	}
}

// TestConfig_UpdatePreservesOthers 验证：更新不影响其他配置项。
func TestConfig_UpdatePreservesOthers(t *testing.T) {
	env := setupConfigTest(t)

	// 先获取默认配置
	resp := env.getConfig(t)
	var before struct {
		Config map[string]interface{} `json:"config"`
	}
	readConfigJSON(t, resp, &before)

	// 更新 default_model
	resp = env.putConfig(t, `{"default_model":"claude-sonnet"}`)

	var after struct {
		Config map[string]interface{} `json:"config"`
	}
	readConfigJSON(t, resp, &after)

	if after.Config["default_model"] != "claude-sonnet" {
		t.Errorf("expected default_model=claude-sonnet, got %v", after.Config["default_model"])
	}

	// log_level 应保持不变
	if after.Config["log_level"] != before.Config["log_level"] {
		t.Errorf("log_level changed unexpectedly: before=%v after=%v", before.Config["log_level"], after.Config["log_level"])
	}

	// auto_claim 应保持不变
	if after.Config["auto_claim"] != before.Config["auto_claim"] {
		t.Errorf("auto_claim changed unexpectedly: before=%v after=%v", before.Config["auto_claim"], after.Config["auto_claim"])
	}
}
