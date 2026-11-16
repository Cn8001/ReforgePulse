package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/platform"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type DriverReader interface {
	ProblemDevices(ctx context.Context) ([]platform.ProblemDevice, error)
	DriverUpdates(ctx context.Context) ([]platform.DriverUpdate, error)
	OpenDriverPlace(place string) error
}

// DriverService shows devices without working drivers and what Windows
// Update offers. Installing stays with Windows Update and the makers.
type DriverService struct {
	r      DriverReader
	runner TaskRunner
}

func NewDriverService(r DriverReader, runner TaskRunner) *DriverService {
	return &DriverService{r: r, runner: runner}
}

func (s *DriverService) Problems(ctx context.Context) ([]platform.ProblemDevice, error) {
	ds, err := s.r.ProblemDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("read devices: %w", err)
	}
	return ds, nil
}

// Offers asks Windows Update, which can take a minute.
func (s *DriverService) Offers(ctx context.Context) ([]platform.DriverUpdate, error) {
	var out []platform.DriverUpdate
	err := s.runner.Do(ctx, "Asking Windows Update for drivers", func(ctx context.Context, _ tasks.Report) error {
		var err error
		out, err = s.r.DriverUpdates(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("search drivers: %w", err)
	}
	return out, nil
}

func (s *DriverService) Open(place string) error { return s.r.OpenDriverPlace(place) }
