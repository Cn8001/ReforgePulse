package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/cn8001/reforgepulse/internal/app"
	"github.com/cn8001/reforgepulse/internal/collectors/system"
	"github.com/cn8001/reforgepulse/internal/journal"
	"github.com/cn8001/reforgepulse/internal/platform/windows"
)

// backend is every service, built once and shared by the window and the MCP
// server so the two cannot drift apart. Wiring only: logic lives in internal/.
type backend struct {
	System  *app.SystemService
	History *app.HistoryService
}

// newBackend keeps its data in %LOCALAPPDATA%\ReforgePulse: os.UserCacheDir
// answers LocalAppData on Windows, which keeps the log off roaming profiles.
func newBackend(host windows.System) (*backend, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("find app data dir: %w", err)
	}
	j, err := journal.Open(filepath.Join(base, "ReforgePulse"))
	if err != nil {
		return nil, fmt.Errorf("open change log: %w", err)
	}
	return &backend{
		System:  app.NewSystemService(system.New(host, host)),
		History: app.NewHistoryService(j),
	}, nil
}

// services lists what the window can call. The binding generator reads the
// static type of each NewService argument, hence the typed fields.
func (b *backend) services() []application.Service {
	return []application.Service{
		application.NewService(b.System),
		application.NewService(b.History),
	}
}
