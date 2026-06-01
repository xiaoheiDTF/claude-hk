// Package store 提供数据持久化层，基于 SQLite 实现。
package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// sqliteProjectStore 是 Project 存储的 SQLite 实现。
type sqliteProjectStore struct {
	db *sql.DB
}

// ListProjects 获取所有项目列表，按 last_seen_at 倒序排列。
func (s *sqliteProjectStore) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_slug, project_cwd, first_seen_at, last_seen_at
		   FROM projects ORDER BY last_seen_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.ProjectSlug, &p.ProjectCwd,
			&p.FirstSeenAt, &p.LastSeenAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}

	logger.Debug("store.project", "list projects: found %d", len(projects))
	return projects, rows.Err()
}

// GetProject 根据 project_slug 获取单个项目信息。
func (s *sqliteProjectStore) GetProject(ctx context.Context, projectSlug string) (*Project, error) {
	var p Project
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_slug, project_cwd, first_seen_at, last_seen_at
		   FROM projects WHERE project_slug = ?`,
		projectSlug,
	).Scan(&p.ID, &p.ProjectSlug, &p.ProjectCwd, &p.FirstSeenAt, &p.LastSeenAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &p, nil
}
