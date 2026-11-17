package app

import (
	"context"
	"fmt"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/export"
	"github.com/cn8001/reforgepulse/internal/recorder"
)

type ExportDeps struct {
	Scans interface {
		Report(context.Context, string) (domain.ScanResult, error)
	}
	Analyses interface {
		Analyses(context.Context) ([]domain.SavedAnalysis, error)
	}
	Recordings interface {
		Recording(context.Context, string) (recorder.Recording, error)
	}
	Checks interface {
		Results(context.Context) ([]domain.CheckResult, error)
	}
	Write func(path string, data []byte) error
}

// ExportService saves a report as a Markdown file where the user chose,
// in the save dialog the window showed.
type ExportService struct {
	d ExportDeps
}

func NewExportService(d ExportDeps) *ExportService { return &ExportService{d: d} }

var checkNames = map[string]string{"sfc": "System files", "dism": "Windows image", "chkdsk": "System drive", "memtest": "Memory"}

func (s *ExportService) Scan(ctx context.Context, id, path string) error {
	r, err := s.d.Scans.Report(ctx, id)
	if err != nil {
		return err
	}
	return s.write(path, export.Scan(r))
}

func (s *ExportService) Analysis(ctx context.Context, name string, dumped time.Time, path string) error {
	all, err := s.d.Analyses.Analyses(ctx)
	if err != nil {
		return err
	}
	for _, a := range all {
		if a.Name == name && a.Dumped.Equal(dumped) {
			return s.write(path, export.Analysis(a))
		}
	}
	return fmt.Errorf("no saved analysis of %s", name)
}

func (s *ExportService) Recording(ctx context.Context, id, path string) error {
	r, err := s.d.Recordings.Recording(ctx, id)
	if err != nil {
		return err
	}
	return s.write(path, export.Recording(r))
}

func (s *ExportService) Check(ctx context.Context, id, path string) error {
	all, err := s.d.Checks.Results(ctx)
	if err != nil {
		return err
	}
	for _, c := range all {
		if c.ID == id {
			return s.write(path, export.Check(c, checkNames[c.Name]))
		}
	}
	return fmt.Errorf("no saved check result %s", id)
}

func (s *ExportService) write(path, md string) error {
	if path == "" {
		return fmt.Errorf("no file chosen")
	}
	if err := s.d.Write(path, []byte(md)); err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	return nil
}
