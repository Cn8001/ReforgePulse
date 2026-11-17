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
	Self    uint32 // our own PID, which End task also refuses
	Exe     string // our own executable: every process running it is ours
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

// List returns the running processes, ReforgePulse's own marked: the
// window, its hardware probe and any agent's MCP server all run its exe.
func (s *ProcessService) List(ctx context.Context) ([]domain.Process, error) {
	ps, err := s.d.Lister.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}
	for i := range ps {
		ps[i].Ours = ps[i].PID == s.d.Self || (s.d.Exe != "" && strings.EqualFold(ps[i].Path, s.d.Exe))
	}
	return ps, nil
}
