package reports

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

func scan(minute int, keys ...string) domain.ScanResult {
	r := domain.ScanResult{At: time.Date(2026, 11, 17, 12, minute, 0, 0, time.UTC)}
	for _, k := range keys {
		r.Findings = append(r.Findings, domain.Finding{Key: k, Title: k, Severity: domain.Medium})
	}
	return r
}

func TestStore_SavesNewestFirstAndKeepsTheLimit(t *testing.T) {
	// Arrange
	ctx := context.Background()
	s, err := Open(t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}

	// Act
	for i := range 3 {
		if _, err := s.Save(ctx, scan(i, "a")); err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.All(ctx)

	// Assert
	if err != nil || len(all) != 2 {
		t.Fatalf("all = %d, %v", len(all), err)
	}
	if all[0].At.Minute() != 2 || all[1].At.Minute() != 1 {
		t.Fatalf("order = %v, %v", all[0].At, all[1].At)
	}
	if all[0].ID != "20261117-120200-000" {
		t.Fatalf("id = %s", all[0].ID)
	}
}

func TestStore_MarksFindingsTheLastScanDidNotHave(t *testing.T) {
	// Arrange
	ctx := context.Background()
	s, _ := Open(t.TempDir(), 5)
	if _, err := s.Save(ctx, scan(0, "a")); err != nil {
		t.Fatal(err)
	}

	// Act
	got, err := s.Save(ctx, scan(1, "a", "b"))

	// Assert
	if err != nil || got.Findings[0].New || !got.Findings[1].New {
		t.Fatalf("findings = %+v, %v", got.Findings, err)
	}
	stored, _ := s.Get(ctx, got.ID)
	if !stored.Findings[1].New {
		t.Fatal("new flag was not stored")
	}
}

func TestStore_GetRejectsPathsAndSkipsTornFiles(t *testing.T) {
	// Arrange
	ctx := context.Background()
	dir := t.TempDir()
	s, _ := Open(dir, 5)
	saved, _ := s.Save(ctx, scan(0, "a"))
	if err := os.WriteFile(filepath.Join(dir, "20261117-130000-000.json"), []byte(`{"id":`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Act
	_, getErr := s.Get(ctx, `..\..\journal`)
	latest, err := s.Latest(ctx)

	// Assert
	if getErr == nil {
		t.Fatal("a path was accepted as an ID")
	}
	if err != nil || latest == nil || latest.ID != saved.ID {
		t.Fatalf("latest = %+v, %v", latest, err)
	}
}

func TestResolutions_ResolveReopenAndComeBack(t *testing.T) {
	// Arrange
	ctx := context.Background()
	r, _ := OpenResolutions(t.TempDir())
	at := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	r.now = func() time.Time { return at }
	if err := r.Resolve(ctx, "a", "A"); err != nil {
		t.Fatal(err)
	}
	if err := r.Resolve(ctx, "b", "B"); err != nil {
		t.Fatal(err)
	}
	if err := r.Reopen(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	res := scan(0, "a", "b", "c")
	res.Findings[2].Key = "a2"
	res.Findings = append(res.Findings, domain.Finding{Key: "a", Latest: at.Add(time.Hour)})

	// Act
	m, err := r.All(ctx)
	Apply(&res, m)

	// Assert
	if err != nil || len(m) != 1 {
		t.Fatalf("resolutions = %v, %v", m, err)
	}
	want := []domain.FindingStatus{domain.Resolved, domain.Open, domain.Open, domain.Returned}
	for i, f := range res.Findings {
		if f.Status != want[i] {
			t.Fatalf("finding %d status = %s, want %s", i, f.Status, want[i])
		}
	}
	if s := Summarize(res); s.Open != 3 || s.Resolved != 1 || s.Worst != domain.Medium {
		t.Fatalf("summary = %+v", s)
	}
}
