package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type ChangeLog interface {
	Log(ctx context.Context) ([]domain.Change, error)
}

// HistoryService shows what ReforgePulse has changed on this PC.
type HistoryService struct {
	log ChangeLog
}

func NewHistoryService(l ChangeLog) *HistoryService {
	return &HistoryService{log: l}
}

func (s *HistoryService) List(ctx context.Context) ([]domain.Change, error) {
	changes, err := s.log.Log(ctx)
	if err != nil {
		return nil, fmt.Errorf("list changes: %w", err)
	}
	return changes, nil
}
