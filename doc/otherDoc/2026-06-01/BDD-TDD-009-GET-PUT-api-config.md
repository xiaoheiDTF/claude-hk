# BDD + TDD: GET/PUT /api/config 系统配置

> 接口: `GET /api/config`, `PUT /api/config`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/config` |
| 方法 | GET, PUT |
| 功能 | 获取/更新系统运行时配置 |
| GET 响应 | `ConfigResponse` |
| PUT 请求体 | `ConfigUpdateRequest` |
| PUT 响应 | `ConfigResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 系统配置管理
  作为管理员
  我需要查看和修改系统运行时配置
  以便调整系统行为

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And 系统默认配置为:
      | 配置项            | 值          |
      | log_level         | info        |
      | session_timeout   | 3600        |
      | max_concurrent    | 10          |
      | auto_claim        | true        |
      | default_model     | claude-3-opus |

  @positive
  Scenario: 获取当前配置
    When 发送 GET 请求到 /api/config
    Then 响应状态码应为 200
    And 响应体应包含:
      """
      {
        "config": {
          "log_level": "info",
          "session_timeout": 3600,
          "max_concurrent": 10,
          "auto_claim": true,
          "default_model": "claude-3-opus"
        }
      }
      """

  @positive
  Scenario: 更新单个配置项
    When 发送 PUT 请求到 /api/config
    And 请求体为:
      """
      {
        "log_level": "debug"
      }
      """
    Then 响应状态码应为 200
    And 响应体中 config.log_level 应为 "debug"
    And 其他配置项保持不变

  @positive
  Scenario: 更新多个配置项
    When 发送 PUT 请求到 /api/config
    And 请求体为:
      """
      {
        "log_level": "warn",
        "session_timeout": 7200
      }
      """
    Then 响应状态码应为 200
    And 响应体中 config.log_level 应为 "warn"
    And 响应体中 config.session_timeout 应为 7200

  @negative
  Scenario: 更新无效的配置值
    When 发送 PUT 请求到 /api/config
    And 请求体为:
      """
      {
        "session_timeout": -1
      }
      """
    Then 响应状态码应为 400
    And 响应体应包含错误码 "invalid_config"

  @negative
  Scenario: 更新不存在的配置项
    When 发送 PUT 请求到 /api/config
    And 请求体为:
      """
      {
        "unknown_key": "value"
      }
      """
    Then 响应状态码应为 400
    And 响应体应包含错误码 "unknown_config_key"

  @negative
  Scenario: 使用非 GET/PUT 方法请求
    When 发送 POST 请求到 /api/config
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义请求/响应类型

```go
// internal/backend/api/response.go

// ConfigResponse 是配置查询的响应体。
type ConfigResponse struct {
	Config map[string]interface{} `json:"config"`
}

// ConfigUpdateRequest 是配置更新的请求体。
type ConfigUpdateRequest struct {
	LogLevel       *string `json:"log_level,omitempty"`
	SessionTimeout *int    `json:"session_timeout,omitempty"`
	MaxConcurrent  *int    `json:"max_concurrent,omitempty"`
	AutoClaim      *bool   `json:"auto_claim,omitempty"`
	DefaultModel   *string `json:"default_model,omitempty"`
}
```

### Step 2: 定义配置存储接口

```go
// internal/backend/store/store.go

// ConfigStore 接口定义配置 CRUD 操作。
type ConfigStore interface {
	GetConfig(ctx context.Context) (map[string]interface{}, error)
	UpdateConfig(ctx context.Context, updates map[string]interface{}) error
}
```

### Step 3: 实现配置存储层

```go
// internal/backend/store/config_store.go

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// configKey 是配置项的键名。
type configKey string

const (
	KeyLogLevel       configKey = "log_level"
	KeySessionTimeout configKey = "session_timeout"
	KeyMaxConcurrent  configKey = "max_concurrent"
	KeyAutoClaim      configKey = "auto_claim"
	KeyDefaultModel   configKey = "default_model"
)

// 默认配置
var defaultConfig = map[string]interface{}{
	string(KeyLogLevel):       "info",
	string(KeySessionTimeout): 3600,
	string(KeyMaxConcurrent):  10,
	string(KeyAutoClaim):      true,
	string(KeyDefaultModel):   "claude-3-opus",
}

type configStore struct {
	db *sql.DB
}

func newConfigStore(db *sql.DB) *configStore {
	return &configStore{db: db}
}

func (s *configStore) initDefaults(ctx context.Context) error {
	for key, value := range defaultConfig {
		var exists int
		err := s.db.QueryRowContext(ctx, `SELECT 1 FROM config WHERE key = ?`, key).Scan(&exists)
		if err == sql.ErrNoRows {
			v, _ := json.Marshal(value)
			_, err := s.db.ExecContext(ctx, `INSERT INTO config(key, value, updated_at) VALUES (?, ?, ?)`,
				key, string(v), time.Now())
			if err != nil {
				return fmt.Errorf("insert default config %s: %w", key, err)
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (s *configStore) GetConfig(ctx context.Context) (map[string]interface{}, error) {
	logger.Debug("store.config", "GetConfig")

	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	config := make(map[string]interface{})
	for rows.Next() {
		var key, valueStr string
		if err := rows.Scan(&key, &valueStr); err != nil {
			return nil, err
		}
		var value interface{}
		if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
			value = valueStr
		}
		config[key] = value
	}
	return config, rows.Err()
}

func (s *configStore) UpdateConfig(ctx context.Context, updates map[string]interface{}) error {
	logger.Debug("store.config", "UpdateConfig: %v", updates)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for key, value := range updates {
		// 验证 key 是否合法
		if !isValidConfigKey(key) {
			return fmt.Errorf("unknown config key: %s", key)
		}

		// 验证值类型
		if err := validateConfigValue(key, value); err != nil {
			return err
		}

		v, _ := json.Marshal(value)
		_, err := tx.ExecContext(ctx,
			`INSERT INTO config(key, value, updated_at) VALUES (?, ?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			key, string(v), time.Now())
		if err != nil {
			return fmt.Errorf("update config %s: %w", key, err)
		}
	}

	return tx.Commit()
}

