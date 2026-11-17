package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cn8001/reforgepulse/internal/diagnosis"
	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
	"github.com/cn8001/reforgepulse/internal/reports"
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

// ReportStore keeps past scans.
type ReportStore interface {
	Save(ctx context.Context, r domain.ScanResult) (domain.ScanResult, error)
	All(ctx context.Context) ([]domain.ScanResult, error)
	Get(ctx context.Context, id string) (domain.ScanResult, error)
	Latest(ctx context.Context) (*domain.ScanResult, error)
	Delete(ctx context.Context, id string) error
	DeleteBefore(ctx context.Context, t time.Time) (int, error)
	DeleteFinding(ctx context.Context, id, key string) (domain.ScanResult, error)
}

// ResolutionStore keeps what the user marked resolved.
type ResolutionStore interface {
	All(ctx context.Context) (map[string]domain.Resolution, error)
	Resolve(ctx context.Context, key, title string) error
	Reopen(ctx context.Context, key string) error
}

// ScanDeps are what a scan reads from and where its report goes.
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
	Reports   ReportStore
	Resolved  ResolutionStore
}

// ScanService runs Scan PC: it reads everything once, applies the rules,
// saves the result as a report and keeps it for Home and Issues. Statuses
// are applied on every read, so a resolution shows in old reports too.
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
	if res, err = s.d.Reports.Save(ctx, res); err != nil {
		return domain.ScanResult{}, fmt.Errorf("save report: %w", err)
	}
	s.mu.Lock()
	s.last = &res
	s.mu.Unlock()
	return s.withStatus(ctx, res)
}

// Last is the latest scan, from the saved reports after a restart, or nil
// before the first.
func (s *ScanService) Last(ctx context.Context) (*domain.ScanResult, error) {
	s.mu.Lock()
	last := s.last
	s.mu.Unlock()
	if last == nil {
		var err error
		if last, err = s.d.Reports.Latest(ctx); err != nil {
			return nil, fmt.Errorf("read last report: %w", err)
		}
		if last == nil {
			return nil, nil
		}
		s.mu.Lock()
		s.last = last
		s.mu.Unlock()
	}
	res, err := s.withStatus(ctx, *last)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Reports lists the kept scans, newest first.
func (s *ScanService) Reports(ctx context.Context) ([]domain.ReportSummary, error) {
	all, err := s.d.Reports.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	resolved, err := s.d.Resolved.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("read resolutions: %w", err)
	}
	out := make([]domain.ReportSummary, 0, len(all))
	for _, r := range all {
		r.Findings = append([]domain.Finding(nil), r.Findings...) // the store's slice stays as saved
		reports.Apply(&r, resolved)
		out = append(out, reports.Summarize(r))
	}
	return out, nil
}

// Report opens one saved scan.
func (s *ScanService) Report(ctx context.Context, id string) (domain.ScanResult, error) {
	r, err := s.d.Reports.Get(ctx, id)
	if err != nil {
		return domain.ScanResult{}, fmt.Errorf("open report: %w", err)
	}
	return s.withStatus(ctx, r)
}

// DeleteReport removes one saved scan.
func (s *ScanService) DeleteReport(ctx context.Context, id string) error {
	if err := s.d.Reports.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete report: %w", err)
	}
	s.forget(func(r *domain.ScanResult) bool { return r.ID == id })
	return nil
}

// DeleteFinding takes one finding out of a saved scan. It is back after the
// next scan if Windows still records it; Resolve is for keeping it away.
func (s *ScanService) DeleteFinding(ctx context.Context, reportID, key string) error {
	r, err := s.d.Reports.DeleteFinding(ctx, reportID, key)
	if err != nil {
		return fmt.Errorf("delete finding: %w", err)
	}
	s.mu.Lock()
	if s.last != nil && s.last.ID == reportID {
		s.last = &r
	}
	s.mu.Unlock()
	return nil
}

// DeleteReportsBefore removes saved scans that ran before t.
func (s *ScanService) DeleteReportsBefore(ctx context.Context, t time.Time) (int, error) {
	n, err := s.d.Reports.DeleteBefore(ctx, t)
	if err != nil {
		return n, fmt.Errorf("delete reports: %w", err)
	}
	s.forget(func(r *domain.ScanResult) bool { return r.At.Before(t) })
	return n, nil
}

// forget drops the kept scan when its report is gone, so Last falls back
// to whichever report is now the newest.
func (s *ScanService) forget(gone func(*domain.ScanResult) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last != nil && gone(s.last) {
		s.last = nil
	}
}

// Resolve marks a finding dealt with until new evidence for it turns up.
func (s *ScanService) Resolve(ctx context.Context, key, title string) error {
	if err := s.d.Resolved.Resolve(ctx, key, title); err != nil {
		return fmt.Errorf("resolve %s: %w", key, err)
	}
	return nil
}

// Reopen undoes Resolve.
func (s *ScanService) Reopen(ctx context.Context, key string) error {
	if err := s.d.Resolved.Reopen(ctx, key); err != nil {
		return fmt.Errorf("reopen %s: %w", key, err)
	}
	return nil
}

// withStatus applies the current resolutions to a copy, so the kept result
// is never changed under a concurrent reader.
func (s *ScanService) withStatus(ctx context.Context, r domain.ScanResult) (domain.ScanResult, error) {
	resolved, err := s.d.Resolved.All(ctx)
	if err != nil {
		return domain.ScanResult{}, fmt.Errorf("read resolutions: %w", err)
	}
	r.Findings = append([]domain.Finding(nil), r.Findings...)
	reports.Apply(&r, resolved)
	return r, nil
}

// Evidence returns the raw crash evidence for the expert view.
func (s *ScanService) Evidence(ctx context.Context) (domain.CrashEvidence, error) {
	ev, err := s.d.Crashes.Collect(ctx)
	if err != nil {
		return ev, fmt.Errorf("read crash evidence: %w", err)
	}
	return ev, nil
}
