package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

// checkID names a run by when it finished, so IDs sort by time.
const checkID = "20060102-150405.000"

// Checks keeps the results of Windows' health checks, the newest keep.
type Checks struct {
	db   *sql.DB
	keep int
}

func NewChecks(db *sql.DB, keep int) *Checks { return &Checks{db: db, keep: keep} }

func (s *Checks) Save(ctx context.Context, r domain.CheckResult) (domain.CheckResult, error) {
	r.ID = r.At.UTC().Format(checkID)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return r, fmt.Errorf("begin save: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO check_results (id, name, exit_code, summary, log) VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.ExitCode, r.Summary, r.Log); err != nil {
		return r, fmt.Errorf("save check result: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM check_results WHERE id NOT IN (SELECT id FROM check_results ORDER BY id DESC LIMIT ?)`, s.keep); err != nil {
		return r, fmt.Errorf("drop old check results: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return r, fmt.Errorf("commit check result: %w", err)
	}
	return r, nil
}

// All returns every kept result, newest first.
func (s *Checks) All(ctx context.Context) ([]domain.CheckResult, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, exit_code, summary, log FROM check_results ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list check results: %w", err)
	}
	defer rows.Close()
	out := []domain.CheckResult{}
	for rows.Next() {
		var r domain.CheckResult
		if err := rows.Scan(&r.ID, &r.Name, &r.ExitCode, &r.Summary, &r.Log); err != nil {
			return nil, fmt.Errorf("read check result: %w", err)
		}
		r.At, _ = time.Parse(checkID, r.ID)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list check results: %w", err)
	}
	return out, nil
}

func (s *Checks) Delete(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM check_results WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete check result: %w", err)
	}
	return nil
}

// DeleteBefore removes results of checks that finished before t.
func (s *Checks) DeleteBefore(ctx context.Context, t time.Time) (int, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM check_results WHERE id < ?`, t.UTC().Format(checkID))
	if err != nil {
		return 0, fmt.Errorf("delete check results: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
