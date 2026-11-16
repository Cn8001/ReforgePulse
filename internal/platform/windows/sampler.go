//go:build windows

package windows

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/cn8001/reforgepulse/internal/platform"
)

var (
	procGetSystemTimes = kernel32.NewProc("GetSystemTimes")

	pdh                             = windows.NewLazySystemDLL("pdh.dll")
	procPdhOpenQuery                = pdh.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounter        = pdh.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData         = pdh.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterArray = pdh.NewProc("PdhGetFormattedCounterArrayW")
	procPdhCloseQuery               = pdh.NewProc("PdhCloseQuery")
)

const (
	pdhFmtDouble = 0x200
	pdhMoreData  = 0x800007D2
	// The 3D engines across every process, which is what Task Manager's GPU
	// column shows.
	gpuCounter = `\GPU Engine(*engtype_3D)\Utilization Percentage`
)

// Sampler measures CPU and GPU load between calls. It keeps the previous
// CPU times and an open PDH query, so it must be closed.
type Sampler struct {
	mu        sync.Mutex
	idle      uint64
	total     uint64
	query     uintptr
	counter   uintptr
	gpuFailed error // set once when the GPU counter is unavailable
}

var _ platform.Sampler = (*Sampler)(nil)

func NewSampler() *Sampler { return &Sampler{} }

func (s *Sampler) CPULoad(ctx context.Context) (float64, error) {
	var idle, kernel, user windows.Filetime
	if ok, _, err := procGetSystemTimes.Call(uintptr(unsafe.Pointer(&idle)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user))); ok == 0 {
		return 0, fmt.Errorf("GetSystemTimes: %w", err)
	}
	// Kernel time already includes idle time.
	i, t := filetime(idle), filetime(kernel)+filetime(user)

	s.mu.Lock()
	defer s.mu.Unlock()
	di, dt := i-s.idle, t-s.total
	s.idle, s.total = i, t
	if dt == 0 {
		return 0, nil
	}
	return 100 * float64(dt-di) / float64(dt), nil
}

func (s *Sampler) GPULoad(ctx context.Context) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gpuFailed != nil {
		return 0, s.gpuFailed
	}
	if s.query == 0 {
		if err := s.openGPU(); err != nil {
			s.gpuFailed = err
			return 0, err
		}
		return 0, nil // a rate counter needs two samples
	}
	if r, _, _ := procPdhCollectQueryData.Call(s.query); r != 0 {
		return 0, fmt.Errorf("PdhCollectQueryData: 0x%x", r)
	}
	var size, count uint32
	r, _, _ := procPdhGetFormattedCounterArray.Call(s.counter, pdhFmtDouble, uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), 0)
	if r != pdhMoreData {
		return 0, nil // no 3D engine instances right now
	}
	buf := make([]byte, size)
	if r, _, _ := procPdhGetFormattedCounterArray.Call(s.counter, pdhFmtDouble, uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buf[0]))); r != 0 {
		return 0, fmt.Errorf("PdhGetFormattedCounterArray: 0x%x", r)
	}
	// PDH_FMT_COUNTERVALUE_ITEM_W: a name pointer, then CStatus, padding and the double.
	type item struct {
		name   *uint16
		status uint32
		_      uint32
		value  float64
	}
	items := unsafe.Slice((*item)(unsafe.Pointer(&buf[0])), count)
	var sum float64
	for _, it := range items {
		if it.status == 0 {
			sum += it.value
		}
	}
	return min(sum, 100), nil
}

func (s *Sampler) openGPU() error {
	var q uintptr
	if r, _, _ := procPdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&q))); r != 0 {
		return fmt.Errorf("PdhOpenQuery: 0x%x: %w", r, errors.ErrUnsupported)
	}
	path, _ := windows.UTF16PtrFromString(gpuCounter)
	var c uintptr
	if r, _, _ := procPdhAddEnglishCounter.Call(q, uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&c))); r != 0 {
		procPdhCloseQuery.Call(q)
		return fmt.Errorf("GPU counter: 0x%x: %w", r, errors.ErrUnsupported)
	}
	procPdhCollectQueryData.Call(q)
	s.query, s.counter = q, c
	return nil
}

// Close releases the PDH query.
func (s *Sampler) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.query != 0 {
		procPdhCloseQuery.Call(s.query)
		s.query = 0
	}
}
