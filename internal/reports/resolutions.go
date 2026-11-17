package reports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/cn8001/reforgepulse/internal/atomicfile"
	"github.com/cn8001/reforgepulse/internal/domain"
)

// Resolutions is what the user marked resolved, kept in one small file by
// finding key. It is the app's own state, not a change to the PC, so it is
// not in the change journal.
type Resolutions struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
}

// OpenResolutions uses dir/resolved.json.
func OpenResolutions(dir string) (*Resolutions, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create resolutions dir: %w", err)
	}
	return &Resolutions{path: filepath.Join(dir, "resolved.json"), now: time.Now}, nil
}

// All returns every resolution by key.
func (r *Resolutions) All(ctx context.Context) (map[string]domain.Resolution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.load()
}

// Resolve marks a finding resolved now.
func (r *Resolutions) Resolve(ctx context.Context, key, title string) error {
	return r.change(ctx, func(m map[string]domain.Resolution) {
		m[key] = domain.Resolution{Key: key, Title: title, At: r.now().UTC()}
	})
}

// Reopen forgets a resolution.
func (r *Resolutions) Reopen(ctx context.Context, key string) error {
	return r.change(ctx, func(m map[string]domain.Resolution) { delete(m, key) })
}

func (r *Resolutions) change(ctx context.Context, edit func(map[string]domain.Resolution)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.load()
	if err != nil {
		return err
	}
	edit(m)
	list := make([]domain.Resolution, 0, len(m))
	for _, v := range m {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Key < list[j].Key })
	b, err := json.Marshal(list)
	if err != nil {
		return fmt.Errorf("encode resolutions: %w", err)
	}
	return atomicfile.Write(r.path, b)
}

func (r *Resolutions) load() (map[string]domain.Resolution, error) {
	m := map[string]domain.Resolution{}
	b, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read resolutions: %w", err)
	}
	var list []domain.Resolution
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("decode resolutions: %w", err)
	}
	for _, v := range list {
		m[v.Key] = v
	}
	return m, nil
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
