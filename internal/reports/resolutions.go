package reports

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

// Resolutions is what the user marked resolved, by finding key. It is the
// app's own state, not a change to the PC, so it is not in the change log.
type Resolutions struct {
	db  *sql.DB
	now func() time.Time
}

func NewResolutions(db *sql.DB) *Resolutions { return &Resolutions{db: db, now: time.Now} }

// All returns every resolution by key.
func (r *Resolutions) All(ctx context.Context) (map[string]domain.Resolution, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, title, at FROM resolutions`)
	if err != nil {
		return nil, fmt.Errorf("read resolutions: %w", err)
	}
	defer rows.Close()
	m := map[string]domain.Resolution{}
	for rows.Next() {
		var v domain.Resolution
		var at string
		if err := rows.Scan(&v.Key, &v.Title, &at); err != nil {
			return nil, fmt.Errorf("read resolution: %w", err)
		}
		v.At, _ = time.Parse(time.RFC3339Nano, at)
		m[v.Key] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read resolutions: %w", err)
	}
	return m, nil
}

// Resolve marks a finding resolved now.
func (r *Resolutions) Resolve(ctx context.Context, key, title string) error {
	_, err := r.db.ExecContext(ctx, `INSERT OR REPLACE INTO resolutions (key, title, at) VALUES (?, ?, ?)`,
		key, title, r.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save resolution: %w", err)
	}
	return nil
}

// Reopen forgets a resolution.
func (r *Resolutions) Reopen(ctx context.Context, key string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM resolutions WHERE key = ?`, key); err != nil {
		return fmt.Errorf("delete resolution: %w", err)
	}
	return nil
}

// Apply sets each finding's status from the resolutions. A resolved finding
// whose newest event is later than the resolution has come back.
func Apply(res *domain.ScanResult, resolved map[string]domain.Resolution) {
	for i := range res.Findings {
		f := &res.Findings[i]
		v, ok := resolved[f.Key]
		switch {
		case !ok:
			f.Status, f.ResolvedAt = domain.Open, time.Time{}
		case f.Latest.After(v.At):
			f.Status, f.ResolvedAt = domain.Returned, v.At
		default:
			f.Status, f.ResolvedAt = domain.Resolved, v.At
		}
	}
}

// Summarize is the one line a report gets in the list of recent scans.
func Summarize(r domain.ScanResult) domain.ReportSummary {
	rank := map[domain.Severity]int{domain.Info: 1, domain.Low: 2, domain.Medium: 3, domain.High: 4, domain.Critical: 5}
	s := domain.ReportSummary{ID: r.ID, At: r.At}
	for _, f := range r.Findings {
		if f.Status == domain.Resolved {
			s.Resolved++
			continue
		}
		s.Open++
		if rank[f.Severity] > rank[s.Worst] {
			s.Worst = f.Severity
		}
	}
	return s
}
