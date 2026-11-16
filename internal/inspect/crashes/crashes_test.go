package crashes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/platform"
)

type fakeMachine struct {
	platform.Machine
	events map[string][]platform.Event
	failOn string
	dumps  string
}

func (f fakeMachine) Events(_ context.Context, _, provider string, _ []uint32, _ time.Time, _ int) ([]platform.Event, error) {
	if provider == f.failOn {
		return nil, errors.New("access denied")
	}
	return f.events[provider], nil
}
func (f fakeMachine) Microcode(context.Context) (uint32, error) { return 0x129, nil }
func (f fakeMachine) Services(context.Context) ([]platform.Service, error) {
	return []platform.Service{{Name: "XtuComponent", Start: 3}, {Name: "WinRing0", Start: 4}, {Name: "Beep", Start: 1}}, nil
}
func (f fakeMachine) DumpDir() string { return f.dumps }

func TestInspector_CollectsAndNormalises(t *testing.T) {
	// Arrange
	dumps := t.TempDir()
	_ = os.WriteFile(filepath.Join(dumps, "111626-1.dmp"), nil, 0o644)
	m := fakeMachine{dumps: dumps, failOn: "Display", events: map[string][]platform.Event{
		"Microsoft-Windows-WER-SystemErrorReporting": {{ID: 1001, Data: map[string]string{"param1": "0x0000003b (0x00000000c0000005, 0x0)"}}},
		"Microsoft-Windows-Kernel-Power":             {{ID: 41, Data: map[string]string{"BugcheckCode": "59"}}, {ID: 41, Data: map[string]string{"BugcheckCode": "0"}}},
		"Microsoft-Windows-WHEA-Logger":              {{ID: 19, Data: map[string]string{"ApicId": "20"}}},
	}}
	in := New(m)

	// Act
	ev, err := in.Collect(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if ev.Bugchecks[0].Detail != "0x3B" || ev.Restarts[0].Detail != "0x3B" || ev.Restarts[1].Detail != "" {
		t.Fatalf("codes = %+v / %+v", ev.Bugchecks, ev.Restarts)
	}
	if ev.WHEA[0].Detail != "APIC 20" || ev.Microcode != 0x129 || len(ev.Minidumps) != 1 {
		t.Fatalf("evidence = %+v", ev)
	}
	if len(ev.Tuning) != 1 || ev.Tuning[0] != "XtuComponent" {
		t.Fatalf("tuning = %v, want only the enabled XTU driver", ev.Tuning)
	}
	if ev.EventsError == "" {
		t.Fatal("a failed source was not reported")
	}
}
