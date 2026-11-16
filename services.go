package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/cn8001/reforgepulse/internal/app"
	"github.com/cn8001/reforgepulse/internal/inspect/processes"
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

	overview *system.Inspector
	journal  *journal.Journal
	sampler  *windows.Sampler
	procs    *processes.Inspector
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
	sampler := windows.NewSampler()
	overview := system.New(host, host, sampler)
	runner := tasks.New(maxTasks, notify)
	procs := processes.New(host, runtime.NumCPU())
	return &backend{
		Procs:    app.NewProcessService(procs),
		About:    app.NewAboutService(about),
		System:   app.NewSystemService(overview, overview, runner),
		Tasks:    app.NewTaskService(runner),
		History:  app.NewHistoryService(j),
		Agent:    app.NewAgentService(exe),
		overview: overview,
		journal:  j,
		sampler:  sampler,
		procs:    procs,
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
	}
}

func (b *backend) tools() []mcp.Tool {
	return mcp.Tools(mcp.Deps{System: b.overview, Live: b.overview, Processes: b.procs, History: b.journal})
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
