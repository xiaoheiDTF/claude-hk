// Package store 提供数据持久化层，基于 SQLite 实现。
package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// sqliteSessionStore 是 Session 存储的 SQLite 实现。
type sqliteSessionStore struct {
	db *sql.DB // SQLite 数据库连接
}

// RegisterSession 注册一个新会话。
// 使用事务同时更新 machines 表（upsert）、projects 表（upsert）和 sessions 表（insert）。
func (s *sqliteSessionStore) RegisterSession(ctx context.Context, sess Session) error {
	logger.Debug("store.session", "register: id=%s machine=%s slug=%s", sess.SessionID, sess.MachineID, sess.ProjectSlug)

	// 开启事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 解析 machine_id 为 username 和 hostname
	username, hostname := parseMachineID(sess.MachineID)

	// Upsert machine：不存在则插入，存在则更新 last_seen_at
	_, err = tx.ExecContext(ctx,
		`INSERT INTO machines (machine_id, os, hostname, username, first_seen_at, last_seen_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT(machine_id) DO UPDATE SET last_seen_at = CURRENT_TIMESTAMP`,
		sess.MachineID, sess.OS, hostname, username)
	if err != nil {
		return fmt.Errorf("upsert machine: %w", err)
	}

	// Upsert project：不存在则插入，存在则更新 last_seen_at
	_, err = tx.ExecContext(ctx,
		`INSERT INTO projects (project_slug, project_cwd, first_seen_at, last_seen_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT(project_slug) DO UPDATE SET last_seen_at = CURRENT_TIMESTAMP`,
		sess.ProjectSlug, sess.ProjectCwd)
	if err != nil {
		return fmt.Errorf("upsert project: %w", err)
	}

	// 插入 session 记录
	_, err = tx.ExecContext(ctx,
		`INSERT INTO sessions (session_id, machine_id, os, project_slug, project_cwd,
		    transcript_path, local_trace_path, model, source, status, registered_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', CURRENT_TIMESTAMP)`,
		sess.SessionID, sess.MachineID, sess.OS, sess.ProjectSlug, sess.ProjectCwd,
		nilIfEmpty(sess.TranscriptPath), nilIfEmpty(sess.LocalTracePath),
		nilIfEmpty(sess.Model), nilIfEmpty(sess.Source))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrSessionExists
		}
		return fmt.Errorf("insert session: %w", err)
	}

	logger.Info("store.session", "session registered: %s slug=%s", sess.SessionID, sess.ProjectSlug)
	return tx.Commit()
}

// CloseSession 关闭指定会话。
// 仅对状态为 active 的会话生效，将其标记为 closed 并记录关闭时间和原因。
func (s *sqliteSessionStore) CloseSession(ctx context.Context, sessionID string, reason string) error {
	logger.Debug("store.session", "close: id=%s reason=%s", sessionID, reason)

	res, err := s.db.ExecContext(ctx,
		`UPDATE sessions
		    SET status = 'closed', closed_at = CURRENT_TIMESTAMP, close_reason = ?
		  WHERE session_id = ? AND status = 'active'`,
		reason, sessionID)
	if err != nil {
		return fmt.Errorf("close session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrSessionNotFound
	}
	logger.Info("store.session", "session closed: %s reason=%s", sessionID, reason)
	return nil
}

// ListSessions 获取会话列表，支持按 machine_id、project_slug、status 过滤。
// 结果按注册时间倒序排列。
func (s *sqliteSessionStore) ListSessions(ctx context.Context, filter SessionFilter) ([]Session, error) {
	logger.Debug("store.session", "list: filter applied")

	query := `SELECT session_id, machine_id, os, project_slug, project_cwd,
	                 transcript_path, local_trace_path, model, source,
	                 status, registered_at, closed_at, close_reason
	            FROM sessions WHERE 1=1`
	args := []any{}

	// 动态构建过滤条件
	if filter.MachineID != nil {
		query += " AND machine_id = ?"
		args = append(args, *filter.MachineID)
	}
	if filter.ProjectSlug != nil {
		query += " AND project_slug = ?"
		args = append(args, *filter.ProjectSlug)
	}
	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}
	query += " ORDER BY registered_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var localTracePath, model, source, closedAt, closeReason sql.NullString
		if err := rows.Scan(
			&sess.SessionID, &sess.MachineID, &sess.OS, &sess.ProjectSlug, &sess.ProjectCwd,
			&sess.TranscriptPath, &localTracePath, &model, &source,
			&sess.Status, &sess.RegisteredAt, &closedAt, &closeReason,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		// 处理 NULL 值
		if localTracePath.Valid {
			sess.LocalTracePath = localTracePath.String
		}
		if model.Valid {
			sess.Model = model.String
		}
		if source.Valid {
			sess.Source = source.String
		}
		if closedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", closedAt.String)
			sess.ClosedAt = &t
		}
		if closeReason.Valid {
			sess.CloseReason = closeReason.String
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// GetSession 获取单个会话的详细信息。
// 如果会话不存在，返回 nil。
func (s *sqliteSessionStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	logger.Debug("store.session", "get: id=%s", sessionID)

	var sess Session
	var localTracePath, model, source, closedAt, closeReason sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT session_id, machine_id, os, project_slug, project_cwd,
		        transcript_path, local_trace_path, model, source,
		        status, registered_at, closed_at, close_reason
		   FROM sessions WHERE session_id = ?`,
		sessionID,
	).Scan(
		&sess.SessionID, &sess.MachineID, &sess.OS, &sess.ProjectSlug, &sess.ProjectCwd,
		&sess.TranscriptPath, &localTracePath, &model, &source,
		&sess.Status, &sess.RegisteredAt, &closedAt, &closeReason,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// 处理 NULL 值
	if localTracePath.Valid {
		sess.LocalTracePath = localTracePath.String
	}
	if model.Valid {
		sess.Model = model.String
	}
	if source.Valid {
		sess.Source = source.String
	}
	if closedAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", closedAt.String)
		sess.ClosedAt = &t
	}
	if closeReason.Valid {
		sess.CloseReason = closeReason.String
	}
	return &sess, nil
}

// CleanupTimedOut 清理注册超过 24 小时仍处于 active 状态的会话。
// 返回被清理的会话数量。
func (s *sqliteSessionStore) CleanupTimedOut(ctx context.Context) (int, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE sessions
		    SET status = 'closed', closed_at = CURRENT_TIMESTAMP, close_reason = 'timeout_cleanup'
		  WHERE status = 'active'
		    AND registered_at < datetime('now', '-24 hours')`)
	if err != nil {
		return 0, fmt.Errorf("cleanup timed out: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// parseMachineID 将 machine_id（格式为 username@hostname）分割为 username 和 hostname。
// 如果不含 @，则整个字符串作为 username，hostname 为空。
func parseMachineID(machineID string) (username, hostname string) {
	parts := strings.SplitN(machineID, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return machineID, ""
}

// nilIfEmpty 如果字符串为空则返回 nil，否则返回原字符串。
// 用于将空字符串存储为 SQL NULL。
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
