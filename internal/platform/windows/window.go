//go:build windows

package windows

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

// windowMark is a named mutex the window holds while it runs. It names
// nothing else; its existence is the signal.
const windowMark = `Local\ReforgePulse.Window`

// MarkWindow creates the mark; release removes it, as does the window's
// process ending in any way, crash included.
func (System) MarkWindow() (release func(), err error) {
	name, _ := windows.UTF16PtrFromString(windowMark)
	h, err := windows.CreateMutex(nil, false, name)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return func() {}, fmt.Errorf("create window mark: %w", err)
	}
	return func() { windows.CloseHandle(h) }, nil
}

// WindowOpen reports whether a window holds the mark. The window runs
// elevated and this may not: being refused access still means it is there.
func (System) WindowOpen() bool {
	name, _ := windows.UTF16PtrFromString(windowMark)
	h, err := windows.OpenMutex(windows.SYNCHRONIZE, false, name)
	if err == nil {
		windows.CloseHandle(h)
		return true
	}
	return errors.Is(err, windows.ERROR_ACCESS_DENIED)
}

// Alive reports whether pid is a process still running.
func (System) Alive(pid uint32) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if windows.GetExitCodeProcess(h, &code) != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}
