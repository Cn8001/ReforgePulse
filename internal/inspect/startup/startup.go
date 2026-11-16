// Package startup lists startup programs with what they cost right now, by
// matching each one's executable against the running processes.
package startup

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

type ProcessLister interface {
	List(ctx context.Context) ([]domain.Process, error)
}

type Inspector struct {
	src   platform.Startup
	procs ProcessLister
}

func New(src platform.Startup, procs ProcessLister) *Inspector {
	return &Inspector{src: src, procs: procs}
}

// List returns startup apps, the ones using the most memory first.
func (in *Inspector) List(ctx context.Context) ([]domain.StartupApp, error) {
	entries, err := in.src.StartupEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list startup apps: %w", err)
	}
	procs, _ := in.procs.List(ctx) // without processes the list still works, just without costs
	byPath, byName := map[string]uint64{}, map[string]uint64{}
	for _, p := range procs {
		if p.Path != "" {
			byPath[strings.ToLower(p.Path)] += p.Private
		}
		byName[strings.ToLower(strings.TrimSuffix(p.Name, ".exe"))] += p.Private
	}

	out := make([]domain.StartupApp, 0, len(entries))
	for _, e := range entries {
		a := domain.StartupApp{ID: e.ID, Name: e.Name, Command: e.Command, Source: e.Source, Machine: e.Machine, Enabled: e.Enabled}
		exe := strings.ToLower(Executable(e.Command))
		mem, ok := byPath[exe]
		if !ok && strings.HasSuffix(exe, ".lnk") {
			// A shortcut's target is unknown without COM; its name usually matches the program.
			mem, ok = byName[strings.ToLower(e.Name)]
		}
		a.Running, a.Memory = ok, mem
		out = append(out, a)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Memory > out[j].Memory })
	return out, nil
}

// Executable takes the program path out of a startup command line, which
// may be quoted and followed by arguments.
func Executable(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if strings.HasPrefix(cmd, `"`) {
		if end := strings.Index(cmd[1:], `"`); end >= 0 {
			return filepath.Clean(cmd[1 : end+1])
		}
	}
	if i := strings.Index(strings.ToLower(cmd), ".exe"); i >= 0 {
		return filepath.Clean(cmd[:i+4])
	}
	if i := strings.IndexByte(cmd, ' '); i >= 0 {
		return cmd[:i]
	}
	return cmd
}
