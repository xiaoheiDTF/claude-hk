// Package store 提供数据持久化层，基于 SQLite 实现。
package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// sqliteMachineStore 是 Machine 存储的 SQLite 实现。
type sqliteMachineStore struct {
	db *sql.DB
}

// ListMachines 获取机器列表，支持按操作系统和主机名过滤。
func (s *sqliteMachineStore) ListMachines(ctx context.Context, filter MachineFilter) ([]Machine, error) {
	query := `SELECT id, machine_id, os, hostname, username, first_seen_at, last_seen_at
	            FROM machines WHERE 1=1`
	args := []any{}

	if filter.OS != nil {
		query += " AND os = ?"
		args = append(args, *filter.OS)
	}
	if filter.Hostname != nil {
		query += " AND hostname = ?"
		args = append(args, *filter.Hostname)
	}
	query += " ORDER BY last_seen_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list machines: %w", err)
	}
	defer rows.Close()

	var machines []Machine
	for rows.Next() {
		var m Machine
		if err := rows.Scan(&m.ID, &m.MachineID, &m.OS, &m.Hostname,
			&m.Username, &m.FirstSeenAt, &m.LastSeenAt); err != nil {
			return nil, fmt.Errorf("scan machine: %w", err)
		}
		machines = append(machines, m)
	}

	logger.Debug("store.machine", "list machines: found %d", len(machines))
	return machines, rows.Err()
}

// GetMachine 根据 machine_id 获取单个机器信息。
func (s *sqliteMachineStore) GetMachine(ctx context.Context, machineID string) (*Machine, error) {
	var m Machine
	err := s.db.QueryRowContext(ctx,
		`SELECT id, machine_id, os, hostname, username, first_seen_at, last_seen_at
		   FROM machines WHERE machine_id = ?`,
		machineID,
	).Scan(&m.ID, &m.MachineID, &m.OS, &m.Hostname,
		&m.Username, &m.FirstSeenAt, &m.LastSeenAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get machine: %w", err)
	}
	return &m, nil
}
