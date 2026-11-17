// Package reports keeps past scans and what the user marked resolved. A
// scan is one row holding its result as compact JSON, and only the latest
// few are kept so the table never grows without bound.
package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

// idLayout makes IDs from scan times, so sorting IDs sorts by time.
const idLayout = "20060102-150405.000"

func idAt(t time.Time) string { return strings.Replace(t.UTC().Format(idLayout), ".", "-", 1) }

type Store struct {
	db   *sql.DB
	keep int
}

// New keeps reports in db and only the newest keep of them.
func New(db *sql.DB, keep int) *Store { return &Store{db: db, keep: keep} }

// Save stores r under an ID taken from its time, marks findings the scan
// before it did not have as new, and drops reports past the limit.
func (s *Store) Save(ctx context.Context, r domain.ScanResult) (domain.ScanResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return r, fmt.Errorf("begin save: %w", err)
	}
	defer tx.Rollback()
	var prev string
	err = tx.QueryRowContext(ctx, `SELECT body FROM reports ORDER BY id DESC LIMIT 1`).Scan(&prev)
	if err == nil {
		var p domain.ScanResult
		if json.Unmarshal([]byte(prev), &p) == nil {
			markNew(&r, p)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return r, fmt.Errorf("read last report: %w", err)
	}
	r.ID = idAt(r.At)
	body, err := json.Marshal(r)
	if err != nil {
		return r, fmt.Errorf("encode report: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO reports (id, at, body) VALUES (?, ?, ?)`, r.ID, r.ID, string(body)); err != nil {
		return r, fmt.Errorf("save report: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM reports WHERE id NOT IN (SELECT id FROM reports ORDER BY id DESC LIMIT ?)`, s.keep); err != nil {
		return r, fmt.Errorf("drop old reports: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return r, fmt.Errorf("commit report: %w", err)
	}
	return r, nil
}

// All returns every kept report, newest first.
func (s *Store) All(ctx context.Context) ([]domain.ScanResult, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT body FROM reports ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()
	out := []domain.ScanResult{}
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, fmt.Errorf("read report: %w", err)
		}
		var r domain.ScanResult
		if json.Unmarshal([]byte(body), &r) != nil {
			continue // one unreadable report should not hide the others
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	return out, nil
}

// Get returns one report by ID.
func (s *Store) Get(ctx context.Context, id string) (domain.ScanResult, error) {
	var r domain.ScanResult
	var body string
	err := s.db.QueryRowContext(ctx, `SELECT body FROM reports WHERE id = ?`, id).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return r, fmt.Errorf("report %q: %w", id, os.ErrNotExist)
	}
	if err != nil {
		return r, fmt.Errorf("read report: %w", err)
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		return r, fmt.Errorf("decode report %s: %w", id, err)
	}
	return r, nil
}

// Latest is the newest report, or nil when there is none.
func (s *Store) Latest(ctx context.Context) (*domain.ScanResult, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM reports ORDER BY id DESC LIMIT 1`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find last report: %w", err)
	}
	r, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Delete removes one report.
func (s *Store) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM reports WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete report: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("report %q: %w", id, os.ErrNotExist)
	}
	return nil
}

// DeleteBefore removes reports of scans that ran before t and returns how
// many went.
func (s *Store) DeleteBefore(ctx context.Context, t time.Time) (int, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM reports WHERE id < ?`, idAt(t))
	if err != nil {
		return 0, fmt.Errorf("delete reports: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func markNew(r *domain.ScanResult, prev domain.ScanResult) {
	seen := map[string]bool{}
	for _, f := range prev.Findings {
		seen[f.Key] = true
	}
	for i := range r.Findings {
		r.Findings[i].New = !seen[r.Findings[i].Key]
	}
}
