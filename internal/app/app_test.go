package app

import (
	"context"
	"errors"
	"testing"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type fakeReader struct {
	o   domain.Overview
	err error
}

func (f fakeReader) Overview(context.Context) (domain.Overview, error) { return f.o, f.err }

func TestSystemService_PassesOverviewThrough(t *testing.T) {
	// Arrange
	s := NewSystemService(fakeReader{o: domain.Overview{OS: "Windows 11"}})

	// Act
	o, err := s.GetOverview(context.Background())

	// Assert
	if err != nil || o.OS != "Windows 11" {
		t.Fatalf("got %+v, %v", o, err)
	}
}

func TestSystemService_WrapsErrors(t *testing.T) {
	// Arrange
	cause := errors.New("boom")
	s := NewSystemService(fakeReader{err: cause})

	// Act
	_, err := s.GetOverview(context.Background())

	// Assert
	if !errors.Is(err, cause) {
		t.Fatalf("err = %v, want it to wrap %v", err, cause)
	}
}

type fakeAccent string

func (f fakeAccent) GetAccentColor() string { return string(f) }

func TestThemeService_ReturnsAccent(t *testing.T) {
	// Arrange
	s := NewThemeService(fakeAccent("rgb(0,120,212)"))

	// Act
	got := s.Accent()

	// Assert
	if got != "rgb(0,120,212)" {
		t.Fatalf("accent = %q", got)
	}
}

type fakeLog struct {
	changes []domain.Change
	err     error
}

func (f fakeLog) Log(context.Context) ([]domain.Change, error) { return f.changes, f.err }

func TestHistoryService_ListsAndWraps(t *testing.T) {
	// Arrange
	cause := errors.New("disk full")
	ok := NewHistoryService(fakeLog{changes: []domain.Change{{ID: "abc1234"}}})
	bad := NewHistoryService(fakeLog{err: cause})

	// Act
	got, err := ok.List(context.Background())
	_, badErr := bad.List(context.Background())

	// Assert
	if err != nil || len(got) != 1 || got[0].ID != "abc1234" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if !errors.Is(badErr, cause) {
		t.Fatalf("err = %v, want it to wrap %v", badErr, cause)
	}
}
