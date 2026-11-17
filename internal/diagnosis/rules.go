// Package diagnosis turns evidence into findings with plain, deterministic
// rules. A rule states what the evidence shows and how sure that is; it
// never presents a correlation as the cause.
package diagnosis

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

// Input is everything a scan gathered.
type Input struct {
	Hardware  platform.HardwareFacts
	Sensors   platform.SensorStatus
	Overview  domain.Overview
	Live      domain.Live
	Crashes   domain.CrashEvidence
	Startup   []domain.StartupApp
	Processes []domain.Process
	Devices   []platform.ProblemDevice
}

type rule func(Input) []domain.Finding

var rules = []rule{
	restarts, bugchecks, whea, intelVmin, gpuResets, diskErrors, appCrashes, serviceFailures,
	tuningTools, memoryPressure, lowDisk, heavyStartup, hungApps, problemDevices, notElevated,
}

// Evaluate runs every rule, most severe first.
func Evaluate(in Input) []domain.Finding {
	var out []domain.Finding
	for _, r := range rules {
		for _, f := range r(in) {
			if f.Key == "" {
				f.Key = f.Rule // one finding per rule, so the rule names it
			}
			out = append(out, f)
		}
	}
	rank := map[domain.Severity]int{domain.Critical: 0, domain.High: 1, domain.Medium: 2, domain.Low: 3, domain.Info: 4}
	sort.SliceStable(out, func(i, j int) bool { return rank[out[i].Severity] < rank[out[j].Severity] })
	return out
}

// BugcheckNames are the stop codes worth a name; the rest show as hex.
var BugcheckNames = map[string]string{
	"0xA": "IRQL_NOT_LESS_OR_EQUAL", "0x1A": "MEMORY_MANAGEMENT", "0x1E": "KMODE_EXCEPTION_NOT_HANDLED",
	"0x3B": "SYSTEM_SERVICE_EXCEPTION", "0x50": "PAGE_FAULT_IN_NONPAGED_AREA", "0x7E": "SYSTEM_THREAD_EXCEPTION_NOT_HANDLED",
	"0x9F": "DRIVER_POWER_STATE_FAILURE", "0xD1": "DRIVER_IRQL_NOT_LESS_OR_EQUAL", "0xEF": "CRITICAL_PROCESS_DIED",
	"0x101": "CLOCK_WATCHDOG_TIMEOUT", "0x116": "VIDEO_TDR_FAILURE", "0x124": "WHEA_UNCORRECTABLE_ERROR",
	"0x133": "DPC_WATCHDOG_VIOLATION", "0x139": "KERNEL_SECURITY_CHECK_FAILURE", "0x154": "UNEXPECTED_STORE_EXCEPTION",
	"0x162": "KERNEL_AUTO_BOOST_INVALID_LOCK_RELEASE", "0x20001": "HYPERVISOR_ERROR",
}

func restarts(in Input) []domain.Finding {
	n := len(in.Crashes.Restarts)
	if n == 0 {
		return nil
	}
	sev := domain.Medium
	if n >= 3 {
		sev = domain.High
	}
	var with int
	for _, r := range in.Crashes.Restarts {
		if r.Detail != "" {
			with++
		}
	}
	return []domain.Finding{{
		Rule: "unexpected-restarts", Severity: sev, Confidence: domain.Confirmed,
		Title:        fmt.Sprintf("Windows restarted unexpectedly %d times in 30 days", n),
		Summary:      fmt.Sprintf("%d of them recorded a blue screen code. The others lost power or froze without one, which points away from software.", with),
		Evidence:     times("Kernel-Power 41", in.Crashes.Restarts),
		Latest:       newest(in.Crashes.Restarts, ""),
		Alternatives: "A power cut, holding the power button, an unstable overclock or undervolt, an overheating or failing PSU, or a driver that froze the PC can all look the same here.",
	}}
}

