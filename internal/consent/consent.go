// Package consent remembers that the user accepted the terms and licence.
// Acceptance is tied to a hash of the texts, so a change to either asks
// again instead of carrying an old yes over to new terms.
package consent

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

const key = "terms.accepted" // its value is the wording that was accepted

type Store struct {
	db *sql.DB
}

// New keeps acceptance in db's settings table.
func New(db *sql.DB) *Store { return &Store{db: db} }

// Version names one wording of the texts.
func Version(texts ...string) string {
	h := sha256.New()
	for _, t := range texts {
		h.Write([]byte(t))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// Accepted reports whether this version was accepted.
func (s *Store) Accepted(ctx context.Context, version string) (bool, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read consent: %w", err)
	}
	return v == version, nil
}

// Accept records that this version was accepted.
func (s *Store) Accept(ctx context.Context, version string) error {
	if _, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, version); err != nil {
		return fmt.Errorf("save consent: %w", err)
	}
	return nil
}
