package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type FeatureLister interface {
	List(ctx context.Context) ([]domain.WindowsFeature, error)
	Known(ctx context.Context, name string) (bool, error)
}

type FeatureSwitch interface {
	SetFeature(ctx context.Context, name string, enabled bool) (bool, error)
}

// FeatureService turns Windows optional features off and back on. DISM is
// slow, so both run as tasks.
type FeatureService struct {
	list    FeatureLister
	sw      FeatureSwitch
	runner  TaskRunner
	journal Recorder
}

// FeatureDeps are the feature service's dependencies.
type FeatureDeps struct {
	List    FeatureLister
	Switch  FeatureSwitch
	Runner  TaskRunner
	Journal Recorder
}

func NewFeatureService(d FeatureDeps) *FeatureService {
	return &FeatureService{list: d.List, sw: d.Switch, runner: d.Runner, journal: d.Journal}
}

func (s *FeatureService) List(ctx context.Context) ([]domain.WindowsFeature, error) {
	var out []domain.WindowsFeature
	err := s.runner.Do(ctx, "Reading Windows features", func(ctx context.Context, _ tasks.Report) error {
		var err error
		out, err = s.list.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	return out, nil
}

// SetEnabled turns a feature on or off and says whether Windows needs a
// restart to finish.
func (s *FeatureService) SetEnabled(ctx context.Context, name string, enabled bool) (bool, error) {
	before, after, verb := "on", "off", "Turn off"
	if enabled {
		before, after, verb = "off", "on", "Turn on"
	}
	return s.apply(ctx, domain.Change{
		Action: "feature", Summary: fmt.Sprintf("%s Windows feature %s", verb, name), Target: name,
		Before: before, After: after, Reversible: true,
	}, enabled)
}

func (s *FeatureService) Revert(ctx context.Context, c domain.Change) error {
	_, err := s.apply(ctx, domain.Change{
		Action: "feature", Summary: "Revert: " + c.Summary, Target: c.Target,
		Before: c.After, After: c.Before, Reversible: true, Reverts: c.ID,
	}, c.Before == "on")
	return err
}

func (s *FeatureService) apply(ctx context.Context, c domain.Change, enabled bool) (bool, error) {
	ok, err := s.list.Known(ctx, c.Target)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, fmt.Errorf("Windows has no feature named %q", c.Target)
	}
	var restart bool
	err = s.runner.Do(ctx, c.Summary, func(ctx context.Context, _ tasks.Report) error {
		id, err := s.journal.Begin(ctx, c)
		if err != nil {
			return fmt.Errorf("log feature change: %w", err)
		}
		restart, err = s.sw.SetFeature(ctx, c.Target, enabled)
		after := c.After
		if restart {
			after += " after a restart"
		}
		if ferr := s.journal.Finish(ctx, id, after, err); ferr != nil && err == nil {
			err = ferr
		}
		return err
	})
	if err != nil {
		return false, fmt.Errorf("change feature: %w", err)
	}
	return restart, nil
}
