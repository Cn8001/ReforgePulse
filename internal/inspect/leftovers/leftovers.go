// Package leftovers finds folders an uninstalled program left behind. A
// folder is a candidate only when nothing on the PC points to it: no
// installed program, running process, service or startup entry, no name
// match, and nothing inside changed for months.
package leftovers

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

const (
	idleFor  = 90 * 24 * time.Hour
	maxFiles = 50_000 // a folder this big is walked no further; it is reported as at least this
)

// system names folders Windows, drivers and shared runtimes own whatever
// is installed.
var system = regexp.MustCompile(`(?i)^(microsoft.*|windows.*|packages|temp|tmp|common files|internet explorer|intel.*|nvidia.*|msbuild|nuget|.*package cache.*|amd|realtek|dell|hp|lenovo|asus|msi|package cache|ssh|regid.*|uninstall information|reference assemblies|dotnet|windowsapps|modifiableswindowsapps|crashdumps|connecteddevicesplatform|d3dscache|comms|history|programs|peernetworking|virtualstore|desktop|usoshared|usoprivate|softwaredistribution|ssl|docker.*|wsl)$`)

type ProcessLister interface {
	List(ctx context.Context) ([]domain.Process, error)
}

// Deps are where the evidence of ownership comes from.
type Deps struct {
	Machine   platform.Machine
	Startup   platform.Startup
	Processes ProcessLister
	Files     platform.Files
}

type Inspector struct {
	d   Deps
	now func() time.Time
}

func New(d Deps) *Inspector { return &Inspector{d: d, now: time.Now} }

func (in *Inspector) List(ctx context.Context) ([]domain.Leftover, error) {
	claims, names, err := in.evidence(ctx)
	if err != nil {
		return nil, err
	}
	cutoff := in.now().Add(-idleFor)
	var out []domain.Leftover
	for _, root := range in.d.Files.LeftoverRoots() {
		entries, err := os.ReadDir(root.Path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			path := filepath.Join(root.Path, e.Name())
			if !e.IsDir() || e.Type()&fs.ModeSymlink != 0 || system.MatchString(e.Name()) || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if claimed(path, claims) || named(e.Name(), names) {
				continue
			}
			l := measure(ctx, path)
			if l.Files == 0 || l.Modified.After(cutoff) {
				continue // empty folders cost nothing; recent ones are in use
			}
			l.Root = root.Name
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	return out, nil
}

// evidence gathers every path something on the PC points at, and every
// program and publisher name, lower-cased.
func (in *Inspector) evidence(ctx context.Context) (claims, names []string, err error) {
	progs, err := in.d.Machine.Programs(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list programs: %w", err)
	}
	for _, p := range progs {
		claims = append(claims, clean(p.InstallLocation), clean(exeDir(p.UninstallString)))
		names = append(names, norm(p.Name), norm(p.Publisher))
	}
	if svcs, err := in.d.Machine.Services(ctx); err == nil {
		for _, s := range svcs {
			claims = append(claims, clean(exeDir(s.ImagePath)))
		}
	}
	if entries, err := in.d.Startup.StartupEntries(ctx); err == nil {
		for _, e := range entries {
			claims = append(claims, clean(exeDir(e.Command)))
		}
	}
	if procs, err := in.d.Processes.List(ctx); err == nil {
		for _, p := range procs {
			claims = append(claims, clean(filepath.Dir(p.Path)))
		}
	}
	return claims, names, nil
}

// claimed is true when something points inside the folder or at it.
func claimed(dir string, claims []string) bool {
	d := strings.ToLower(filepath.Clean(dir))
	for _, c := range claims {
		if c != "" && (c == d || strings.HasPrefix(c, d+string(filepath.Separator))) {
			return true
		}
	}
	return false
}

// named is true when the folder name is part of an installed program's or
// publisher's name, or the other way round, ignoring case and punctuation.
func named(folder string, names []string) bool {
	f := norm(folder)
	if len(f) < 3 {
		return true // too short to tell apart; leave it alone
	}
	for _, n := range names {
		if len(n) >= 3 && (strings.Contains(n, f) || strings.Contains(f, n)) {
			return true
		}
	}
	return false
}

func measure(ctx context.Context, dir string) domain.Leftover {
	l := domain.Leftover{Path: dir}
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || ctx.Err() != nil || l.Files >= maxFiles {
			return fs.SkipAll
		}
		if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // a folder's own time changes whenever a file is added, so only files count
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(l.Modified) {
			l.Modified = info.ModTime()
		}
		l.Files++
		l.Size += uint64(info.Size())
		return nil
	})
	return l
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func norm(s string) string { return nonAlnum.ReplaceAllString(strings.ToLower(s), "") }

func clean(p string) string {
	p = strings.Trim(strings.TrimSpace(p), `"`)
	if p == "" || p == "." {
		return ""
	}
	return strings.ToLower(filepath.Clean(p))
}

// exeDir is the folder of the program a command line starts.
func exeDir(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if strings.HasPrefix(cmd, `"`) {
		if end := strings.Index(cmd[1:], `"`); end >= 0 {
			return filepath.Dir(cmd[1 : end+1])
		}
	}
	if i := strings.Index(strings.ToLower(cmd), ".exe"); i >= 0 {
		return filepath.Dir(cmd[:i+4])
	}
	return ""
}
