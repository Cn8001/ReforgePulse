package startup

import (
	"context"
	"testing"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

type fakeStartup []platform.StartupEntry

func (f fakeStartup) StartupEntries(context.Context) ([]platform.StartupEntry, error) { return f, nil }
func (f fakeStartup) SetStartupEnabled(context.Context, string, bool) error           { return nil }

type fakeProcs []domain.Process

func (f fakeProcs) List(context.Context) ([]domain.Process, error) { return f, nil }

func TestExecutable_ParsesCommandLines(t *testing.T) {
	cases := map[string]string{
		`"C:\Program Files\App\app.exe" --minimized`: `C:\Program Files\App\app.exe`,
		`C:\Tools\tool.exe /background`:              `C:\Tools\tool.exe`,
		`rundll32 shell32.dll,Control_RunDLL`:        `rundll32`,
	}
	for in, want := range cases {
		// Act
		got := Executable(in)

		// Assert
		if got != want {
			t.Errorf("Executable(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInspector_MatchesRunningProcesses(t *testing.T) {
	// Arrange
	src := fakeStartup{
		{ID: "user-run|Steam", Name: "Steam", Command: `"C:\Steam\steam.exe" -silent`, Enabled: true},
		{ID: "user-folder|Discord.lnk", Name: "Discord", Command: `C:\Startup\Discord.lnk`, Enabled: true},
		{ID: "user-run|Idle", Name: "Idle", Command: `C:\Idle\idle.exe`, Enabled: false},
	}
	procs := fakeProcs{
		{Name: "steam.exe", Path: `C:\Steam\steam.exe`, Private: 300},
		{Name: "Discord.exe", Path: `C:\Users\x\Discord.exe`, Private: 500},
	}
	in := New(src, procs)

	// Act
	got, err := in.List(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "Discord" || !got[0].Running || got[0].Memory != 500 {
		t.Fatalf("first = %+v, want Discord running at 500", got[0])
	}
	if got[1].Name != "Steam" || got[1].Memory != 300 {
		t.Fatalf("second = %+v", got[1])
	}
	if got[2].Running || got[2].Enabled {
		t.Fatalf("idle = %+v", got[2])
	}
}