func bugchecks(in Input) []domain.Finding {
	if len(in.Crashes.Bugchecks) == 0 {
		return nil
	}
	counts := map[string]int{}
	for _, b := range in.Crashes.Bugchecks {
		counts[b.Detail]++
	}
	var codes []string
	for c, n := range counts {
		name := BugcheckNames[c]
		if name == "" {
			name = "stop code"
		}
		codes = append(codes, fmt.Sprintf("%s %s ×%d", c, name, n))
	}
	sort.Strings(codes)
	f := domain.Finding{
		Rule: "bugchecks", Severity: domain.High, Confidence: domain.Confirmed,
		Title:    fmt.Sprintf("%d blue screens in 30 days", len(in.Crashes.Bugchecks)),
		Summary:  "Codes: " + strings.Join(codes, ", ") + ".",
		Evidence: append(times("WER 1001", in.Crashes.Bugchecks), in.Crashes.Minidumps...),
		Latest:   newest(in.Crashes.Bugchecks, ""),
		Alternatives: "Different codes from one PC usually mean one unstable part (CPU, memory, power) rather than many bad drivers; " +
			"the same code every time points more to one driver.",
	}
	if counts["0x124"] > 0 || counts["0x101"] > 0 {
		f.Summary += " WHEA_UNCORRECTABLE_ERROR and CLOCK_WATCHDOG_TIMEOUT are raised by the processor itself."
	}
	return []domain.Finding{f}
}

func whea(in Input) []domain.Finding {
	n := len(in.Crashes.WHEA)
	if n == 0 {
		return nil
	}
	apic := map[string]int{}
	for _, e := range in.Crashes.WHEA {
		if strings.HasPrefix(e.Detail, "APIC ") {
			apic[e.Detail]++
		}
	}
	f := domain.Finding{
		Rule: "whea", Severity: domain.High, Confidence: domain.Likely,
		Title:    fmt.Sprintf("Windows recorded %d hardware errors", n),
		Summary:  "The processor, memory or PCIe bus reported errors. Repeated errors are a stability problem even when Windows corrected them.",
		Evidence: times("WHEA-Logger", in.Crashes.WHEA),
		Latest:   newest(in.Crashes.WHEA, ""),
		Alternatives: "An overclock, undervolt or XMP profile, an outdated BIOS, heat, power delivery, " +
			"or a degraded CPU can all cause these; the evidence does not name one part.",
	}
	if len(apic) > 0 {
		f.Summary += " By processor: " + apicSummary(apic) + "."
		f.Summary += " Errors that keep coming from the same few processors make a degraded core possible, but not proven."
	}
	return []domain.Finding{f}
}

// desktopRaptorLake matches 13th and 14th generation desktop Core i5/i7/i9,
// the parts Intel's Vmin Shift Instability affects; laptop suffixes do not.
var desktopRaptorLake = regexp.MustCompile(`(?i)\bi[579]-1[34]\d{3}(K|KF|KS|F|T)?\b`)

// fixedMicrocode is Intel's 0x12B update for the Vmin Shift Instability.
const fixedMicrocode = 0x12B

func intelVmin(in Input) []domain.Finding {
	m := in.Crashes.Microcode
	if !desktopRaptorLake.MatchString(in.Overview.CPUModel) || m == 0 || m >= fixedMicrocode {
		return nil
	}
	return []domain.Finding{{
		Rule: "intel-vmin", Severity: domain.High, Confidence: domain.Likely,
		Title: "Your CPU needs Intel's stability microcode",
		Summary: fmt.Sprintf("%s runs microcode 0x%X. Intel's fix for 13th and 14th gen instability is 0x12B or newer, "+
			"delivered by a BIOS update from your board maker. Until then, damage can build up over time.", in.Overview.CPUModel, m),
		Evidence: []string{fmt.Sprintf("microcode 0x%X < 0x12B", m), in.Overview.CPUModel},
		Action:   &domain.Action{Label: "Cap CPU frequency until the BIOS is updated", Kind: "cpu-cap"},
	}}
}

