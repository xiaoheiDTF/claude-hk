// Package service 提供后端业务逻辑层。
package service

import (
	"context"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ProxyService 处理 Proxy 相关的业务逻辑。
type ProxyService struct {
	store store.ProxyStore
}

// NewProxyService 创建 ProxyService 实例。
func NewProxyService(s store.ProxyStore) *ProxyService {
	return &ProxyService{store: s}
}

// List 获取代理列表。
func (svc *ProxyService) List(ctx context.Context, filter store.ProxyFilter) ([]store.Proxy, error) {
	logger.Debug("svc.proxy", "List: status=%v project=%v", filter.Status, filter.Project)
	return svc.store.ListProxies(ctx, filter)
}
