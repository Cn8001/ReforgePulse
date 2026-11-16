package app

import (
	"context"
	"errors"
	"testing"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type fakeReader struct {
	o   domain.Overview
	err error
}

func (f fakeReader) Overview(context.Context) (domain.Overview, error) { return f.o, f.err }
func (f fakeReader) Live(context.Context) (domain.Live, error)         { return domain.Live{}, f.err }

func TestSystemService_PassesOverviewThrough(t *testing.T) {
	// Arrange
	s := NewSystemService(fakeReader{o: domain.Overview{OS: "Windows 11"}}, fakeReader{}, tasks.New(1, func(tasks.Status) {}))

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
	s := NewSystemService(fakeReader{err: cause}, fakeReader{}, tasks.New(1, func(tasks.Status) {}))

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

func TestSystemService_RunsAsTask(t *testing.T) {
	// Arrange
	var titles []string
	runner := tasks.New(1, func(s tasks.Status) { titles = append(titles, s.Title) })
	s := NewSystemService(fakeReader{}, fakeReader{}, runner)

	// Act
	_, err := s.GetOverview(context.Background())

	// Assert
	if err != nil || len(titles) == 0 || titles[0] != "Reading system overview" {
		t.Fatalf("titles = %v, err = %v", titles, err)
	}
}

type fakeLister struct {
	ps  []domain.Process
	err error
}

func (f fakeLister) List(context.Context) ([]domain.Process, error) { return f.ps, f.err }

func TestProcessService_ListsAndWraps(t *testing.T) {
	// Arrange
	cause := errors.New("snapshot failed")
	ok := NewProcessService(fakeLister{ps: []domain.Process{{PID: 4, Name: "System"}}})
	bad := NewProcessService(fakeLister{err: cause})

	// Act
	got, err := ok.List(context.Background())
	_, badErr := bad.List(context.Background())

	// Assert
	if err != nil || len(got) != 1 || got[0].PID != 4 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if !errors.Is(badErr, cause) {
		t.Fatalf("err = %v", badErr)
	}
}

func TestVersionFrom_ReadsInfoVersion(t *testing.T) {
	// Arrange
	config := []byte("version: '3'\ninfo:\n  productName: \"ReforgePulse\"\n  version: \"0.1.0\" # The application version\nios:\n  version: \"9.9.9\"\n")

	// Act
	got := VersionFrom(config)

	// Assert
	if got != "0.1.0" {
		t.Fatalf("version = %q, want 0.1.0", got)
	}
}

func TestVersionFrom_MissingIsDev(t *testing.T) {
	// Act
	got := VersionFrom([]byte("version: '3'\n"))

	// Assert
	if got != "dev" {
		t.Fatalf("version = %q, want dev", got)
	}
}
