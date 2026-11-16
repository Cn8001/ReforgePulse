// Package programs finds installed programs nobody has started in a long
// time. The evidence is Windows' own launch record (Prefetch); a missing
// record is weaker evidence and is labelled as such.
package programs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

// idleAfter is how long without a launch makes a program forgotten.
const idleAfter = 90 * 24 * time.Hour

// shared are packages other programs depend on; their own exes never run,
// so a missing launch record says nothing about them.
var shared = regexp.MustCompile(`(?i)redistributable|runtime|\bsdk\b|driver|visual c\+\+|\.net|webview2|directx|chipset|firmware|update for`)

type Inspector struct {
	m        platform.Machine
	listExes func(dir string) []string
	now      func() time.Time
}

func New(m platform.Machine) *Inspector {
	return &Inspector{m: m, listExes: exesIn, now: time.Now}
}

// List returns programs, forgotten ones first and the biggest first.
func (in *Inspector) List(ctx context.Context) ([]domain.InstalledProgram, error) {
	progs, err := in.m.Programs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list programs: %w", err)
	}
	runs, runsErr := in.m.LastRuns(ctx) // without it nothing is flagged
	now := in.now()

	out := make([]domain.InstalledProgram, 0, len(progs))
	for _, p := range progs {
		ip := domain.InstalledProgram{
			ID: p.ID, Name: p.Name, Publisher: p.Publisher, Version: p.Version, Size: p.SizeKB * 1024,
			Location: p.InstallLocation, DaysIdle: -1, Uninstaller: p.UninstallString != "",
		}
		for _, exe := range in.listExes(p.InstallLocation) {
			if t := runs[strings.ToUpper(exe)]; t.After(ip.LastRun) {
				ip.LastRun = t
			}
		}
		installed, _ := time.Parse("20060102", p.InstallDate)
		switch {
		case runsErr != nil:
			ip.Basis = "launch history unavailable (needs administrator)"
		case shared.MatchString(p.Name):
			ip.Basis = "shared component other programs rely on"
		case !ip.LastRun.IsZero():
			ip.DaysIdle = int(now.Sub(ip.LastRun).Hours() / 24)
			ip.Forgotten = now.Sub(ip.LastRun) > idleAfter
			ip.Basis = fmt.Sprintf("last started %d days ago", ip.DaysIdle)
		case p.InstallLocation == "":
			ip.Basis = "install folder unknown, so no launch record can be matched"
		case !installed.IsZero() && now.Sub(installed) > idleAfter:
			ip.Forgotten = true
			ip.Basis = fmt.Sprintf("no launch record; installed %d days ago", int(now.Sub(installed).Hours()/24))
		default:
			ip.Basis = "no launch record yet"
		}
		out = append(out, ip)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Forgotten != out[j].Forgotten {
			return out[i].Forgotten
		}
		return out[i].Size > out[j].Size
	})
	return out, nil
}

// Find looks a program up again by ID, so an uninstall never runs a command
// line that came from the window.
func (in *Inspector) Find(ctx context.Context, id string) (platform.Program, error) {
	progs, err := in.m.Programs(ctx)
	if err != nil {
		return platform.Program{}, fmt.Errorf("list programs: %w", err)
	}
	for _, p := range progs {
		if p.ID == id {
			return p, nil
		}
	}
	return platform.Program{}, fmt.Errorf("program %q not found", id)
}

// exesIn names the executables in a folder and its direct subfolders, which
// is where nearly every program keeps the exe users start.
func exesIn(dir string) []string {
	if dir == "" {
		return nil
	}
	dir = strings.Trim(dir, `"`)
	var out []string
	add := func(d string) []os.DirEntry {
		entries, _ := os.ReadDir(d)
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".exe") {
				out = append(out, e.Name())
			}
		}
		return entries
	}
	for _, e := range add(dir) {
		if e.IsDir() && e.Type()&os.ModeSymlink == 0 {
			add(filepath.Join(dir, e.Name()))
		}
	}
	return out
}
