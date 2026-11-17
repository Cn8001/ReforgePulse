package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

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

func (f fakeLog) Log(context.Context) ([]domain.Change, error)         { return f.changes, f.err }
func (f fakeLog) Delete(context.Context, ...string) (int, error)       { return 0, f.err }
func (f fakeLog) DeleteBefore(context.Context, time.Time) (int, error) { return 0, f.err }

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
	ok := NewProcessService(ProcessDeps{Lister: fakeLister{ps: []domain.Process{{PID: 4, Name: "System"}}}, Self: 1})
	bad := NewProcessService(ProcessDeps{Lister: fakeLister{err: cause}, Self: 1})

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
	s := NewProcessService(ProcessDeps{Killer: k, Journal: &memJournal{}, Self: 999})

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
	s := NewProcessService(ProcessDeps{Killer: k, Journal: j, Self: 999})

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

type memReports struct{ saved []domain.ScanResult }

func (m *memReports) Save(_ context.Context, r domain.ScanResult) (domain.ScanResult, error) {
	r.ID = fmt.Sprint(len(m.saved))
	m.saved = append([]domain.ScanResult{r}, m.saved...)
	return r, nil
}
func (m *memReports) All(context.Context) ([]domain.ScanResult, error) { return m.saved, nil }
func (m *memReports) Get(_ context.Context, id string) (domain.ScanResult, error) {
	for _, r := range m.saved {
		if r.ID == id {
			return r, nil
		}
	}
	return domain.ScanResult{}, os.ErrNotExist
}
func (m *memReports) Delete(_ context.Context, id string) error {
	for i, r := range m.saved {
		if r.ID == id {
			m.saved = append(m.saved[:i], m.saved[i+1:]...)
			return nil
		}
	}
	return os.ErrNotExist
}
func (m *memReports) DeleteBefore(context.Context, time.Time) (int, error) { return 0, nil }
func (m *memReports) DeleteFinding(_ context.Context, id, key string) (domain.ScanResult, error) {
	for i, r := range m.saved {
		if r.ID == id {
			var kept []domain.Finding
			for _, f := range r.Findings {
				if f.Key != key {
					kept = append(kept, f)
				}
			}
			m.saved[i].Findings = kept
			return m.saved[i], nil
		}
	}
	return domain.ScanResult{}, os.ErrNotExist
}
func (m *memReports) Latest(context.Context) (*domain.ScanResult, error) {
	if len(m.saved) == 0 {
		return nil, nil
	}
	return &m.saved[0], nil
}

type memResolved map[string]domain.Resolution

func (m memResolved) All(context.Context) (map[string]domain.Resolution, error) { return m, nil }
func (m memResolved) Resolve(_ context.Context, key, title string) error {
	m[key] = domain.Resolution{Key: key, Title: title, At: time.Now()}
	return nil
}
func (m memResolved) Reopen(_ context.Context, key string) error {
	delete(m, key)
	return nil
}

func TestScanService_SavesReportsAndAppliesResolutions(t *testing.T) {
	// Arrange
	store := &memReports{}
	deps := ScanDeps{
		Hardware:  fakeHardware{},
		Sensors:   fakeHardware{},
		Devices:   fakeHardware{},
		Overview:  fakeReader{o: domain.Overview{Elevated: true}},
		Live:      fakeReader{},
		Crashes:   fakeCrashes{ev: domain.CrashEvidence{Restarts: []domain.CrashEvent{{}, {}}}},
		Startup:   fakeStartupList{},
		Processes: fakeLister{},
		Runner:    tasks.New(1, func(tasks.Status) {}),
		Reports:   store,
		Resolved:  memResolved{},
	}
	ctx := context.Background()
	s := NewScanService(deps)

	// Act
	res, err := s.Scan(ctx)
	resolveErr := s.Resolve(ctx, "unexpected-restarts", "restarts")
	restarted := NewScanService(deps)
	last, lastErr := restarted.Last(ctx)
	list, listErr := s.Reports(ctx)

	// Assert
	if err != nil || len(res.Findings) != 1 || res.Findings[0].Status != domain.Open || res.ID != "0" {
		t.Fatalf("scan = %+v, %v", res, err)
	}
	if resolveErr != nil || lastErr != nil || last == nil || last.Findings[0].Status != domain.Resolved {
		t.Fatalf("last after restart = %+v, %v %v", last, resolveErr, lastErr)
	}
	if res.Health.Score != 93 || last.Health.Score != 100 { // two restarts are a medium finding
		t.Fatalf("health before resolving %d, after %d", res.Health.Score, last.Health.Score)
	}
	if store.saved[0].Findings[0].Status != "" {
		t.Fatal("a status leaked into the stored report")
	}
	if listErr != nil || len(list) != 1 || list[0].Resolved != 1 || list[0].Open != 0 {
		t.Fatalf("reports = %+v, %v", list, listErr)
	}
}

