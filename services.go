package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/cn8001/reforgepulse/internal/app"
	"github.com/cn8001/reforgepulse/internal/inspect/system"
	"github.com/cn8001/reforgepulse/internal/journal"
	"github.com/cn8001/reforgepulse/internal/mcp"
	"github.com/cn8001/reforgepulse/internal/platform/windows"
)

// backend is every service, built once and shared by the window and the MCP
// server so the two cannot drift apart. Wiring only: logic lives in internal/.
type backend struct {
	System  *app.SystemService
	History *app.HistoryService
	Agent   *app.AgentService

	overview *system.Inspector
	journal  *journal.Journal
}

func newBackend(host windows.System, dataDir string) (*backend, error) {
	j, err := journal.Open(dataDir)
	if err != nil {
		return nil, fmt.Errorf("open change log: %w", err)
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("find executable: %w", err)
	}
	overview := system.New(host, host)
	return &backend{
		System:   app.NewSystemService(overview),
		History:  app.NewHistoryService(j),
		Agent:    app.NewAgentService(exe),
		overview: overview,
		journal:  j,
	}, nil
}

// services lists what the window can call. The binding generator reads the
// static type of each NewService argument, hence the typed fields.
func (b *backend) services() []application.Service {
	return []application.Service{
		application.NewService(b.System),
		application.NewService(b.History),
		application.NewService(b.Agent),
	}
}

func (b *backend) tools() []mcp.Tool {
	return mcp.Tools(mcp.Deps{System: b.overview, History: b.journal})
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
