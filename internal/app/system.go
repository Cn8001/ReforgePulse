// Package app holds the services the window and the MCP server call. It
// knows nothing of Wails: each service depends only on a small interface.
package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type OverviewReader interface {
	Overview(ctx context.Context) (domain.Overview, error)
}

type SystemService struct {
	reader OverviewReader
}

func NewSystemService(r OverviewReader) *SystemService {
	return &SystemService{reader: r}
}

func (s *SystemService) GetOverview(ctx context.Context) (domain.Overview, error) {
	o, err := s.reader.Overview(ctx)
	if err != nil {
		return domain.Overview{}, fmt.Errorf("get overview: %w", err)
	}
	return o, nil
}
