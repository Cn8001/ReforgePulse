package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

// UpdateSource finds, fetches and starts a newer release.
type UpdateSource interface {
	Check(ctx context.Context) (*domain.Release, error)                      // nil when this is the latest
	Download(ctx context.Context, progress func(written, total int64)) error // fetches and verifies what Check found
	Restart(ctx context.Context) error
}

// UpdatesUnavailable is the source for a build that cannot update, such as
// a development build: every call answers why.
func UpdatesUnavailable(why error) UpdateSource { return noUpdates{why} }

type noUpdates struct{ why error }

func (n noUpdates) Check(context.Context) (*domain.Release, error)     { return nil, n.why }
func (n noUpdates) Download(context.Context, func(int64, int64)) error { return n.why }
func (n noUpdates) Restart(context.Context) error                      { return n.why }

// SettingStore keeps the user's choices as text.
type SettingStore interface {
	Get(ctx context.Context, key, def string) (string, error)
	Set(ctx context.Context, key, value string) error
}

type UpdateDeps struct {
	Source   UpdateSource
	Runner   TaskRunner
	Journal  Recorder
	Settings SettingStore
	Version  string
}

const checkOnStartKey = "updates.check_on_start"

// UpdateService updates ReforgePulse itself. It looks for a new version
// when the app starts, unless the user turned that off, and when asked from
// Settings; it downloads and restarts only when the user says so. The
// download is logged like any other change.
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

// OnStart is the quiet check when the window opens: no task in the status
// bar, and nil when the user turned it off or nothing is newer.
func (s *UpdateService) OnStart(ctx context.Context) (*domain.Release, error) {
	on, err := s.CheckOnStart(ctx)
	if err != nil || !on {
		return nil, err
	}
	rel, err := s.d.Source.Check(ctx)
	if err != nil {
		return nil, fmt.Errorf("check for updates: %w", err)
	}
	return rel, nil
}

// CheckOnStart reports whether OnStart looks for updates; it does unless
// turned off.
func (s *UpdateService) CheckOnStart(ctx context.Context) (bool, error) {
	v, err := s.d.Settings.Get(ctx, checkOnStartKey, "on")
	if err != nil {
		return false, fmt.Errorf("read update setting: %w", err)
	}
	return v != "off", nil
}

func (s *UpdateService) SetCheckOnStart(ctx context.Context, on bool) error {
	v := "off"
	if on {
		v = "on"
	}
	if err := s.d.Settings.Set(ctx, checkOnStartKey, v); err != nil {
		return fmt.Errorf("save update setting: %w", err)
	}
	return nil
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
		return s.d.Source.Download(ctx, downloadProgress(report))
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

// downloadProgress turns byte counts into task progress. The source calls it
// every 64 KB, so it reports only when the whole percent changes, keeping
// the window to about a hundred updates per download.
func downloadProgress(report tasks.Report) func(written, total int64) {
	last := -1
	return func(written, total int64) {
		if total <= 0 {
			return // the size is unknown, so the bar keeps sliding
		}
		pct := int(written * 100 / total)
		if pct == last {
			return
		}
		last = pct
		report(float64(written)/float64(total), fmt.Sprintf("%.1f of %.1f MB", float64(written)/(1<<20), float64(total)/(1<<20)))
	}
}

// Restart closes ReforgePulse, swaps the new version in and starts it.
func (s *UpdateService) Restart(ctx context.Context) error {
	if err := s.d.Source.Restart(ctx); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	return nil
}
