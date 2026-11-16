package leftovers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

type fakeMachine struct {
	platform.Machine
	progs []platform.Program
}

func (f fakeMachine) Programs(context.Context) ([]platform.Program, error) { return f.progs, nil }
func (f fakeMachine) Services(context.Context) ([]platform.Service, error) { return nil, nil }

type noStartup struct{ platform.Startup }

func (noStartup) StartupEntries(context.Context) ([]platform.StartupEntry, error) { return nil, nil }

type procs []domain.Process

func (p procs) List(context.Context) ([]domain.Process, error) { return p, nil }

type files struct {
	platform.Files
	root string
}

func (f files) LeftoverRoots() []platform.LeftoverRoot {
	return []platform.LeftoverRoot{{Name: "AppData (Roaming)", Path: f.root}}
}

func folder(t *testing.T, dir string, age time.Duration) {
	t.Helper()
	_ = os.MkdirAll(dir, 0o755)
	f := filepath.Join(dir, "data.bin")
	_ = os.WriteFile(f, make([]byte, 64), 0o644)
	old := time.Now().Add(-age)
	_ = os.Chtimes(f, old, old)
}

func TestInspector_FindsOnlyUnclaimedOldFolders(t *testing.T) {
	// Arrange
	root := t.TempDir()
	year := 365 * 24 * time.Hour
	folder(t, filepath.Join(root, "GoneApp"), year)      // leftover
	folder(t, filepath.Join(root, "Steam Helper"), year) // name matches an installed program
	folder(t, filepath.Join(root, "Installed"), year)    // an install location
	folder(t, filepath.Join(root, "Running"), year)      // a running process lives here
	folder(t, filepath.Join(root, "Fresh"), time.Hour)   // changed recently
	folder(t, filepath.Join(root, "Microsoft"), year)    // Windows' own
	m := fakeMachine{progs: []platform.Program{
		{Name: "Steam", Publisher: "Valve"},
		{Name: "Tool", InstallLocation: filepath.Join(root, "Installed")},
	}}
	in := New(Deps{Machine: m, Startup: noStartup{}, Processes: procs{{Path: filepath.Join(root, "Running", "r.exe")}}, Files: files{root: root}})

	// Act
	got, err := in.List(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(got[0].Path) != "GoneApp" || got[0].Size != 64 || got[0].Root != "AppData (Roaming)" {
		t.Fatalf("got %+v", got)
	}
}
