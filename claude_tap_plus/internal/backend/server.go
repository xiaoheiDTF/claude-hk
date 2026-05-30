// Package backend 是后端服务的主包，提供 HTTP API 服务器及其配置。
package backend

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/api"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// Server 是后端 HTTP 服务的主结构体，聚合配置和数据存储。
type Server struct {
	cfg   Config         // 服务配置
	store *store.SQLiteStore // SQLite 数据存储
}

// NewServer 创建并初始化后端服务器。
// 根据配置打开 SQLite 数据库，失败时返回错误。
func NewServer(cfg Config) (*Server, error) {
	logger.Debug("backend", "creating server: host=%s port=%d db=%s", cfg.Host, cfg.Port, cfg.DBPath)
	s, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &Server{cfg: cfg, store: s}, nil
}

// Start 启动 HTTP 服务器。
// 启动前会清理超时会话，然后注册路由并开始监听。支持优雅关闭（SIGINT/SIGTERM）。
func (s *Server) Start() error {
	// 启动清理：将超时会话标记为已关闭
	cleanupSvc := service.NewCleanupService(s.store.Sessions())
	cleanupSvc.CleanupTimedOutSessions(context.Background())
	logger.Info("backend", "startup cleanup done")

	// 初始化业务服务和处理器
	issueSvc := service.NewIssueService(s.store.Issues())
	sessionSvc := service.NewSessionService(s.store.Sessions())

	router := api.NewRouter(api.Handlers{
		Issue:   api.NewIssueHandler(issueSvc),
		Session: api.NewSessionHandler(sessionSvc),
	})

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:    s.cfg.Addr(),
		Handler: router,
	}

	// 监听系统信号以实现优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Info("backend", "shutdown signal received")
		srv.Shutdown(context.Background())
	}()

	logger.Info("backend", "backend listening on %s", s.cfg.Addr())
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Close 关闭后端服务器及其数据库连接。
func (s *Server) Close() error {
	err := s.store.Close()
	logger.Info("backend", "backend server closed")
	return err
}
