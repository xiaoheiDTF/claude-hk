package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db            *sql.DB
	issueStore    *sqliteIssueStore
	sessionStore  *sqliteSessionStore
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &SQLiteStore{
		db:           db,
		issueStore:   &sqliteIssueStore{db: db},
		sessionStore: &sqliteSessionStore{db: db},
	}, nil
}

func (s *SQLiteStore) Issues() IssueStore    { return s.issueStore }
func (s *SQLiteStore) Sessions() SessionStore { return s.sessionStore }
func (s *SQLiteStore) DB() *sql.DB            { return s.db }
func (s *SQLiteStore) Close() error           { return s.db.Close() }
