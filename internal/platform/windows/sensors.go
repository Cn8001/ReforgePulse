//go:build windows

package windows

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

// SensorReader reads NVIDIA GPUs through the driver's own nvml.dll, other
// GPUs through the graphics kernel, and everything else from HWiNFO's
// shared memory when HWiNFO publishes it.
// Windows itself has no reliable CPU temperature; reading it directly
// needs a kernel driver, which ReforgePulse does not install.
type SensorReader struct {
	mu      sync.Mutex
	nvml    *windows.LazyDLL
	nvmlErr error
	started bool
	kmt     kmtAdapters
}

var _ platform.Sensors = (*SensorReader)(nil)

func NewSensorReader() *SensorReader {
	return &SensorReader{nvml: windows.NewLazySystemDLL("nvml.dll")}
}

// Sensors reads every source, one caller at a time. nvml.dll was seen to
// crash on a NULL read when another read overlapped it, reproducibly on an
// RTX 4070 Ti SUPER, so nothing here runs beside anything else.
func (r *SensorReader) Sensors(ctx context.Context) platform.SensorStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	var st platform.SensorStatus
	gpu, nvmlErr, kmtErr := r.graphics()
	if nvmlErr != nil {
		st.NVML = nvmlErr.Error()
	}
	if kmtErr != nil {
		st.Graphics = kmtErr.Error()
	}
	hw, err := hwinfo()
	if err != nil {
		st.HWiNFO = err.Error()
	}
	st.Sensors = append(gpu, hw...)
	return st
}

// graphics reads NVML where NVIDIA's driver is installed and the graphics
// kernel everywhere else; the caller holds r.mu. Never both in one process:
// with the kernel's adapters open, nvml.dll crashed on a NULL read now and
// then (reproduced on an RTX 4070 Ti SUPER, never without them). NVML also
// says more, watts and load, so on a PC with NVIDIA beside another card that
// other card goes without.
func (r *SensorReader) graphics() (_ []platform.Sensor, nvmlErr, kmtErr error) {
	gpu, nvmlErr := r.gpus()
	if nvmlErr == nil && len(gpu) > 0 {
		return gpu, nil, nil
	}
	other, kmtErr := r.kmt.sensors()
	return other, nvmlErr, kmtErr
}

// nvmlUtilization mirrors nvmlUtilization_t; nvmlMemory mirrors nvmlMemory_t.
type nvmlUtilization struct{ GPU, Memory uint32 }
type nvmlMemory struct{ Total, Free, Used uint64 }

// gpus reads NVIDIA cards; the caller holds r.mu.
func (r *SensorReader) gpus() ([]platform.Sensor, error) {
	if r.nvmlErr != nil {
		return nil, r.nvmlErr
	}
	call := func(name string, args ...uintptr) uintptr {
		ret, _, _ := r.nvml.NewProc(name).Call(args...)
		return ret
	}
	if !r.started {
		if err := r.nvml.Load(); err != nil {
			r.nvmlErr = fmt.Errorf("no NVIDIA driver library (nvml.dll)")
			return nil, r.nvmlErr
		}
		if ret := call("nvmlInit_v2"); ret != 0 {
			r.nvmlErr = fmt.Errorf("nvmlInit failed: %d", ret)
			return nil, r.nvmlErr
		}
		r.started = true
	}
	var count uint32
	call("nvmlDeviceGetCount_v2", uintptr(unsafe.Pointer(&count)))
	var out []platform.Sensor
	for i := uint32(0); i < count; i++ {
		var dev uintptr
		if call("nvmlDeviceGetHandleByIndex_v2", uintptr(i), uintptr(unsafe.Pointer(&dev))) != 0 {
			continue
		}
		nameBuf := make([]byte, 96)
		call("nvmlDeviceGetName", dev, uintptr(unsafe.Pointer(&nameBuf[0])), uintptr(len(nameBuf)))
		group := strings.TrimRight(string(nameBuf), "\x00")
		add := func(name, kind, unit string, v float64) {
			out = append(out, platform.Sensor{Source: "nvml", Group: group, Name: name, Kind: kind, Value: v, Unit: unit})
		}
		var temp, clock, mem, power, fan uint32
		if call("nvmlDeviceGetTemperature", dev, 0, uintptr(unsafe.Pointer(&temp))) == 0 {
			add("GPU temperature", "temp", "°C", float64(temp))
		}
		if call("nvmlDeviceGetClockInfo", dev, 0, uintptr(unsafe.Pointer(&clock))) == 0 {
			add("Core clock", "clock", "MHz", float64(clock))
		}
		if call("nvmlDeviceGetClockInfo", dev, 2, uintptr(unsafe.Pointer(&mem))) == 0 {
			add("Memory clock", "clock", "MHz", float64(mem))
		}
		if call("nvmlDeviceGetPowerUsage", dev, uintptr(unsafe.Pointer(&power))) == 0 {
			add("Board power", "power", "W", float64(power)/1000)
		}
		if call("nvmlDeviceGetFanSpeed", dev, uintptr(unsafe.Pointer(&fan))) == 0 {
			add("Fan", "fan", "%", float64(fan))
		}
		var u nvmlUtilization
		if call("nvmlDeviceGetUtilizationRates", dev, uintptr(unsafe.Pointer(&u))) == 0 {
			add("GPU load", "load", "%", float64(u.GPU))
		}
		var m nvmlMemory
		if call("nvmlDeviceGetMemoryInfo", dev, uintptr(unsafe.Pointer(&m))) == 0 && m.Total > 0 {
			add("VRAM used", "load", "%", 100*float64(m.Used)/float64(m.Total))
		}
	}
	return out, nil
}

