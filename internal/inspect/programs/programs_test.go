package programs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/platform"
)

type fakeMachine struct {
	platform.Machine
	progs   []platform.Program
	runs    map[string]time.Time
	runsErr error
}

func (f fakeMachine) Programs(context.Context) ([]platform.Program, error) { return f.progs, nil }
func (f fakeMachine) LastRuns(context.Context) (map[string]time.Time, error) {
	return f.runs, f.runsErr
}

var now = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func newTest(m fakeMachine, exes map[string][]string) *Inspector {
	in := New(m)
	in.now = func() time.Time { return now }
	in.listExes = func(dir string) []string { return exes[dir] }
	return in
}

func TestInspector_FlagsLongUnusedPrograms(t *testing.T) {
	// Arrange
	m := fakeMachine{
		progs: []platform.Program{
			{ID: "a", Name: "Old Game", InstallLocation: `C:\Game`, SizeKB: 1000},
			{ID: "b", Name: "Daily Tool", InstallLocation: `C:\Tool`, SizeKB: 5000},
		},
		runs: map[string]time.Time{"GAME.EXE": now.AddDate(0, -6, 0), "TOOL.EXE": now.AddDate(0, 0, -2)},
	}
	in := newTest(m, map[string][]string{`C:\Game`: {"game.exe"}, `C:\Tool`: {"tool.exe"}})

	// Act
	got, err := in.List(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "Old Game" || !got[0].Forgotten || got[0].Size != 1000*1024 {
		t.Fatalf("first = %+v", got[0])
	}
	if got[1].Forgotten || got[1].DaysIdle != 2 {
		t.Fatalf("second = %+v", got[1])
	}
}

func TestInspector_NeverFlagsSharedComponents(t *testing.T) {
	// Arrange
	m := fakeMachine{progs: []platform.Program{
		{ID: "r", Name: "Microsoft Visual C++ 2015-2022 Redistributable (x64)", InstallLocation: `C:\VC`, InstallDate: "20200101"},
	}, runs: map[string]time.Time{}}
	in := newTest(m, nil)

	// Act
	got, _ := in.List(context.Background())

	// Assert
	if got[0].Forgotten {
		t.Fatalf("a shared runtime was flagged: %+v", got[0])
	}
}

func TestInspector_NoHistoryFlagsNothing(t *testing.T) {
	// Arrange
	m := fakeMachine{progs: []platform.Program{{ID: "a", Name: "App", InstallLocation: `C:\App`, InstallDate: "20200101"}},
		runsErr: errors.New("access denied")}
	in := newTest(m, nil)

	// Act
	got, err := in.List(context.Background())

	// Assert
	if err != nil || got[0].Forgotten {
		t.Fatalf("got %+v, err = %v", got, err)
	}
}

func TestInspector_OldInstallWithoutRecordIsFlagged(t *testing.T) {
	// Arrange
	m := fakeMachine{progs: []platform.Program{{ID: "a", Name: "Trial", InstallLocation: `C:\Trial`, InstallDate: "20240101"}},
		runs: map[string]time.Time{}}
	in := newTest(m, map[string][]string{`C:\Trial`: {"trial.exe"}})

	// Act
	got, _ := in.List(context.Background())

	// Assert
	if !got[0].Forgotten || got[0].DaysIdle != -1 {
		t.Fatalf("got %+v", got[0])
	}
}
