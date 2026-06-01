// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"net/http"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ProjectHandler 处理 Project 相关的 HTTP 请求。
type ProjectHandler struct {
	svc *service.ProjectService
}

// NewProjectHandler 创建新的 ProjectHandler 实例。
func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// List 处理获取项目列表的请求。
// 接收 GET 请求，返回所有项目信息。
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	logger.Debug("api.project", "GET /api/projects")

	projects, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list projects")
		return
	}

	// 确保返回空数组而非 null
	if projects == nil {
		projects = []store.Project{}
	}

	items := make([]ProjectListItem, len(projects))
	for i, p := range projects {
		items[i] = ProjectListItem{
			ID:          p.ID,
			ProjectSlug: p.ProjectSlug,
			ProjectCwd:  p.ProjectCwd,
			FirstSeenAt: p.FirstSeenAt,
			LastSeenAt:  p.LastSeenAt,
		}
	}

	writeJSON(w, http.StatusOK, ProjectsListResponse{Projects: items})
}
