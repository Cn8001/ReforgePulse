package app

import (
	"context"
	"fmt"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type CheckRunner interface {
	RunCheck(ctx context.Context, name string) (int, string, error)
}

// CheckStore keeps check results for Reports.
type CheckStore interface {
	Save(ctx context.Context, r domain.CheckResult) (domain.CheckResult, error)
	All(ctx context.Context) ([]domain.CheckResult, error)
	Delete(ctx context.Context, id string) error
}

type CheckDeps struct {
	Run    CheckRunner
	Runner TaskRunner
	Saved  CheckStore
	Now    func() time.Time
}

// CheckService runs Windows' read-only health checks as tasks; some take
// many minutes, and the status bar can cancel them. Results are saved: a
// check that took twenty minutes should not vanish with the page.
type CheckService struct {
	d CheckDeps
}

func NewCheckService(d CheckDeps) *CheckService { return &CheckService{d: d} }

// unsaved are checks with nothing to keep: Memory Diagnostic only schedules
// a test for the next restart, and the scan reads its result from the log.
var unsaved = map[string]bool{"memtest": true}

var checkTitles = map[string]string{
	"sfc": "Checking system files", "dism": "Checking the Windows image",
	"chkdsk": "Checking the system drive", "memtest": "Opening Memory Diagnostic",
}

func (s *CheckService) Run(ctx context.Context, name string) (domain.CheckResult, error) {
	title, ok := checkTitles[name]
	if !ok {
		return domain.CheckResult{}, fmt.Errorf("unknown check %q", name)
	}
	r := domain.CheckResult{Name: name}
	err := s.d.Runner.Do(ctx, title, func(ctx context.Context, _ tasks.Report) error {
		var err error
		r.ExitCode, r.Output, err = s.d.Run.RunCheck(ctx, name)
		return err
	})
	if err != nil {
		return r, fmt.Errorf("%s: %w", name, err)
	}
	r.At = s.d.Now()
	if unsaved[name] {
		return r, nil
	}
	saved, err := s.d.Saved.Save(ctx, r)
	if err != nil {
		return r, fmt.Errorf("save %s result: %w", name, err)
	}
	return saved, nil
}

// Results lists saved results, newest first.
func (s *CheckService) Results(ctx context.Context) ([]domain.CheckResult, error) {
	return s.d.Saved.All(ctx)
}

func (s *CheckService) DeleteResult(ctx context.Context, id string) error {
	return s.d.Saved.Delete(ctx, id)
}
