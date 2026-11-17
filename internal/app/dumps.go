package app

import (
	"context"
	"fmt"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type DumpReader interface {
	List(ctx context.Context) ([]domain.Dump, error)
	Analyze(ctx context.Context, name string) (domain.DumpAnalysis, error)
	HasDebugger() bool
}

// AnalysisStore keeps finished dump analyses.
type AnalysisStore interface {
	Save(ctx context.Context, name string, dumped time.Time, a domain.DumpAnalysis) error
	Get(ctx context.Context, name string, dumped time.Time) (*domain.DumpAnalysis, error)
	All(ctx context.Context) ([]domain.SavedAnalysis, error)
	Delete(ctx context.Context, name string, dumped time.Time) error
}

// DumpService is the Minidumps tool. Analyses are saved, so one that took
// minutes is still there after leaving the page or restarting.
type DumpService struct {
	r      DumpReader
	runner TaskRunner
	saved  AnalysisStore
}

func NewDumpService(r DumpReader, runner TaskRunner, saved AnalysisStore) *DumpService {
	return &DumpService{r: r, runner: runner, saved: saved}
}

// List returns the dumps, each with its saved analysis when it has one.
func (s *DumpService) List(ctx context.Context) ([]domain.Dump, error) {
	ds, err := s.r.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dumps: %w", err)
	}
	for i := range ds {
		if ds[i].Analysis, err = s.saved.Get(ctx, ds[i].Name, ds[i].Time); err != nil {
			return nil, err
		}
	}
	return ds, nil
}

// Analyses lists every saved analysis, for Reports; the dump file may be
// gone by now, the analysis stays until deleted.
func (s *DumpService) Analyses(ctx context.Context) ([]domain.SavedAnalysis, error) {
	return s.saved.All(ctx)
}

func (s *DumpService) DeleteAnalysis(ctx context.Context, name string, dumped time.Time) error {
	return s.saved.Delete(ctx, name, dumped)
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
	a.At = time.Now()
	ds, err := s.r.List(ctx)
	if err != nil {
		return a, fmt.Errorf("find dump %s: %w", name, err)
	}
	for _, d := range ds {
		if d.Name == name {
			if err := s.saved.Save(ctx, name, d.Time, a); err != nil {
				return a, err
			}
		}
	}
	return a, nil
}
