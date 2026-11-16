package diagnosis

import (
	"strings"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

func suspect(ss []domain.Suspect, area string) domain.Suspect {
	for _, s := range ss {
		if s.Area == area {
			return s
		}
	}
	return domain.Suspect{}
}

func TestSuspects_QuietPCHasNone(t *testing.T) {
	// Act
	got := Suspects(Input{})

	// Assert
	if got != nil {
		t.Fatalf("suspects = %+v", got)
	}
}

func TestSuspects_WeighsEveryAreaNotJustCPU(t *testing.T) {
	// Arrange: a PC like the one this was built on.
	day := func(d int) time.Time { return time.Date(2026, 11, d, 12, 0, 0, 0, time.UTC) }
	in := Input{
		Overview: domain.Overview{CPUModel: "Intel(R) Core(TM) i9-14900KS"},
		Hardware: platform.HardwareFacts{
			Hypervisor: true,
			Memory: []platform.MemoryModule{
				{Slot: "A1", Part: "F5-6400", RatedMHz: 4000, ConfiguredMHz: 4000}, {Slot: "A2", Part: "F5-6400", RatedMHz: 4000, ConfiguredMHz: 4000},
				{Slot: "B1", Part: "F5-6400", RatedMHz: 4000, ConfiguredMHz: 4000}, {Slot: "B2", Part: "F5-6400", RatedMHz: 4000, ConfiguredMHz: 4000},
			},
			Disks: []platform.DiskHealth{{Name: "990 PRO", Health: "Healthy", TempC: -1, Wear: -1}},
		},
		Crashes: domain.CrashEvidence{
			Microcode:      0x133,
			WHEA:           []domain.CrashEvent{{Detail: "APIC 32"}, {Detail: "APIC 32"}, {Detail: "APIC 33"}},
			Bugchecks:      []domain.CrashEvent{{Time: day(10), Detail: "0x3B"}, {Time: day(11), Detail: "0x20001"}, {Time: day(12), Detail: "0x50"}},
			Restarts:       []domain.CrashEvent{{Time: day(10)}, {Time: day(11), Detail: "power button"}},
			Tuning:         []string{"XTUComponent"},
			DriverInstalls: []domain.CrashEvent{{Time: day(9), Detail: "NVIDIA nvlddmkm 32.0"}},
		},
	}

	// Act
	got := Suspects(in)

	// Assert
	if len(got) != 7 {
		t.Fatalf("got %d areas, want all 7", len(got))
	}
	if s := suspect(got, "cpu"); s.Verdict != strong || !strings.Contains(strings.Join(s.Against, " "), "0x133") {
		t.Fatalf("cpu = %+v", s)
	}
	if s := suspect(got, "memory"); s.Verdict != some || !strings.Contains(strings.Join(s.For, " "), "two per channel") {
		t.Fatalf("memory = %+v", s)
	}
	if s := suspect(got, "storage"); s.Verdict != none {
		t.Fatalf("storage = %+v", s)
	}
	if s := suspect(got, "heat"); s.Verdict != unmeasured {
		t.Fatalf("heat = %+v, want unmeasured, never clean", s)
	}
	if s := suspect(got, "power"); s.Verdict != some || !strings.Contains(strings.Join(s.Against, " "), "power button") {
		t.Fatalf("power = %+v", s)
	}
	if s := suspect(got, "drivers"); !strings.Contains(strings.Join(s.For, " "), "nvlddmkm") {
		t.Fatalf("drivers = %+v", s)
	}
	if s := suspect(got, "software"); s.Verdict != strong {
		t.Fatalf("software = %+v, want strong (tuning tools and hypervisor errors)", s)
	}
}

func TestSuspects_HotSensorIsEvidence(t *testing.T) {
	// Arrange
	in := Input{
		Crashes: domain.CrashEvidence{Restarts: []domain.CrashEvent{{}}},
		Sensors: platform.SensorStatus{Sensors: []platform.Sensor{
			{Source: "hwinfo", Group: "CPU [#0]", Name: "Core Max", Kind: "temp", Value: 70, Max: 100, Unit: "°C"},
		}},
	}

	// Act
	s := suspect(Suspects(in), "heat")

	// Assert
	if s.Verdict != strong || !strings.Contains(strings.Join(s.For, " "), "reached 100") {
		t.Fatalf("heat = %+v", s)
	}
}

func TestSuspects_CoolSensorsRuleHeatOut(t *testing.T) {
	// Arrange
	in := Input{
		Crashes: domain.CrashEvidence{Restarts: []domain.CrashEvent{{}}},
		Sensors: platform.SensorStatus{Sensors: []platform.Sensor{{Source: "nvml", Kind: "temp", Value: 40, Unit: "°C"}}},
	}

	// Act
	s := suspect(Suspects(in), "heat")

	// Assert
	if s.Verdict != none {
		t.Fatalf("heat = %+v", s)
	}
}
