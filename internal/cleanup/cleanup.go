// Package cleanup finds and deletes disposable files in known locations.
// It deletes only regular files older than a location's minimum age, never
// follows links or junctions, and never removes a location's root folder.
package cleanup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

// Recorder is the change journal: every cleanup is logged before it starts.
type Recorder interface {
	Begin(ctx context.Context, c domain.Change) (string, error)
	Finish(ctx context.Context, id, after string, err error) error
}

type Engine struct {
	locs    []platform.Location
	journal Recorder
	now     func() time.Time
}

func New(locs []platform.Location, journal Recorder) *Engine {
	return &Engine{locs: locs, journal: journal, now: time.Now}
}

// Analyze counts what each location would free. Nothing is changed.
func (e *Engine) Analyze(ctx context.Context) ([]domain.CleanupArea, error) {
	out := make([]domain.CleanupArea, 0, len(e.locs))
	for _, l := range e.locs {
		a := domain.CleanupArea{ID: l.ID, Name: l.Name, Note: l.Note, Paths: l.Paths}
		err := e.walk(ctx, l, func(_ string, size int64) {
			a.Files++
			a.Bytes += uint64(size)
		})
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// Clean deletes the eligible files of the chosen locations and reports what
// was really freed. Files in use are skipped, not forced. report hears how
// far it is, by files; cancelling ctx stops it between two files.
func (e *Engine) Clean(ctx context.Context, ids []string, report func(float64, string)) (domain.CleanupResult, error) {
	chosen := map[string]bool{}
	for _, id := range ids {
		chosen[id] = true
	}
	var locs []platform.Location
	var names, paths []string
	for _, l := range e.locs {
		if chosen[l.ID] {
			locs = append(locs, l)
			names = append(names, l.Name)
			paths = append(paths, l.Paths...)
		}
	}
	if len(locs) == 0 {
		return domain.CleanupResult{}, errors.New("nothing selected")
	}

	before, err := e.Analyze(ctx)
	if err != nil {
		return domain.CleanupResult{}, err
	}
	var files int
	var bytes uint64
	for _, a := range before {
		if chosen[a.ID] {
			files, bytes = files+a.Files, bytes+a.Bytes
		}
	}
	id, err := e.journal.Begin(ctx, domain.Change{
		Action:  "cleanup",
		Summary: "Clean up " + strings.Join(names, ", "),
		Target:  strings.Join(paths, "; "),
		Before:  fmt.Sprintf("%d files, %d bytes", files, bytes),
	})
	if err != nil {
		return domain.CleanupResult{}, fmt.Errorf("log cleanup: %w", err)
	}

	r := domain.CleanupResult{ChangeID: id}
	last := time.Now()
	for _, l := range locs {
		var dirs []string
		report(progress(r, files), l.Name)
		err = e.walk(ctx, l, func(path string, size int64) {
			if os.Remove(path) != nil {
				r.Skipped++
			} else {
				r.Deleted++
				r.Freed += uint64(size)
				dirs = append(dirs, filepath.Dir(path))
			}
			if time.Since(last) > 200*time.Millisecond {
				last = time.Now()
				report(progress(r, files), fmt.Sprintf("%s, %d of %d files", l.Name, r.Deleted+r.Skipped, files))
			}
		})
		if err != nil {
			break
		}
		removeEmpty(dirs, l.Paths)
	}
	after := fmt.Sprintf("deleted %d files, freed %d bytes, skipped %d", r.Deleted, r.Freed, r.Skipped)
	if errors.Is(err, context.Canceled) {
		after = "stopped by the user: " + after
	}
	if ferr := e.journal.Finish(ctx, id, after, err); ferr != nil && err == nil {
		err = fmt.Errorf("log cleanup result: %w", ferr)
	}
	return r, err
}

// progress is the share of planned files handled so far; files that came
// into being after the analysis may push it past one.
func progress(r domain.CleanupResult, planned int) float64 {
	if planned == 0 {
		return -1
	}
	return min(1, float64(r.Deleted+r.Skipped)/float64(planned))
}

// walk calls fn for every regular file old enough to delete. Folders that
// are links or junctions are skipped, so the walk never leaves the location.
func (e *Engine) walk(ctx context.Context, l platform.Location, fn func(path string, size int64)) error {
	cutoff := e.now().Add(-l.MinAge)
	for _, root := range expand(l.Paths) {
		if !filepath.IsAbs(root) {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if cerr := ctx.Err(); cerr != nil {
				return cerr
			}
			if err != nil {
				if d != nil && d.IsDir() {
					return fs.SkipDir // unreadable folder: leave it alone
				}
				return nil
			}
			if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			info, err := d.Info()
			if err != nil || info.ModTime().After(cutoff) {
				return nil
			}
			fn(path, info.Size())
			return nil
		})
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
	}
	return nil
}

// expand resolves glob paths such as a browser's User Data\*\Cache, one
// root per match, so each profile is cleaned without its parent.
func expand(paths []string) []string {
	var out []string
	for _, p := range paths {
		if !strings.ContainsAny(p, "*?[") {
			out = append(out, p)
			continue
		}
		matches, _ := filepath.Glob(p)
		out = append(out, matches...)
	}
	return out
}

// removeEmpty drops folders the cleanup emptied, deepest first, but never a
// location root. os.Remove refuses a folder that still holds anything.
func removeEmpty(dirs, roots []string) {
	isRoot := map[string]bool{}
	for _, r := range expand(roots) {
		isRoot[filepath.Clean(r)] = true
	}
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		for !isRoot[filepath.Clean(d)] && os.Remove(d) == nil {
			d = filepath.Dir(d)
		}
	}
}
