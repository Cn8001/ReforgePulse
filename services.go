package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/cn8001/reforgepulse/internal/agents"
	"github.com/cn8001/reforgepulse/internal/app"
	"github.com/cn8001/reforgepulse/internal/cleanup"
	"github.com/cn8001/reforgepulse/internal/consent"
	"github.com/cn8001/reforgepulse/internal/coretest"
	"github.com/cn8001/reforgepulse/internal/diagnosis"
	"github.com/cn8001/reforgepulse/internal/inspect/cores"
	"github.com/cn8001/reforgepulse/internal/inspect/crashes"
	"github.com/cn8001/reforgepulse/internal/inspect/dumps"
	"github.com/cn8001/reforgepulse/internal/inspect/events"
	"github.com/cn8001/reforgepulse/internal/inspect/features"
	"github.com/cn8001/reforgepulse/internal/inspect/leftovers"
	"github.com/cn8001/reforgepulse/internal/inspect/oldfiles"
	"github.com/cn8001/reforgepulse/internal/inspect/processes"
	"github.com/cn8001/reforgepulse/internal/inspect/programs"
	"github.com/cn8001/reforgepulse/internal/inspect/startup"
	"github.com/cn8001/reforgepulse/internal/inspect/system"
	"github.com/cn8001/reforgepulse/internal/journal"
	"github.com/cn8001/reforgepulse/internal/mcp"
	"github.com/cn8001/reforgepulse/internal/platform/windows"
	"github.com/cn8001/reforgepulse/internal/probe"
	"github.com/cn8001/reforgepulse/internal/recorder"
	"github.com/cn8001/reforgepulse/internal/reports"
	"github.com/cn8001/reforgepulse/internal/store"
	"github.com/cn8001/reforgepulse/internal/tasks"
	"github.com/cn8001/reforgepulse/internal/updates"
)

// backend is every service, built once and shared by the window and the MCP
// server so the two cannot drift apart. Wiring only: logic lives in internal/.
type backend struct {
	System  *app.SystemService
	History *app.HistoryService
	Agent   *app.AgentService
	Tasks   *app.TaskService
	Procs   *app.ProcessService
	About   *app.AboutService
	Cleanup *app.CleanupService
	Startup *app.StartupService
	Progs   *app.ProgramService
	Left    *app.LeftoverService
	Old     *app.OldFilesService
	Scan    *app.ScanService
	CPUCap  *app.CPUCapService
	Feats   *app.FeatureService
	HW      *app.HardwareService
	Sens    *app.SensorService
	Events  *app.EventService
	Checks  *app.CheckService
	Rec     *app.RecorderService
	Dumps   *app.DumpService
	Drivers *app.DriverService
	Keep    *app.RetentionService
	Export  *app.ExportService

	overview *system.Inspector
	journal  *journal.Journal
	db       *sql.DB
	mcpLog   *store.McpLog
	runner   *tasks.Runner
	sampler  *windows.Sampler
	hw       hardwareReads
	host     windows.System
	procs    *processes.Inspector
	startup  *startup.Inspector
	programs *programs.Inspector
	left     *leftovers.Inspector
}

// keepReports is how many past scans stay on disk: a month of daily scans,
// a few hundred KB.
const keepReports = 30

// keepRecordings is how many recorder sessions stay saved; each is at most
// a few hundred points per series, so they stay small.
const keepRecordings = 20

// keepMcpLog is how many lines of agents' MCP activity stay; enough to see
// the last few sessions of each agent.
const keepMcpLog = 500

// keepChecks is how many health check results stay saved.
const keepChecks = 50

// keepChanges caps the change log; older entries are dropped at start-up.
// Years of normal use, and still only a few hundred KB.
const keepChanges = 500

// maxTasks bounds how much work runs at once, so a scan never starves the
// PC it is diagnosing.
const maxTasks = 4