func gpuResets(in Input) []domain.Finding {
	n := len(in.Crashes.GPUResets)
	if n == 0 {
		return nil
	}
	return []domain.Finding{{
		Rule: "gpu-resets", Severity: domain.Medium, Confidence: domain.Confirmed,
		Title:        fmt.Sprintf("The graphics driver stopped responding %d times", n),
		Summary:      "Windows reset the display driver to recover. Repeats usually mean a driver problem, an unstable GPU overclock, or power to the card.",
		Evidence:     times("Display 4101", in.Crashes.GPUResets),
		Latest:       newest(in.Crashes.GPUResets, ""),
		Alternatives: "A game or app pushing the GPU too hard can trigger this once without anything being wrong.",
	}}
}

func diskErrors(in Input) []domain.Finding {
	n := len(in.Crashes.DiskErrors)
	if n == 0 {
		return nil
	}
	return []domain.Finding{{
		Rule: "disk-errors", Severity: domain.Critical, Confidence: domain.Likely,
		Title:        fmt.Sprintf("A drive reported %d read or file system errors", n),
		Summary:      "Back up important files first. Bad blocks and failed reads can mean a failing drive or a loose cable.",
		Evidence:     times("disk/Ntfs", in.Crashes.DiskErrors),
		Latest:       newest(in.Crashes.DiskErrors, ""),
		Alternatives: "A USB drive pulled out without ejecting, or a sleeping external disk, also logs these.",
	}}
}

// manyApps is when crashes stop being about the apps: this many different
// programs failing on one PC that also blue-screens points at the PC.
const manyApps = 5

func appCrashes(in Input) []domain.Finding {
	distinct := map[string]bool{}
	for _, e := range in.Crashes.AppCrashes {
		distinct[strings.SplitN(e.Detail, " in ", 2)[0]] = true
	}
	unstable := len(in.Crashes.Bugchecks)+len(in.Crashes.WHEA)+len(in.Crashes.Restarts) > 0
	if len(distinct) >= manyApps && unstable {
		return []domain.Finding{{
			Rule: "system-instability", Severity: domain.High, Confidence: domain.Likely,
			Title: fmt.Sprintf("%d different programs crashed %d times", len(distinct), len(in.Crashes.AppCrashes)),
			Summary: "Together with the blue screens and hardware errors, this points at the PC rather than the programs: " +
				"processor, memory, power or a tuning profile. Reinstalling apps will not fix it.",
			Evidence:     times("Application Error", in.Crashes.AppCrashes),
			Latest:       newest(in.Crashes.AppCrashes, ""),
			Alternatives: "A broken system file or a bad shared library (such as a .NET or Visual C++ runtime) can also crash many apps.",
		}}
	}
	var out []domain.Finding
	for app, n := range top(in.Crashes.AppCrashes, 3) {
		out = append(out, domain.Finding{
			Rule: "app-crash", Key: "app-crash:" + app, Latest: newest(in.Crashes.AppCrashes, app),
			Severity: domain.Low, Confidence: domain.Confirmed,
			Title:   fmt.Sprintf("%s crashed %d times", app, n),
			Summary: "Updating or reinstalling it usually helps. If many different apps crash, look at system stability instead.",
		})
	}
	for app, n := range top(in.Crashes.AppHangs, 3) {
		out = append(out, domain.Finding{
			Rule: "app-hang", Key: "app-hang:" + app, Latest: newest(in.Crashes.AppHangs, app),
			Severity: domain.Low, Confidence: domain.Confirmed,
			Title:   fmt.Sprintf("%s stopped responding %d times", app, n),
			Summary: "Windows closed it after it froze.",
		})
	}
	return out
}

func serviceFailures(in Input) []domain.Finding {
	var out []domain.Finding
	for svc, n := range top(in.Crashes.Services, 3) {
		out = append(out, domain.Finding{
			Rule: "service-failure", Key: "service-failure:" + svc, Latest: newest(in.Crashes.Services, svc),
			Severity: domain.Low, Confidence: domain.Confirmed,
			Title:   fmt.Sprintf("The %s service failed %d times", svc, n),
			Summary: "It crashed or did not start. If it belongs to a program you removed, its leftovers may still register it.",
		})
	}
	return out
}

