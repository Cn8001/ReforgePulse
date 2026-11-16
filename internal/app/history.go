package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type ChangeLog interface {
	Log(ctx context.Context) ([]domain.Change, error)
}

// Reverter undoes one kind of change, writing its own journal entry.
type Reverter interface {
	Revert(ctx context.Context, c domain.Change) error
}

// HistoryService shows what ReforgePulse has changed on this PC and undoes
// a change when its kind knows how.
type HistoryService struct {
	log       ChangeLog
	reverters map[string]Reverter // by Change.Action
}

func NewHistoryService(l ChangeLog, reverters map[string]Reverter) *HistoryService {
	return &HistoryService{log: l, reverters: reverters}
}

// Revert undoes an applied, reversible change.
func (s *HistoryService) Revert(ctx context.Context, id string) error {
	changes, err := s.log.Log(ctx)
	if err != nil {
		return fmt.Errorf("read change log: %w", err)
	}
	for _, c := range changes {
		if c.ID != id {
			continue
		}
		r, ok := s.reverters[c.Action]
		if !c.Reversible || c.Status != domain.Applied || !ok {
			return fmt.Errorf("change %s cannot be reverted", id)
		}
		return r.Revert(ctx, c)
	}
	return fmt.Errorf("change %s not found", id)
}

func (s *HistoryService) List(ctx context.Context) ([]domain.Change, error) {
	changes, err := s.log.Log(ctx)
	if err != nil {
		return nil, fmt.Errorf("list changes: %w", err)
	}
	return changes, nil
}
