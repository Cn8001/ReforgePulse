package app

import (
	"context"
	"errors"
	"testing"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type fakeReader struct {
	o   domain.Overview
	err error
}

func (f fakeReader) Overview(context.Context) (domain.Overview, error) { return f.o, f.err }
func (f fakeReader) Live(context.Context) (domain.Live, error)         { return domain.Live{}, f.err }

func TestSystemService_PassesOverviewThrough(t *testing.T) {
	// Arrange
	s := NewSystemService(fakeReader{o: domain.Overview{OS: "Windows 11"}}, fakeReader{}, tasks.New(1, func(tasks.Status) {}))

	// Act
	o, err := s.GetOverview(context.Background())

	// Assert
	if err != nil || o.OS != "Windows 11" {
		t.Fatalf("got %+v, %v", o, err)
	}
}

func TestSystemService_WrapsErrors(t *testing.T) {
	// Arrange
	cause := errors.New("boom")
	s := NewSystemService(fakeReader{err: cause}, fakeReader{}, tasks.New(1, func(tasks.Status) {}))

	// Act
	_, err := s.GetOverview(context.Background())

	// Assert
	if !errors.Is(err, cause) {
		t.Fatalf("err = %v, want it to wrap %v", err, cause)
	}
}

type fakeAccent string

func (f fakeAccent) GetAccentColor() string { return string(f) }

func TestThemeService_ReturnsAccent(t *testing.T) {
	// Arrange
	s := NewThemeService(fakeAccent("rgb(0,120,212)"))

	// Act
	got := s.Accent()

	// Assert
	if got != "rgb(0,120,212)" {
		t.Fatalf("accent = %q", got)
	}
}

type fakeLog struct {
	changes []domain.Change
	err     error
}

func (f fakeLog) Log(context.Context) ([]domain.Change, error) { return f.changes, f.err }

func TestHistoryService_ListsAndWraps(t *testing.T) {
	// Arrange
	cause := errors.New("disk full")
	ok := NewHistoryService(fakeLog{changes: []domain.Change{{ID: "abc1234"}}}, nil)
	bad := NewHistoryService(fakeLog{err: cause}, nil)

	// Act
	got, err := ok.List(context.Background())
	_, badErr := bad.List(context.Background())

	// Assert
	if err != nil || len(got) != 1 || got[0].ID != "abc1234" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if !errors.Is(badErr, cause) {
		t.Fatalf("err = %v, want it to wrap %v", badErr, cause)
	}
}

func TestSystemService_RunsAsTask(t *testing.T) {
	// Arrange
	var titles []string
	runner := tasks.New(1, func(s tasks.Status) { titles = append(titles, s.Title) })
	s := NewSystemService(fakeReader{}, fakeReader{}, runner)

	// Act
	_, err := s.GetOverview(context.Background())

	// Assert
	if err != nil || len(titles) == 0 || titles[0] != "Reading system overview" {
		t.Fatalf("titles = %v, err = %v", titles, err)
	}
}

type fakeLister struct {
	ps  []domain.Process
	err error
}

func (f fakeLister) List(context.Context) ([]domain.Process, error) { return f.ps, f.err }

func TestProcessService_ListsAndWraps(t *testing.T) {
	// Arrange
	cause := errors.New("snapshot failed")
	ok := NewProcessService(fakeLister{ps: []domain.Process{{PID: 4, Name: "System"}}}, nil, nil, 1)
	bad := NewProcessService(fakeLister{err: cause}, nil, nil, 1)

	// Act
	got, err := ok.List(context.Background())
	_, badErr := bad.List(context.Background())

	// Assert
	if err != nil || len(got) != 1 || got[0].PID != 4 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if !errors.Is(badErr, cause) {
		t.Fatalf("err = %v", badErr)
	}
}

func TestVersionFrom_ReadsInfoVersion(t *testing.T) {
	// Arrange
	config := []byte("version: '3'\ninfo:\n  productName: \"ReforgePulse\"\n  version: \"0.0.1\" # The application version\nios:\n  version: \"9.9.9\"\n")

	// Act
	got := VersionFrom(config)

	// Assert
	if got != "0.0.1" {
		t.Fatalf("version = %q, want 0.0.1", got)
	}
}

func TestVersionFrom_MissingIsDev(t *testing.T) {
	// Act
	got := VersionFrom([]byte("version: '3'\n"))

	// Assert
	if got != "dev" {
		t.Fatalf("version = %q, want dev", got)
	}
}

type memJournal struct{ begun []domain.Change }

func (m *memJournal) Begin(_ context.Context, c domain.Change) (string, error) {
	m.begun = append(m.begun, c)
	return "id" + string(rune('0'+len(m.begun))), nil
}
func (m *memJournal) Finish(context.Context, string, string, error) error { return nil }

type fakeSwitch struct{ calls []bool }

func (f *fakeSwitch) SetStartupEnabled(_ context.Context, _ string, on bool) error {
	f.calls = append(f.calls, on)
	return nil
}

func TestStartupService_DisableIsLoggedAndRevertible(t *testing.T) {
	// Arrange
	j, sw := &memJournal{}, &fakeSwitch{}
	s := NewStartupService(nil, sw, j)

	// Act
	err := s.SetEnabled(context.Background(), "user-run|Steam", "Steam", false)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	c := j.begun[0]
	if c.Action != "startup" || !c.Reversible || c.Before != "enabled" || c.After != "disabled" || sw.calls[0] {
		t.Fatalf("change = %+v, calls = %v", c, sw.calls)
	}
}

func TestHistoryService_RevertDispatchesByAction(t *testing.T) {
	// Arrange
	j, sw := &memJournal{}, &fakeSwitch{}
	startup := NewStartupService(nil, sw, j)
	applied := domain.Change{ID: "abc1234", Action: "startup", Target: "user-run|Steam", Before: "enabled", After: "disabled",
		Reversible: true, Status: domain.Applied}
	h := NewHistoryService(fakeLog{changes: []domain.Change{applied}}, map[string]Reverter{"startup": startup})

	// Act
	err := h.Revert(context.Background(), "abc1234")

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.calls) != 1 || !sw.calls[0] || j.begun[0].Reverts != "abc1234" {
		t.Fatalf("calls = %v, change = %+v", sw.calls, j.begun)
	}
}

