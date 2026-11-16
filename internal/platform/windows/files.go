//go:build windows

package windows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var procSHFileOperation = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

var _ platform.Files = System{}

func (System) LeftoverRoots() []platform.LeftoverRoot {
	var out []platform.LeftoverRoot
	for _, r := range []struct{ name, env string }{
		{"Program Files", "ProgramFiles"},
		{"Program Files (x86)", "ProgramFiles(x86)"},
		{"ProgramData", "ProgramData"},
		{"AppData (Roaming)", "APPDATA"},
		{"AppData (Local)", "LOCALAPPDATA"},
	} {
		if p := os.Getenv(r.env); p != "" {
			out = append(out, platform.LeftoverRoot{Name: r.name, Path: p})
		}
	}
	return out
}

// shFileOpStruct mirrors SHFILEOPSTRUCTW on 64-bit Windows.
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

const (
	foDelete          = 3
	fofSilent         = 0x4
	fofNoConfirmation = 0x10
	fofAllowUndo      = 0x40
	fofNoErrorUI      = 0x400
)

// Recycle moves paths to the Recycle Bin in one shell operation.
func (System) Recycle(ctx context.Context, paths []string) error {
	var from []uint16
	for _, p := range paths {
		if !filepath.IsAbs(p) || strings.ContainsRune(p, 0) {
			return fmt.Errorf("refusing path %q", p)
		}
		u, err := windows.UTF16FromString(p) // ends with one NUL
		if err != nil {
			return fmt.Errorf("encode %q: %w", p, err)
		}
		from = append(from, u...)
	}
	from = append(from, 0) // the list ends with a second NUL
	op := shFileOpStruct{wFunc: foDelete, pFrom: &from[0], fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI}
	if r, _, _ := procSHFileOperation.Call(uintptr(unsafe.Pointer(&op))); r != 0 {
		return fmt.Errorf("move to Recycle Bin: error 0x%x", r)
	}
	if op.fAnyOperationsAborted != 0 {
		return fmt.Errorf("move to Recycle Bin was aborted")
	}
	return nil
}
