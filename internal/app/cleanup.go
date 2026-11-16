package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type Cleaner interface {
	Analyze(ctx context.Context) ([]domain.CleanupArea, error)
	Clean(ctx context.Context, ids []string) (domain.CleanupResult, error)
}

// CleanupService runs disk cleanup as tasks, so the status bar shows it and
// the user can cancel it.
type CleanupService struct {
	cleaner Cleaner
	runner  TaskRunner
}

func NewCleanupService(c Cleaner, runner TaskRunner) *CleanupService {
	return &CleanupService{cleaner: c, runner: runner}
}

func (s *CleanupService) Analyze(ctx context.Context) ([]domain.CleanupArea, error) {
	var areas []domain.CleanupArea
	err := s.runner.Do(ctx, "Analysing disk space", func(ctx context.Context, _ tasks.Report) error {
		var err error
		areas, err = s.cleaner.Analyze(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("analyse cleanup: %w", err)
	}
	return areas, nil
}

// Clean deletes what the user approved in the preview.
func (s *CleanupService) Clean(ctx context.Context, ids []string) (domain.CleanupResult, error) {
	var r domain.CleanupResult
	err := s.runner.Do(ctx, "Cleaning up", func(ctx context.Context, _ tasks.Report) error {
		var err error
		r, err = s.cleaner.Clean(ctx, ids)
		return err
	})
	if err != nil {
		return r, fmt.Errorf("clean up: %w", err)
	}
	return r, nil
}