func TestScanService_DeletingTheLatestReportFallsBackToTheOneBefore(t *testing.T) {
	// Arrange
	ctx := context.Background()
	store := &memReports{}
	s := NewScanService(ScanDeps{
		Hardware: fakeHardware{}, Sensors: fakeHardware{}, Devices: fakeHardware{},
		Overview: fakeReader{o: domain.Overview{Elevated: true}}, Live: fakeReader{},
		Crashes: fakeCrashes{}, Startup: fakeStartupList{}, Processes: fakeLister{},
		Runner: tasks.New(1, func(tasks.Status) {}), Reports: store, Resolved: memResolved{},
	})
	first, _ := s.Scan(ctx)
	second, _ := s.Scan(ctx)

	// Act
	err := s.DeleteReport(ctx, second.ID)
	last, lastErr := s.Last(ctx)

	// Assert
	if err != nil || lastErr != nil || last == nil || last.ID != first.ID {
		t.Fatalf("last = %+v, errs %v %v", last, err, lastErr)
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

type fakeUpdates struct {
	rel        *domain.Release
	downloaded bool
}

func (f *fakeUpdates) Check(context.Context) (*domain.Release, error) { return f.rel, nil }
func (f *fakeUpdates) Download(context.Context) error                 { f.downloaded = true; return nil }
func (f *fakeUpdates) Restart(context.Context) error                  { return nil }

type memSettings map[string]string

func (m memSettings) Get(_ context.Context, key, def string) (string, error) {
	if v, ok := m[key]; ok {
		return v, nil
	}
	return def, nil
}
func (m memSettings) Set(_ context.Context, key, value string) error { m[key] = value; return nil }

func TestUpdateService_OnStartChecksUntilTurnedOff(t *testing.T) {
	// Arrange
	ctx := context.Background()
	src := &fakeUpdates{rel: &domain.Release{Version: "0.2.0"}}
	s := NewUpdateService(UpdateDeps{Source: src, Settings: memSettings{}, Version: "0.1.0"})

	// Act
	first, err := s.OnStart(ctx)
	setErr := s.SetCheckOnStart(ctx, false)
	second, err2 := s.OnStart(ctx)

	// Assert
	if err != nil || first == nil || first.Version != "0.2.0" {
		t.Fatalf("first = %+v, %v", first, err)
	}
	if setErr != nil || err2 != nil || second != nil {
		t.Fatalf("after turning off: %+v, %v %v", second, setErr, err2)
	}
}

func TestUpdateService_DownloadIsLoggedAndDevBuildsSayWhy(t *testing.T) {
	// Arrange
	ctx := context.Background()
	src := &fakeUpdates{rel: &domain.Release{Version: "0.2.0"}}
	j := &memJournal{}
	runner := tasks.New(1, func(tasks.Status) {})
	s := NewUpdateService(UpdateDeps{Source: src, Runner: runner, Journal: j, Version: "0.1.0"})
	dev := NewUpdateService(UpdateDeps{Source: UpdatesUnavailable(errors.New("dev build")), Runner: runner, Journal: j})

	// Act
	rel, err := s.Check(ctx)
	dlErr := s.Download(ctx, rel.Version)
	_, devErr := dev.Check(ctx)

	// Assert
	if err != nil || dlErr != nil || !src.downloaded {
		t.Fatalf("check %v, download %v, downloaded %v", err, dlErr, src.downloaded)
	}
	if len(j.begun) != 1 || j.begun[0].Before != "0.1.0" || j.begun[0].After != "0.2.0" {
		t.Fatalf("journal = %+v", j.begun)
	}
	if devErr == nil || !strings.Contains(devErr.Error(), "dev build") {
		t.Fatalf("dev build check: %v", devErr)
	}
}

func TestRetentionService_OffUntilSetThenDeletesOlderThanTheDays(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	var cuts []time.Time
	prune := func(_ context.Context, before time.Time) (int, error) { cuts = append(cuts, before); return 2, nil }
	s := NewRetentionService(RetentionDeps{Settings: memSettings{}, Prune: []Pruner{prune, prune}, Now: func() time.Time { return now }})

	// Act
	offN, offErr := s.Apply(ctx)
	n, err := s.SetDays(ctx, 30)
	_, badErr := s.SetDays(ctx, -1)

	// Assert
	if offN != 0 || offErr != nil || len(cuts) != 2 {
		t.Fatalf("off: %d %v, cuts %v", offN, offErr, cuts)
	}
	if n != 4 || err != nil || !cuts[0].Equal(now.AddDate(0, 0, -30)) {
		t.Fatalf("30 days: %d %v, cut %v", n, err, cuts[0])
	}
	if badErr == nil {
		t.Fatal("negative days were accepted")
	}
}

type fakeCheck struct{}

func (fakeCheck) RunCheck(context.Context, string) (int, string, error) {
	return 0, "Beginning system scan.\r\nVerification 100% complete.\r\n\r\nWindows Resource Protection did not find any integrity violations.\r\n", nil
}
func (fakeCheck) CheckLog(string) string    { return `C:\Windows\Logs\CBS\CBS.log` }
func (fakeCheck) ShowInFolder(string) error { return nil }

type fakeMemory struct{}

func (fakeMemory) MemoryTests(context.Context, time.Time) ([]domain.CheckResult, error) {
	return []domain.CheckResult{{ID: "memtest-1", Name: "memtest", At: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), FromWindows: true}}, nil
}

type memChecks struct{ saved []domain.CheckResult }

func (m *memChecks) Save(_ context.Context, r domain.CheckResult) (domain.CheckResult, error) {
	r.ID = "id"
	m.saved = append(m.saved, r)
	return r, nil
}
func (m *memChecks) All(context.Context) ([]domain.CheckResult, error) { return m.saved, nil }
func (m *memChecks) Delete(context.Context, string) error              { return nil }

func TestCheckService_SavesChecksButNotTheMemoryTestBooking(t *testing.T) {
	// Arrange
	ctx := context.Background()
	store := &memChecks{}
	s := NewCheckService(CheckDeps{Run: fakeCheck{}, Runner: tasks.New(1, func(tasks.Status) {}), Saved: store, Memory: fakeMemory{}, Now: time.Now})

	// Act
	sfc, err := s.Run(ctx, "sfc")
	_, memErr := s.Run(ctx, "memtest")

	// Assert
	if err != nil || memErr != nil || sfc.ID != "id" || len(store.saved) != 1 || store.saved[0].Name != "sfc" {
		t.Fatalf("sfc %+v %v, memtest %v, saved %+v", sfc, err, memErr, store.saved)
	}
	if store.saved[0].Summary != "Beginning system scan.\nWindows Resource Protection did not find any integrity violations." || store.saved[0].Log == "" {
		t.Fatalf("summary %q, log %q", store.saved[0].Summary, store.saved[0].Log)
	}
	all, _ := s.Results(ctx)
	delErr := s.DeleteResult(ctx, "memtest-1")
	if len(all) != 2 || all[1].Name != "memtest" || delErr == nil {
		t.Fatalf("results %+v, deleting a memory test: %v", all, delErr)
	}
}

type fakeScans struct{}

func (fakeScans) Report(context.Context, string) (domain.ScanResult, error) {
	return domain.ScanResult{Health: domain.Health{Score: 64}}, nil
}

func TestExportService_WritesTheReportWhereChosen(t *testing.T) {
	// Arrange
	files := map[string]string{}
	s := NewExportService(ExportDeps{Scans: fakeScans{}, Write: func(p string, b []byte) error { files[p] = string(b); return nil }})

	// Act
	err := s.Scan(context.Background(), "20261117-090000-000", `C:\Users\me\scan.md`)
	noPath := s.Scan(context.Background(), "x", "")

	// Assert
	if err != nil || !strings.Contains(files[`C:\Users\me\scan.md`], "64 out of 100") {
		t.Fatalf("err %v, files %v", err, files)
	}
	if noPath == nil {
		t.Fatal("an empty path was accepted")
	}
}

func TestProcessService_MarksOurPIDAndWhatItStarted(t *testing.T) {
	// Arrange
	ps := []domain.Process{
		{PID: 10, ParentPID: 1, Name: "reforgepulse.exe"},
		{PID: 11, ParentPID: 10, Name: "reforgepulse.exe"}, // the probe
		{PID: 12, ParentPID: 11, Name: "cdb.exe"},
		{PID: 20, ParentPID: 30, Name: "reforgepulse.exe"}, // an agent's MCP server, the agent's child
		{PID: 30, ParentPID: 30, Name: "claude.exe"},
	}
	s := NewProcessService(ProcessDeps{Lister: fakeLister{ps: ps}, Self: 10})

	// Act
	got, err := s.List(context.Background())

	// Assert
	want := []bool{true, true, true, false, false}
	for i, w := range want {
		if err != nil || got[i].Ours != w {
			t.Fatalf("%s %d ours = %v, want %v (%v)", got[i].Name, got[i].PID, got[i].Ours, w, err)
		}
	}
}

type countingIcons struct{ asked []string }

func (c *countingIcons) Icons(_ context.Context, paths []string) (map[string]string, error) {
	c.asked = append(c.asked, paths...)
	out := map[string]string{}
	for _, p := range paths {
		out[p] = "data:" + p
	}
	return out, nil
}

func TestProcessService_IconsAreAskedForOnce(t *testing.T) {
	// Arrange
	src := &countingIcons{}
	s := NewProcessService(ProcessDeps{Icons: src})
	ctx := context.Background()

	// Act
	_, err1 := s.Icons(ctx, []string{"a.exe", "b.exe"})
	got, err2 := s.Icons(ctx, []string{"a.exe", "c.exe", ""})

	// Assert
	if err1 != nil || err2 != nil || len(src.asked) != 3 || got["a.exe"] != "data:a.exe" || got["c.exe"] != "data:c.exe" {
		t.Fatalf("asked %v, got %v", src.asked, got)
	}
}

func TestProcessService_MarksWhatEndTaskWouldRefuse(t *testing.T) {
	// Arrange
	ps := []domain.Process{
		{PID: 4, Name: "System", Access: true},
		{PID: 700, Name: "csrss.exe", Access: true},
		{PID: 800, Name: "MsMpEng.exe"}, // Windows denies access
		{PID: 900, Name: "notepad.exe", Access: true},
		{PID: 10, Name: "reforgepulse.exe", Access: true},
	}
	s := NewProcessService(ProcessDeps{Lister: fakeLister{ps: ps}, Self: 10})

	// Act
	got, err := s.List(context.Background())

	// Assert
	want := []bool{true, true, true, false, true}
	for i, w := range want {
		if err != nil || got[i].Protected != w {
			t.Fatalf("%s protected = %v, want %v", got[i].Name, got[i].Protected, w)
		}
	}
}

func TestCoreTestSummary_NamesOnlyTheProcessorsWorthALook(t *testing.T) {
	// Arrange
	bad := domain.CoreTest{Seconds: 30, NewWHEA: 1, Results: []domain.CoreTestResult{
		{Index: 0, Rounds: 40}, {Index: 8, Rounds: 40, Errors: 3}, {Index: 9, Skipped: "outside group 0"},
	}}
	good := domain.CoreTest{Seconds: 30, Results: []domain.CoreTestResult{{Index: 0, Rounds: 40}, {Index: 1, Rounds: 40}}}

	// Act
	badText, badCode := coreTestSummary(bad)
	goodText, goodCode := coreTestSummary(good)

	// Assert
	if badCode != 1 || !strings.Contains(badText, "Processor 8 disagreed with the others in 3 of 40 rounds.") ||
		!strings.Contains(badText, "Processor 9 was skipped") || !strings.Contains(badText, "1 hardware errors") || strings.Contains(badText, "Processor 0") {
		t.Fatalf("bad run: %d\n%s", badCode, badText)
	}
	if goodCode != 0 || !strings.Contains(goodText, "Every processor gave the same results") {
		t.Fatalf("good run: %d\n%s", goodCode, goodText)
	}
}

type fakeConn struct{}

func (fakeConn) Connect(context.Context, string) (string, error) { return "", nil }
func (fakeConn) Registered(_ context.Context, agent string) domain.AgentStatus {
	return domain.AgentStatus{Installed: true, Registered: agent != "opencode"}
}

type fakeMcpLog []domain.McpEvent

func (f fakeMcpLog) Recent(context.Context, int) ([]domain.McpEvent, error) { return f, nil }

func TestAgentService_LiveOnlyWhileItsServerRuns(t *testing.T) {
	// Arrange: newest first, as the log reads
	log := fakeMcpLog{
		{PID: 20, Client: "codex-mcp-client", Event: "end"},
		{PID: 10, Client: "claude-code", Event: "call", Tool: "scan_pc"},
		{PID: 20, Client: "codex-mcp-client", Event: "start"},
		{PID: 10, Client: "claude-code", Event: "start"},
	}
	procs := fakeLister{ps: []domain.Process{
		{PID: 90, ParentPID: 1, Name: "opencode.exe"},
		{PID: 91, ParentPID: 90, Name: "reforgepulse.exe"}, // an older copy that keeps no log
	}}
	s := NewAgentService(AgentDeps{Conn: fakeConn{}, Log: log, Procs: procs, Alive: func(pid uint32) bool { return pid == 10 || pid == 20 }})

	// Act
	got, err := s.Status(context.Background())

	// Assert
	if err != nil || len(got) != 3 {
		t.Fatalf("status = %+v, %v", got, err)
	}
	claude, codex, opencode := got[0], got[1], got[2]
	if !claude.Live || len(claude.Events) != 2 || codex.Live || len(codex.Events) != 2 || opencode.Registered || !opencode.Live {
		t.Fatalf("claude %+v\ncodex %+v\nopencode %+v", claude, codex, opencode)
	}
}
