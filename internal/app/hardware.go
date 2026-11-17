package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type CoreLister interface {
	List(ctx context.Context) ([]domain.CoreInfo, error)
}

type CoreLoads interface {
	CoreLoads(ctx context.Context) ([]platform.CoreLoad, error)
}

type CoreTester interface {
	Run(ctx context.Context, cpus []int, perCore time.Duration, report func(float64, string)) (domain.CoreTest, error)
}

// HardwareDeps are the hardware service's dependencies.
type HardwareDeps struct {
	Facts  HardwareReader
	Cores  CoreLister
	Loads  CoreLoads
	Tester CoreTester
	Runner TaskRunner
	Saved  CheckStore // core tests are kept with Windows' health checks
	Now    func() time.Time
}

// HardwareService shows what the PC is made of, each core live, and runs
// the per-core test.
type HardwareService struct{ d HardwareDeps }

func NewHardwareService(d HardwareDeps) *HardwareService { return &HardwareService{d: d} }

func (s *HardwareService) Facts(ctx context.Context) (platform.HardwareFacts, error) {
	var f platform.HardwareFacts
	err := s.d.Runner.Do(ctx, "Reading hardware", func(ctx context.Context, _ tasks.Report) error {
		var err error
		f, err = s.d.Facts.HardwareFacts(ctx)
		return err
	})
	if err != nil {
		return f, fmt.Errorf("read hardware: %w", err)
	}
	return f, nil
}

func (s *HardwareService) Cores(ctx context.Context) ([]domain.CoreInfo, error) {
	cs, err := s.d.Cores.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("read cores: %w", err)
	}
	return cs, nil
}

// Loads is polled every second for the live core view, outside the runner.
func (s *HardwareService) Loads(ctx context.Context) ([]platform.CoreLoad, error) {
	ls, err := s.d.Loads.CoreLoads(ctx)
	if err != nil {
		return nil, fmt.Errorf("read core loads: %w", err)
	}
	return ls, nil
}

// TestCores stresses the chosen processors one at a time. It can crash an
// unstable PC, which the window says before it starts.
func (s *HardwareService) TestCores(ctx context.Context, cpus []int, seconds int) (domain.CoreTest, error) {
	if seconds < 5 || seconds > 600 {
		return domain.CoreTest{}, fmt.Errorf("%d seconds per processor is outside 5 to 600", seconds)
	}
	var t domain.CoreTest
	err := s.d.Runner.Do(ctx, "Testing cores", func(ctx context.Context, report tasks.Report) error {
		var err error
		t, err = s.d.Tester.Run(ctx, cpus, time.Duration(seconds)*time.Second, report)
		return err
	})
	if err != nil {
		return t, fmt.Errorf("core test: %w", err)
	}
	summary, code := coreTestSummary(t)
	if _, err := s.d.Saved.Save(ctx, domain.CheckResult{Name: "coretest", At: s.d.Now(), ExitCode: code, Summary: summary}); err != nil {
		return t, fmt.Errorf("save core test: %w", err)
	}
	return t, nil
}

// coreTestSummary is the verdict kept for Reports: what ran, and only the
// processors worth a look. Exit code 1 means one disagreed with the others
// or Windows logged a hardware error during the test.
func coreTestSummary(t domain.CoreTest) (string, int) {
	lines := []string{fmt.Sprintf("Tested %d processors for %d seconds each.", len(t.Results), t.Seconds)}
	code := 0
	for _, r := range t.Results {
		switch {
		case r.Skipped != "":
			lines = append(lines, fmt.Sprintf("Processor %d was skipped: %s.", r.Index, r.Skipped))
		case r.Errors > 0:
			lines = append(lines, fmt.Sprintf("Processor %d disagreed with the others in %d of %d rounds.", r.Index, r.Errors, r.Rounds))
			code = 1
		}
	}
	if t.NewWHEA > 0 {
		lines = append(lines, fmt.Sprintf("Windows logged %d hardware errors during the test.", t.NewWHEA))
		code = 1
	}
	if code == 0 {
		lines = append(lines, "Every processor gave the same results as the others.")
	}
	return strings.Join(lines, "\n"), code
}
