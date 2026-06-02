// Package store 提供数据持久化层，基于 SQLite 实现。
package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// sqliteProxyStore 是 Proxy 存储的 SQLite 实现。
type sqliteProxyStore struct {
	db *sql.DB
}

// ListProxies 获取代理列表，支持按状态和项目过滤。
func (s *sqliteProxyStore) ListProxies(ctx context.Context, filter ProxyFilter) ([]Proxy, error) {
	query := `SELECT proxy_id, project_slug, status, registered_at, last_ping_at
	            FROM proxies WHERE 1=1`
	args := []any{}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}
	if filter.Project != nil {
		query += " AND project_slug = ?"
		args = append(args, *filter.Project)
	}
	query += " ORDER BY registered_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list proxies: %w", err)
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		var p Proxy
		var lastPing sql.NullTime
		if err := rows.Scan(&p.ProxyID, &p.ProjectSlug, &p.Status, &p.RegisteredAt, &lastPing); err != nil {
			return nil, fmt.Errorf("scan proxy: %w", err)
		}
		if lastPing.Valid {
			p.LastPingAt = &lastPing.Time
		}
		proxies = append(proxies, p)
	}

	logger.Debug("store.proxy", "list proxies: found %d", len(proxies))
	return proxies, rows.Err()
}
