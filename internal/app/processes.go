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
type ProcessService struct {
	lister  ProcessLister
	killer  ProcessKiller
	journal Recorder
	self    uint32 // our own PID, which End task also refuses
}

func NewProcessService(l ProcessLister, k ProcessKiller, j Recorder, self uint32) *ProcessService {
	return &ProcessService{lister: l, killer: k, journal: j, self: self}
}

// End ends a process after the user confirmed it. It cannot be undone, and
// the change log says so.
func (s *ProcessService) End(ctx context.Context, pid uint32, name string) error {
	if pid <= 4 || pid == s.self || protected[strings.ToLower(name)] {
		return fmt.Errorf("%s is a Windows process that must keep running", name)
	}
	id, err := s.journal.Begin(ctx, domain.Change{
		Action: "end-task", Summary: fmt.Sprintf("End task: %s (PID %d)", name, pid), Target: name, Before: "running",
	})
	if err != nil {
		return fmt.Errorf("log end task: %w", err)
	}
	err = s.killer.Kill(ctx, pid)
	if ferr := s.journal.Finish(ctx, id, "ended", err); ferr != nil && err == nil {
		err = ferr
	}
	if err != nil {
		return fmt.Errorf("end task: %w", err)
	}
	return nil
}

func (s *ProcessService) List(ctx context.Context) ([]domain.Process, error) {
	ps, err := s.lister.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}
	return ps, nil
}
