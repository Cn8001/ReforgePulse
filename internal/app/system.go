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

// LiveReader answers the current load. It is polled every second, so it
// runs outside the task runner to keep the status bar quiet.
type LiveReader interface {
	Live(ctx context.Context) (domain.Live, error)
}

type SystemService struct {
	reader OverviewReader
	live   LiveReader
	runner TaskRunner
}

func NewSystemService(r OverviewReader, live LiveReader, runner TaskRunner) *SystemService {
	return &SystemService{reader: r, live: live, runner: runner}
}

func (s *SystemService) GetLive(ctx context.Context) (domain.Live, error) {
	l, err := s.live.Live(ctx)
	if err != nil {
		return domain.Live{}, fmt.Errorf("get live load: %w", err)
	}
	return l, nil
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
