package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

// UpdateSource finds, fetches and starts a newer release.
type UpdateSource interface {
	Check(ctx context.Context) (*domain.Release, error) // nil when this is the latest
	Download(ctx context.Context) error                 // fetches and verifies what Check found
	Restart(ctx context.Context) error
}

// UpdatesUnavailable is the source for a build that cannot update, such as
// a development build: every call answers why.
func UpdatesUnavailable(why error) UpdateSource { return noUpdates{why} }

type noUpdates struct{ why error }

func (n noUpdates) Check(context.Context) (*domain.Release, error) { return nil, n.why }
func (n noUpdates) Download(context.Context) error                 { return n.why }
func (n noUpdates) Restart(context.Context) error                  { return n.why }

type UpdateDeps struct {
	Source  UpdateSource
	Runner  TaskRunner
	Journal Recorder
	Version string
}

// UpdateService updates ReforgePulse itself, only when the user asks from
// Settings. The download is logged like any other change.
type UpdateService struct {
	d UpdateDeps
}

func NewUpdateService(d UpdateDeps) *UpdateService { return &UpdateService{d: d} }

// Check asks GitHub for a newer release.
func (s *UpdateService) Check(ctx context.Context) (*domain.Release, error) {
	var rel *domain.Release
	err := s.d.Runner.Do(ctx, "Checking for updates", func(ctx context.Context, report tasks.Report) error {
		var err error
		rel, err = s.d.Source.Check(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("check for updates: %w", err)
	}
	return rel, nil
}

// Download fetches the release Check found and verifies its checksum. It
// takes effect on Restart.
func (s *UpdateService) Download(ctx context.Context, version string) error {
	id, err := s.d.Journal.Begin(ctx, domain.Change{
		Action: "update", Summary: "Update ReforgePulse to " + version, Target: "reforgepulse.exe",
		Before: s.d.Version, After: version,
	})
	if err != nil {
		return fmt.Errorf("record update: %w", err)
	}
	err = s.d.Runner.Do(ctx, "Downloading ReforgePulse "+version, func(ctx context.Context, report tasks.Report) error {
		return s.d.Source.Download(ctx)
	})
	after := "downloaded and verified, installs on restart"
	if err != nil {
		after = ""
	}
	if ferr := s.d.Journal.Finish(ctx, id, after, err); ferr != nil && err == nil {
		err = ferr
	}
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	return nil
}

// Restart closes ReforgePulse, swaps the new version in and starts it.
func (s *UpdateService) Restart(ctx context.Context) error {
	if err := s.d.Source.Restart(ctx); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	return nil
}
