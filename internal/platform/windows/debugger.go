//go:build windows

package windows

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var _ platform.Debugger = System{}

// DebuggerPath finds cdb.exe from the Windows SDK or the WinDbg app.
func (System) DebuggerPath() string {
	candidates := []string{filepath.Join(os.Getenv("ProgramFiles(x86)"), `Windows Kits\10\Debuggers\x64\cdb.exe`)}
	if apps, _ := filepath.Glob(filepath.Join(os.Getenv("ProgramFiles"), `WindowsApps\Microsoft.WinDbg_*_x64__8wekyb3d8bbwe\amd64\cdb.exe`)); len(apps) > 0 {
		candidates = append(candidates, apps[len(apps)-1])
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// Analyze runs !analyze -v with symbols cached in the given folder and
// fetched from Microsoft's symbol server, the one network use, which the
// window asks about first.
func (s System) Analyze(ctx context.Context, dump, symbols string) (string, error) {
	cdb := s.DebuggerPath()
	if cdb == "" {
		return "", fmt.Errorf("no debugger installed (WinDbg or the Windows SDK debugging tools)")
	}
	if err := os.MkdirAll(symbols, 0o755); err != nil {
		return "", fmt.Errorf("create symbol cache: %w", err)
	}
	cmd := exec.CommandContext(ctx, cdb, "-z", dump,
		"-y", "srv*"+symbols+"*https://msdl.microsoft.com/download/symbols",
		"-c", ".symfix+ "+symbols+"; .reload; !analyze -v; q")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return "", fmt.Errorf("run debugger: %w", err)
	}
	return string(out), nil
}