func TestHistoryService_RefusesIrreversible(t *testing.T) {
	// Arrange
	c := domain.Change{ID: "x", Action: "cleanup", Status: domain.Applied}
	h := NewHistoryService(fakeLog{changes: []domain.Change{c}}, map[string]Reverter{})

	// Act
	err := h.Revert(context.Background(), "x")

	// Assert
	if err == nil {
		t.Fatal("want an error for an irreversible change")
	}
}

type fakeKiller struct{ killed []uint32 }

func (f *fakeKiller) Kill(_ context.Context, pid uint32) error {
	f.killed = append(f.killed, pid)
	return nil
}

func TestProcessService_EndRefusesWindowsProcesses(t *testing.T) {
	// Arrange
	k := &fakeKiller{}
	s := NewProcessService(nil, k, &memJournal{}, 999)

	// Act
	errs := []error{
		s.End(context.Background(), 4, "System"),
		s.End(context.Background(), 700, "csrss.exe"),
		s.End(context.Background(), 999, "reforgepulse.exe"),
	}

	// Assert
	for i, err := range errs {
		if err == nil {
			t.Fatalf("case %d: want a refusal", i)
		}
	}
	if len(k.killed) != 0 {
		t.Fatalf("killed %v", k.killed)
	}
}

func TestProcessService_EndIsLogged(t *testing.T) {
	// Arrange
	k, j := &fakeKiller{}, &memJournal{}
	s := NewProcessService(nil, k, j, 999)

	// Act
	err := s.End(context.Background(), 4242, "hungapp.exe")

	// Assert
	if err != nil || len(k.killed) != 1 || j.begun[0].Action != "end-task" || j.begun[0].Reversible {
		t.Fatalf("err = %v, killed = %v, log = %+v", err, k.killed, j.begun)
	}
}

type fakeFinder []domain.Leftover

func (f fakeFinder) List(context.Context) ([]domain.Leftover, error) { return f, nil }

type fakeBin struct{ got []string }

func (b *fakeBin) Recycle(_ context.Context, paths []string) error {
	b.got = append(b.got, paths...)
	return nil
}

