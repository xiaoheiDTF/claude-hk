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
