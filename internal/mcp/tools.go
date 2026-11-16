package mcp

import (
	"context"
	"encoding/json"

	"github.com/cn8001/reforgepulse/internal/app"
)

// Deps are the read-only services the tools draw on. Tools never change the
// system: agents may suggest a change, the user approves it in the window.
type Deps struct {
	System  app.OverviewReader
	History app.ChangeLog
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
			Name:        "get_change_log",
			Description: "Every change ReforgePulse made on this PC, newest first, with before and after values and whether it was applied, failed or reverted.",
			Schema:      noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.History.Log(ctx)
			},
		},
	}
}
