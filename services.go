package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/cn8001/reforgepulse/internal/agents"
	"github.com/cn8001/reforgepulse/internal/app"
	"github.com/cn8001/reforgepulse/internal/cleanup"
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
	"github.com/cn8001/reforgepulse/internal/recorder"
	"github.com/cn8001/reforgepulse/internal/tasks"
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

	overview *system.Inspector
	journal  *journal.Journal
	sampler  *windows.Sampler
	cpus     *windows.CPUs
	sensors  *windows.SensorReader
	host     windows.System
	cores    *cores.Inspector
	procs    *processes.Inspector
	startup  *startup.Inspector
	programs *programs.Inspector
	left     *leftovers.Inspector
}

// maxTasks bounds how much work runs at once, so a scan never starves the
// PC it is diagnosing.
const maxTasks = 4

func newBackend(host windows.System, dataDir string, notify tasks.Notifier) (*backend, error) {
	about := app.About{Version: app.VersionFrom(buildConfig), DataDir: dataDir, Terms: terms}
	j, err := journal.Open(dataDir)
	if err != nil {
		return nil, fmt.Errorf("open change log: %w", err)
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("find executable: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("find home folder: %w", err)
	}
	sampler := windows.NewSampler()
	cpus := windows.NewCPUs()
	sensors := windows.NewSensorReader()
	coreMap := cores.New(cpus, host)
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
		Hardware: host, Sensors: sensors, Overview: overview, Live: overview, Crashes: crashes.New(host), Startup: startupApps, Processes: procs, Runner: runner,
	})
	return &backend{
		Startup: startupSvc,
		Scan:    scan,
		CPUCap:  cpuCap,
		Feats:   feats,
		Sens:    app.NewSensorService(sensors),
		Events:  app.NewEventService(events.New(host), runner),
		Checks:  app.NewCheckService(host, runner),
		Rec:     app.NewRecorderService(recorder.New(overview, procs), runner),
		Dumps: app.NewDumpService(dumps.New(dumps.Deps{
			Dir: host.DumpDir(), Symbols: filepath.Join(dataDir, "symbols"), Debugger: host, Names: diagnosis.BugcheckNames,
		}), runner),
		HW: app.NewHardwareService(app.HardwareDeps{
			Facts: host, Cores: coreMap, Loads: cpus, Runner: runner,
			Tester: coretest.New(coretest.Deps{CPUs: cpus, WHEA: coreMap.WHEAByAPIC}),
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
		Procs:   app.NewProcessService(procs, procs, j, uint32(os.Getpid())),
		About:   app.NewAboutService(about),
		Cleanup: app.NewCleanupService(cleanup.New(host.CleanupLocations(), j), runner),
		System:  app.NewSystemService(overview, overview, runner),
		Tasks:   app.NewTaskService(runner),
		History: app.NewHistoryService(j, map[string]app.Reverter{"startup": startupSvc, "cpu-cap": cpuCap, "feature": feats}),
		Agent: app.NewAgentService(exe, agents.New(agents.Deps{
			Exe: exe, Home: home, LookPath: exec.LookPath, Run: runProgram,
		}), j),
		overview: overview,
		journal:  j,
		sampler:  sampler,
		cpus:     cpus,
		sensors:  sensors,
		host:     host,
		cores:    coreMap,
		procs:    procs,
		startup:  startupApps,
		programs: progs,
		left:     left,
	}, nil
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
	}
}

func (b *backend) tools() []mcp.Tool {
	return mcp.Tools(mcp.Deps{System: b.overview, Live: b.overview, Processes: b.procs, Startup: b.startup,
		Programs: b.programs, Leftovers: b.left, Scanner: b.Scan, Power: b.CPUCap,
		Hardware: hwInfo{b.host, b.cores}, History: b.journal})
}

// Close releases what the backend holds open.
func (b *backend) Close() {
	b.sampler.Close()
	b.cpus.Close()
	b.sensors.Close()
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
	*cores.Inspector
}
