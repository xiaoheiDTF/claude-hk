// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"net/http"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// Handlers 聚合所有 HTTP 处理器的结构体。
type Handlers struct {
	Issue   *IssueHandler   // Issue 相关接口处理器
	Session *SessionHandler // Session 相关接口处理器
	Proxy   *ProxyHandler   // 代理注册和 trace-init 转发处理器
	Machine *MachineHandler // Machine 相关接口处理器
	Project *ProjectHandler // Project 相关接口处理器
	Log     *LogHandler     // Log 相关接口处理器
	Config  *ConfigHandler  // Config 相关接口处理器
	Status  *StatusHandler  // 系统状态处理器
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
	mux.HandleFunc("/api/issues", h.Issue.ListIssues)               // 列出所有 Issue

	// Session 相关路由。
	mux.HandleFunc("/api/session/register", h.Session.Register) // 注册会话
	mux.HandleFunc("/api/session/close", h.Session.Close)       // 关闭会话
	mux.HandleFunc("/api/sessions", h.Session.List)             // 获取会话列表
	mux.HandleFunc("/api/session/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/issues"):
			h.Session.GetIssues(w, r)
		case strings.HasSuffix(path, "/tokens"):
			h.Session.GetTokens(w, r)
		case strings.HasSuffix(path, "/traces"):
			h.Session.GetTraces(w, r)
		default:
			h.Session.Get(w, r)
		}
	})

	// Proxy 相关路由（代理注册 + trace-init 转发 + 代理列表）。
	if h.Proxy != nil {
		mux.HandleFunc("/api/proxy/register", h.Proxy.Register)     // 代理注册
		mux.HandleFunc("/api/proxy/unregister", h.Proxy.Unregister) // 代理注销
		mux.HandleFunc("/api/proxy/trace-init", h.Proxy.TraceInit)  // 转发 trace-init
		mux.HandleFunc("/api/proxies", h.Proxy.List)                // 代理列表
	}

	// Machine 相关路由。
	if h.Machine != nil {
		mux.HandleFunc("/api/machines", h.Machine.List) // 列出所有机器
	}

	// Project 相关路由。
	if h.Project != nil {
		mux.HandleFunc("/api/projects", h.Project.List) // 列出所有项目
	}

	// Log 相关路由。
	if h.Log != nil {
		mux.HandleFunc("/api/logs", h.Log.Query) // 查询日志
	}

	// Config 相关路由（GET + PUT 分发）。
	if h.Config != nil {
		mux.HandleFunc("/api/config", h.Config.ServeHTTP) // 获取/更新配置
	}

	// Status 相关路由。
	if h.Status != nil {
		mux.HandleFunc("/api/status", h.Status.Get) // 系统状态
	}

	logger.Debug("api", "routes registered: %d endpoints", 15)
	return mux
}
