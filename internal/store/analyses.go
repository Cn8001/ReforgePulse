package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

// Analyses keeps debugger analyses of crash dumps, which take minutes and
// a symbol download, so they are worth keeping.
type Analyses struct {
	db *sql.DB
}

func NewAnalyses(db *sql.DB) *Analyses { return &Analyses{db: db} }

// Save keeps a for the dump called name that Windows wrote at dumped.
func (s *Analyses) Save(ctx context.Context, name string, dumped time.Time, a domain.DumpAnalysis) error {
	body, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("encode analysis: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO dump_analyses (name, dumped, body) VALUES (?, ?, ?)`,
		name, dumped.UTC().Format(Stamp), string(body)); err != nil {
		return fmt.Errorf("save analysis of %s: %w", name, err)
	}
	return nil
}

// All returns every saved analysis, newest dump first, each with the time
// Windows wrote its dump.
func (s *Analyses) All(ctx context.Context) ([]domain.SavedAnalysis, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT dumped, body FROM dump_analyses ORDER BY dumped DESC`)
	if err != nil {
		return nil, fmt.Errorf("list analyses: %w", err)
	}
	defer rows.Close()
	out := []domain.SavedAnalysis{}
	for rows.Next() {
		var dumped, body string
		if err := rows.Scan(&dumped, &body); err != nil {
			return nil, fmt.Errorf("read analysis: %w", err)
		}
		var a domain.SavedAnalysis
		if json.Unmarshal([]byte(body), &a.DumpAnalysis) != nil {
			continue
		}
		a.Dumped, _ = time.Parse(Stamp, dumped)
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list analyses: %w", err)
	}
	return out, nil
}

// Delete forgets one analysis. The dump file itself stays.
func (s *Analyses) Delete(ctx context.Context, name string, dumped time.Time) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM dump_analyses WHERE name = ? AND dumped = ?`, name, dumped.UTC().Format(Stamp)); err != nil {
		return fmt.Errorf("delete analysis of %s: %w", name, err)
	}
	return nil
}

// Get returns the saved analysis of that dump, or nil.
func (s *Analyses) Get(ctx context.Context, name string, dumped time.Time) (*domain.DumpAnalysis, error) {
	var body string
	err := s.db.QueryRowContext(ctx, `SELECT body FROM dump_analyses WHERE name = ? AND dumped = ?`,
		name, dumped.UTC().Format(Stamp)).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read analysis of %s: %w", name, err)
	}
	var a domain.DumpAnalysis
	if err := json.Unmarshal([]byte(body), &a); err != nil {
		return nil, fmt.Errorf("decode analysis of %s: %w", name, err)
	}
	return &a, nil
}
