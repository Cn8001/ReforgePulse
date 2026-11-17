package app

import (
	"context"
	"fmt"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type ChangeLog interface {
	Log(ctx context.Context) ([]domain.Change, error)
	Delete(ctx context.Context, ids ...string) (int, error)
	DeleteBefore(ctx context.Context, t time.Time) (int, error)
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

// Delete removes one entry, and the entries that revert it, from the log.
// The PC is not touched, but a deleted change can no longer be reverted here.
func (s *HistoryService) Delete(ctx context.Context, id string) (int, error) {
	n, err := s.log.Delete(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("delete change: %w", err)
	}
	return n, nil
}

// DeleteBefore removes entries of changes made before t.
func (s *HistoryService) DeleteBefore(ctx context.Context, t time.Time) (int, error) {
	n, err := s.log.DeleteBefore(ctx, t)
	if err != nil {
		return 0, fmt.Errorf("delete changes: %w", err)
	}
	return n, nil
}

func (s *HistoryService) List(ctx context.Context) ([]domain.Change, error) {
	changes, err := s.log.Log(ctx)
	if err != nil {
		return nil, fmt.Errorf("list changes: %w", err)
	}
	return changes, nil
}
