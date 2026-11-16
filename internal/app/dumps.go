package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type DumpReader interface {
	List(ctx context.Context) ([]domain.Dump, error)
	Analyze(ctx context.Context, name string) (domain.DumpAnalysis, error)
	HasDebugger() bool
}

// DumpService is the Minidumps tool.
type DumpService struct {
	r      DumpReader
	runner TaskRunner
}

func NewDumpService(r DumpReader, runner TaskRunner) *DumpService {
	return &DumpService{r: r, runner: runner}
}

func (s *DumpService) List(ctx context.Context) ([]domain.Dump, error) {
	ds, err := s.r.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dumps: %w", err)
	}
	return ds, nil
}

func (s *DumpService) HasDebugger() bool { return s.r.HasDebugger() }

// Analyze can take minutes the first time, while symbols download.
func (s *DumpService) Analyze(ctx context.Context, name string) (domain.DumpAnalysis, error) {
	var a domain.DumpAnalysis
	err := s.runner.Do(ctx, "Analysing "+name, func(ctx context.Context, _ tasks.Report) error {
		var err error
		a, err = s.r.Analyze(ctx, name)
		return err
	})
	if err != nil {
		return a, fmt.Errorf("analyse dump: %w", err)
	}
	return a, nil
}
