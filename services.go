package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/cn8001/reforgepulse/internal/app"
	"github.com/cn8001/reforgepulse/internal/collectors/system"
	"github.com/cn8001/reforgepulse/internal/platform/windows"
)

// backend is every service, built once and shared by the window and the MCP
// server so the two cannot drift apart. Wiring only: logic lives in internal/.
type backend struct {
	System *app.SystemService
}

func newBackend(host windows.System) *backend {
	return &backend{
		System: app.NewSystemService(system.New(host, host)),
	}
}

// services lists what the window can call. The binding generator reads the
// static type of each NewService argument, hence the typed fields.
func (b *backend) services() []application.Service {
	return []application.Service{
		application.NewService(b.System),
	}
}
