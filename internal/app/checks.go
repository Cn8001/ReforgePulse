package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/tasks"
)

type CheckRunner interface {
	RunCheck(ctx context.Context, name string) (int, string, error)
}

// CheckResult is what a Windows health check said.
type CheckResult struct {
	Name     string `json:"name"`
	ExitCode int    `json:"exitCode"`
	Output   string `json:"output"`
}

// CheckService runs Windows' read-only health checks as tasks; some take
// many minutes, and the status bar can cancel them.
type CheckService struct {
	run    CheckRunner
	runner TaskRunner
}

func NewCheckService(r CheckRunner, runner TaskRunner) *CheckService {
	return &CheckService{run: r, runner: runner}
}

var checkTitles = map[string]string{
	"sfc": "Checking system files", "dism": "Checking the Windows image",
	"chkdsk": "Checking the system drive", "memtest": "Opening Memory Diagnostic",
}

func (s *CheckService) Run(ctx context.Context, name string) (CheckResult, error) {
	title, ok := checkTitles[name]
	if !ok {
		return CheckResult{}, fmt.Errorf("unknown check %q", name)
	}
	r := CheckResult{Name: name}
	err := s.runner.Do(ctx, title, func(ctx context.Context, _ tasks.Report) error {
		var err error
		r.ExitCode, r.Output, err = s.run.RunCheck(ctx, name)
		return err
	})
	if err != nil {
		return r, fmt.Errorf("%s: %w", name, err)
	}
	return r, nil
}
