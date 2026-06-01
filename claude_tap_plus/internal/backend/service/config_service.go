package service

import (
	"context"
	"fmt"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ConfigService 提供系统配置管理的业务逻辑。
type ConfigService struct {
	store store.ConfigStore
}

// NewConfigService 创建配置管理服务实例。
func NewConfigService(s store.ConfigStore) *ConfigService {
	return &ConfigService{store: s}
}

// Get 返回所有配置项。
func (svc *ConfigService) Get(ctx context.Context) (map[string]interface{}, error) {
	logger.Debug("svc.config", "Get")
	return svc.store.GetConfig(ctx)
}

// Update 更新指定的配置项。
func (svc *ConfigService) Update(ctx context.Context, updates map[string]interface{}) error {
	logger.Debug("svc.config", "Update: %v", updates)
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}
	return svc.store.UpdateConfig(ctx, updates)
}
