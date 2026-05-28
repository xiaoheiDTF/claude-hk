package store

import "database/sql"

func runMigrations(db *sql.DB) error {
	stmts := []string{
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

		`CREATE TABLE IF NOT EXISTS projects (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			project_slug    TEXT NOT NULL UNIQUE,
			project_cwd     TEXT NOT NULL,
			first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at    DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_slug ON projects(project_slug)`,

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
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
