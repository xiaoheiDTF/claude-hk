package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type sqliteSessionStore struct {
	db *sql.DB
}

func (s *sqliteSessionStore) RegisterSession(ctx context.Context, sess Session) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Parse machine_id into username and hostname.
	username, hostname := parseMachineID(sess.MachineID)

	// Upsert machine.
	_, err = tx.ExecContext(ctx,
		`INSERT INTO machines (machine_id, os, hostname, username, first_seen_at, last_seen_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT(machine_id) DO UPDATE SET last_seen_at = CURRENT_TIMESTAMP`,
		sess.MachineID, sess.OS, hostname, username)
	if err != nil {
		return fmt.Errorf("upsert machine: %w", err)
	}

	// Upsert project.
	_, err = tx.ExecContext(ctx,
		`INSERT INTO projects (project_slug, project_cwd, first_seen_at, last_seen_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT(project_slug) DO UPDATE SET last_seen_at = CURRENT_TIMESTAMP`,
		sess.ProjectSlug, sess.ProjectCwd)
	if err != nil {
		return fmt.Errorf("upsert project: %w", err)
	}

	// Insert session.
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

	return tx.Commit()
}

func (s *sqliteSessionStore) CloseSession(ctx context.Context, sessionID string, reason string) error {
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
	return nil
}

func (s *sqliteSessionStore) ListSessions(ctx context.Context, filter SessionFilter) ([]Session, error) {
	query := `SELECT session_id, machine_id, os, project_slug, project_cwd,
	                 transcript_path, local_trace_path, model, source,
	                 status, registered_at, closed_at, close_reason
	            FROM sessions WHERE 1=1`
	args := []any{}

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

func (s *sqliteSessionStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
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

func parseMachineID(machineID string) (username, hostname string) {
	parts := strings.SplitN(machineID, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return machineID, ""
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
