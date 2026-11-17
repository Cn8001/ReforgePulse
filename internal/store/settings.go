package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Settings are the user's choices, one text value per key.
type Settings struct {
	db *sql.DB
}

func NewSettings(db *sql.DB) *Settings { return &Settings{db: db} }

// Get returns the value for key, or def when it was never set.
func (s *Settings) Get(ctx context.Context, key, def string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return def, nil
	}
	if err != nil {
		return "", fmt.Errorf("read setting %s: %w", key, err)
	}
	return v, nil
}

func (s *Settings) Set(ctx context.Context, key, value string) error {
	if _, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value); err != nil {
		return fmt.Errorf("save setting %s: %w", key, err)
	}
	return nil
}
