// Package oldfiles finds what has sat untouched in a folder like Downloads
// for months: installers run once, archives already extracted.
package oldfiles

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

const (
	idleFor = 180 * 24 * time.Hour
	minSize = 1 << 20 // smaller items are not worth a decision
)

// Folder is a place to look in, with the name the window shows.
type Folder struct{ Name, Path string }

type Inspector struct {
	folders []Folder
	now     func() time.Time
}

func New(folders []Folder) *Inspector { return &Inspector{folders: folders, now: time.Now} }

// List returns top-level items, files or folders, whose newest file is older
// than the idle limit, biggest first.
func (in *Inspector) List(ctx context.Context) ([]domain.Leftover, error) {
	cutoff := in.now().Add(-idleFor)
	var out []domain.Leftover
	for _, f := range in.folders {
		entries, err := os.ReadDir(f.Path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if e.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
				continue
			}
			l := measure(filepath.Join(f.Path, e.Name()))
			if l.Size >= minSize && !l.Modified.IsZero() && l.Modified.Before(cutoff) {
				l.Root = f.Name
				out = append(out, l)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	return out, nil
}

func measure(path string) domain.Leftover {
	l := domain.Leftover{Path: path}
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
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
		if err != nil {
			return nil
		}
		l.Files++
		l.Size += uint64(info.Size())
		if info.ModTime().After(l.Modified) {
			l.Modified = info.ModTime()
		}
		return nil
	})
	return l
}
