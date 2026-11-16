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
	"github.com/cn8001/reforgepulse/internal/inspect/processes"
	"github.com/cn8001/reforgepulse/internal/inspect/programs"
	"github.com/cn8001/reforgepulse/internal/inspect/startup"
	"github.com/cn8001/reforgepulse/internal/inspect/system"
	"github.com/cn8001/reforgepulse/internal/journal"
	"github.com/cn8001/reforgepulse/internal/mcp"
	"github.com/cn8001/reforgepulse/internal/platform/windows"
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

	overview *system.Inspector
	journal  *journal.Journal
	sampler  *windows.Sampler
	procs    *processes.Inspector
	startup  *startup.Inspector
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
	overview := system.New(host, host, sampler)
	runner := tasks.New(maxTasks, notify)
	procs := processes.New(processes.Deps{Processes: host, GPU: sampler, CPUs: runtime.NumCPU()})
	startupApps := startup.New(host, procs)
	startupSvc := app.NewStartupService(startupApps, host, j)
	return &backend{
		Startup: startupSvc,
		Progs: app.NewProgramService(app.ProgramDeps{
			Finder: programs.New(host), Uninstaller: host, Runner: runner, Journal: j,
		}),
		Procs:   app.NewProcessService(procs, procs, j, uint32(os.Getpid())),
		About:   app.NewAboutService(about),
		Cleanup: app.NewCleanupService(cleanup.New(host.CleanupLocations(), j), runner),
		System:  app.NewSystemService(overview, overview, runner),
		Tasks:   app.NewTaskService(runner),
		History: app.NewHistoryService(j, map[string]app.Reverter{"startup": startupSvc}),
		Agent: app.NewAgentService(exe, agents.New(agents.Deps{
			Exe: exe, Home: home, LookPath: exec.LookPath, Run: runProgram,
		}), j),
		overview: overview,
		journal:  j,
		sampler:  sampler,
		procs:    procs,
		startup:  startupApps,
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
	}
}

func (b *backend) tools() []mcp.Tool {
	return mcp.Tools(mcp.Deps{System: b.overview, Live: b.overview, Processes: b.procs, Startup: b.startup, History: b.journal})
}

// Close releases what the backend holds open.
func (b *backend) Close() { b.sampler.Close() }

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
