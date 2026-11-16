package coretest

import (
	"context"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/platform"
)

type fakeCPUs struct{ pinned []int }

func (f *fakeCPUs) Topology(context.Context) ([]platform.LogicalCPU, error) { return nil, nil }
func (f *fakeCPUs) CoreLoads(context.Context) ([]platform.CoreLoad, error) {
	return []platform.CoreLoad{{Index: 3, MHz: 5400}}, nil
}
func (f *fakeCPUs) RunPinned(cpu int, fn func()) error {
	f.pinned = append(f.pinned, cpu)
	fn()
	return nil
}

func TestEngine_FlagsTheCoreThatDisagrees(t *testing.T) {
	// Arrange: core 3 gets round 2 wrong.
	cpus := &fakeCPUs{}
	e := New(Deps{CPUs: cpus, WHEA: func(context.Context, time.Time) (map[uint32]int, error) {
		return map[uint32]int{32: 2}, nil
	}})
	current := 0
	e.work = func(seed uint64) [32]byte {
		var h [32]byte
		h[0] = byte(seed)
		if current == 3 && seed == 2 {
			h[1] = 0xFF
		}
		return h
	}
	clock := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	ticks := 0
	e.now = func() time.Time {
		ticks++
		if ticks%5 == 0 {
			clock = clock.Add(time.Hour) // ends each processor's window after a few rounds
		}
		return clock
	}

	// Act
	var got []int
	res, err := e.Run(context.Background(), []int{1, 2, 3}, time.Minute, func(_ float64, note string) {
		got = append(got, len(note))
		current = len(got) // processors run in order: 1, 2, 3
	})

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res.Results {
		if r.Rounds == 0 {
			t.Fatalf("processor %d ran no rounds", r.Index)
		}
		want := 0
		if r.Index == 3 {
			want = 1
		}
		if r.Errors != want {
			t.Fatalf("processor %d: %d errors, want %d (results %+v)", r.Index, r.Errors, want, res.Results)
		}
	}
	if res.NewWHEA != 2 || res.Results[2].MHz != 5400 {
		t.Fatalf("test = %+v", res)
	}
}

func TestWork_IsDeterministic(t *testing.T) {
	// Act
	a, b := work(7), work(7)

	// Assert
	if a != b || a == work(8) {
		t.Fatal("work is not deterministic per seed")
	}
}

func TestEngine_NeedsProcessors(t *testing.T) {
	// Act
	_, err := New(Deps{}).Run(context.Background(), nil, time.Second, func(float64, string) {})

	// Assert
	if err == nil {
		t.Fatal("want an error")
	}
}
