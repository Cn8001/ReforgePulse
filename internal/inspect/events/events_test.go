package events

import (
	"context"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/platform"
)

type fakeLog map[string][]platform.Event

func (f fakeLog) Recent(_ context.Context, ch string, _ int, _ time.Time, _ int) ([]platform.Event, error) {
	return f[ch], nil
}

func TestInspector_GroupsAndExplains(t *testing.T) {
	// Arrange
	t1 := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	t2 := t1.Add(48 * time.Hour)
	log := fakeLog{
		"System": {
			{Provider: "Microsoft-Windows-Kernel-Power", ID: 41, Level: 1, Time: t1, Message: "old"},
			{Provider: "Microsoft-Windows-Kernel-Power", ID: 41, Level: 1, Time: t2, Message: "new"},
			{Provider: "Microsoft-Windows-DistributedCOM", ID: 10016, Level: 2, Time: t1},
		},
		"Application": {{Provider: "Application Error", ID: 1000, Level: 2, Time: t1.Add(time.Hour)}},
	}
	in := New(log)

	// Act
	got, err := in.List(context.Background(), 30, false)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].ID != 41 || got[0].Count != 2 || got[0].Message != "new" || !got[0].First.Equal(t1) || got[0].Explain == "" {
		t.Fatalf("groups = %+v", got)
	}
	for _, g := range got {
		if g.ID == 10016 && !g.Noise {
			t.Fatal("DCOM 10016 was not marked as noise")
		}
	}
}