func newBackend(host windows.System, dataDir string, notify tasks.Notifier, hw hardwareReads) (*backend, error) {
	about := app.About{
		Version: app.VersionFrom(buildConfig), DataDir: dataDir, Terms: terms, License: license,
		Source: "https://github.com/" + updates.Repository, Wording: consent.Version(terms, license),
	}
	ctx := context.Background()
	db, err := store.Open(ctx, filepath.Join(dataDir, "reforgepulse.db"))
	if err != nil {
		return nil, err
	}
	if err := store.ImportFiles(ctx, db, dataDir); err != nil {
		db.Close()
		return nil, fmt.Errorf("import older data: %w", err)
	}
	accepted := consent.New(db)
	j := journal.New(db)
	if _, err := j.Prune(ctx, keepChanges); err != nil {
		return nil, fmt.Errorf("trim change log: %w", err)
	}
	recordings, analyses := store.NewRecordings(db, keepRecordings), store.NewAnalyses(db)
	checks := store.NewChecks(db, keepChecks)
	mcpLog := store.NewMcpLog(db, keepMcpLog)
	reportStore := reports.New(db, keepReports)
	resolved := reports.NewResolutions(db)
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("find executable: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("find home folder: %w", err)
	}
	sampler := windows.NewSampler()
	overview := system.New(host, host, sampler)
	runner := tasks.New(maxTasks, notify)
	procs := processes.New(processes.Deps{Processes: host, GPU: sampler, CPUs: runtime.NumCPU()})
	startupApps := startup.New(host, procs)
	startupSvc := app.NewStartupService(startupApps, host, j)
	cpuCap := app.NewCPUCapService(host, j)
	feats := app.NewFeatureService(app.FeatureDeps{List: features.New(host), Switch: host, Runner: runner, Journal: j})
	progs := programs.New(host)
	left := leftovers.New(leftovers.Deps{Machine: host, Startup: host, Processes: procs, Files: host})
	scan := app.NewScanService(app.ScanDeps{
		Hardware: host, Sensors: hw.Sensors, Devices: host, Overview: overview, Live: overview, Crashes: crashes.New(host), Startup: startupApps, Processes: procs, Runner: runner,
		Reports: reportStore, Resolved: resolved,
	})
	b := &backend{
		Startup: startupSvc,
		Scan:    scan,
		CPUCap:  cpuCap,
		Feats:   feats,
		Sens:    app.NewSensorService(hw.Sensors),
		Events:  app.NewEventService(events.New(host), runner),
		Checks:  app.NewCheckService(app.CheckDeps{Run: host, Runner: runner, Saved: checks, Memory: host, Now: time.Now}),
		Drivers: app.NewDriverService(host, runner),
		Rec:     app.NewRecorderService(recorder.New(overview, procs), runner, recordings),
		Keep: app.NewRetentionService(app.RetentionDeps{
			Settings: store.NewSettings(db), Now: time.Now,
			Prune: []app.Pruner{j.DeleteBefore, scan.DeleteReportsBefore, recordings.DeleteBefore, analyses.DeleteBefore, checks.DeleteBefore},
		}),
		Dumps: app.NewDumpService(dumps.New(dumps.Deps{
			Dir: host.DumpDir(), Symbols: filepath.Join(dataDir, "symbols"), Debugger: host, Names: diagnosis.BugcheckNames,
		}), runner, analyses),
		HW: app.NewHardwareService(app.HardwareDeps{
			Facts: host, Cores: hw.Cores, Loads: hw.Loads, Tester: hw.Tester, Runner: runner, Saved: checks, Now: time.Now,
		}),
		Old: app.NewOldFilesService(app.LeftoverDeps{
			What:   "old downloads",
			Finder: oldfiles.New([]oldfiles.Folder{{Name: "Downloads", Path: filepath.Join(home, "Downloads")}}),
			Bin:    host, Runner: runner, Journal: j,
		}),
		Left: app.NewLeftoverService(app.LeftoverDeps{
			What:   "leftovers",
			Finder: left,
			Bin:    host, Runner: runner, Journal: j,
		}),
		Progs: app.NewProgramService(app.ProgramDeps{
			Finder: progs, Uninstaller: host, Runner: runner, Journal: j,
		}),
		Procs:   app.NewProcessService(app.ProcessDeps{Lister: procs, Killer: procs, Journal: j, Icons: hw.Icons, Self: uint32(os.Getpid())}),
		About:   app.NewAboutService(about, accepted),
		Cleanup: app.NewCleanupService(cleanup.New(host.CleanupLocations(), j), runner),
		System:  app.NewSystemService(overview, overview, runner),
		Tasks:   app.NewTaskService(runner),
		History: app.NewHistoryService(j, map[string]app.Reverter{"startup": startupSvc, "cpu-cap": cpuCap, "feature": feats}),
		Agent: app.NewAgentService(app.AgentDeps{
			Exe: exe, Journal: j, Log: mcpLog, Alive: host.Alive,
			Conn: agents.New(agents.Deps{Exe: exe, Home: home, LookPath: exec.LookPath, Run: runProgram}),
		}),
		mcpLog:   mcpLog,
		overview: overview,
		journal:  j,
		db:       db,
		runner:   runner,
		sampler:  sampler,
		hw:       hw,
		host:     host,
		procs:    procs,
		startup:  startupApps,
		programs: progs,
		left:     left,
	}
	b.Export = app.NewExportService(app.ExportDeps{
		Scans: b.Scan, Analyses: b.Dumps, Recordings: b.Rec, Checks: b.Checks,
		Write: func(path string, data []byte) error { return os.WriteFile(path, data, 0o644) },
	})
	return b, nil
}