func tuningTools(in Input) []domain.Finding {
	if len(in.Crashes.Tuning) == 0 {
		return nil
	}
	f := domain.Finding{
		Rule: "tuning-tools", Severity: domain.Info, Confidence: domain.Confirmed,
		Title:    "Hardware tuning tools are installed",
		Summary:  "These drivers can change clocks and voltages at every boot and override limits set elsewhere.",
		Evidence: in.Crashes.Tuning,
	}
	if len(in.Crashes.WHEA)+len(in.Crashes.Bugchecks) > 0 {
		f.Severity = domain.Medium
		f.Summary += " While the PC is unstable, uninstall or disable them so their profiles are not part of the problem."
		f.Action = &domain.Action{Label: "Review programs", Kind: "open-programs"}
	}
	return []domain.Finding{f}
}

// apicSummary lists processors by error count and pairs IDs that differ
// only in the last bit, which on Intel are the two threads of one core.
func apicSummary(apic map[string]int) string {
	type c struct {
		id string
		n  int
	}
	var cs []c
	for id, n := range apic {
		cs = append(cs, c{id, n})
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].n > cs[j].n || cs[i].n == cs[j].n && cs[i].id < cs[j].id })
	parts := make([]string, len(cs))
	for i, x := range cs {
		parts[i] = fmt.Sprintf("%s ×%d", x.id, x.n)
	}
	out := strings.Join(parts, ", ")
	for _, a := range cs {
		var id int
		if _, err := fmt.Sscanf(a.id, "APIC %d", &id); err == nil && id%2 == 0 {
			if _, ok := apic[fmt.Sprintf("APIC %d", id+1)]; ok {
				out += fmt.Sprintf(" (APIC %d and %d are likely two threads of one core)", id, id+1)
			}
		}
	}
	return out
}

func memoryPressure(in Input) []domain.Finding {
	m := in.Live.Memory
	if m.Status != domain.Measured || m.Value < 90 {
		return nil
	}
	return []domain.Finding{{
		Rule: "memory-pressure", Severity: domain.Medium, Confidence: domain.Confirmed,
		Title:   fmt.Sprintf("Memory is %.0f%% used", m.Value),
		Summary: "Windows is short of free memory, so it pages to disk and everything slows down. Close or disable what you do not need.",
		Action:  &domain.Action{Label: "Show processes by memory", Kind: "open-processes", Arg: "private"},
	}}
}

func lowDisk(in Input) []domain.Finding {
	var out []domain.Finding
	for _, v := range in.Overview.Volumes {
		if v.Total.Status != domain.Measured || v.Total.Value == 0 {
			continue
		}
		if free := v.Free.Value / v.Total.Value; free < 0.10 {
			out = append(out, domain.Finding{
				Rule: "low-disk", Key: "low-disk:" + v.Root, Severity: domain.Medium, Confidence: domain.Confirmed,
				Title:   fmt.Sprintf("Drive %s is %.0f%% full", v.Root, 100*(1-free)),
				Summary: "Windows needs free space for updates, the page file and temporary files.",
				Action:  &domain.Action{Label: "Free up space", Kind: "open-cleanup"},
			})
		}
	}
	return out
}

func heavyStartup(in Input) []domain.Finding {
	var on int
	var mem uint64
	for _, a := range in.Startup {
		if a.Enabled {
			on++
			mem += a.Memory
		}
	}
	if on < 8 && mem < 1<<30 {
		return nil
	}
	return []domain.Finding{{
		Rule: "heavy-startup", Severity: domain.Low, Confidence: domain.Likely,
		Title:   fmt.Sprintf("%d apps start with Windows", on),
		Summary: fmt.Sprintf("Together they use %.1f GB of memory right now and slow sign-in.", float64(mem)/(1<<30)),
		Action:  &domain.Action{Label: "Review startup apps", Kind: "open-startup"},
	}}
}

