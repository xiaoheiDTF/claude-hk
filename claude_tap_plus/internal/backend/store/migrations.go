// Package store 提供数据持久化层，基于 SQLite 实现。
package store

import (
	"database/sql"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// runMigrations 执行数据库迁移，创建必要的表和索引。
// 如果表或索引已存在则跳过（使用 IF NOT EXISTS）。
func runMigrations(db *sql.DB) error {
	stmts := []string{
		// machines 表：记录使用 claude-tap-plus 的机器信息
		`CREATE TABLE IF NOT EXISTS machines (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			machine_id      TEXT NOT NULL UNIQUE,
			os              TEXT NOT NULL,
			hostname        TEXT NOT NULL,
			username        TEXT NOT NULL,
			first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at    DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_machines_hostname ON machines(hostname)`,

		// projects 表：记录 Claude Code 工作过的项目
		`CREATE TABLE IF NOT EXISTS projects (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			project_slug    TEXT NOT NULL UNIQUE,
			project_cwd     TEXT NOT NULL,
			first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at    DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_slug ON projects(project_slug)`,

		// sessions 表：记录每次 Claude Code 会话的元数据
		`CREATE TABLE IF NOT EXISTS sessions (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id       TEXT NOT NULL UNIQUE,
			machine_id       TEXT NOT NULL,
			os               TEXT NOT NULL,
			project_slug     TEXT NOT NULL,
			project_cwd      TEXT NOT NULL,
			transcript_path  TEXT NOT NULL,
			local_trace_path TEXT,
			model            TEXT,
			source           TEXT,
			status           TEXT NOT NULL DEFAULT 'active',
			registered_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			closed_at        DATETIME,
			close_reason     TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_machine ON sessions(machine_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_slug)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_registered ON sessions(registered_at)`,

		// issue_claims 表：存储 Issue 领取关系和状态流转
		`CREATE TABLE IF NOT EXISTS issue_claims (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_full_name  TEXT NOT NULL,
			issue_number    INTEGER NOT NULL,
			issue_title     TEXT,
			status          TEXT NOT NULL DEFAULT 'idle',
			session_id      TEXT,
			claimed_at      DATETIME,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(repo_full_name, issue_number)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_issue_claims_repo ON issue_claims(repo_full_name)`,
		`CREATE INDEX IF NOT EXISTS idx_issue_claims_session ON issue_claims(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_issue_claims_status ON issue_claims(status)`,

		// config 表：存储系统运行时配置
		`CREATE TABLE IF NOT EXISTS config (
			key        TEXT PRIMARY KEY,
			value      TEXT NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		// proxies 表：记录已注册的代理信息
		`CREATE TABLE IF NOT EXISTS proxies (
			proxy_id      TEXT PRIMARY KEY,
			project_slug  TEXT NOT NULL,
			status        TEXT NOT NULL DEFAULT 'active',
			registered_at DATETIME NOT NULL,
			last_ping_at  DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_proxies_status ON proxies(status)`,
		`CREATE INDEX IF NOT EXISTS idx_proxies_project ON proxies(project_slug)`,
	}

	logger.Debug("store", "running %d migration statements", len(stmts))
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			logger.Error("store", "migration failed: %s: %v", stmt[:60], err)
			return err
		}
		logger.Debug("store", "migration OK: %.60s", stmt)
	}
	return nil
}
