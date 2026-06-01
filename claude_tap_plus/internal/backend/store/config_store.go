package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// --- Config key constants ---

// configKey 是配置项键名的类型。
type configKey string

const (
	KeyLogLevel       configKey = "log_level"
	KeySessionTimeout configKey = "session_timeout"
	KeyMaxConcurrent  configKey = "max_concurrent"
	KeyAutoClaim      configKey = "auto_claim"
	KeyDefaultModel   configKey = "default_model"
)

// defaultConfig 是系统默认配置。
var defaultConfig = map[string]interface{}{
	string(KeyLogLevel):       "info",
	string(KeySessionTimeout): 3600,
	string(KeyMaxConcurrent):  10,
	string(KeyAutoClaim):      true,
	string(KeyDefaultModel):   "claude-3-opus",
}

// sqliteConfigStore 是 ConfigStore 的 SQLite 实现。
type sqliteConfigStore struct {
	db *sql.DB
}

// InitDefaults 将缺失的默认配置插入数据库。
func (s *sqliteConfigStore) InitDefaults(ctx context.Context) error {
	for key, value := range defaultConfig {
		var exists int
		err := s.db.QueryRowContext(ctx, `SELECT 1 FROM config WHERE key = ?`, key).Scan(&exists)
		if err == sql.ErrNoRows {
			v, _ := json.Marshal(value)
			_, err := s.db.ExecContext(ctx,
				`INSERT INTO config(key, value, updated_at) VALUES (?, ?, ?)`,
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

// GetConfig 返回所有配置项。
func (s *sqliteConfigStore) GetConfig(ctx context.Context) (map[string]interface{}, error) {
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

// UpdateConfig 更新指定的配置项。
func (s *sqliteConfigStore) UpdateConfig(ctx context.Context, updates map[string]interface{}) error {
	logger.Debug("store.config", "UpdateConfig: %v", updates)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for key, value := range updates {
		if !isValidConfigKey(key) {
			return fmt.Errorf("unknown config key: %s", key)
		}
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

// isValidConfigKey 检查配置键是否合法。
func isValidConfigKey(key string) bool {
	switch configKey(key) {
	case KeyLogLevel, KeySessionTimeout, KeyMaxConcurrent, KeyAutoClaim, KeyDefaultModel:
		return true
	}
	return false
}

// validateConfigValue 校验配置值的合法性。
func validateConfigValue(key string, value interface{}) error {
	switch configKey(key) {
	case KeyLogLevel:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("invalid log_level: must be string")
		}
		switch v {
		case "debug", "info", "warn", "error":
			// ok
		default:
			return fmt.Errorf("invalid log_level: %s", v)
		}
	case KeySessionTimeout:
		v, ok := toFloat64(value)
		if !ok || v <= 0 {
			return fmt.Errorf("invalid session_timeout: %v", value)
		}
	case KeyMaxConcurrent:
		v, ok := toFloat64(value)
		if !ok || v <= 0 {
			return fmt.Errorf("invalid max_concurrent: %v", value)
		}
	case KeyAutoClaim:
		_, ok := value.(bool)
		if !ok {
			return fmt.Errorf("invalid auto_claim: must be bool")
		}
	case KeyDefaultModel:
		v, ok := value.(string)
		if !ok || v == "" {
			return fmt.Errorf("invalid default_model: %v", value)
		}
	}
	return nil
}

// toFloat64 将 JSON 数值转换为 float64（兼容 int/float64）。
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
