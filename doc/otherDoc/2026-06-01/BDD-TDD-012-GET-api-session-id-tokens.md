# BDD + TDD: GET /api/session/{id}/tokens Token 使用统计

> 接口: `GET /api/session/{id}/tokens`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/session/{id}/tokens` |
| 方法 | GET |
| 功能 | 获取指定会话的 Token 使用统计 |
| 路径参数 | `id` — session_id |
| 响应 | `SessionTokensResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 获取会话 Token 使用统计
  作为监控用户
  我需要查看某个会话的 Token 消耗情况
  以便了解 API 使用成本

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And 存在会话 "session-abc123"
    And 该会话有以下 Token 使用记录:
      | timestamp           | input_tokens | output_tokens | model         |
      | 2026-06-01T10:00:00Z | 1024         | 512           | claude-3-opus |
      | 2026-06-01T10:05:00Z | 2048         | 1024          | claude-3-opus |
      | 2026-06-01T10:10:00Z | 512          | 256           | claude-3-opus |

  @positive
  Scenario: 获取会话 Token 统计
    When 发送 GET 请求到 /api/session/session-abc123/tokens
    Then 响应状态码应为 200
    And 响应体应包含:
      """
      {
        "session_id": "session-abc123",
        "total_input_tokens": 3584,
        "total_output_tokens": 1792,
        "total_tokens": 5376,
        "request_count": 3,
        "model": "claude-3-opus",
        "breakdown": [
          {
            "timestamp": "2026-06-01T10:00:00Z",
            "input_tokens": 1024,
            "output_tokens": 512
          },
          {
            "timestamp": "2026-06-01T10:05:00Z",
            "input_tokens": 2048,
            "output_tokens": 1024
          },
          {
            "timestamp": "2026-06-01T10:10:00Z",
            "input_tokens": 512,
            "output_tokens": 256
          }
        ]
      }
      """

  @positive
  Scenario: 会话无 Token 记录
    Given 存在会话 "session-no-tokens"
    And 该会话无 Token 使用记录
    When 发送 GET 请求到 /api/session/session-no-tokens/tokens
    Then 响应状态码应为 200
    And 响应体应包含:
      """
      {
        "session_id": "session-no-tokens",
        "total_input_tokens": 0,
        "total_output_tokens": 0,
        "total_tokens": 0,
        "request_count": 0,
        "breakdown": []
      }
      """

  @negative
  Scenario: 获取不存在的会话
    When 发送 GET 请求到 /api/session/session-not-exist/tokens
    Then 响应状态码应为 404
    And 响应体应包含错误码 "not_found"

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/session/session-abc123/tokens
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义响应类型

```go
// internal/backend/api/response.go

// SessionTokensResponse 是会话 Token 统计的响应体。
type SessionTokensResponse struct {
	SessionID         string           `json:"session_id"`
	TotalInputTokens  int64            `json:"total_input_tokens"`
	TotalOutputTokens int64            `json:"total_output_tokens"`
	TotalTokens       int64            `json:"total_tokens"`
	RequestCount      int              `json:"request_count"`
	Model             string           `json:"model,omitempty"`
	Breakdown         []TokenBreakdown `json:"breakdown"`
}

// TokenBreakdown 是单次请求的 Token 明细。
type TokenBreakdown struct {
	Timestamp    time.Time `json:"timestamp"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
}
```

### Step 2: 定义 Token 存储接口

```go
// internal/backend/store/store.go

// TokenStore 接口定义 Token 统计的存储操作。
type TokenStore interface {
	RecordTokenUsage(ctx context.Context, sessionID string, inputTokens, outputTokens int64, model string) error
	GetSessionTokens(ctx context.Context, sessionID string) (*TokenStats, error)
}

// TokenStats 是会话的 Token 统计。
type TokenStats struct {
	SessionID         string
	TotalInputTokens  int64
	TotalOutputTokens int64
	TotalTokens       int64
	RequestCount      int
	Model             string
	Breakdown         []TokenUsageRecord
}

// TokenUsageRecord 是单次 Token 使用记录。
type TokenUsageRecord struct {
	Timestamp    time.Time
	InputTokens  int64
	OutputTokens int64
}
```

### Step 3: 实现 Token 存储层

```go
// internal/backend/store/token_store.go

package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type tokenStore struct {
	db *sql.DB
}

func newTokenStore(db *sql.DB) *tokenStore {
	return &tokenStore{db: db}
}

