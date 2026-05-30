// Package store 提供数据持久化层，基于 SQLite 实现。
package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// sqliteIssueStore 是 Issue 存储的 SQLite 实现。
type sqliteIssueStore struct {
	db *sql.DB // SQLite 数据库连接
}

// CheckIssues 检查指定仓库中多个 Issue 的状态。
// 如果 Issue 记录不存在，会自动创建 idle 状态的初始记录。
func (s *sqliteIssueStore) CheckIssues(ctx context.Context, repo string, numbers []int) ([]IssueCheckResult, error) {
	if len(numbers) == 0 {
		return nil, nil
	}

	logger.Debug("store.issue", "check issues: repo=%s numbers=%v", repo, numbers)

	// 为不存在的 Issue 自动创建 idle 初始记录
	insertStmt, err := s.db.PrepareContext(ctx,
		`INSERT OR IGNORE INTO issue_claims (repo_full_name, issue_number, status, updated_at)
		 VALUES (?, ?, 'idle', CURRENT_TIMESTAMP)`)
	if err != nil {
		return nil, fmt.Errorf("prepare insert: %w", err)
	}
	defer insertStmt.Close()

	autoCreated := 0
	for _, num := range numbers {
		res, err := insertStmt.ExecContext(ctx, repo, num)
		if err != nil {
			return nil, fmt.Errorf("insert issue %d: %w", num, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			autoCreated++
		}
	}
	if autoCreated > 0 {
		logger.Debug("store.issue", "auto-created idle records for %d issues", autoCreated)
	}

	// 查询所有匹配的记录
	placeholders := make([]string, len(numbers))
	args := make([]any, 0, len(numbers)+1)
	args = append(args, repo)
	for i, num := range numbers {
		placeholders[i] = "?"
		args = append(args, num)
	}

	query := fmt.Sprintf(
		`SELECT issue_number, status, session_id, claimed_at
		   FROM issue_claims
		  WHERE repo_full_name = ? AND issue_number IN (%s)`,
		strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query issues: %w", err)
	}
	defer rows.Close()

	results := make([]IssueCheckResult, 0, len(numbers))
	for rows.Next() {
		var r IssueCheckResult
		var sessionID sql.NullString
		var claimedAt sql.NullString
		if err := rows.Scan(&r.Number, &r.Status, &sessionID, &claimedAt); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		if sessionID.Valid {
			r.SessionID = &sessionID.String
		}
		if claimedAt.Valid {
			r.ClaimedAt = &claimedAt.String
		}
		results = append(results, r)
	}
	logger.Info("store.issue", "check issues: repo=%s found %d records", repo, len(results))
	return results, rows.Err()
}

// ClaimIssue 尝试领取指定 Issue。
// 使用乐观锁：仅在当前状态为 idle 时才能领取成功。
// 支持幂等：同一 session 重复领取返回成功。
func (s *sqliteIssueStore) ClaimIssue(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*ClaimResult, error) {
	logger.Debug("store.issue", "claim: repo=%s #%d session=%s", repo, number, sessionID)

	// 确保记录存在（不存在时创建 idle 记录）
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO issue_claims (repo_full_name, issue_number, status, updated_at)
		 VALUES (?, ?, 'idle', CURRENT_TIMESTAMP)`,
		repo, number)
	if err != nil {
		return nil, fmt.Errorf("ensure issue: %w", err)
	}

	// 乐观锁：仅在 status = 'idle' 时更新为 claimed
	res, err := s.db.ExecContext(ctx,
		`UPDATE issue_claims
		    SET status = 'claimed', session_id = ?, claimed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		  WHERE repo_full_name = ? AND issue_number = ? AND status = 'idle'`,
		sessionID, repo, number)
	if err != nil {
		return nil, fmt.Errorf("claim update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		var claimedAt string
		s.db.QueryRowContext(ctx,
			`SELECT claimed_at FROM issue_claims WHERE repo_full_name = ? AND issue_number = ?`,
			repo, number).Scan(&claimedAt)
		logger.Info("store.issue", "issue claimed: repo=%s #%d by %s", repo, number, sessionID)
		return &ClaimResult{Success: true, Status: "claimed", ClaimedAt: &claimedAt}, nil
	}

	// 领取失败，查询当前状态
	var curStatus string
	var curSession sql.NullString
	var curClaimedAt sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT status, session_id, claimed_at FROM issue_claims WHERE repo_full_name = ? AND issue_number = ?`,
		repo, number).Scan(&curStatus, &curSession, &curClaimedAt)
	if err != nil {
		return nil, fmt.Errorf("query state: %w", err)
	}

	// 幂等：同一 session 已领取过
	if curSession.Valid && curSession.String == sessionID {
		logger.Debug("store.issue", "claim idempotent: same session %s", sessionID)
		result := &ClaimResult{Success: true, Status: curStatus}
		if curClaimedAt.Valid {
			result.ClaimedAt = &curClaimedAt.String
		}
		return result, nil
	}

	// 已被其他 session 领取或处于终态
	logger.Warn("store.issue", "claim conflict: repo=%s #%d already claimed by %s", repo, number, curSession.String)
	result := &ClaimResult{Success: false, Status: curStatus}
	if curSession.Valid {
		result.ClaimedBy = &curSession.String
	}
	if curClaimedAt.Valid {
		result.ClaimedAt = &curClaimedAt.String
	}
	return result, nil
}

