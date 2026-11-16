// Package events folds the event logs into kinds of events with counts and
// plain explanations, the part of Event Viewer people actually need.
package events

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

const perChannel = 3000

type known struct {
	explain string
	noise   bool
}

// catalog explains the events that matter most, and marks the ones Windows
// logs constantly for no reason anyone needs to act on.
var catalog = map[string]known{
	"Microsoft-Windows-Kernel-Power/41":               {explain: "Windows restarted without shutting down cleanly: a crash, a freeze, or lost power."},
	"EventLog/6008":                                   {explain: "The shutdown before this start was unexpected."},
	"Microsoft-Windows-WER-SystemErrorReporting/1001": {explain: "A blue screen. The stop code is in the message."},
	"Microsoft-Windows-WHEA-Logger/17":                {explain: "A PCIe device reported a corrected error: often a GPU, NVMe drive or its slot."},
	"Microsoft-Windows-WHEA-Logger/18":                {explain: "A fatal hardware error. Windows had to stop."},
	"Microsoft-Windows-WHEA-Logger/19":                {explain: "The processor corrected a hardware error. Repeats mean instability: voltage, heat, an overclock, or a degraded core."},
	"Microsoft-Windows-WHEA-Logger/47":                {explain: "Memory corrected an error. Repeats point at RAM, its speed profile, or the memory controller."},
	"Display/4101":                                    {explain: "The graphics driver stopped responding and recovered."},
	"disk/7":                                          {explain: "A drive has a bad block. Back up, then check the drive."},
	"disk/51":                                         {explain: "A paging read or write failed: a drive or cable problem."},
	"disk/153":                                        {explain: "Windows had to retry a read or write on a drive."},
	"Microsoft-Windows-Ntfs/55":                       {explain: "The file system found corruption on a drive."},
	"volmgr/161":                                      {explain: "Windows could not write a crash dump, so the blue screen left no file."},
	"Service Control Manager/7000":                    {explain: "A service failed to start."},
	"Service Control Manager/7009":                    {explain: "A service took too long to start."},
	"Service Control Manager/7031":                    {explain: "A service crashed; Windows restarted it."},
	"Service Control Manager/7034":                    {explain: "A service crashed."},
	"Service Control Manager/7023":                    {explain: "A service stopped with an error."},
	"Application Error/1000":                          {explain: "A program crashed. The message names it and the module that failed."},
	"Application Hang/1002":                           {explain: "A program stopped responding and was closed."},
	".NET Runtime/1026":                               {explain: "A .NET program crashed with an unhandled error."},
	"Microsoft-Windows-Kernel-Processor-Power/37":     {explain: "Firmware limited the processor's speed, for heat or power."},
	"Microsoft-Windows-DistributedCOM/10016":          {explain: "A permissions note Windows logs constantly. Harmless; ignore it.", noise: true},
	"Microsoft-Windows-DistributedCOM/10010":          {explain: "A background component did not register in time. Usually harmless.", noise: true},
	"Schannel/36871":                                  {explain: "A secure connection failed to set up. Usually harmless.", noise: true},
	"Microsoft-Windows-Security-SPP/16385":            {explain: "Licensing scheduler noise. Harmless.", noise: true},
	"Microsoft-Windows-Kernel-EventTracing/2":         {explain: "A logging session could not start. Harmless.", noise: true},
}

type Inspector struct {
	log platform.EventLog
	now func() time.Time
}

func New(log platform.EventLog) *Inspector { return &Inspector{log: log, now: time.Now} }

// List groups System and Application events from the last days, newest
// group first. Warnings are included only when asked.
func (in *Inspector) List(ctx context.Context, days int, warnings bool) ([]domain.EventGroup, error) {
	level := 2
	if warnings {
		level = 3
	}
	since := in.now().AddDate(0, 0, -days)
	groups := map[string]*domain.EventGroup{}
	var firstErr error
	for _, ch := range []string{"System", "Application"} {
		evs, err := in.log.Recent(ctx, ch, level, since, perChannel)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("read %s log: %w", ch, err)
		}
		for _, e := range evs {
			key := fmt.Sprintf("%s/%d", e.Provider, e.ID)
			g := groups[ch+"|"+key]
			if g == nil {
				k := catalog[key]
				g = &domain.EventGroup{Channel: ch, Provider: e.Provider, ID: e.ID, Level: e.Level, First: e.Time, Last: e.Time,
					Message: e.Message, Explain: k.explain, Noise: k.noise}
				groups[ch+"|"+key] = g
			}
			g.Count++
			if e.Time.Before(g.First) {
				g.First = e.Time
			}
			if e.Time.After(g.Last) {
				g.Last, g.Message = e.Time, e.Message
			}
			g.Level = min(g.Level, e.Level)
		}
	}
	out := make([]domain.EventGroup, 0, len(groups))
	for _, g := range groups {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Last.After(out[j].Last) })
	if len(out) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}
