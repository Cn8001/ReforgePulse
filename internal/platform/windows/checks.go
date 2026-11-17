//go:build windows

package windows

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

var _ platform.Checks = System{}

// checkCommands is the whole allowlist; nothing from the window reaches a
// command line.
func checkCommands() map[string][]string {
	drive := os.Getenv("SystemDrive")
	if drive == "" {
		drive = "C:"
	}
	return map[string][]string{
		"sfc":    {"sfc.exe", "/verifyonly"},
		"dism":   {"dism.exe", "/Online", "/Cleanup-Image", "/ScanHealth", "/English"},
		"chkdsk": {"chkdsk.exe", drive, "/scan"},
	}
}

func (System) RunCheck(ctx context.Context, name string) (int, string, error) {
	if name == "memtest" {
		// It asks the user when to restart; nothing to wait for.
		if err := exec.Command("mdsched.exe").Start(); err != nil {
			return -1, "", fmt.Errorf("open Windows Memory Diagnostic: %w", err)
		}
		return 0, "Windows Memory Diagnostic is open. Choose to restart now or at the next start; the result appears in Events and the next scan.", nil
	}
	args, ok := checkCommands()[name]
	if !ok {
		return -1, "", fmt.Errorf("unknown check %q", name)
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code, err = ee.ExitCode(), nil // a non-zero exit is a result, not a failure to run
	}
	if err != nil {
		return -1, "", fmt.Errorf("run %s: %w", name, err)
	}
	return code, decode(out), nil
}

// decode reads sfc's UTF-16 output as well as the others' 8-bit output,
// and keeps only the last lines, where the verdict is.
func decode(b []byte) string {
	var s string
	if len(b) >= 4 && b[1] == 0 && b[3] == 0 {
		u := make([]uint16, len(b)/2)
		for i := range u {
			u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
		}
		s = string(utf16.Decode(u))
	} else {
		s = string(b)
	}
	s = strings.ReplaceAll(s, "\r", "\n")
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasSuffix(l, "%") && !strings.Contains(l, "complete.") {
			lines = append(lines, l) // progress lines say nothing once it is done
		}
	}
	if len(lines) > 30 {
		lines = lines[len(lines)-30:]
	}
	return strings.Join(lines, "\n")
}

// CheckLog is where Windows writes a check's full log; CHKDSK's online scan
// writes only to the event log, so it has none.
func (System) CheckLog(name string) string {
	win := os.Getenv("SystemRoot")
	switch name {
	case "sfc":
		return win + `\Logs\CBS\CBS.log`
	case "dism":
		return win + `\Logs\DISM\dism.log`
	}
	return ""
}

// MemoryTests are Windows Memory Diagnostic's verdicts, which it writes to
// the System log after the restart it tests in.
func (s System) MemoryTests(ctx context.Context, since time.Time) ([]domain.CheckResult, error) {
	// 1101 to 1103 on Windows 10 and 11, 1201 and 1202 before.
	events, err := s.Events(ctx, "System", "Microsoft-Windows-MemoryDiagnostics-Results", []uint32{1101, 1102, 1103, 1201, 1202}, since, 100)
	if err != nil {
		return nil, fmt.Errorf("read memory test results: %w", err)
	}
	out := make([]domain.CheckResult, 0, len(events))
	for _, e := range events {
		r := domain.CheckResult{
			ID: fmt.Sprintf("memtest-%d", e.Time.UnixNano()), Name: "memtest", At: e.Time, FromWindows: true,
			Summary: "Windows Memory Diagnostic found no errors.",
		}
		switch e.ID {
		case 1102, 1202:
			r.ExitCode, r.Summary = 1, "Windows Memory Diagnostic found errors. Test each memory stick on its own to find which one."
		case 1103:
			r.ExitCode, r.Summary = 2, "The memory test was cancelled before it finished, so it says nothing about the memory."
		}
		out = append(out, r)
	}
	return out, nil
}
