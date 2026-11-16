package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type ProcessLister interface {
	List(ctx context.Context) ([]domain.Process, error)
}

// ProcessService feeds the Processes page. It is polled, so it runs outside
// the task runner like the live load.
type ProcessService struct {
	lister ProcessLister
}

func NewProcessService(l ProcessLister) *ProcessService {
	return &ProcessService{lister: l}
}

func (s *ProcessService) List(ctx context.Context) ([]domain.Process, error) {
	ps, err := s.lister.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}
	return ps, nil
}
