// Package store 提供数据持久化层，基于 SQLite 实现。
package store

import (
	"database/sql"
	"fmt"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"

	_ "modernc.org/sqlite" // 注册 SQLite 驱动
)

// SQLiteStore 是统一的 SQLite 存储入口，聚合了 Issue 和 Session 两个子存储。
type SQLiteStore struct {
	db           *sql.DB          // SQLite 数据库连接
	issueStore   *sqliteIssueStore // Issue 存储实现
	sessionStore *sqliteSessionStore // Session 存储实现
}

// NewSQLiteStore 创建并初始化 SQLite 存储。
// 打开数据库、设置 WAL 模式、执行迁移，然后初始化各子存储。
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	logger.Debug("store", "opening sqlite: %s", dbPath)
	// 打开数据库，设置 busy timeout 为 5 秒
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// 启用 WAL 模式以提高并发性能
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	logger.Debug("store", "WAL mode set")

	// 执行数据库迁移
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	logger.Debug("store", "migrations complete")

	logger.Info("store", "sqlite store ready: %s", dbPath)
	return &SQLiteStore{
		db:           db,
		issueStore:   &sqliteIssueStore{db: db},
		sessionStore: &sqliteSessionStore{db: db},
	}, nil
}

// Issues 返回 Issue 存储接口。
func (s *SQLiteStore) Issues() IssueStore { return s.issueStore }

// Sessions 返回 Session 存储接口。
func (s *SQLiteStore) Sessions() SessionStore { return s.sessionStore }

// DB 返回底层的 *sql.DB 连接（主要用于测试）。
func (s *SQLiteStore) DB() *sql.DB { return s.db }

// Close 关闭数据库连接。
func (s *SQLiteStore) Close() error {
	err := s.db.Close()
	logger.Info("store", "sqlite store closed")
	return err
}
