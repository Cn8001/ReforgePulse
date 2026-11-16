package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cn8001/reforgepulse/internal/diagnosis"
	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type CrashCollector interface {
	Collect(ctx context.Context) (domain.CrashEvidence, error)
}

type HardwareReader interface {
	HardwareFacts(ctx context.Context) (platform.HardwareFacts, error)
}

type DeviceReader interface {
	ProblemDevices(ctx context.Context) ([]platform.ProblemDevice, error)
}

// ScanDeps are what a scan reads from.
type ScanDeps struct {
	Devices   DeviceReader
	Hardware  HardwareReader
	Sensors   SensorReader
	Overview  OverviewReader
	Live      LiveReader
	Crashes   CrashCollector
	Startup   StartupLister
	Processes ProcessLister
	Runner    TaskRunner
}

// ScanService runs Scan PC: it reads everything once, applies the rules,
// and keeps the last result for Home and Fix Issues.
type ScanService struct {
	d    ScanDeps
	mu   sync.Mutex
	last *domain.ScanResult
}

func NewScanService(d ScanDeps) *ScanService { return &ScanService{d: d} }

func (s *ScanService) Scan(ctx context.Context) (domain.ScanResult, error) {
	var res domain.ScanResult
	err := s.d.Runner.Do(ctx, "Scanning PC", func(ctx context.Context, report tasks.Report) error {
		var in diagnosis.Input
		var err error
		report(0.1, "system")
		if in.Overview, err = s.d.Overview.Overview(ctx); err != nil {
			return err
		}
		report(0.25, "load")
		in.Live, _ = s.d.Live.Live(ctx) // missing load only drops the load rules
		report(0.4, "event logs")
		if in.Crashes, err = s.d.Crashes.Collect(ctx); err != nil {
			return err
		}
		report(0.65, "memory and drives")
		in.Hardware, _ = s.d.Hardware.HardwareFacts(ctx) // missing facts leave those areas unmeasured
		in.Sensors = s.d.Sensors.Sensors(ctx)
		in.Devices, _ = s.d.Devices.ProblemDevices(ctx)
		report(0.8, "startup apps")
		in.Startup, _ = s.d.Startup.List(ctx)
		report(0.9, "processes")
		in.Processes, _ = s.d.Processes.List(ctx)
		res = domain.ScanResult{At: time.Now(), Findings: diagnosis.Evaluate(in), Suspects: diagnosis.Suspects(in), Elevated: in.Overview.Elevated}
		return nil
	})
	if err != nil {
		return domain.ScanResult{}, fmt.Errorf("scan: %w", err)
	}
	s.mu.Lock()
	s.last = &res
	s.mu.Unlock()
	return res, nil
}

// Last is the latest scan, or nil before the first.
func (s *ScanService) Last() *domain.ScanResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

// Evidence returns the raw crash evidence for the expert view.
func (s *ScanService) Evidence(ctx context.Context) (domain.CrashEvidence, error) {
	ev, err := s.d.Crashes.Collect(ctx)
	if err != nil {
		return ev, fmt.Errorf("read crash evidence: %w", err)
	}
	return ev, nil
}