// UpdateIssueStatus 更新指定 Issue 的状态（仅当前持有者可更新）。
func (s *sqliteIssueStore) UpdateIssueStatus(ctx context.Context, repo string, number int, sessionID string, newStatus string) (*UpdateStatusResult, error) {
	logger.Debug("store.issue", "update status: repo=%s #%d -> %s", repo, number, newStatus)

	// 查询当前状态
	var curStatus string
	var curOwner sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT status, session_id FROM issue_claims WHERE repo_full_name = ? AND issue_number = ?`,
		repo, number,
	).Scan(&curStatus, &curOwner)
	if err == sql.ErrNoRows {
		return &UpdateStatusResult{Updated: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query status: %w", err)
	}

	// 非持有者无法更新
	if !curOwner.Valid || curOwner.String != sessionID {
		logger.Warn("store.issue", "status update rejected: not owner")
		return &UpdateStatusResult{PreviousStatus: curStatus, Updated: false}, nil
	}

	// 执行更新
	res, err := s.db.ExecContext(ctx,
		`UPDATE issue_claims SET status = ?, updated_at = CURRENT_TIMESTAMP
		  WHERE repo_full_name = ? AND issue_number = ? AND session_id = ?`,
		newStatus, repo, number, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		logger.Info("store.issue", "status updated: repo=%s #%d %s->%s", repo, number, curStatus, newStatus)
	}
	return &UpdateStatusResult{PreviousStatus: curStatus, NewStatus: newStatus, Updated: n > 0}, nil
}

// ReleaseIssue 释放指定 session 持有的某个 Issue。
// 仅对非终态（非 merged/rejected）的 Issue 有效。
func (s *sqliteIssueStore) ReleaseIssue(ctx context.Context, repo string, number int, sessionID string) (bool, error) {
	logger.Debug("store.issue", "release: repo=%s #%d session=%s", repo, number, sessionID)

	// 查询当前持有者
	var owner sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT session_id FROM issue_claims WHERE repo_full_name = ? AND issue_number = ?`,
		repo, number,
	).Scan(&owner)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query owner: %w", err)
	}
	if !owner.Valid || owner.String != sessionID {
		return false, nil
	}

	// 释放：仅对非终态状态生效
	res, err := s.db.ExecContext(ctx,
		`UPDATE issue_claims
		    SET status = 'idle', session_id = NULL, claimed_at = NULL, updated_at = CURRENT_TIMESTAMP
		  WHERE repo_full_name = ? AND issue_number = ? AND session_id = ?
		    AND status NOT IN ('merged', 'rejected')`,
		repo, number, sessionID,
	)
	if err != nil {
		return false, fmt.Errorf("update: %w", err)
	}
	n, _ := res.RowsAffected()
	released := n > 0
	logger.Info("store.issue", "issue released: repo=%s #%d released=%v", repo, number, released)
	return released, nil
}

// ReleaseSessionIssues 释放指定 session 持有的所有非终态 Issue。
// 返回被释放的 Issue 编号列表。
func (s *sqliteIssueStore) ReleaseSessionIssues(ctx context.Context, sessionID string) ([]int, error) {
	logger.Debug("store.issue", "release session: %s", sessionID)

	// 先收集将要释放的 Issue 编号
	rows, err := s.db.QueryContext(ctx,
		`SELECT issue_number FROM issue_claims
		  WHERE session_id = ? AND status NOT IN ('merged', 'rejected')`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query session issues: %w", err)
	}

	var numbers []int
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan: %w", err)
		}
		numbers = append(numbers, n)
	}
	rows.Close()

	if len(numbers) == 0 {
		return nil, nil
	}

	// 批量更新释放
	_, err = s.db.ExecContext(ctx,
		`UPDATE issue_claims
		    SET status = 'idle', session_id = NULL, claimed_at = NULL, updated_at = CURRENT_TIMESTAMP
		  WHERE session_id = ? AND status NOT IN ('merged', 'rejected')`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("batch release: %w", err)
	}

	logger.Info("store.issue", "released %d issues for session %s", len(numbers), sessionID)
	return numbers, nil
}
