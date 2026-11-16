//go:build windows

// Package windows implements the platform interfaces with Win32 calls.
package windows

import (
	"context"
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/cn8001/reforgepulse/internal/platform"
)

// x/sys does not wrap these two.
var (
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetTickCount64       = kernel32.NewProc("GetTickCount64")
)

// System reads the overview facts. It holds no state, so the zero value is ready.
type System struct{}

var _ platform.Source = System{}

// OSName names the release by build number: the registry's ProductName
// still says "Windows 10" on Windows 11.
func (System) OSName(ctx context.Context) (string, error) {
	v := windows.RtlGetVersion()
	name := "Windows 10"
	if v.MajorVersion == 10 && v.BuildNumber >= 22000 {
		name = "Windows 11"
	} else if v.MajorVersion != 10 {
		name = fmt.Sprintf("Windows %d.%d", v.MajorVersion, v.MinorVersion)
	}
	return fmt.Sprintf("%s (build %d)", name, v.BuildNumber), nil
}

func (System) CPU(ctx context.Context) (string, int, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err != nil {
		return "", runtime.NumCPU(), fmt.Errorf("open cpu key: %w", err)
	}
	defer k.Close()
	name, _, err := k.GetStringValue("ProcessorNameString")
	if err != nil {
		return "", runtime.NumCPU(), fmt.Errorf("read cpu name: %w", err)
	}
	return name, runtime.NumCPU(), nil
}

// memoryStatusEx mirrors MEMORYSTATUSEX; Length must be set before the call.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func (System) Memory(ctx context.Context) (uint64, uint64, error) {
	m := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	if ok, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m))); ok == 0 {
		return 0, 0, fmt.Errorf("GlobalMemoryStatusEx: %w", err)
	}
	return m.TotalPhys, m.AvailPhys, nil
}

func (System) Uptime(ctx context.Context) (time.Duration, error) {
	if err := procGetTickCount64.Find(); err != nil {
		return 0, fmt.Errorf("GetTickCount64: %w", err)
	}
	ms, _, _ := procGetTickCount64.Call()
	return time.Duration(ms) * time.Millisecond, nil
}

// Volumes lists fixed drives only: removable and network drives are out of scope.
func (System) Volumes(ctx context.Context) ([]platform.Volume, error) {
	buf := make([]uint16, 254) // 26 letters of "X:\\\x00" plus the final NUL fit easily
	n, err := windows.GetLogicalDriveStrings(uint32(len(buf)), &buf[0])
	if err != nil {
		return nil, fmt.Errorf("GetLogicalDriveStrings: %w", err)
	}
	var vols []platform.Volume
	for _, root := range splitNull(buf[:n]) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		p, _ := windows.UTF16PtrFromString(root)
		if windows.GetDriveType(p) != windows.DRIVE_FIXED {
			continue
		}
		var free, total, totalFree uint64
		if err := windows.GetDiskFreeSpaceEx(p, &free, &total, &totalFree); err != nil {
			continue // an unreadable drive is left out, not fatal
		}
		vols = append(vols, platform.Volume{Root: root, Total: total, Free: totalFree})
	}
	return vols, nil
}

// splitNull splits a double-NUL-terminated list of UTF-16 strings.
func splitNull(b []uint16) []string {
	var out []string
	start := 0
	for i, c := range b {
		if c == 0 {
			if i > start {
				out = append(out, windows.UTF16ToString(b[start:i]))
			}
			start = i + 1
		}
	}
	return out
}
