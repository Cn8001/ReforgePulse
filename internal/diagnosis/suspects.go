package diagnosis

import (
	"fmt"
	"strings"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

// Verdicts, from most to least evidence. Unmeasured is not a clean bill:
// Windows simply cannot see that part.
const (
	strong     = "strong"
	some       = "some"
	none       = "none"
	unmeasured = "unmeasured"
)

// driverCodes are stop codes a misbehaving driver usually raises.
var driverCodes = map[string]bool{"0xA": true, "0xD1": true, "0x1E": true, "0x7E": true, "0x9F": true, "0x133": true, "0x116": true, "0x3B": true}

// Suspects weighs every place instability can come from, so the scan does
// not assume one. It returns nothing on a PC with nothing to explain.
func Suspects(in Input) []domain.Suspect {
	c := in.Crashes
	if len(c.Restarts)+len(c.Bugchecks)+len(c.WHEA) == 0 && len(c.AppCrashes) < 10 {
		return nil
	}
	codes := map[string]int{}
	for _, b := range c.Bugchecks {
		codes[b.Detail]++
	}
	first := firstCrash(c)
	return []domain.Suspect{
		cpuSuspect(in, codes), memorySuspect(in, codes), storageSuspect(in), heatSuspect(in),
		powerSuspect(in), driverSuspect(in, codes, first), softwareSuspect(in, codes, first),
	}
}

func cpuSuspect(in Input, codes map[string]int) domain.Suspect {
	s := domain.Suspect{Area: "cpu", Title: "Processor",
		Checks: []string{"Cap the CPU frequency a few hundred MHz lower for a week and see whether crashes stop",
			"Update the BIOS and load its defaults, removing any overclock or undervolt"}}
	var mca int
	for _, e := range in.Crashes.WHEA {
		if strings.HasPrefix(e.Detail, "APIC ") {
			mca++
		}
	}
	if mca > 0 {
		s.For = append(s.For, fmt.Sprintf("%d machine-check errors reported by processor cores", mca))
	}
	if n := codes["0x124"] + codes["0x101"]; n > 0 {
		s.For = append(s.For, fmt.Sprintf("%d blue screens the processor raises itself (0x124, 0x101)", n))
	}
	if desktopRaptorLake.MatchString(in.Overview.CPUModel) {
		if m := in.Crashes.Microcode; m != 0 && m < fixedMicrocode {
			s.For = append(s.For, fmt.Sprintf("13th/14th gen Intel on microcode 0x%X, before Intel's 0x12B fix", m))
		} else if m != 0 {
			s.Against = append(s.Against, fmt.Sprintf("microcode 0x%X already has Intel's stability fix", m))
		}
	}
	if mca == 0 && codes["0x124"]+codes["0x101"] == 0 {
		s.Against = append(s.Against, "no processor machine-check errors")
	}
	return verdict(s, mca >= 3 || codes["0x124"]+codes["0x101"] > 0)
}

func memorySuspect(in Input, codes map[string]int) domain.Suspect {
	s := domain.Suspect{Area: "memory", Title: "Memory (RAM)",
		Checks: []string{"Run Windows Memory Diagnostic (extended) or MemTest86 overnight",
			"Turn XMP/EXPO off, or test one stick or one pair at a time"}}
	if n := codes["0x1A"] + codes["0x50"] + codes["0x4E"]; n > 0 {
		s.For = append(s.For, fmt.Sprintf("%d memory-management blue screens (0x1A, 0x50, 0x4E); drivers can cause these too", n))
	}
	var bad, good int
	for _, t := range in.Crashes.MemoryTests {
		if t.Detail == "errors found" {
			bad++
		} else {
			good++
		}
	}
	if bad > 0 {
		s.For = append(s.For, "Windows Memory Diagnostic found errors")
	}
	if good > 0 && bad == 0 {
		s.Against = append(s.Against, "Windows Memory Diagnostic passed (it misses some faults)")
	}
	mods := in.Hardware.Memory
	ddr5Sticks, parts := 0, map[string]bool{}
	for _, m := range mods {
		parts[m.Part] = true
		if m.ConfiguredMHz > m.RatedMHz && m.RatedMHz > 0 {
			s.For = append(s.For, fmt.Sprintf("%s runs at %d MT/s, above its rated %d: an XMP/EXPO overclock", m.Slot, m.ConfiguredMHz, m.RatedMHz))
		}
		if m.ConfiguredMHz >= 4000 {
			ddr5Sticks++
		}
	}
	if ddr5Sticks >= 4 {
		s.For = append(s.For, fmt.Sprintf("%d DDR5 sticks, two per channel: the hardest setup for the memory controller", ddr5Sticks))
	}
	if len(parts) > 1 {
		s.For = append(s.For, "sticks of different models are mixed")
	}
	return verdict(s, bad > 0)
}

func storageSuspect(in Input) domain.Suspect {
	s := domain.Suspect{Area: "storage", Title: "Drives",
		Checks: []string{"Back up first, then run a disk check (chkdsk /scan)", "Read the drive's SMART data with the maker's tool"}}
	if n := len(in.Crashes.DiskErrors); n > 0 {
		s.For = append(s.For, fmt.Sprintf("%d read or file system errors in the event log", n))
	}
	measured := false
	for _, d := range in.Hardware.Disks {
		if d.Health != "" && d.Health != "Healthy" {
			s.For = append(s.For, fmt.Sprintf("%s reports %s (%s)", d.Name, d.Health, d.Status))
		}
		if d.ReadErrors > 0 || d.WriteErrors > 0 {
			s.For = append(s.For, fmt.Sprintf("%s has %d uncorrected read and %d write errors", d.Name, d.ReadErrors, d.WriteErrors))
		}
		if d.Wear >= 90 {
			s.For = append(s.For, fmt.Sprintf("%s is %d%% worn", d.Name, d.Wear))
		}
		if d.TempMaxC > 0 && d.TempC >= d.TempMaxC {
			s.For = append(s.For, fmt.Sprintf("%s is at %d°C, its limit", d.Name, d.TempC))
		}
		if d.Health == "Healthy" {
			measured = true
			s.Against = append(s.Against, d.Name+" reports healthy")
		}
	}
	if len(in.Crashes.DiskErrors) == 0 {
		s.Against = append(s.Against, "no disk or NTFS errors in the event log")
	}
	if !measured && len(in.Crashes.DiskErrors) == 0 {
		s.Verdict = unmeasured
		return s
	}
	return verdict(s, len(in.Crashes.DiskErrors) >= 3)
}

func heatSuspect(in Input) domain.Suspect {
	s := domain.Suspect{Area: "heat", Title: "Temperature",
		Checks: []string{"Watch CPU and GPU temperatures with HWiNFO while you do what usually crashes",
			"Check fans, dust and the CPU cooler's mounting and paste"}}
	if n := len(in.Crashes.Throttling); n > 0 {
		s.For = append(s.For, fmt.Sprintf("firmware limited processor speed %d times (heat or power limits)", n))
	}
	measured, hot := false, false
	for _, x := range in.Sensors.Sensors {
		if x.Kind != "temp" {
			continue
		}
		measured = true
		peak := max(x.Value, x.Max)
		// CPUs throttle near 100 °C and GPUs near 90 °C; a few degrees short
		// of that already means the cooling has no headroom.
		limit := 95.0
		if x.Source == "nvml" || strings.Contains(strings.ToLower(x.Group), "gpu") {
			limit = 85
		}
		if peak >= limit {
			hot = true
			s.For = append(s.For, fmt.Sprintf("%s %s reached %.0f%s", x.Group, x.Name, peak, x.Unit))
		}
	}
	if measured && !hot {
		s.Against = append(s.Against, "temperatures measured now are within limits")
	}
	// Windows has no reliable CPU temperature, so without throttling records
	// or sensors this stays unmeasured rather than clean.
	if len(s.For) == 0 && !measured {
		s.Against = append(s.Against, "no firmware throttling recorded")
		s.Verdict = unmeasured
		return s
	}
	if len(s.For) == 0 {
		s.Verdict = none
		return s
	}
	return verdict(s, hot || len(in.Crashes.Throttling) >= 10)
}

func powerSuspect(in Input) domain.Suspect {
	s := domain.Suspect{Area: "power", Title: "Power and voltage",
		Checks: []string{"Remove undervolts and load-line changes in the BIOS",
			"Check the PSU cables, and try another PSU or outlet if restarts stay codeless"}}
	var silent, button int
	for _, r := range in.Crashes.Restarts {
		switch r.Detail {
		case "":
			silent++
		case "power button":
			button++
		}
	}
	if silent > 0 {
		s.For = append(s.For, fmt.Sprintf("%d restarts with no blue screen: power was lost or the PC froze solid", silent))
	}
	for _, e := range in.Crashes.WHEA {
		if strings.HasPrefix(e.Detail, "APIC ") {
			s.For = append(s.For, "processor machine-check errors also appear when the core voltage is too low (undervolt, load-line, degraded VRM)")
			break
		}
	}
	if button > 0 {
		s.Against = append(s.Against, fmt.Sprintf("%d restarts were the power button being held", button))
	}
	s.Against = append(s.Against, "voltages are not measured: Windows does not expose them")
	if silent == 0 {
		s.Verdict = unmeasured
		return s
	}
	return verdict(s, silent >= 3)
}

func driverSuspect(in Input, codes map[string]int, first time.Time) domain.Suspect {
	s := domain.Suspect{Area: "drivers", Title: "Drivers",
		Checks: []string{"Roll back or reinstall the drivers installed just before the crashes began",
			"Open the newest minidump in WinDbg (!analyze -v) to see the module in charge"}}
	if n := len(in.Crashes.GPUResets); n > 0 {
		s.For = append(s.For, fmt.Sprintf("the graphics driver reset %d times", n))
	}
	total, top, topCode := 0, 0, ""
	for code, n := range codes {
		total += n
		if n > top {
			top, topCode = n, code
		}
	}
	if total >= 3 && top*10 >= total*7 && driverCodes[topCode] {
		s.For = append(s.For, fmt.Sprintf("%d of %d blue screens share %s, a code drivers usually raise", top, total, topCode))
	} else if total >= 3 && len(codes) >= 3 {
		s.Against = append(s.Against, fmt.Sprintf("%d different blue screen codes, which fits unstable hardware better than one driver", len(codes)))
	}
	if near := before(in.Crashes.DriverInstalls, first, 3*24*time.Hour); len(near) > 0 {
		s.For = append(s.For, "drivers installed in the 3 days before the first crash: "+strings.Join(near, "; "))
	}
	return verdict(s, len(s.For) >= 2)
}

func softwareSuspect(in Input, codes map[string]int, first time.Time) domain.Suspect {
	s := domain.Suspect{Area: "software", Title: "Software and Windows",
		Checks: []string{"Uninstall tuning and RGB tools while testing", "Check Windows' system files (sfc /verifyonly, DISM /ScanHealth)"}}
	if len(in.Crashes.Tuning) > 0 {
		s.For = append(s.For, "tuning or RGB drivers loaded: "+strings.Join(in.Crashes.Tuning, ", "))
	}
	if n := codes["0x20001"]; n > 0 {
		hv := ""
		if in.Hardware.Hypervisor {
			hv = "; the hypervisor (Hyper-V, WSL 2 or VBS) is on"
		}
		s.For = append(s.For, fmt.Sprintf("%d HYPERVISOR_ERROR blue screens%s", n, hv))
	}
	if near := before(realUpdates(in.Crashes.Updates), first, 3*24*time.Hour); len(near) > 0 {
		s.For = append(s.For, "Windows updates installed in the 3 days before the first crash: "+strings.Join(near, "; "))
	}
	return verdict(s, len(s.For) >= 2)
}

// verdict sets strong when the decisive test passed, some when anything
// points this way, none otherwise.
func verdict(s domain.Suspect, decisive bool) domain.Suspect {
	switch {
	case decisive:
		s.Verdict = strong
	case len(s.For) > 0:
		s.Verdict = some
	default:
		s.Verdict = none
	}
	return s
}

func firstCrash(c domain.CrashEvidence) time.Time {
	var first time.Time
	for _, list := range [][]domain.CrashEvent{c.Bugchecks, c.Restarts, c.WHEA} {
		for _, e := range list {
			if e.Time.IsZero() {
				continue // an event without a time cannot start the timeline
			}
			if first.IsZero() || e.Time.Before(first) {
				first = e.Time
			}
		}
	}
	return first
}

// before lists what happened within d before t, newest first as logged.
func before(events []domain.CrashEvent, t time.Time, d time.Duration) []string {
	if t.IsZero() {
		return nil
	}
	var out []string
	for _, e := range events {
		if !e.Time.After(t) && t.Sub(e.Time) <= d && e.Detail != "" {
			out = append(out, e.Detail)
		}
	}
	return out
}

// realUpdates drops the daily Defender signature updates, which would line
// up with any crash and explain none.
func realUpdates(events []domain.CrashEvent) []domain.CrashEvent {
	var out []domain.CrashEvent
	for _, e := range events {
		if !strings.Contains(e.Detail, "Security Intelligence Update") && !strings.Contains(e.Detail, "Defender") {
			out = append(out, e)
		}
	}
	return out
}
