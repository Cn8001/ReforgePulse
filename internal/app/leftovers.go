package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type LeftoverFinder interface {
	List(ctx context.Context) ([]domain.Leftover, error)
}

type Recycler interface {
	Recycle(ctx context.Context, paths []string) error
}

// LeftoverService finds things the user may not need, such as folders an
// uninstalled program left behind or downloads untouched for months, and
// moves the ones the user picks to the Recycle Bin.
type LeftoverService struct {
	what    string // what it looks for, for the status bar and the log
	finder  LeftoverFinder
	bin     Recycler
	runner  TaskRunner
	journal Recorder

	mu    sync.Mutex
	found map[string]domain.Leftover // the last scan; only these may be recycled
}

// LeftoverDeps are the leftover service's dependencies.
type LeftoverDeps struct {
	What    string // "leftovers", "old downloads"
	Finder  LeftoverFinder
	Bin     Recycler
	Runner  TaskRunner
	Journal Recorder
}

func NewLeftoverService(d LeftoverDeps) *LeftoverService {
	return &LeftoverService{what: d.What, finder: d.Finder, bin: d.Bin, runner: d.Runner, journal: d.Journal}
}

func (s *LeftoverService) List(ctx context.Context) ([]domain.Leftover, error) {
	var out []domain.Leftover
	err := s.runner.Do(ctx, "Looking for "+s.what, func(ctx context.Context, _ tasks.Report) error {
		var err error
		out, err = s.finder.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("find %s: %w", s.what, err)
	}
	found := make(map[string]domain.Leftover, len(out))
	for _, l := range out {
		found[l.Path] = l
	}
	s.mu.Lock()
	s.found = found
	s.mu.Unlock()
	return out, nil
}

// Recycle moves leftovers to the Recycle Bin. A path the last scan did not
// report is refused, whatever the window sends.
func (s *LeftoverService) Recycle(ctx context.Context, paths []string) error {
	s.mu.Lock()
	var total uint64
	for _, p := range paths {
		l, ok := s.found[p]
		if !ok {
			s.mu.Unlock()
			return fmt.Errorf("%s was not found by the last scan for %s", p, s.what)
		}
		total += l.Size
	}
	s.mu.Unlock()
	if len(paths) == 0 {
		return fmt.Errorf("nothing selected")
	}
	return s.runner.Do(ctx, "Moving "+s.what+" to the Recycle Bin", func(ctx context.Context, report tasks.Report) error {
		id, err := s.journal.Begin(ctx, domain.Change{
			Action: "recycle", Summary: fmt.Sprintf("Move %d %s to the Recycle Bin", len(paths), s.what),
			Target: strings.Join(paths, "; "), Before: fmt.Sprintf("%d bytes", total),
		})
		if err != nil {
			return fmt.Errorf("log recycle: %w", err)
		}
		// One item at a time: Windows moves a whole list in one call that
		// can neither report nor stop, and a big folder takes a while.
		var moved []string
		for i, p := range paths {
			if err = ctx.Err(); err != nil {
				break
			}
			report(float64(i)/float64(len(paths)), filepath.Base(p))
			if err = s.bin.Recycle(ctx, []string{p}); err != nil {
				break
			}
			moved = append(moved, p)
		}
		after := fmt.Sprintf("%d of %d in the Recycle Bin; restore from there", len(moved), len(paths))
		if ferr := s.journal.Finish(ctx, id, after, err); ferr != nil && err == nil {
			err = ferr
		}
		s.mu.Lock()
		for _, p := range moved {
			delete(s.found, p)
		}
		s.mu.Unlock()
		return err
	})
}

// OldFilesService is the same service pointed at old downloads. It is its
// own type because the window binds services by type.
type OldFilesService struct{ *LeftoverService }

func NewOldFilesService(d LeftoverDeps) *OldFilesService {
	return &OldFilesService{NewLeftoverService(d)}
}
