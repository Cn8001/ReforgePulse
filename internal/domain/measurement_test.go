package domain

import (
	"testing"
	"time"
)

func TestMeasure_KeepsUnitAndSource(t *testing.T) {
	// Arrange
	at := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	// Act
	m := Measure(42, Bytes, "GlobalMemoryStatusEx", at)

	// Assert
	if m.Status != Measured || m.Unit != Bytes || m.Source != "GlobalMemoryStatusEx" || !m.At.Equal(at) || m.Value != 42 {
		t.Fatalf("unexpected measurement: %+v", m)
	}
}

func TestMissing_NeverReportsMeasured(t *testing.T) {
	for _, s := range []Availability{Measured, NotMeasured, Unsupported, Unknown} {
		// Act
		m := Missing(s, "x")

		// Assert
		if m.Status == Measured {
			t.Fatalf("Missing(%q) reported measured", s)
		}
		if m.Value != 0 {
			t.Fatalf("Missing(%q) has value %v", s, m.Value)
		}
	}
}