func isValidConfigKey(key string) bool {
	switch configKey(key) {
	case KeyLogLevel, KeySessionTimeout, KeyMaxConcurrent, KeyAutoClaim, KeyDefaultModel:
		return true
	}
	return false
}

func validateConfigValue(key string, value interface{}) error {
	switch configKey(key) {
	case KeyLogLevel:
		v, ok := value.(string)
		if !ok || (v != "debug" && v != "info" && v != "warn" && v != "error") {
			return fmt.Errorf("invalid log_level: %v", value)
		}
	case KeySessionTimeout:
		v, ok := value.(float64)
		if !ok || v <= 0 {
			return fmt.Errorf("invalid session_timeout: %v", value)
		}
	case KeyMaxConcurrent:
		v, ok := value.(float64)
		if !ok || v <= 0 {
			return fmt.Errorf("invalid max_concurrent: %v", value)
		}
	case KeyAutoClaim:
		_, ok := value.(bool)
		if !ok {
			return fmt.Errorf("invalid auto_claim: %v", value)
		}
	case KeyDefaultModel:
		_, ok := value.(string)
		if !ok || value.(string) == "" {
			return fmt.Errorf("invalid default_model: %v", value)
		}
	}
	return nil
}
```

### Step 4: 实现配置服务层

```go
// internal/backend/service/config_service.go

package service

import (
	"context"
	"fmt"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type ConfigService struct {
	store store.ConfigStore
}

func NewConfigService(s store.ConfigStore) *ConfigService {
	return &ConfigService{store: s}
}

func (svc *ConfigService) Get(ctx context.Context) (map[string]interface{}, error) {
	logger.Debug("svc.config", "Get")
	return svc.store.GetConfig(ctx)
}

func (svc *ConfigService) Update(ctx context.Context, updates map[string]interface{}) error {
	logger.Debug("svc.config", "Update: %v", updates)
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}
	return svc.store.UpdateConfig(ctx, updates)
}
```

### Step 5: 实现 Handler 层

```go
// internal/backend/api/config_handler.go

package api

import (
	"encoding/json"
	"net/http"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type ConfigHandler struct {
	svc *service.ConfigService
}

func NewConfigHandler(svc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

func (h *ConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	logger.Debug("api.config", "GET /api/config")

	config, err := h.svc.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get config")
		return
	}

	writeJSON(w, http.StatusOK, ConfigResponse{Config: config})
}

func (h *ConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use PUT")
		return
	}

	var req ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	logger.Debug("api.config", "PUT /api/config: %+v", req)

	// 转换为 map
	updates := make(map[string]interface{})
	if req.LogLevel != nil {
		updates["log_level"] = *req.LogLevel
	}
	if req.SessionTimeout != nil {
		updates["session_timeout"] = *req.SessionTimeout
	}
	if req.MaxConcurrent != nil {
		updates["max_concurrent"] = *req.MaxConcurrent
	}
	if req.AutoClaim != nil {
		updates["auto_claim"] = *req.AutoClaim
	}
	if req.DefaultModel != nil {
		updates["default_model"] = *req.DefaultModel
	}

	if len(updates) == 0 {
		writeError(w, http.StatusBadRequest, "no_updates", "no valid fields to update")
		return
	}

	if err := h.svc.Update(r.Context(), updates); err != nil {
		code := "internal_error"
		if err.Error() == "unknown config key" {
			code = "unknown_config_key"
		} else if err.Error() == "invalid config value" {
			code = "invalid_config"
		}
		writeError(w, http.StatusBadRequest, code, err.Error())
		return
	}

	// 返回更新后的配置
	config, err := h.svc.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get updated config")
		return
	}

	writeJSON(w, http.StatusOK, ConfigResponse{Config: config})
}
```

### Step 6: 注册路由

```go
// internal/backend/api/router.go

type Handlers struct {
	Issue   *IssueHandler
	Session *SessionHandler
	Proxy   *ProxyHandler
	Machine *MachineHandler
	Project *ProjectHandler
	Log     *LogHandler
	Config  *ConfigHandler // 新增
}

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	// ... 已有路由
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.Config.Get(w, r)
		case http.MethodPut:
			h.Config.Update(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or PUT")
		}
	})
	return mux
}
```

---

## 四、Todo List

- [ ] Step 1: 定义 ConfigResponse 和 ConfigUpdateRequest 类型
- [ ] Step 2: 定义 ConfigStore 接口
- [ ] Step 3: 实现 configStore（含默认值初始化）
- [ ] Step 4: 实现 ConfigService
- [ ] Step 5: 实现 ConfigHandler（GET + PUT）
- [ ] Step 6: 在 router.go 中注册 /api/config 路由
- [ ] Step 7: 添加 config 表迁移脚本
- [ ] Step 8: 编写单元测试 config_api_test.go
- [ ] Step 9: 运行 BDD 测试验证
- [ ] Step 10: 更新 API 文档

---

*文档结束*
