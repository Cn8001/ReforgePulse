package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
)

// Recorder is the change journal. Every change is begun before the system
// is touched and finished after, with what it really did.
type Recorder interface {
	Begin(ctx context.Context, c domain.Change) (string, error)
	Finish(ctx context.Context, id, after string, err error) error
}

type StartupLister interface {
	List(ctx context.Context) ([]domain.StartupApp, error)
}

type StartupSwitch interface {
	SetStartupEnabled(ctx context.Context, id string, enabled bool) error
}

type StartupService struct {
	list    StartupLister
	sw      StartupSwitch
	journal Recorder
}

func NewStartupService(l StartupLister, sw StartupSwitch, j Recorder) *StartupService {
	return &StartupService{list: l, sw: sw, journal: j}
}

func (s *StartupService) List(ctx context.Context) ([]domain.StartupApp, error) {
	apps, err := s.list.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list startup apps: %w", err)
	}
	return apps, nil
}

// SetEnabled turns a startup app on or off, logged and reversible.
func (s *StartupService) SetEnabled(ctx context.Context, id, name string, enabled bool) error {
	verb, before, after := "Disable", "enabled", "disabled"
	if enabled {
		verb, before, after = "Enable", "disabled", "enabled"
	}
	return s.apply(ctx, domain.Change{
		Action: "startup", Summary: fmt.Sprintf("%s at startup: %s", verb, name),
		Target: id, Before: before, After: after, Reversible: true,
	}, enabled)
}

// Revert puts a startup app back the way the change found it.
func (s *StartupService) Revert(ctx context.Context, c domain.Change) error {
	enabled := c.Before == "enabled"
	return s.apply(ctx, domain.Change{
		Action: "startup", Summary: "Revert: " + c.Summary, Target: c.Target,
		Before: c.After, After: c.Before, Reversible: true, Reverts: c.ID,
	}, enabled)
}

func (s *StartupService) apply(ctx context.Context, c domain.Change, enabled bool) error {
	id, err := s.journal.Begin(ctx, c)
	if err != nil {
		return fmt.Errorf("log startup change: %w", err)
	}
	err = s.sw.SetStartupEnabled(ctx, c.Target, enabled)
	if ferr := s.journal.Finish(ctx, id, c.After, err); ferr != nil && err == nil {
		err = ferr
	}
	if err != nil {
		return fmt.Errorf("change startup app: %w", err)
	}
	return nil
}
