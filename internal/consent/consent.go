// Package consent remembers that the user accepted the terms and licence.
// Acceptance is tied to a hash of the texts, so a change to either asks
// again instead of carrying an old yes over to new terms.
package consent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cn8001/reforgepulse/internal/atomicfile"
)

type record struct {
	Version string    `json:"version"`
	At      time.Time `json:"at"`
}

type Store struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
}

// Open uses dir/accepted.json.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create consent dir: %w", err)
	}
	return &Store{path: filepath.Join(dir, "accepted.json"), now: time.Now}, nil
}

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
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read consent: %w", err)
	}
	var r record
	if json.Unmarshal(b, &r) != nil {
		return false, nil // a damaged file just asks again
	}
	return r.Version == version, nil
}

// Accept records that this version was accepted now.
func (s *Store) Accept(ctx context.Context, version string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b, err := json.Marshal(record{Version: version, At: s.now().UTC()})
	if err != nil {
		return fmt.Errorf("encode consent: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := atomicfile.Write(s.path, b); err != nil {
		return fmt.Errorf("save consent: %w", err)
	}
	return nil
}
