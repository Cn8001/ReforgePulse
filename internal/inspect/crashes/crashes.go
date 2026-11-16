// Package crashes collects what Windows recorded about crashes and hardware
// errors, the same evidence crash-diag.ps1 gathered by hand.
package crashes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

const (
	window   = 30 * 24 * time.Hour
	maxEvent = 200
)

// tuning matches drivers of overclocking, monitoring and RGB tools that
// write to the CPU or the board directly; crash dumps often show them loaded.
var tuning = regexp.MustCompile(`(?i)iocbios|ntiolib|winring0|rtcore|xtu|throttle|asio|glckio|eneio|magdrv|inpoutx|cpuz|hwinfo`)

type Inspector struct {
	m   platform.Machine
	now func() time.Time
}

func New(m platform.Machine) *Inspector { return &Inspector{m: m, now: time.Now} }

// Collect reads the last 30 days. A source that fails is left empty and
// the first such error is kept, so the rest still counts.
func (in *Inspector) Collect(ctx context.Context) (domain.CrashEvidence, error) {
	since := in.now().Add(-window)
	ev := domain.CrashEvidence{Since: since}
	var errs []error
	read := func(provider string, ids []uint32, detail func(platform.Event) string) []domain.CrashEvent {
		return in.read(ctx, "System", provider, ids, since, detail, &errs)
	}
	readApp := func(provider string, ids []uint32, detail func(platform.Event) string) []domain.CrashEvent {
		return in.read(ctx, "Application", provider, ids, since, detail, &errs)
	}
	ev.Bugchecks = read("Microsoft-Windows-WER-SystemErrorReporting", []uint32{1001}, func(e platform.Event) string {
		code, _, _ := strings.Cut(e.Data["param1"], " ")
		return hexCode(code, 16)
	})
	ev.Restarts = read("Microsoft-Windows-Kernel-Power", []uint32{41}, func(e platform.Event) string {
		return hexCode(e.Data["BugcheckCode"], 10)
	})
	ev.WHEA = read("Microsoft-Windows-WHEA-Logger", nil, func(e platform.Event) string {
		if a, ok := e.Data["ApicId"]; ok {
			return "APIC " + a
		}
		return e.Data["PrimaryDeviceName"]
	})
	ev.GPUResets = read("Display", []uint32{4101}, func(e platform.Event) string { return e.Data["param1"] })
	ev.DiskErrors = append(read("disk", []uint32{7, 51, 153}, device), read("Microsoft-Windows-Ntfs", []uint32{55}, device)...)
	ev.Services = read("Service Control Manager", []uint32{7000, 7009, 7031, 7034}, func(e platform.Event) string {
		if e.ID == 7009 {
			return e.Data["param2"] // 7009 puts the timeout first and the service second
		}
		return e.Data["param1"]
	})
	ev.AppCrashes = append(readApp("Application Error", []uint32{1000}, func(e platform.Event) string {
		app, mod := field(e, "AppName", "param1"), field(e, "ModuleName", "param4")
		if mod != "" && !strings.EqualFold(mod, app) {
			return app + " in " + mod
		}
		return app
	}), readApp(".NET Runtime", []uint32{1026}, func(e platform.Event) string { return ".NET app" })...)
	ev.AppHangs = readApp("Application Hang", []uint32{1002}, func(e platform.Event) string { return field(e, "AppName", "param1") })
	if err := ctx.Err(); err != nil {
		return domain.CrashEvidence{}, err
	}
	if len(errs) > 0 {
		ev.EventsError = errs[0].Error()
	}

	ev.Minidumps = minidumps(in.m.DumpDir())
	ev.Microcode, _ = in.m.Microcode(ctx)
	if svcs, err := in.m.Services(ctx); err == nil {
		for _, s := range svcs {
			if s.Start != 4 && (tuning.MatchString(s.Name) || tuning.MatchString(s.ImagePath)) {
				ev.Tuning = append(ev.Tuning, s.Name)
			}
		}
	}
	return ev, nil
}

func (in *Inspector) read(ctx context.Context, channel, provider string, ids []uint32, since time.Time,
	detail func(platform.Event) string, errs *[]error) []domain.CrashEvent {
	events, err := in.m.Events(ctx, channel, provider, ids, since, maxEvent)
	if err != nil {
		*errs = append(*errs, err)
	}
	out := make([]domain.CrashEvent, 0, len(events))
	for _, e := range events {
		out = append(out, domain.CrashEvent{Time: e.Time, ID: e.ID, Detail: detail(e)})
	}
	return out
}

// field reads EventData by name, falling back to its position: older event
// manifests leave the fields unnamed.
func field(e platform.Event, name, position string) string {
	if v := e.Data[name]; v != "" {
		return v
	}
	return e.Data[position]
}

func device(e platform.Event) string { return field(e, "DeviceName", "param1") }

// hexCode turns "0x0000003b" or "59" into "0x3B"; zero and junk become "".
func hexCode(s string, base int) string {
	s = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s)), "0x")
	n, err := strconv.ParseUint(s, base, 64)
	if err != nil || n == 0 {
		return ""
	}
	return fmt.Sprintf("0x%X", n)
}

func minidumps(dir string) []string {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) || err != nil {
		return nil
	}
	type dump struct {
		name string
		at   time.Time
	}
	var ds []dump
	for _, e := range entries {
		if info, err := e.Info(); err == nil && !e.IsDir() {
			ds = append(ds, dump{e.Name(), info.ModTime()})
		}
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i].at.After(ds[j].at) })
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = fmt.Sprintf("%s (%s)", d.name, d.at.Format("2006-01-02 15:04"))
	}
	return out
}
