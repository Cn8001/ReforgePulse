package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type EventGrouper interface {
	List(ctx context.Context, days int, warnings bool) ([]domain.EventGroup, error)
}

// EventService is the Events tool: Event Viewer, folded and explained.
type EventService struct {
	g      EventGrouper
	runner TaskRunner
}

func NewEventService(g EventGrouper, runner TaskRunner) *EventService {
	return &EventService{g: g, runner: runner}
}

func (s *EventService) List(ctx context.Context, days int, warnings bool) ([]domain.EventGroup, error) {
	if days < 1 || days > 90 {
		return nil, fmt.Errorf("%d days is outside 1 to 90", days)
	}
	var out []domain.EventGroup
	err := s.runner.Do(ctx, "Reading event logs", func(ctx context.Context, _ tasks.Report) error {
		var err error
		out, err = s.g.List(ctx, days, warnings)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read events: %w", err)
	}
	return out, nil
}
