// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"net/http"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// Handlers 聚合所有 HTTP 处理器的结构体。
type Handlers struct {
	Issue   *IssueHandler   // Issue 相关接口处理器
	Session *SessionHandler // Session 相关接口处理器
	Proxy   *ProxyHandler   // 代理注册和 trace-init 转发处理器
}

// NewRouter 创建并返回 HTTP 路由器，注册所有 API 端点。
func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	// 健康检查端点
	mux.HandleFunc("/health", Health)

	// Issue 相关路由。
	mux.HandleFunc("/api/issue/check", h.Issue.CheckIssues)         // 检查 Issue 状态
	mux.HandleFunc("/api/issue/claim", h.Issue.ClaimIssue)          // 领取 Issue
	mux.HandleFunc("/api/issue/release", h.Issue.ReleaseIssue)      // 释放 Issue
	mux.HandleFunc("/api/issue/release-session", h.Issue.ReleaseSession) // 释放会话下所有 Issue
	mux.HandleFunc("/api/issue/status", h.Issue.UpdateStatus)       // 更新 Issue 状态

	// Session 相关路由。
	mux.HandleFunc("/api/session/register", h.Session.Register) // 注册会话
	mux.HandleFunc("/api/session/close", h.Session.Close)       // 关闭会话
	mux.HandleFunc("/api/sessions", h.Session.List)             // 获取会话列表
	mux.HandleFunc("/api/session/", h.Session.Get)              // 获取单个会话详情

	// Proxy 相关路由（代理注册 + trace-init 转发）。
	if h.Proxy != nil {
		mux.HandleFunc("/api/proxy/register", h.Proxy.Register)     // 代理注册
		mux.HandleFunc("/api/proxy/unregister", h.Proxy.Unregister) // 代理注销
		mux.HandleFunc("/api/proxy/trace-init", h.Proxy.TraceInit)  // 转发 trace-init
	}

	logger.Debug("api", "routes registered: %d endpoints", 9)
	return mux
}
