// Package app holds the services the window and the MCP server call. It
// knows nothing of Wails: each service depends only on a small interface.
package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type OverviewReader interface {
	Overview(ctx context.Context) (domain.Overview, error)
}

type SystemService struct {
	reader OverviewReader
	runner TaskRunner
}

func NewSystemService(r OverviewReader, runner TaskRunner) *SystemService {
	return &SystemService{reader: r, runner: runner}
}

func (s *SystemService) GetOverview(ctx context.Context) (domain.Overview, error) {
	var o domain.Overview
	err := s.runner.Do(ctx, "Reading system overview", func(ctx context.Context, _ tasks.Report) error {
		var err error
		o, err = s.reader.Overview(ctx)
		return err
	})
	if err != nil {
		return domain.Overview{}, fmt.Errorf("get overview: %w", err)
	}
	return o, nil
}
