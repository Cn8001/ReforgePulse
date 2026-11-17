package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/app"
	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

// Deps are the read-only services the tools draw on. Tools never change the
// system: agents may suggest a change, the user approves it in the window.
type Deps struct {
	System    app.OverviewReader
	Live      app.LiveReader
	Processes app.ProcessLister
	Startup   app.StartupLister
	Programs  ProgramLister
	Leftovers app.LeftoverFinder
	Scanner   Scanner
	Power     PowerReader
	Hardware  HardwareInfo
	Devices   app.DeviceReader
	History   app.ChangeLog
}

type HardwareInfo interface {
	HardwareFacts(ctx context.Context) (platform.HardwareFacts, error)
	List(ctx context.Context) ([]domain.CoreInfo, error)
}

type PowerReader interface {
	Plans(ctx context.Context) ([]platform.PowerPlan, error)
}

type Scanner interface {
	Scan(ctx context.Context) (domain.ScanResult, error)
	Evidence(ctx context.Context) (domain.CrashEvidence, error)
	Reports(ctx context.Context) ([]domain.ReportSummary, error)
	Report(ctx context.Context, id string) (domain.ScanResult, error)
}

type ProgramLister interface {
	List(ctx context.Context) ([]domain.InstalledProgram, error)
}

var noArgs = json.RawMessage(`{"type":"object","properties":{}}`)

func Tools(d Deps) []Tool {
	return []Tool{
		{
			Name: "scan_pc",
			Description: "Runs ReforgePulse's full read-only scan and returns findings: crashes, blue screens, WHEA hardware errors, " +
				"Intel microcode, disk errors, app and service failures, memory pressure, full drives and heavy startup. " +
				"Each finding has severity, confidence, evidence and other explanations the evidence allows.",
			Schema: noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.Scanner.Scan(ctx)
			},
		},
		{
			Name: "list_scan_reports",
			Description: "Past scans, newest first: when each ran, how many findings are open and how many the user marked resolved. " +
				"Compare them to see whether a fix held.",
			Schema: noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.Scanner.Reports(ctx)
			},
		},
		{
			Name: "get_scan_report",
			Description: "One past scan by the id list_scan_reports gives. Each finding has a status: open, resolved, or returned " +
				"(marked resolved, then newer evidence turned up). New means the scan before it did not have it.",
			Schema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
			Call: func(ctx context.Context, args json.RawMessage) (any, error) {
				var a struct{ ID string }
				if err := json.Unmarshal(args, &a); err != nil {
					return nil, fmt.Errorf("read arguments: %w", err)
				}
				return d.Scanner.Report(ctx, a.ID)
			},
		},
		{
			Name:        "get_crash_evidence",
			Description: "The raw crash evidence of the last 30 days: blue screen codes, unexpected restarts, WHEA errors with APIC IDs, GPU driver resets, app crashes and hangs, disk errors, failed services, minidumps, microcode and tuning drivers.",
			Schema:      noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.Scanner.Evidence(ctx)
			},
		},
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
			Name:        "list_programs",
			Description: "Installed programs with size, when Windows last saw them start (Prefetch) and whether they look forgotten (unused for 90+ days). Launch history needs ReforgePulse's window running as administrator.",
			Schema:      noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.Programs.List(ctx)
			},
		},
		{
			Name:        "list_leftovers",
			Description: "Folders in Program Files, ProgramData and AppData that no installed program, process, service or startup entry claims and that have not changed for 90 days: likely remains of uninstalled programs. Matching is by name, so review before removing.",
			Schema:      noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.Leftovers.List(ctx)
			},
		},
		{
			Name: "get_hardware",
			Description: "CPU, motherboard, BIOS, every RAM stick (rated vs configured speed), GPUs with driver versions, disk health, " +
				"and every logical processor with its core, P/E type, x2APIC ID and how many WHEA errors named it in 30 days.",
			Schema: noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				f, err := d.Hardware.HardwareFacts(ctx)
				if err != nil {
					return nil, err
				}
				cs, _ := d.Hardware.List(ctx)
				return map[string]any{"hardware": f, "cores": cs}, nil
			},
		},
		{
			Name:        "list_problem_devices",
			Description: "Devices Device Manager marks with a problem, with their hardware ID and error code (28 = no driver installed). Disconnected devices are left out.",
			Schema:      noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.Devices.ProblemDevices(ctx)
			},
		},
		{
			Name:        "get_power_plans",
			Description: "Every Windows power plan with its processor frequency caps in MHz (0 = none): P-cores and E-cores, plugged in and on battery. Changing them is done by the user in ReforgePulse.",
			Schema:      noArgs,
			Call: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return d.Power.Plans(ctx)
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
