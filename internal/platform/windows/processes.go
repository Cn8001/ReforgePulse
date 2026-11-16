//go:build windows

package windows

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var procGetProcessMemoryInfo = kernel32.NewProc("K32GetProcessMemoryInfo")

var _ platform.Processes = System{}

// processMemoryCounters mirrors PROCESS_MEMORY_COUNTERS_EX.
type processMemoryCounters struct {
	Cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

func (System) Processes(ctx context.Context) ([]platform.ProcessInfo, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("process snapshot: %w", err)
	}
	defer windows.CloseHandle(snap)

	var out []platform.ProcessInfo
	e := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		p := platform.ProcessInfo{
			PID:       e.ProcessID,
			ParentPID: e.ParentProcessID,
			Threads:   e.Threads,
			Name:      windows.UTF16ToString(e.ExeFile[:]),
		}
		readProcess(&p)
		out = append(out, p)
	}
	return out, nil
}

// readProcess fills in what needs a handle. Limited query rights are enough
// for times and the path; memory also needs VM read, so it is tried first.
func readProcess(p *platform.ProcessInfo) {
	if p.PID == 0 {
		return // the System Idle Process cannot be opened
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ, false, p.PID)
	if err != nil {
		if h, err = windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, p.PID); err != nil {
			return
		}
	}
	defer windows.CloseHandle(h)
	p.Opened = true

	var created, exited, kernel, user windows.Filetime
	if windows.GetProcessTimes(h, &created, &exited, &kernel, &user) == nil {
		p.CPUTime = time.Duration((filetime(kernel) + filetime(user)) * 100) // FILETIME counts 100 ns units
	}

	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	if windows.QueryFullProcessImageName(h, 0, &buf[0], &size) == nil {
		p.Path = windows.UTF16ToString(buf[:size])
	}

	m := processMemoryCounters{Cb: uint32(unsafe.Sizeof(processMemoryCounters{}))}
	if ok, _, _ := procGetProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&m)), uintptr(m.Cb)); ok != 0 {
		p.WorkingSet, p.Private = uint64(m.WorkingSetSize), uint64(m.PrivateUsage)
	}
}

func filetime(f windows.Filetime) uint64 {
	return uint64(f.HighDateTime)<<32 | uint64(f.LowDateTime)
}
