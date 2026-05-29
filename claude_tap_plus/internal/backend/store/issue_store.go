package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type sqliteIssueStore struct {
	db *sql.DB
}

func (s *sqliteIssueStore) CheckIssues(ctx context.Context, repo string, numbers []int) ([]IssueCheckResult, error) {
	if len(numbers) == 0 {
		return nil, nil
	}

	// Auto-create idle records for any issue_number not yet in the table.
	insertStmt, err := s.db.PrepareContext(ctx,
		`INSERT OR IGNORE INTO issue_claims (repo_full_name, issue_number, status, updated_at)
		 VALUES (?, ?, 'idle', CURRENT_TIMESTAMP)`)
	if err != nil {
		return nil, fmt.Errorf("prepare insert: %w", err)
	}
	defer insertStmt.Close()

	for _, num := range numbers {
		if _, err := insertStmt.ExecContext(ctx, repo, num); err != nil {
			return nil, fmt.Errorf("insert issue %d: %w", num, err)
		}
	}

	// Query all matching records.
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
	return results, rows.Err()
}

func (s *sqliteIssueStore) ClaimIssue(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*ClaimResult, error) {
	// Ensure the record exists.
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO issue_claims (repo_full_name, issue_number, status, updated_at)
		 VALUES (?, ?, 'idle', CURRENT_TIMESTAMP)`,
		repo, number)
	if err != nil {
		return nil, fmt.Errorf("ensure issue: %w", err)
	}

	// Optimistic lock: try to claim only if status = 'idle'.
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
		return &ClaimResult{Success: true, Status: "claimed", ClaimedAt: &claimedAt}, nil
	}

	// Claim failed — check current state.
	var curStatus string
	var curSession sql.NullString
	var curClaimedAt sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT status, session_id, claimed_at FROM issue_claims WHERE repo_full_name = ? AND issue_number = ?`,
		repo, number).Scan(&curStatus, &curSession, &curClaimedAt)
	if err != nil {
		return nil, fmt.Errorf("query state: %w", err)
	}

	// Idempotent: same session already claimed it.
	if curSession.Valid && curSession.String == sessionID {
		result := &ClaimResult{Success: true, Status: curStatus}
		if curClaimedAt.Valid {
			result.ClaimedAt = &curClaimedAt.String
		}
		return result, nil
	}

	// Already claimed by another session or in terminal state.
	result := &ClaimResult{Success: false, Status: curStatus}
	if curSession.Valid {
		result.ClaimedBy = &curSession.String
	}
	if curClaimedAt.Valid {
		result.ClaimedAt = &curClaimedAt.String
	}
	return result, nil
}

func (s *sqliteIssueStore) UpdateIssueStatus(ctx context.Context, repo string, number int, sessionID string, newStatus string) (*UpdateStatusResult, error) {
	// Check current state.
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

	// Only the owning session can update status.
	if !curOwner.Valid || curOwner.String != sessionID {
		return &UpdateStatusResult{PreviousStatus: curStatus, Updated: false}, nil
	}

	// Update.
	res, err := s.db.ExecContext(ctx,
		`UPDATE issue_claims SET status = ?, updated_at = CURRENT_TIMESTAMP
		  WHERE repo_full_name = ? AND issue_number = ? AND session_id = ?`,
		newStatus, repo, number, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}
	n, _ := res.RowsAffected()
	return &UpdateStatusResult{PreviousStatus: curStatus, NewStatus: newStatus, Updated: n > 0}, nil
}

func (s *sqliteIssueStore) ReleaseIssue(ctx context.Context, repo string, number int, sessionID string) (bool, error) {
	// Check current owner.
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

	// Release: only non-terminal statuses.
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
	return n > 0, nil
}

func (s *sqliteIssueStore) ReleaseSessionIssues(ctx context.Context, sessionID string) ([]int, error) {
	// Collect issue numbers that will be released.
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

	// Batch update.
	_, err = s.db.ExecContext(ctx,
		`UPDATE issue_claims
		    SET status = 'idle', session_id = NULL, claimed_at = NULL, updated_at = CURRENT_TIMESTAMP
		  WHERE session_id = ? AND status NOT IN ('merged', 'rejected')`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("batch release: %w", err)
	}

	return numbers, nil
}