// services lists what the window can call. The binding generator reads the
// static type of each NewService argument, hence the typed fields.
func (b *backend) services() []application.Service {
	return []application.Service{
		application.NewService(b.System),
		application.NewService(b.History),
		application.NewService(b.Agent),
		application.NewService(b.Tasks),
		application.NewService(b.Procs),
		application.NewService(b.About),
		application.NewService(b.Cleanup),
		application.NewService(b.Startup),
		application.NewService(b.Progs),
		application.NewService(b.Left),
		application.NewService(b.Old),
		application.NewService(b.Scan),
		application.NewService(b.CPUCap),
		application.NewService(b.Feats),
		application.NewService(b.HW),
		application.NewService(b.Sens),
		application.NewService(b.Events),
		application.NewService(b.Checks),
		application.NewService(b.Rec),
		application.NewService(b.Dumps),
		application.NewService(b.Drivers),
		application.NewService(b.Keep),
		application.NewService(b.Export),
	}
}

func (b *backend) tools() []mcp.Tool {
	return mcp.Tools(mcp.Deps{System: b.overview, Live: b.overview, Processes: b.procs, Startup: b.startup,
		Programs: b.programs, Leftovers: b.left, Scanner: b.Scan, Power: b.CPUCap,
		Hardware: hwInfo{b.host, b.hw.Cores}, Devices: b.host, History: b.journal})
}

// Close releases what the backend holds open.
func (b *backend) Close() {
	b.sampler.Close()
	b.hw.Close()
	b.db.Close()
}

// dataDir is the one folder ReforgePulse keeps everything in, as bmeditor
// does: %LOCALAPPDATA%\ReforgePulse, which os.UserCacheDir answers on
// Windows. Local rather than roaming, so it never follows the user around.
func dataDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find app data folder: %w", err)
	}
	return filepath.Join(base, "ReforgePulse"), nil
}

// runProgram runs an agent CLI and returns what it printed.
func runProgram(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

// hwInfo joins the two hardware readers the MCP tool needs.
type hwInfo struct {
	windows.System
	app.CoreLister
}

// hardwareReads are the reads that touch the hardware directly. The window
// gets them from the probe child, so a crash there cannot close it; the MCP
// server, already a process of its own, reads in process.
type hardwareReads struct {
	Cores   app.CoreLister
	Loads   app.CoreLoads
	Sensors app.SensorReader
	Tester  app.CoreTester
	Icons   app.IconSource
	Close   func()
}

// localIcons reads icons in this process, for the MCP server and the probe.
type localIcons struct{ host windows.System }

func (l localIcons) Icons(ctx context.Context, paths []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range paths {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		out[p] = ""
		if b, err := l.host.IconPNG(p); err == nil {
			out[p] = "data:image/png;base64," + base64.StdEncoding.EncodeToString(b)
		}
	}
	return out, nil
}

// localHardware reads in this process. The probe child serves these.
func localHardware(host windows.System) hardwareReads {
	cpus := windows.NewCPUs()
	sensors := windows.NewSensorReader()
	coreMap := cores.New(cpus, host)
	return hardwareReads{
		Cores: coreMap, Loads: cpus, Sensors: sensors, Icons: localIcons{host},
		Tester: coretest.New(coretest.Deps{CPUs: cpus, WHEA: coreMap.WHEAByAPIC}),
		Close:  func() { cpus.Close(); sensors.Close() },
	}
}

// probeHardware reads through a `reforgepulse.exe probe` child, started on
// first use and again after it dies. Its crash reports go to the log.
func probeHardware(exe string, logw io.Writer, logger *slog.Logger) hardwareReads {
	c := probe.NewClient(probe.Command(exe, logw, "probe"), func(err error) {
		logger.Warn("hardware probe ended", "error", err)
	})
	r := probe.NewRemote(c)
	return hardwareReads{Cores: r, Loads: r, Sensors: r, Tester: r, Icons: r, Close: c.Close}
}