// Close shuts NVML down, then lets go of the graphics adapters.
func (r *SensorReader) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		r.nvml.NewProc("nvmlShutdown").Call()
		r.started = false
	}
	r.kmt.close()
}

const hwinfoSignature = 0x53695748 // "HWiS"

var procOpenFileMapping = kernel32.NewProc("OpenFileMappingW")

// hwinfoKinds maps HWiNFO's reading types to ours; the rest are skipped.
var hwinfoKinds = map[uint32]string{1: "temp", 2: "volt", 3: "fan", 5: "power", 6: "clock", 7: "load"}

// hwinfo reads HWiNFO's shared memory (HWiNFO_SENSORS_SHARED_MEM2, packed):
// a header with section offsets, sensor names, then readings.
func hwinfo() ([]platform.Sensor, error) {
	name, _ := windows.UTF16PtrFromString(`Global\HWiNFO_SENS_SM2`)
	hv, _, _ := procOpenFileMapping.Call(windows.FILE_MAP_READ, 0, uintptr(unsafe.Pointer(name)))
	if hv == 0 {
		return nil, fmt.Errorf("HWiNFO is not running with Shared Memory Support on")
	}
	h := windows.Handle(hv)
	defer windows.CloseHandle(h)
	addr, err := windows.MapViewOfFile(h, windows.FILE_MAP_READ, 0, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("map HWiNFO memory: %w", err)
	}
	defer windows.UnmapViewOfFile(addr)
	var info windows.MemoryBasicInformation
	if err := windows.VirtualQuery(addr, &info, unsafe.Sizeof(info)); err != nil {
		return nil, fmt.Errorf("size HWiNFO memory: %w", err)
	}
	// The view stays mapped until the deferred unmap, so reading through the
	// address is safe; the double conversion keeps vet from flagging it.
	mem := unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&addr))), info.RegionSize)
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(mem[off:]) }
	if u32(0) != hwinfoSignature {
		return nil, fmt.Errorf("HWiNFO memory is not ready")
	}
	sensOff, sensSize, sensN := int(u32(20)), int(u32(24)), int(u32(28))
	readOff, readSize, readN := int(u32(32)), int(u32(36)), int(u32(40))
	if readOff+readSize*readN > len(mem) || sensOff+sensSize*sensN > len(mem) {
		return nil, fmt.Errorf("HWiNFO memory layout not understood")
	}
	sensorName := func(i int) string {
		if i < 0 || i >= sensN {
			return ""
		}
		return latin1(mem[sensOff+i*sensSize+136 : sensOff+i*sensSize+264]) // the user's name for the sensor
	}
	var out []platform.Sensor
	for i := 0; i < readN; i++ {
		r := mem[readOff+i*readSize:]
		kind, ok := hwinfoKinds[binary.LittleEndian.Uint32(r)]
		if !ok {
			continue
		}
		out = append(out, platform.Sensor{
			Source: "hwinfo", Group: sensorName(int(binary.LittleEndian.Uint32(r[4:]))), Kind: kind,
			Name:  latin1(r[140:268]),
			Unit:  latin1(r[268:284]),
			Value: math.Float64frombits(binary.LittleEndian.Uint64(r[284:])),
			Max:   math.Float64frombits(binary.LittleEndian.Uint64(r[300:])),
		})
	}
	return out, nil
}

// latin1 decodes HWiNFO's ANSI strings, which put ° as byte 0xB0.
func latin1(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		if c == 0 {
			break
		}
		sb.WriteRune(rune(c))
	}
	return strings.TrimSpace(sb.String())
}