func hungApps(in Input) []domain.Finding {
	var names []string
	for _, p := range in.Processes {
		if p.Hung {
			names = append(names, p.Name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return []domain.Finding{{
		Rule: "hung-now", Severity: domain.Medium, Confidence: domain.Confirmed,
		Title:   "Not responding: " + strings.Join(names, ", "),
		Summary: "These apps have a window that stopped answering. Wait a moment, or end the task.",
		Action:  &domain.Action{Label: "Show processes", Kind: "open-processes", Arg: "cpu"},
	}}
}

// DeviceProblems explains Device Manager's error codes in plain words.
var DeviceProblems = map[int]string{
	1: "not set up correctly", 3: "driver may be corrupt", 10: "cannot start", 12: "not enough resources",
	14: "needs a restart", 18: "reinstall its driver", 19: "registry entry is damaged", 21: "being removed",
	22: "disabled", 24: "not present or not working", 28: "no driver installed", 31: "not working properly",
	32: "driver service disabled", 37: "driver failed to load", 39: "driver missing or corrupt",
	43: "Windows stopped it after it reported a problem", 48: "driver blocked as incompatible", 52: "driver signature not verified",
}

func problemDevices(in Input) []domain.Finding {
	var missing, broken []string
	for _, d := range in.Devices {
		if d.ErrorCode == 22 {
			continue // someone turned it off on purpose
		}
		line := fmt.Sprintf("%s: %s (code %d)", d.Name, DeviceProblems[d.ErrorCode], d.ErrorCode)
		if d.ErrorCode == 28 || d.ErrorCode == 39 || d.ErrorCode == 18 {
			missing = append(missing, line)
		} else {
			broken = append(broken, line)
		}
	}
	if len(missing)+len(broken) == 0 {
		return nil
	}
	sev := domain.Low
	if len(missing) > 0 {
		sev = domain.Medium
	}
	return []domain.Finding{{
		Rule: "problem-devices", Severity: sev, Confidence: domain.Confirmed,
		Title:    fmt.Sprintf("%d devices have driver problems", len(missing)+len(broken)),
		Summary:  "Windows Update or the maker's site usually has the driver. Tools > Drivers shows what Windows Update offers and opens the right page.",
		Evidence: append(missing, broken...),
		Action:   &domain.Action{Label: "Find drivers", Kind: "open-tools", Arg: "drivers"},
	}}
}

func notElevated(in Input) []domain.Finding {
	if in.Overview.Elevated {
		return nil
	}
	return []domain.Finding{{
		Rule: "not-elevated", Severity: domain.Info, Confidence: domain.Confirmed,
		Title:   "Some checks were skipped",
		Summary: "ReforgePulse runs without administrator rights, so launch history and some logs were not readable. Restart it and accept the prompt.",
	}}
}

// top counts events by detail and keeps those seen at least min times.
func top(events []domain.CrashEvent, min int) map[string]int {
	counts := map[string]int{}
	for _, e := range events {
		if e.Detail != "" {
			counts[e.Detail]++
		}
	}
	for k, n := range counts {
		if n < min {
			delete(counts, k)
		}
	}
	return counts
}

// newest is the latest event, or the latest with this detail when detail is
// set. It is what tells a resolved finding that it has come back.
func newest(events []domain.CrashEvent, detail string) time.Time {
	var t time.Time
	for _, e := range events {
		if (detail == "" || e.Detail == detail) && e.Time.After(t) {
			t = e.Time
		}
	}
	return t
}

// shownEvents is how many occurrences a finding's details list; the rest are
// counted. Enough to see a pattern, small enough for a saved scan.
const shownEvents = 15

// times lists the first occurrences for the evidence panel.
func times(source string, events []domain.CrashEvent) []string {
	var out []string
	for i, e := range events {
		if i == shownEvents {
			out = append(out, fmt.Sprintf("… and %d more", len(events)-shownEvents))
			break
		}
		line := fmt.Sprintf("%s · %s", e.Time.Local().Format("2006-01-02 15:04"), source)
		if e.Detail != "" {
			line += " · " + e.Detail
		}
		out = append(out, line)
	}
	return out
}
