//go:build windows

package windows

import (
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

// The graphics kernel answers these for every vendor whose driver fills
// them in (WDDM 2.4 and later); Task Manager reads its GPU temperature the
// same way, so nothing has to be installed.
var (
	procEnumAdapters2    = gdi32.NewProc("D3DKMTEnumAdapters2")
	procQueryAdapterInfo = gdi32.NewProc("D3DKMTQueryAdapterInfo")
	procCloseAdapter     = gdi32.NewProc("D3DKMTCloseAdapter")
	errNoAdapterPerfData = errors.New("the graphics driver does not report temperature, fan or power")
)

const (
	adapterInfoSize           = 20          // D3DKMT_ADAPTERINFO: handle, LUID, sources, a BOOL
	adapterRegistryInfoLen    = 4 * 260 * 2 // D3DKMT_ADAPTERREGISTRYINFO: four MAX_PATH wide strings
	kmtqaiAdapterRegistryInfo = 8
	kmtqaiAdapterPerfData     = 62
	kmtqaiAdapterPerfDataCaps = 63
)

type d3dkmtEnumAdapters2 struct {
	Count    uint32
	Adapters unsafe.Pointer
}

type d3dkmtQueryAdapterInfo struct {
	Adapter uint32
	Type    int32
	Data    unsafe.Pointer
	Size    uint32
}

// adapterSensors reads temperature, fan and power for each graphics adapter
// whose driver reports them, named the way Device Manager names it. Adapters
// that report nothing, such as most integrated GPUs, are left out.
func adapterSensors() ([]platform.Sensor, error) {
	if err := procEnumAdapters2.Find(); err != nil {
		return nil, fmt.Errorf("no D3DKMTEnumAdapters2: %w", err)
	}
	var e d3dkmtEnumAdapters2
	if st, _, _ := procEnumAdapters2.Call(uintptr(unsafe.Pointer(&e))); st != 0 || e.Count == 0 {
		return nil, fmt.Errorf("list graphics adapters: status %#x", st)
	}
	buf := make([]byte, adapterInfoSize*int(e.Count))
	e.Adapters = unsafe.Pointer(&buf[0])
	st, _, _ := procEnumAdapters2.Call(uintptr(unsafe.Pointer(&e)))
	runtime.KeepAlive(buf)
	if st != 0 {
		return nil, fmt.Errorf("list graphics adapters: status %#x", st)
	}
	var out []platform.Sensor
	for i := 0; i < int(e.Count); i++ {
		h := binary.LittleEndian.Uint32(buf[i*adapterInfoSize:])
		out = append(out, readAdapter(h)...)
		closeArg := struct{ Adapter uint32 }{h}
		procCloseAdapter.Call(uintptr(unsafe.Pointer(&closeArg)))
	}
	if len(out) == 0 {
		return nil, errNoAdapterPerfData
	}
	return out, nil
}

func queryAdapter(h uint32, typ int32, data []byte) bool {
	q := d3dkmtQueryAdapterInfo{Adapter: h, Type: typ, Data: unsafe.Pointer(&data[0]), Size: uint32(len(data))}
	st, _, _ := procQueryAdapterInfo.Call(uintptr(unsafe.Pointer(&q)))
	runtime.KeepAlive(data)
	return st == 0
}

func readAdapter(h uint32) []platform.Sensor {
	reg := make([]byte, adapterRegistryInfoLen) // D3DKMT_ADAPTERREGISTRYINFO; the name comes first
	if !queryAdapter(h, kmtqaiAdapterRegistryInfo, reg) {
		return nil // Windows' own software renderer has no registry entry
	}
	name := windows.UTF16ToString(unsafe.Slice((*uint16)(unsafe.Pointer(&reg[0])), 260))
	perf := make([]byte, 64) // D3DKMT_ADAPTER_PERFDATA
	caps := make([]byte, 40) // D3DKMT_ADAPTER_PERFDATACAPS
	if name == "" || !queryAdapter(h, kmtqaiAdapterPerfData, perf) {
		return nil
	}
	queryAdapter(h, kmtqaiAdapterPerfDataCaps, caps)
	u32 := func(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }
	memHz := binary.LittleEndian.Uint64(perf[8:])
	fan, power, temp := u32(perf, 48), u32(perf, 52), u32(perf, 56)
	maxFan, tempMax := u32(caps, 24), u32(caps, 28)
	if temp == 0 && tempMax == 0 && power == 0 && maxFan == 0 {
		return nil
	}
	var out []platform.Sensor
	add := func(n, kind, unit string, v float64) {
		out = append(out, platform.Sensor{Source: "wddm", Group: name, Name: n, Kind: kind, Value: v, Unit: unit})
	}
	if temp > 0 {
		add("GPU temperature", "temp", "°C", float64(temp)/10) // tenths of a degree
	}
	if maxFan > 0 { // a fan standing still at idle still reads 0
		add("Fan", "fan", "RPM", float64(fan))
	}
	if power > 0 {
		add("Power use", "power", "% of limit", float64(power)/10) // tenths of a percent
	}
	if memHz > 0 {
		add("Memory clock", "clock", "MHz", float64(memHz)/1e6)
	}
	return out
}
