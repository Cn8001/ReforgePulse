//go:build windows && amd64

package windows

import (
	"context"
	"encoding/binary"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

// cpuid runs the CPUID instruction on the current processor.
func cpuid(leaf, subleaf uint32) (eax, ebx, ecx, edx uint32)

var (
	procGetLogicalProcessorInformationEx = kernel32.NewProc("GetLogicalProcessorInformationEx")
	procSetThreadAffinityMask            = kernel32.NewProc("SetThreadAffinityMask")
)

const relationProcessorCore = 0

// CPUs reads topology and per-core load. It keeps an open PDH query, so it
// must be closed.
type CPUs struct {
	mu      sync.Mutex
	query   uintptr
	util    uintptr
	perf    uintptr
	freq    uintptr
	openErr error
}

var _ platform.CPUs = (*CPUs)(nil)

func NewCPUs() *CPUs { return &CPUs{} }

// Topology walks the cores Windows reports, then asks each logical
// processor its own x2APIC ID by running CPUID pinned to it.
func (c *CPUs) Topology(ctx context.Context) ([]platform.LogicalCPU, error) {
	var size uint32
	procGetLogicalProcessorInformationEx.Call(relationProcessorCore, 0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return nil, fmt.Errorf("processor topology: no data")
	}
	buf := make([]byte, size)
	if ok, _, err := procGetLogicalProcessorInformationEx.Call(relationProcessorCore, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size))); ok == 0 {
		return nil, fmt.Errorf("processor topology: %w", err)
	}
	var out []platform.LogicalCPU
	// SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX: Size at 4, then
	// PROCESSOR_RELATIONSHIP: Flags at 8, EfficiencyClass at 9, the first
	// GROUP_AFFINITY's mask at 32 and group at 40.
	for off, core := 0, 0; off < int(size); core++ {
		rec := buf[off:]
		n := int(binary.LittleEndian.Uint32(rec[4:]))
		smt, eff := rec[8]&1 == 1, rec[9]
		mask := binary.LittleEndian.Uint64(rec[32:])
		group := binary.LittleEndian.Uint16(rec[40:])
		for i := 0; i < 64; i++ {
			if mask&(1<<i) != 0 && group == 0 {
				out = append(out, platform.LogicalCPU{Index: i, Core: core, Efficiency: eff, SMT: smt})
			}
		}
		off += n
	}
	for i := range out {
		var apic uint32
		_ = c.RunPinned(out[i].Index, func() { _, _, _, apic = cpuid(0x0B, 0) })
		out[i].APIC = apic
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out, nil
}

// RunPinned locks the goroutine to its OS thread and that thread to one
// logical processor, runs fn, then lets both go.
func (c *CPUs) RunPinned(cpu int, fn func()) error {
	if cpu < 0 || cpu >= 64 {
		return fmt.Errorf("processor %d is outside group 0", cpu)
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	thread := uintptr(windows.CurrentThread())
	old, _, err := procSetThreadAffinityMask.Call(thread, uintptr(1)<<cpu)
	if old == 0 {
		return fmt.Errorf("pin to processor %d: %w", cpu, err)
	}
	defer procSetThreadAffinityMask.Call(thread, old)
	fn()
	return nil
}

// CoreLoads reads the counters Task Manager uses: utility for load, and
// base frequency times performance for the effective clock.
func (c *CPUs) CoreLoads(ctx context.Context) ([]platform.CoreLoad, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.openErr != nil {
		return nil, c.openErr
	}
	if c.query == 0 {
		if err := c.open(); err != nil {
			c.openErr = err
			return nil, err
		}
	}
	if r, _, _ := procPdhCollectQueryData.Call(c.query); r != 0 {
		return nil, fmt.Errorf("PdhCollectQueryData: 0x%x", r)
	}
	util, perf, freq := pdhArray(c.util), pdhArray(c.perf), pdhArray(c.freq)
	var out []platform.CoreLoad
	for inst, u := range util {
		group, num, ok := strings.Cut(inst, ",")
		idx, err := strconv.Atoi(num)
		if !ok || group != "0" || err != nil {
			continue // totals and other processor groups
		}
		out = append(out, platform.CoreLoad{Index: idx, Load: min(u, 100), MHz: freq[inst] * perf[inst] / 100})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out, nil
}

func (c *CPUs) open() error {
	var q uintptr
	if r, _, _ := procPdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&q))); r != 0 {
		return fmt.Errorf("PdhOpenQuery: 0x%x", r)
	}
	add := func(path string) (uintptr, error) {
		p, _ := windows.UTF16PtrFromString(path)
		var h uintptr
		if r, _, _ := procPdhAddEnglishCounter.Call(q, uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(&h))); r != 0 {
			return 0, fmt.Errorf("counter %s: 0x%x", path, r)
		}
		return h, nil
	}
	var err error
	if c.util, err = add(`\Processor Information(*)\% Processor Utility`); err == nil {
		if c.perf, err = add(`\Processor Information(*)\% Processor Performance`); err == nil {
			c.freq, err = add(`\Processor Information(*)\Processor Frequency`)
		}
	}
	if err != nil {
		procPdhCloseQuery.Call(q)
		return err
	}
	procPdhCollectQueryData.Call(q)
	c.query = q
	return nil
}

func (c *CPUs) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.query != 0 {
		procPdhCloseQuery.Call(c.query)
		c.query = 0
	}
}

// pdhArray reads a wildcard counter as instance name to value.
func pdhArray(counter uintptr) map[string]float64 {
	var size, count uint32
	r, _, _ := procPdhGetFormattedCounterArray.Call(counter, pdhFmtDouble|pdhFmtNoCap100, uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), 0)
	if r != pdhMoreData {
		return nil
	}
	buf := make([]byte, size)
	if r, _, _ := procPdhGetFormattedCounterArray.Call(counter, pdhFmtDouble|pdhFmtNoCap100, uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buf[0]))); r != 0 {
		return nil
	}
	type item struct {
		name   *uint16
		status uint32
		_      uint32
		value  float64
	}
	out := make(map[string]float64, count)
	for _, it := range unsafe.Slice((*item)(unsafe.Pointer(&buf[0])), count) {
		if it.status == 0 {
			out[windows.UTF16PtrToString(it.name)] = it.value
		}
	}
	return out
}
