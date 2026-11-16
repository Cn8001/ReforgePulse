package diagnosis

import (
	"strings"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

func find(fs []domain.Finding, rule string) *domain.Finding {
	for i := range fs {
		if fs[i].Rule == rule {
			return &fs[i]
		}
	}
	return nil
}

func events(n int, detail string) []domain.CrashEvent {
	out := make([]domain.CrashEvent, n)
	for i := range out {
		out[i] = domain.CrashEvent{Time: time.Date(2026, 11, i+1, 12, 0, 0, 0, time.UTC), Detail: detail}
	}
	return out
}

func TestEvaluate_HealthyPCHasNoFindings(t *testing.T) {
	// Arrange
	in := Input{Overview: domain.Overview{Elevated: true, CPUModel: "AMD Ryzen 7 7800X3D"}}

	// Act
	got := Evaluate(in)

	// Assert
	if len(got) != 0 {
		t.Fatalf("findings = %+v", got)
	}
}

func TestEvaluate_RaptorLakeOldMicrocode(t *testing.T) {
	// Arrange
	in := Input{
		Overview: domain.Overview{Elevated: true, CPUModel: "Intel(R) Core(TM) i9-14900KS"},
		Crashes:  domain.CrashEvidence{Microcode: 0x129, WHEA: events(4, "APIC 20")},
	}

	// Act
	got := Evaluate(in)

	// Assert
	v := find(got, "intel-vmin")
	if v == nil || v.Action == nil || v.Action.Kind != "cpu-cap" {
		t.Fatalf("intel-vmin = %+v", v)
	}
	w := find(got, "whea")
	if w == nil || w.Confidence != domain.Likely || !strings.Contains(w.Summary, "APIC 20") {
		t.Fatalf("whea = %+v", w)
	}
}

func TestEvaluate_FixedMicrocodeAndLaptopsAreQuiet(t *testing.T) {
	for _, in := range []Input{
		{Overview: domain.Overview{Elevated: true, CPUModel: "Intel(R) Core(TM) i9-14900K"}, Crashes: domain.CrashEvidence{Microcode: 0x12F}},
		{Overview: domain.Overview{Elevated: true, CPUModel: "Intel(R) Core(TM) i7-13700HX"}, Crashes: domain.CrashEvidence{Microcode: 0x120}},
	} {
		// Act
		got := Evaluate(in)

		// Assert
		if find(got, "intel-vmin") != nil {
			t.Fatalf("flagged %s", in.Overview.CPUModel)
		}
	}
}

func TestEvaluate_SeverityOrderAndRestartWording(t *testing.T) {
	// Arrange
	in := Input{
		Overview: domain.Overview{Elevated: true},
		Crashes:  domain.CrashEvidence{Restarts: events(3, ""), DiskErrors: events(1, `\Device\Harddisk1\DR1`)},
		Live:     domain.Live{Memory: domain.Measurement{Status: domain.Measured, Value: 95}},
	}

	// Act
	got := Evaluate(in)

	// Assert
	if got[0].Rule != "disk-errors" || got[0].Severity != domain.Critical {
		t.Fatalf("first = %+v, want the critical disk finding", got[0])
	}
	r := find(got, "unexpected-restarts")
	if r == nil || r.Severity != domain.High || r.Alternatives == "" {
		t.Fatalf("restarts = %+v", r)
	}
	if find(got, "memory-pressure") == nil {
		t.Fatal("memory pressure at 95% was not reported")
	}
}

func TestEvaluate_AppCrashesNeedRepeats(t *testing.T) {
	// Arrange
	in := Input{Overview: domain.Overview{Elevated: true}, Crashes: domain.CrashEvidence{
		AppCrashes: append(events(3, "game.exe in d3d12.dll"), events(1, "once.exe")...),
	}}

	// Act
	got := Evaluate(in)

	// Assert
	if len(got) != 1 || !strings.Contains(got[0].Title, "game.exe") {
		t.Fatalf("findings = %+v", got)
	}
}
