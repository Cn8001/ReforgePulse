// Package lifetime ties helper processes to the window: an agent's
// `reforgepulse.exe mcp` ends when the window it was serving alongside
// closes, so nothing of ReforgePulse lingers (or keeps its exe locked).
package lifetime

import (
	"context"
	"time"
)

// Follow calls gone once the window, after having been open, is closed. A
// helper started while no window was open keeps running: ending it at once
// would only break the agent that started it.
func Follow(ctx context.Context, windowOpen func() bool, every time.Duration, gone func()) {
	seen := false
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		open := windowOpen()
		if open {
			seen = true
		} else if seen {
			gone()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
