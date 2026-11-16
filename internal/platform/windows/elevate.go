//go:build windows

package windows

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var _ platform.Privileges = System{}

func (System) Elevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// Relaunch starts this executable again through the UAC prompt. The caller
// exits on success; the elevated copy takes over.
func (System) Relaunch(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}
	cwd, _ := os.Getwd()
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = syscall.EscapeArg(a)
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exe)
	params, _ := windows.UTF16PtrFromString(strings.Join(quoted, " "))
	dir, _ := windows.UTF16PtrFromString(cwd)
	err = windows.ShellExecute(0, verb, file, params, dir, windows.SW_NORMAL)
	if errors.Is(err, windows.ERROR_CANCELLED) {
		return platform.ErrDenied
	}
	if err != nil {
		return fmt.Errorf("relaunch elevated: %w", err)
	}
	return nil
}
