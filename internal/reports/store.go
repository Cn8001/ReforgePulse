// Package reports keeps past scans and what the user marked resolved. A scan
// is one compact JSON file, named by when it ran, and only the latest few
// are kept so the folder never grows without bound.
package reports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/cn8001/reforgepulse/internal/domain"
)

const idLayout = "20060102-150405.000"

// validID guards Get: an ID comes from the window and becomes a file name.
var validID = regexp.MustCompile(`^\d{8}-\d{6}-\d{3}$`)

type Store struct {
	mu   sync.Mutex
	dir  string
	keep int
}

// Open uses dir for reports, creating it when needed, and keeps the newest
// keep of them.
func Open(dir string, keep int) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create reports dir: %w", err)
	}
	return &Store{dir: dir, keep: keep}, nil
}

// Save writes r under an ID taken from its time, marks findings the scan
// before it did not have as new, and drops reports past the limit.
func (s *Store) Save(ctx context.Context, r domain.ScanResult) (domain.ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return r, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ids, err := s.ids()
	if err != nil {
		return r, err
	}
	if len(ids) > 0 {
		prev, err := s.read(ids[0])
		if err == nil {
			markNew(&r, prev)
		}
	}
	r.ID = strings.Replace(r.At.UTC().Format(idLayout), ".", "-", 1)
	if err := s.write(r); err != nil {
		return r, err
	}
	ids = append([]string{r.ID}, ids...)
	for _, old := range ids[min(len(ids), s.keep):] {
		if err := os.Remove(s.path(old)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return r, fmt.Errorf("remove old report: %w", err)
		}
	}
	return r, nil
}

// All returns every kept report, newest first.
func (s *Store) All(ctx context.Context) ([]domain.ScanResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids, err := s.ids()
	if err != nil {
		return nil, err
	}
	out := make([]domain.ScanResult, 0, len(ids))
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		r, err := s.read(id)
		if err != nil {
			continue // a file torn by a crash mid-write is skipped, not fatal
		}
		out = append(out, r)
	}
	return out, nil
}

// Get returns one report by ID.
func (s *Store) Get(ctx context.Context, id string) (domain.ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.ScanResult{}, err
	}
	if !validID.MatchString(id) {
		return domain.ScanResult{}, fmt.Errorf("report %q: %w", id, os.ErrNotExist)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read(id)
}

// Latest is the newest report, or nil when there is none.
func (s *Store) Latest(ctx context.Context) (*domain.ScanResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ids, err := s.ids()
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if r, err := s.read(id); err == nil {
			return &r, nil
		}
	}
	return nil, nil
}

// ids lists report IDs, newest first: the layout sorts by time.
func (s *Store) ids() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	var ids []string
	for _, e := range entries {
		id := strings.TrimSuffix(e.Name(), ".json")
		if !e.IsDir() && id != e.Name() && validID.MatchString(id) {
			ids = append(ids, id)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))
	return ids, nil
}

func (s *Store) path(id string) string { return filepath.Join(s.dir, id+".json") }

func (s *Store) read(id string) (domain.ScanResult, error) {
	var r domain.ScanResult
	b, err := os.ReadFile(s.path(id))
	if err != nil {
		return r, fmt.Errorf("read report: %w", err)
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return r, fmt.Errorf("decode report %s: %w", id, err)
	}
	return r, nil
}

func (s *Store) write(r domain.ScanResult) error {
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	return writeFile(s.dir, s.path(r.ID), b)
}

// writeFile replaces path through a temporary file, so a crash leaves either
// the old file or the new one, never half of one.
func writeFile(dir, path string, b []byte) error {
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync %s: %w", filepath.Base(path), err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", filepath.Base(path), err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replace %s: %w", filepath.Base(path), err)
	}
	return nil
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
