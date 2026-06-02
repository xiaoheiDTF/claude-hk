// Package service 提供后端业务逻辑层。
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// SystemStats 是系统统计信息。
type SystemStats struct {
	ActiveSessions int64 // 活跃会话数
	ActiveProxies  int64 // 活跃代理数
	PendingIssues  int64 // 待处理 Issue 数
	TotalMachines  int64 // 机器总数
	TotalProjects  int64 // 项目总数
}

// SystemStatus 表示系统整体状态。
type SystemStatus struct {
	Status        string       // 系统状态：healthy
	Version       string       // 版本号
	UptimeSeconds int64        // 运行时间（秒）
	Stats         SystemStats  // 统计信息
	Timestamp     time.Time    // 当前时间戳
}

// StatusService 聚合各子系统的统计数据。
type StatusService struct {
	sessionStore store.SessionStore
	proxyStore   store.ProxyStore
	issueStore   store.IssueStore
	machineStore store.MachineStore
	projectStore store.ProjectStore
	startTime    time.Time
}

// NewStatusService 创建 StatusService 实例。
func NewStatusService(
	sessionStore store.SessionStore,
	proxyStore store.ProxyStore,
	issueStore store.IssueStore,
	machineStore store.MachineStore,
	projectStore store.ProjectStore,
	startTime time.Time,
) *StatusService {
	return &StatusService{
		sessionStore: sessionStore,
		proxyStore:   proxyStore,
		issueStore:   issueStore,
		machineStore: machineStore,
		projectStore: projectStore,
		startTime:    startTime,
	}
}

// Get 获取系统整体状态。
func (svc *StatusService) Get(ctx context.Context) (*SystemStatus, error) {
	logger.Debug("svc.status", "Get")

	activeSessions, err := svc.countActiveSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("count active sessions: %w", err)
	}

	activeProxies, err := svc.countActiveProxies(ctx)
	if err != nil {
		return nil, fmt.Errorf("count active proxies: %w", err)
	}

	pendingIssues, err := svc.countPendingIssues(ctx)
	if err != nil {
		return nil, fmt.Errorf("count pending issues: %w", err)
	}

	totalMachines, err := svc.countMachines(ctx)
	if err != nil {
		return nil, fmt.Errorf("count machines: %w", err)
	}

	totalProjects, err := svc.countProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("count projects: %w", err)
	}

	uptime := int64(time.Since(svc.startTime).Seconds())

	return &SystemStatus{
		Status:        "healthy",
		Version:       "1.0.0",
		UptimeSeconds: uptime,
		Stats: SystemStats{
			ActiveSessions: activeSessions,
			ActiveProxies:  activeProxies,
			PendingIssues:  pendingIssues,
			TotalMachines:  totalMachines,
			TotalProjects:  totalProjects,
		},
		Timestamp: time.Now().UTC(),
	}, nil
}

func (svc *StatusService) countActiveSessions(ctx context.Context) (int64, error) {
	active := "active"
	sessions, err := svc.sessionStore.ListSessions(ctx, store.SessionFilter{Status: &active})
	if err != nil {
		return 0, err
	}
	return int64(len(sessions)), nil
}

func (svc *StatusService) countActiveProxies(ctx context.Context) (int64, error) {
	active := "active"
	proxies, err := svc.proxyStore.ListProxies(ctx, store.ProxyFilter{Status: &active})
	if err != nil {
		return 0, err
	}
	return int64(len(proxies)), nil
}

func (svc *StatusService) countPendingIssues(ctx context.Context) (int64, error) {
	idle := "idle"
	_, total, err := svc.issueStore.ListIssues(ctx, store.IssueFilter{Status: &idle})
	if err != nil {
		return 0, err
	}
	return int64(total), nil
}

func (svc *StatusService) countMachines(ctx context.Context) (int64, error) {
	machines, err := svc.machineStore.ListMachines(ctx, store.MachineFilter{})
	if err != nil {
		return 0, err
	}
	return int64(len(machines)), nil
}

func (svc *StatusService) countProjects(ctx context.Context) (int64, error) {
	projects, err := svc.projectStore.ListProjects(ctx)
	if err != nil {
		return 0, err
	}
	return int64(len(projects)), nil
}