func (s *tokenStore) RecordTokenUsage(ctx context.Context, sessionID string, inputTokens, outputTokens int64, model string) error {
	logger.Debug("store.token", "RecordTokenUsage: session=%s input=%d output=%d model=%s",
		sessionID, inputTokens, outputTokens, model)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO token_usage(session_id, input_tokens, output_tokens, model, timestamp)
		 VALUES (?, ?, ?, ?, ?)`,
		sessionID, inputTokens, outputTokens, model, time.Now())
	return err
}

func (s *tokenStore) GetSessionTokens(ctx context.Context, sessionID string) (*TokenStats, error) {
	logger.Debug("store.token", "GetSessionTokens: session=%s", sessionID)

	// 获取汇总统计
	var stats TokenStats
	var model sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT 
		   COALESCE(SUM(input_tokens), 0),
		   COALESCE(SUM(output_tokens), 0),
		   COALESCE(SUM(input_tokens + output_tokens), 0),
		   COUNT(*),
		   MAX(model)
		 FROM token_usage WHERE session_id = ?`,
		sessionID).Scan(
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.TotalTokens,
		&stats.RequestCount,
		&model,
	)
	if err != nil {
		return nil, err
	}

	stats.SessionID = sessionID
	if model.Valid {
		stats.Model = model.String
	}

	// 获取明细记录
	rows, err := s.db.QueryContext(ctx,
		`SELECT timestamp, input_tokens, output_tokens 
		 FROM token_usage WHERE session_id = ? ORDER BY timestamp`,
		sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r TokenUsageRecord
		if err := rows.Scan(&r.Timestamp, &r.InputTokens, &r.OutputTokens); err != nil {
			return nil, err
		}
		stats.Breakdown = append(stats.Breakdown, r)
	}

	return &stats, rows.Err()
}
```

### Step 4: 实现 Token 服务层

```go
// internal/backend/service/token_service.go

package service

import (
	"context"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type TokenService struct {
	store store.TokenStore
}

func NewTokenService(s store.TokenStore) *TokenService {
	return &TokenService{store: s}
}

func (svc *TokenService) GetSessionTokens(ctx context.Context, sessionID string) (*store.TokenStats, error) {
	logger.Debug("svc.token", "GetSessionTokens: session=%s", sessionID)
	return svc.store.GetSessionTokens(ctx, sessionID)
}
```

### Step 5: 实现 Handler 层

```go
// internal/backend/api/session_handler.go

// GetTokens 处理获取会话 Token 统计的请求。
func (h *SessionHandler) GetTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	path := r.URL.Path
	prefix := "/api/session/"
	suffix := "/tokens"
	sessionID := path[len(prefix) : len(path)-len(suffix)]
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session_id is required")
		return
	}

	logger.Debug("api.session", "GET /api/session/%s/tokens", sessionID)

	// 检查会话是否存在
	sess, err := h.svc.Get(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get session")
		return
	}
	if sess == nil {
		writeError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}

	// 获取 Token 统计
	stats, err := h.tokenSvc.GetSessionTokens(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get tokens")
		return
	}

	breakdown := make([]TokenBreakdown, len(stats.Breakdown))
	for i, b := range stats.Breakdown {
		breakdown[i] = TokenBreakdown{
			Timestamp:    b.Timestamp,
			InputTokens:  b.InputTokens,
			OutputTokens: b.OutputTokens,
		}
	}

	writeJSON(w, http.StatusOK, SessionTokensResponse{
		SessionID:         sessionID,
		TotalInputTokens:  stats.TotalInputTokens,
		TotalOutputTokens: stats.TotalOutputTokens,
		TotalTokens:       stats.TotalTokens,
		RequestCount:      stats.RequestCount,
		Model:             stats.Model,
		Breakdown:         breakdown,
	})
}
```

### Step 6: 修改 SessionHandler 注入 TokenService

```go
// internal/backend/api/session_handler.go

type SessionHandler struct {
	svc      *service.SessionService
	issueSvc *service.IssueService
	tokenSvc *service.TokenService // 新增
	traceSvc *service.TraceService
}

func NewSessionHandler(svc *service.SessionService, issueSvc *service.IssueService, tokenSvc *service.TokenService, traceSvc *service.TraceService) *SessionHandler {
	return &SessionHandler{svc: svc, issueSvc: issueSvc, tokenSvc: tokenSvc, traceSvc: traceSvc}
}
```

### Step 7: 注册路由

```go
// internal/backend/api/router.go

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	// ... 已有路由
	mux.HandleFunc("/api/session/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/issues"):
			h.Session.GetIssues(w, r)
		case strings.HasSuffix(path, "/tokens"):
			h.Session.GetTokens(w, r) // 新增
		case strings.HasSuffix(path, "/traces"):
			h.Session.GetTraces(w, r)
		default:
			h.Session.Get(w, r)
		}
	})
	return mux
}
```

---

## 四、Todo List

- [ ] Step 1: 定义 SessionTokensResponse 和 TokenBreakdown 类型
- [ ] Step 2: 定义 TokenStore 接口和 TokenStats 模型
- [ ] Step 3: 实现 tokenStore（含 RecordTokenUsage 和 GetSessionTokens）
- [ ] Step 4: 实现 TokenService
- [ ] Step 5: 修改 SessionHandler 注入 TokenService
- [ ] Step 6: 实现 SessionHandler.GetTokens
- [ ] Step 7: 在 router.go 中注册 /api/session/{id}/tokens 路由
- [ ] Step 8: 添加 token_usage 表迁移脚本
- [ ] Step 9: 编写单元测试 token_api_test.go
- [ ] Step 10: 运行 BDD 测试验证
- [ ] Step 11: 更新 API 文档

---

*文档结束*
