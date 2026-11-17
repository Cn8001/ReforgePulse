package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/cn8001/reforgepulse/internal/recorder"
)

// Recordings keeps recorder sessions, the newest keep of them.
type Recordings struct {
	db   *sql.DB
	keep int
}

func NewRecordings(db *sql.DB, keep int) *Recordings { return &Recordings{db: db, keep: keep} }

// Save stores r under an ID from its start time and returns it with the ID.
func (s *Recordings) Save(ctx context.Context, r recorder.Recording) (recorder.Recording, error) {
	r.ID = r.Started.UTC().Format("20060102-150405")
	body, err := json.Marshal(r)
	if err != nil {
		return r, fmt.Errorf("encode recording: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return r, fmt.Errorf("begin save: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO recordings (id, body) VALUES (?, ?)`, r.ID, string(body)); err != nil {
		return r, fmt.Errorf("save recording: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM recordings WHERE id NOT IN (SELECT id FROM recordings ORDER BY id DESC LIMIT ?)`, s.keep); err != nil {
		return r, fmt.Errorf("drop old recordings: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return r, fmt.Errorf("commit recording: %w", err)
	}
	return r, nil
}

// All returns the saved sessions, newest first, without their load series:
// a list needs the summary, not every point.
func (s *Recordings) All(ctx context.Context) ([]recorder.Recording, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT body FROM recordings ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list recordings: %w", err)
	}
	defer rows.Close()
	out := []recorder.Recording{}
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, fmt.Errorf("read recording: %w", err)
		}
		var r recorder.Recording
		if json.Unmarshal([]byte(body), &r) != nil {
			continue
		}
		r.CPU, r.GPU, r.Memory = nil, nil, nil
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list recordings: %w", err)
	}
	return out, nil
}

func (s *Recordings) Get(ctx context.Context, id string) (recorder.Recording, error) {
	var r recorder.Recording
	var body string
	err := s.db.QueryRowContext(ctx, `SELECT body FROM recordings WHERE id = ?`, id).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return r, fmt.Errorf("recording %q: %w", id, os.ErrNotExist)
	}
	if err != nil {
		return r, fmt.Errorf("read recording: %w", err)
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		return r, fmt.Errorf("decode recording %s: %w", id, err)
	}
	return r, nil
}

func (s *Recordings) Delete(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM recordings WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete recording: %w", err)
	}
	return nil
}
