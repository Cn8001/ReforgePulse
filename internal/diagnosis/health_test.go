package diagnosis

import (
	"testing"

	"github.com/cn8001/reforgepulse/internal/domain"
)

func TestHealth_DeductsOpenFindingsBySeverity(t *testing.T) {
	// Arrange
	findings := []domain.Finding{
		{Key: "low", Severity: domain.Low, Status: domain.Open},
		{Key: "high", Severity: domain.High, Status: domain.Returned},
		{Key: "done", Severity: domain.Critical, Status: domain.Resolved},
		{Key: "note", Severity: domain.Info, Status: domain.Open},
	}

	// Act
	h := Health(findings)

	// Assert
	if h.Score != 83 || len(h.Deductions) != 2 || h.Deductions[0].Key != "high" {
		t.Fatalf("health = %+v", h)
	}
}

func TestHealth_NeverBelowZero(t *testing.T) {
	// Arrange
	var findings []domain.Finding
	for range 6 {
		findings = append(findings, domain.Finding{Severity: domain.Critical, Status: domain.Open})
	}

	// Act
	h := Health(findings)

	// Assert
	if h.Score != 0 {
		t.Fatalf("score = %d", h.Score)
	}
}
