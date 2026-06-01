// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// IssueHandler 处理 Issue 相关的 HTTP 请求。
type IssueHandler struct {
	svc *service.IssueService // Issue 业务逻辑服务
}

// NewIssueHandler 创建新的 IssueHandler 实例。
func NewIssueHandler(svc *service.IssueService) *IssueHandler {
	return &IssueHandler{svc: svc}
}

// CheckIssues 处理检查 Issue 状态的请求。
// 接收 POST 请求，返回指定仓库中 Issue 的当前状态列表。
func (h *IssueHandler) CheckIssues(w http.ResponseWriter, r *http.Request) {
	// 只允许 POST 方法
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	// 解析请求体
	var req CheckIssuesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	// 校验必填字段
	if req.RepoFullName == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "repo_full_name is required")
		return
	}
	if req.IssueNumbers == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "issue_numbers is required")
		return
	}

	logger.Debug("api.issue", "POST /api/issue/check repo=%s", req.RepoFullName)

	// 调用服务层查询 Issue 状态
	results, err := h.svc.Check(r.Context(), req.RepoFullName, req.IssueNumbers)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to check issues")
		return
	}

	// 转换为响应格式
	items := make([]IssueStatusItem, len(results))
	for i, r := range results {
		items[i] = IssueStatusItem{
			Number:    r.Number,
			Status:    r.Status,
			SessionID: r.SessionID,
			ClaimedAt: r.ClaimedAt,
		}
	}

	// 确保返回空数组而非 null
	if items == nil {
		items = []IssueStatusItem{}
	}

	writeJSON(w, http.StatusOK, CheckIssuesResponse{Issues: items})
}

// ClaimIssue 处理领取 Issue 的请求。
// 接收 POST 请求，将指定 Issue 标记为已领取状态。
func (h *IssueHandler) ClaimIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req ClaimIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.RepoFullName == "" || req.IssueNumber == 0 || req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "repo_full_name, issue_number and session_id are required")
		return
	}

	logger.Debug("api.issue", "POST /api/issue/claim repo=%s #%d", req.RepoFullName, req.IssueNumber)

	// 调用服务层领取 Issue
	result, err := h.svc.Claim(r.Context(), req.RepoFullName, req.IssueNumber, req.SessionID, req.IssueTitle)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to claim issue")
		return
	}

	// 领取成功
	if result.Success {
		writeJSON(w, http.StatusOK, ClaimIssueResponse{
			Success:   true,
			Status:    result.Status,
			ClaimedAt: result.ClaimedAt,
		})
		return
	}

	// 已被其他会话领取，返回冲突信息
	writeJSON(w, http.StatusConflict, ClaimIssueResponse{
		Success:   false,
		Error:     "already_claimed",
		ClaimedBy: result.ClaimedBy,
		ClaimedAt: result.ClaimedAt,
	})
}

// ReleaseIssue 处理释放单个 Issue 的请求。
// 接收 POST 请求，将指定 Issue 恢复为空闲状态（仅限当前持有者）。
func (h *IssueHandler) ReleaseIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req ReleaseIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.RepoFullName == "" || req.IssueNumber == 0 || req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "repo_full_name, issue_number and session_id are required")
		return
	}

	logger.Debug("api.issue", "POST /api/issue/release repo=%s #%d", req.RepoFullName, req.IssueNumber)

	// 调用服务层释放 Issue
	released, err := h.svc.Release(r.Context(), req.RepoFullName, req.IssueNumber, req.SessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to release issue")
		return
	}

	// 不是该会话持有的 Issue
	if !released {
		writeJSON(w, http.StatusOK, ReleaseIssueResponse{Success: false, Error: "not_owner"})
		return
	}

	writeJSON(w, http.StatusOK, ReleaseIssueResponse{Success: true, Released: boolPtr(true)})
}

// ReleaseSession 处理释放某会话下所有 Issue 的请求。
// 接收 POST 请求，将该 session 持有的所有非终态 Issue 恢复为空闲。
func (h *IssueHandler) ReleaseSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req ReleaseSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session_id is required")
		return
	}

	logger.Debug("api.issue", "POST /api/issue/release-session session=%s", req.SessionID)

	// 调用服务层释放该会话下的所有 Issue
	numbers, err := h.svc.ReleaseSession(r.Context(), req.SessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to release session issues")
		return
	}

	// 确保返回空数组而非 null
	if numbers == nil {
		numbers = []int{}
	}

	writeJSON(w, http.StatusOK, ReleaseSessionResponse{Released: numbers, Count: len(numbers)})
}

// UpdateStatus 处理更新 Issue 状态的请求。
// 接收 POST 请求，仅允许当前持有者更新 Issue 状态。
func (h *IssueHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.RepoFullName == "" || req.IssueNumber == 0 || req.SessionID == "" || req.Status == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "repo_full_name, issue_number, session_id and status are required")
		return
	}

	logger.Debug("api.issue", "POST /api/issue/status repo=%s #%d -> %s", req.RepoFullName, req.IssueNumber, req.Status)

	// 调用服务层更新状态
	result, err := h.svc.UpdateStatus(r.Context(), req.RepoFullName, req.IssueNumber, req.SessionID, req.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update status")
		return
	}

	// 更新失败（非持有者或不存在）
	if !result.Updated {
		status := http.StatusOK
		errMsg := ""
		if result.PreviousStatus == "" {
			errMsg = "not_found"
		} else {
			errMsg = "not_owner"
		}
		writeJSON(w, status, UpdateStatusResponse{
			Success:        false,
			PreviousStatus: result.PreviousStatus,
			Error:          errMsg,
		})
		return
	}

	// 更新成功
	writeJSON(w, http.StatusOK, UpdateStatusResponse{
		Success:        true,
		PreviousStatus: result.PreviousStatus,
		NewStatus:      result.NewStatus,
	})
}

// boolPtr 返回 bool 的指针，用于 JSON 响应中区分零值和未设置。
func boolPtr(b bool) *bool { return &b }

// ListIssues 处理获取 Issue 列表的请求。
// 接收 GET 请求，支持按仓库、状态、session_id 过滤和分页。
func (h *IssueHandler) ListIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	// 解析查询参数
	var filter store.IssueFilter
	if v := r.URL.Query().Get("repo"); v != "" {
		filter.RepoFullName = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("session_id"); v != "" {
		filter.SessionID = &v
	}
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.Page = n
		}
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.PageSize = n
		}
	}

	logger.Debug("api.issue", "GET /api/issues filter=%+v", filter)

	items, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list issues")
		return
	}

	// 确保返回空数组而非 null
	if items == nil {
		items = []store.IssueListItem{}
	}

	// 分页参数
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	// 转换为响应格式
	respItems := make([]IssueListItem, len(items))
	for i, item := range items {
		respItems[i] = IssueListItem{
			ID:           item.ID,
			RepoFullName: item.RepoFullName,
			IssueNumber:  item.IssueNumber,
			IssueTitle:   item.IssueTitle,
			Status:       item.Status,
			SessionID:    item.SessionID,
			ClaimedAt:    item.ClaimedAt,
			UpdatedAt:    item.UpdatedAt,
		}
	}

	writeJSON(w, http.StatusOK, IssuesListResponse{
		Issues:     respItems,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}
