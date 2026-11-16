// Package journal is the change log: an append-only JSON Lines file. A
// change is written before ReforgePulse touches the system and finished with
// a one-line status after, so a crash halfway still leaves a trace.
package journal

import (
	"bufio"
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

	"github.com/cn8001/reforgepulse/internal/domain"
)

type Journal struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
}

// Open uses dir/journal.jsonl, creating dir when needed.
func Open(dir string) (*Journal, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create journal dir: %w", err)
	}
	return &Journal{path: filepath.Join(dir, "journal.jsonl"), now: time.Now}, nil
}

// Begin records a change about to be applied and returns its ID.
func (j *Journal) Begin(ctx context.Context, c domain.Change) (string, error) {
	c.At = j.now().UTC()
	c.Status = domain.Started
	c.ID = shortID(c)
	if err := j.append(ctx, c); err != nil {
		return "", err
	}
	return c.ID, nil
}

// Finish records how the change went: applied when err is nil, else failed.
// after says what the change actually did, such as the bytes it freed.
func (j *Journal) Finish(ctx context.Context, id, after string, err error) error {
	c := domain.Change{ID: id, At: j.now().UTC(), Status: domain.Applied, After: after}
	if err != nil {
		c.Status, c.Error = domain.Failed, err.Error()
	}
	return j.append(ctx, c)
}

// Log returns every change, newest first, with its latest status.
func (j *Journal) Log(ctx context.Context) ([]domain.Change, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	f, err := os.Open(j.path)
	if errors.Is(err, os.ErrNotExist) {
		return []domain.Change{}, nil // an empty list, not null, for the UI and agents
	}
	if err != nil {
		return nil, fmt.Errorf("open journal: %w", err)
	}
	defer f.Close()

	var order []string
	byID := map[string]*domain.Change{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var c domain.Change
		if json.Unmarshal(sc.Bytes(), &c) != nil || c.ID == "" {
			continue // a torn line from a crash mid-write
		}
		if prev, ok := byID[c.ID]; ok {
			prev.Status, prev.Error = c.Status, c.Error
			if c.After != "" {
				prev.After = c.After
			}
			continue
		}
		byID[c.ID] = &c
		order = append(order, c.ID)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read journal: %w", err)
	}

	for _, c := range byID {
		if undone, ok := byID[c.Reverts]; ok && c.Status == domain.Applied {
			undone.Status = domain.Reverted
		}
	}
	out := make([]domain.Change, 0, len(order))
	for i := len(order) - 1; i >= 0; i-- {
		out = append(out, *byID[order[i]])
	}
	return out, nil
}

// append writes one line and syncs it: the entry must survive a crash in the
// change that follows.
func (j *Journal) append(ctx context.Context, c domain.Change) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	line, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode change: %w", err)
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	f, err := os.OpenFile(j.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open journal: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return fmt.Errorf("write journal: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("sync journal: %w", err)
	}
	return f.Close()
}

// shortID hashes the change's content, like a short git hash.
func shortID(c domain.Change) string {
	h := sha256.Sum256(fmt.Appendf(nil, "%d|%s|%s|%s|%s|%s", c.At.UnixNano(), c.Summary, c.Target, c.Before, c.After, c.Reverts))
	return hex.EncodeToString(h[:])[:7]
}
