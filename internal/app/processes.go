package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type ProcessLister interface {
	List(ctx context.Context) ([]domain.Process, error)
}

type ProcessKiller interface {
	Kill(ctx context.Context, pid uint32) error
}

// protected are processes whose end crashes or locks Windows; End task
// refuses them whatever the caller asks.
var protected = map[string]bool{
	"system": true, "registry": true, "smss.exe": true, "csrss.exe": true, "wininit.exe": true,
	"winlogon.exe": true, "services.exe": true, "lsass.exe": true, "svchost.exe": true,
	"dwm.exe": true, "fontdrvhost.exe": true, "memory compression": true, "msmpeng.exe": true,
}

// ProcessService feeds the Processes page. Listing is polled, so it runs
// outside the task runner like the live load.
type ProcessDeps struct {
	Lister  ProcessLister
	Killer  ProcessKiller
	Journal Recorder
	Self    uint32 // our own PID: End task refuses it, and it and what it started are marked ours
}

type ProcessService struct {
	d ProcessDeps
}

func NewProcessService(d ProcessDeps) *ProcessService { return &ProcessService{d: d} }

// End ends a process after the user confirmed it. It cannot be undone, and
// the change log says so.
func (s *ProcessService) End(ctx context.Context, pid uint32, name string) error {
	if pid <= 4 || pid == s.d.Self || protected[strings.ToLower(name)] {
		return fmt.Errorf("%s is a Windows process that must keep running", name)
	}
	id, err := s.d.Journal.Begin(ctx, domain.Change{
		Action: "end-task", Summary: fmt.Sprintf("End task: %s (PID %d)", name, pid), Target: name, Before: "running",
	})
	if err != nil {
		return fmt.Errorf("log end task: %w", err)
	}
	err = s.d.Killer.Kill(ctx, pid)
	if ferr := s.d.Journal.Finish(ctx, id, "ended", err); ferr != nil && err == nil {
		err = ferr
	}
	if err != nil {
		return fmt.Errorf("end task: %w", err)
	}
	return nil
}

// List returns the running processes with this window's own marked, by
// PID: the window and every process under it, such as the hardware probe.
func (s *ProcessService) List(ctx context.Context) ([]domain.Process, error) {
	ps, err := s.d.Lister.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}
	parent := make(map[uint32]uint32, len(ps))
	for _, p := range ps {
		parent[p.PID] = p.ParentPID
	}
	for i := range ps {
		ps[i].Ours = s.descends(ps[i].PID, parent)
	}
	return ps, nil
}

// descends follows parents up to our PID. The walk is bounded: a reused
// PID can make the parent chain loop.
func (s *ProcessService) descends(pid uint32, parent map[uint32]uint32) bool {
	for range 64 {
		if pid == s.d.Self {
			return true
		}
		next, ok := parent[pid]
		if !ok || next == pid || next == 0 {
			return false
		}
		pid = next
	}
	return false
}
