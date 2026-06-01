// Package service 提供后端业务逻辑层。
package service

import (
	"context"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// MachineService 处理 Machine 相关的业务逻辑。
type MachineService struct {
	store store.MachineStore
}

// NewMachineService 创建 MachineService 实例。
func NewMachineService(s store.MachineStore) *MachineService {
	return &MachineService{store: s}
}

// List 获取机器列表。
func (svc *MachineService) List(ctx context.Context, filter store.MachineFilter) ([]store.Machine, error) {
	logger.Debug("svc.machine", "List: filter applied")
	return svc.store.ListMachines(ctx, filter)
}

// Get 根据 machine_id 获取单个机器信息。
func (svc *MachineService) Get(ctx context.Context, machineID string) (*store.Machine, error) {
	logger.Debug("svc.machine", "Get: id=%s", machineID)
	return svc.store.GetMachine(ctx, machineID)
}
