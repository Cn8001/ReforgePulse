//go:build windows

package windows

import (
	"context"
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var (
	powrprof                   = windows.NewLazySystemDLL("powrprof.dll")
	procPowerEnumerate         = powrprof.NewProc("PowerEnumerate")
	procPowerReadFriendlyName  = powrprof.NewProc("PowerReadFriendlyName")
	procPowerReadACValueIndex  = powrprof.NewProc("PowerReadACValueIndex")
	procPowerReadDCValueIndex  = powrprof.NewProc("PowerReadDCValueIndex")
	procPowerWriteACValueIndex = powrprof.NewProc("PowerWriteACValueIndex")
	procPowerWriteDCValueIndex = powrprof.NewProc("PowerWriteDCValueIndex")
	procPowerGetActiveScheme   = powrprof.NewProc("PowerGetActiveScheme")
	procPowerSetActiveScheme   = powrprof.NewProc("PowerSetActiveScheme")
)

// The processor subgroup and its maximum frequency settings, as powercfg
// names them SUB_PROCESSOR, PROCFREQMAX and PROCFREQMAX1.
var (
	subProcessor = mustGUID("{54533251-82be-4824-96c1-47b60b740d00}")
	procFreqMax  = mustGUID("{75b0ae3f-bce0-45a7-8c89-c9611c25e100}") // efficiency class 0
	procFreqMax1 = mustGUID("{75b0ae3f-bce0-45a7-8c89-c9611c25e101}") // efficiency class 1
)

const accessScheme = 16

var _ platform.Power = System{}

func mustGUID(s string) windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic(err) // a typo in the constants above, caught by the first run
	}
	return g
}

func (System) PowerPlans(ctx context.Context) ([]platform.PowerPlan, error) {
	var active *windows.GUID
	if r, _, _ := procPowerGetActiveScheme.Call(0, uintptr(unsafe.Pointer(&active))); r == 0 && active != nil {
		defer windows.LocalFree(windows.Handle(unsafe.Pointer(active)))
	}
	var out []platform.PowerPlan
	for i := uint32(0); ; i++ {
		var g windows.GUID
		size := uint32(unsafe.Sizeof(g))
		r, _, _ := procPowerEnumerate.Call(0, 0, 0, accessScheme, uintptr(i), uintptr(unsafe.Pointer(&g)), uintptr(unsafe.Pointer(&size)))
		if r == uintptr(windows.ERROR_NO_MORE_ITEMS) {
			break
		}
		if r != 0 {
			return out, fmt.Errorf("list power plans: error %d", r)
		}
		p := platform.PowerPlan{GUID: g.String(), Name: friendlyName(&g), Active: active != nil && *active == g}
		p.ECoreAC, p.ECoreDC = readCap(&g, &procFreqMax)
		p.PCoreAC, p.PCoreDC = readCap(&g, &procFreqMax1)
		out = append(out, p)
	}
	return out, nil
}

func (System) SetCPUCaps(ctx context.Context, p platform.PowerPlan) error {
	g, err := windows.GUIDFromString(p.GUID)
	if err != nil {
		return fmt.Errorf("plan %q: %w", p.GUID, err)
	}
	for _, w := range []struct {
		proc    *windows.LazyProc
		setting *windows.GUID
		value   uint32
	}{
		{procPowerWriteACValueIndex, &procFreqMax, p.ECoreAC}, {procPowerWriteDCValueIndex, &procFreqMax, p.ECoreDC},
		{procPowerWriteACValueIndex, &procFreqMax1, p.PCoreAC}, {procPowerWriteDCValueIndex, &procFreqMax1, p.PCoreDC},
	} {
		r, _, _ := w.proc.Call(0, uintptr(unsafe.Pointer(&g)), uintptr(unsafe.Pointer(&subProcessor)), uintptr(unsafe.Pointer(w.setting)), uintptr(w.value))
		if r != 0 {
			return fmt.Errorf("write cap on %s: %w", p.Name, windows.Errno(r))
		}
	}
	// Windows applies changed values only when the active plan is set again.
	var active *windows.GUID
	if r, _, _ := procPowerGetActiveScheme.Call(0, uintptr(unsafe.Pointer(&active))); r != 0 || active == nil {
		return errors.New("read active power plan")
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(active)))
	if r, _, _ := procPowerSetActiveScheme.Call(0, uintptr(unsafe.Pointer(active))); r != 0 {
		return fmt.Errorf("reapply active plan: %w", windows.Errno(r))
	}
	return nil
}

// readCap answers 0 (no cap) when a plan does not define the setting.
func readCap(scheme, setting *windows.GUID) (ac, dc uint32) {
	procPowerReadACValueIndex.Call(0, uintptr(unsafe.Pointer(scheme)), uintptr(unsafe.Pointer(&subProcessor)), uintptr(unsafe.Pointer(setting)), uintptr(unsafe.Pointer(&ac)))
	procPowerReadDCValueIndex.Call(0, uintptr(unsafe.Pointer(scheme)), uintptr(unsafe.Pointer(&subProcessor)), uintptr(unsafe.Pointer(setting)), uintptr(unsafe.Pointer(&dc)))
	return ac, dc
}

func friendlyName(g *windows.GUID) string {
	var size uint32
	procPowerReadFriendlyName.Call(0, uintptr(unsafe.Pointer(g)), 0, 0, 0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return g.String()
	}
	buf := make([]uint16, size/2+1)
	procPowerReadFriendlyName.Call(0, uintptr(unsafe.Pointer(g)), 0, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	return windows.UTF16ToString(buf)
}