func TestLeftoverService_RefusesUnscannedPaths(t *testing.T) {
	// Arrange
	bin := &fakeBin{}
	s := NewLeftoverService(LeftoverDeps{Finder: fakeFinder{{Path: `C:\Old`}}, Bin: bin, Runner: tasks.New(1, func(tasks.Status) {}), Journal: &memJournal{}})
	_, _ = s.List(context.Background())

	// Act
	err := s.Recycle(context.Background(), []string{`C:\Windows`})
	okErr := s.Recycle(context.Background(), []string{`C:\Old`})

	// Assert
	if err == nil {
		t.Fatal("a path outside the scan was accepted")
	}
	if okErr != nil || len(bin.got) != 1 || bin.got[0] != `C:\Old` {
		t.Fatalf("err = %v, recycled = %v", okErr, bin.got)
	}
}

type fakeCrashes struct{ ev domain.CrashEvidence }

func (f fakeCrashes) Collect(context.Context) (domain.CrashEvidence, error) { return f.ev, nil }

type fakeStartupList []domain.StartupApp

func (f fakeStartupList) List(context.Context) ([]domain.StartupApp, error) { return f, nil }

func TestScanService_EvaluatesAndKeepsLast(t *testing.T) {
	// Arrange
	s := NewScanService(ScanDeps{
		Hardware:  fakeHardware{},
		Sensors:   fakeHardware{},
		Devices:   fakeHardware{},
		Overview:  fakeReader{o: domain.Overview{Elevated: true}},
		Live:      fakeReader{},
		Crashes:   fakeCrashes{ev: domain.CrashEvidence{Restarts: []domain.CrashEvent{{}, {}}}},
		Startup:   fakeStartupList{},
		Processes: fakeLister{},
		Runner:    tasks.New(1, func(tasks.Status) {}),
	})

	// Act
	res, err := s.Scan(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 1 || res.Findings[0].Rule != "unexpected-restarts" {
		t.Fatalf("findings = %+v", res.Findings)
	}
	if s.Last() == nil || len(s.Last().Findings) != 1 {
		t.Fatal("the result was not kept")
	}
}

type fakePower struct{ plans []platform.PowerPlan }

func (f *fakePower) PowerPlans(context.Context) ([]platform.PowerPlan, error) {
	return append([]platform.PowerPlan(nil), f.plans...), nil
}

func (f *fakePower) SetCPUCaps(_ context.Context, p platform.PowerPlan) error {
	for i := range f.plans {
		if f.plans[i].GUID == p.GUID {
			f.plans[i] = p
		}
	}
	return nil
}

func TestCPUCapService_ApplyThenRevertRestoresEveryPlan(t *testing.T) {
	// Arrange
	power := &fakePower{plans: []platform.PowerPlan{{GUID: "a", Name: "Balanced"}, {GUID: "b", Name: "MSI", PCoreAC: 5800}}}
	j := &memJournal{}
	s := NewCPUCapService(power, j)

	// Act
	err := s.Apply(context.Background(), 5500, 0)
	applied := j.begun[0]
	applied.ID, applied.Status = "abc", domain.Applied
	rerr := s.Revert(context.Background(), applied)

	// Assert
	if err != nil || rerr != nil {
		t.Fatalf("apply = %v, revert = %v", err, rerr)
	}
	if power.plans[0].PCoreAC != 0 || power.plans[1].PCoreAC != 5800 {
		t.Fatalf("plans after revert = %+v", power.plans)
	}
	if j.begun[1].Reverts != "abc" {
		t.Fatalf("revert entry = %+v", j.begun[1])
	}
}

func TestCPUCapService_RejectsSillyValues(t *testing.T) {
	// Arrange
	s := NewCPUCapService(&fakePower{}, &memJournal{})

	// Act
	err := s.Apply(context.Background(), 50, 0)

	// Assert
	if err == nil {
		t.Fatal("50 MHz was accepted")
	}
}

type fakeHardware struct{}

func (fakeHardware) HardwareFacts(context.Context) (platform.HardwareFacts, error) {
	return platform.HardwareFacts{}, nil
}

func (fakeHardware) Sensors(context.Context) platform.SensorStatus { return platform.SensorStatus{} }

func (fakeHardware) ProblemDevices(context.Context) ([]platform.ProblemDevice, error) {
	return nil, nil
}
