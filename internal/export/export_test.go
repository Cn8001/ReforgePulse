package export

import (
	"strings"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/recorder"
)

func TestScan_HasScoreFindingsEvidenceAndTheEstimate(t *testing.T) {
	// Arrange
	r := domain.ScanResult{
		At:       time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC),
		Elevated: true,
		Health:   domain.Health{Score: 70, Deductions: []domain.Deduction{{Title: "A | B", Points: 15}}},
		Findings: []domain.Finding{{Title: "20 blue screens", Severity: domain.High, Confidence: domain.Confirmed, Summary: "Codes: 0x3B.", Evidence: []string{"2026-11-16 10:00 WER 1001 0x3B"}}},
	}

	// Act
	md := Scan(r)

	// Assert
	for _, want := range []string{"**70 out of 100**", `| -15 | A \| B |`, "### 20 blue screens", "High severity, confirmed.", "- `2026-11-16 10:00 WER 1001 0x3B`", Estimate} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in:\n%s", want, md)
		}
	}
}

func TestRecordingAndCheck_ReadAsTables(t *testing.T) {
	// Arrange
	rec := recorder.Recording{Seconds: 600, CPU: []float64{10, 30}, Top: []recorder.ProcessTrend{{Name: "game.exe", PID: 4, PeakMB: 900, Leak: true}}}
	check := domain.CheckResult{Name: "sfc", Summary: "No integrity violations.", Log: `C:\Windows\Logs\CBS\CBS.log`}

	// Act
	recMD, checkMD := Recording(rec), Check(check, "System files")

	// Assert
	if !strings.Contains(recMD, "| Processor | 30% | 20% |") || !strings.Contains(recMD, "likely leak") {
		t.Fatalf("recording:\n%s", recMD)
	}
	if !strings.Contains(checkMD, "# System files check") || !strings.Contains(checkMD, "CBS.log") {
		t.Fatalf("check:\n%s", checkMD)
	}
}
