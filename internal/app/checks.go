package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type CheckRunner interface {
	RunCheck(ctx context.Context, name string) (int, string, error)
	CheckLog(name string) string // where Windows keeps the full log, or ""
	ShowInFolder(path string) error
}

// MemoryTestReader reads Windows Memory Diagnostic's verdicts, which only
// Windows' event log has.
type MemoryTestReader interface {
	MemoryTests(ctx context.Context, since time.Time) ([]domain.CheckResult, error)
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
	Memory MemoryTestReader
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
	r.Summary, r.Log = summarize(r.Output), s.d.Run.CheckLog(name)
	if unsaved[name] {
		return r, nil
	}
	saved, err := s.d.Saved.Save(ctx, r)
	if err != nil {
		return r, fmt.Errorf("save %s result: %w", name, err)
	}
	return saved, nil
}

// Results lists saved results and the memory tests Windows recorded in the
// last year, newest first.
func (s *CheckService) Results(ctx context.Context) ([]domain.CheckResult, error) {
	out, err := s.d.Saved.All(ctx)
	if err != nil {
		return nil, err
	}
	// Without the event log (no administrator rights) the saved results
	// still stand on their own.
	if mem, err := s.d.Memory.MemoryTests(ctx, s.d.Now().AddDate(-1, 0, 0)); err == nil {
		out = append(out, mem...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out, nil
}

// ShowLog opens Explorer at the full log Windows keeps for a check. It takes
// the check's name, not a path, so the window cannot point it anywhere else.
func (s *CheckService) ShowLog(name string) error {
	path := s.d.Run.CheckLog(name)
	if path == "" {
		return fmt.Errorf("%s keeps no log of its own", name)
	}
	return s.d.Run.ShowInFolder(path)
}

// DeleteResult forgets a saved result. Memory tests are Windows' own records
// and stay in its event log.
func (s *CheckService) DeleteResult(ctx context.Context, id string) error {
	if strings.HasPrefix(id, "memtest-") {
		return fmt.Errorf("memory test results are kept by Windows in its event log")
	}
	return s.d.Saved.Delete(ctx, id)
}

// maxSummary bounds what is kept of a check's verdict.
const maxSummary = 600

// summarize keeps the verdict: the last few lines that are not progress.
// SFC and DISM print a line per percent before it, in the user's language,
// so the rule is "no percent sign" rather than any wording.
func summarize(output string) string {
	var keep []string
	lines := strings.Split(strings.ReplaceAll(output, "\r", "\n"), "\n")
	for i := len(lines) - 1; i >= 0 && len(keep) < 4; i-- {
		l := strings.TrimSpace(lines[i])
		if l == "" || strings.Contains(l, "%") || strings.Trim(l, "=-.[] ") == "" {
			continue
		}
		keep = append([]string{l}, keep...)
	}
	s := strings.Join(keep, "\n")
	if len(s) > maxSummary {
		s = s[len(s)-maxSummary:]
	}
	return s
}
