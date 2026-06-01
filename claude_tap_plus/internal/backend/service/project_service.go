// Package service 提供后端业务逻辑层。
package service

import (
	"context"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ProjectService 处理 Project 相关的业务逻辑。
type ProjectService struct {
	store store.ProjectStore
}

// NewProjectService 创建 ProjectService 实例。
func NewProjectService(s store.ProjectStore) *ProjectService {
	return &ProjectService{store: s}
}

// List 获取项目列表。
func (svc *ProjectService) List(ctx context.Context) ([]store.Project, error) {
	logger.Debug("svc.project", "List: all projects")
	return svc.store.ListProjects(ctx)
}

// Get 根据 project_slug 获取单个项目信息。
func (svc *ProjectService) Get(ctx context.Context, projectSlug string) (*store.Project, error) {
	logger.Debug("svc.project", "Get: slug=%s", projectSlug)
	return svc.store.GetProject(ctx, projectSlug)
}
