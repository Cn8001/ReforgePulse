package mcp

import (
	"context"
	"encoding/json"

	"github.com/cn8001/reforgepulse/internal/app"
)

// Deps are the read-only services the tools draw on. Tools never change the
// system: agents may suggest a change, the user approves it in the window.
type Deps struct {
	System    app.OverviewReader
	Live      app.LiveReader
	Processes app.ProcessLister
	Startup   app.StartupLister
	History   app.ChangeLog
}

var noArgs = json.RawMessage(`{"type":"object","properties":{}}`)

func Tools(d Deps) []Tool {
	return []Tool{
		{
			Name:        "get_system_overview",
			Description: "OS, CPU, memory use, uptime and fixed drives of this PC. Each value says whether it was measured, unsupported or unknown.",
			Schema:      noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.System.Overview(ctx)
			},
		},
		{
			Name:        "get_live_load",
			Description: "Current CPU, GPU (3D engines) and memory load in percent. CPU and GPU compare against the previous call, so call twice a second apart for a fresh reading.",
			Schema:      noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.Live.Live(ctx)
			},
		},
		{
			Name:        "list_processes",
			Description: "Running processes, busiest first: PID, parent, name, path, threads, CPU percent of the whole machine, working set and private bytes. CPU compares against the previous call. access=false means Windows refused to open the process.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"limit":{"type":"integer","description":"How many to return, default 25"}}}`),
			Call: func(ctx context.Context, args json.RawMessage) (any, error) {
				var a struct{ Limit int }
				_ = json.Unmarshal(args, &a) // no arguments means the default
				if a.Limit <= 0 {
					a.Limit = 25
				}
				ps, err := d.Processes.List(ctx)
				if err != nil {
					return nil, err
				}
				return ps[:min(a.Limit, len(ps))], nil
			},
		},
		{
			Name:        "list_startup_apps",
			Description: "Programs Windows starts at sign-in: whether each is enabled, where it is registered, and whether it is running now with how much memory.",
			Schema:      noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.Startup.List(ctx)
			},
		},
		{
			Name:        "get_change_log",
			Description: "Every change ReforgePulse made on this PC, newest first, with before and after values and whether it was applied, failed or reverted.",
			Schema:      noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.History.Log(ctx)
			},
		},
	}
}
