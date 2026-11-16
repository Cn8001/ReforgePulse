package processes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/platform"
)

type fakeSource struct {
	snaps [][]platform.ProcessInfo
	ports []platform.PortInfo
	err   error
}

func (f *fakeSource) Ports(context.Context) ([]platform.PortInfo, error) { return f.ports, nil }
func (f *fakeSource) Kill(context.Context, uint32) error                 { return nil }

type fakeGPU map[uint32]float64

func (f fakeGPU) GPUProcesses(context.Context) (map[uint32]float64, error) { return f, nil }

func (f *fakeSource) Processes(context.Context) ([]platform.ProcessInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	s := f.snaps[0]
	f.snaps = f.snaps[1:]
	return s, nil
}

// newTest returns an inspector whose clock moves one second per List call.
func newTest(src platform.Processes, cpus int) *Inspector {
	in := New(Deps{Processes: src, CPUs: cpus})
	t := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	in.now = func() time.Time { t = t.Add(time.Second); return t }
	return in
}

func TestInspector_CPUFromDelta(t *testing.T) {
	// Arrange
	src := &fakeSource{snaps: [][]platform.ProcessInfo{
		{{PID: 10, Name: "game.exe", CPUTime: 0}, {PID: 20, Name: "idle.exe", CPUTime: 0, Private: 5}},
		{{PID: 10, Name: "game.exe", CPUTime: 2 * time.Second}, {PID: 20, Name: "idle.exe", CPUTime: 0, Private: 5}},
	}}
	in := newTest(src, 4)

	// Act
	first, _ := in.List(context.Background())
	second, err := in.List(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if first[0].CPU != 0 {
		t.Fatalf("first sample cpu = %v, want 0", first[0].CPU)
	}
	if second[0].Name != "game.exe" || second[0].CPU != 50 {
		t.Fatalf("got %+v, want game.exe at 50%% (2 s over 1 s on 4 CPUs)", second[0])
	}
}

func TestInspector_ReusedPIDShowsNoSpike(t *testing.T) {
	// Arrange
	src := &fakeSource{snaps: [][]platform.ProcessInfo{
		{{PID: 10, Name: "old.exe", CPUTime: time.Hour}},
		{{PID: 10, Name: "new.exe", CPUTime: time.Second}},
	}}
	in := newTest(src, 1)

	// Act
	_, _ = in.List(context.Background())
	got, _ := in.List(context.Background())

	// Assert
	if got[0].CPU != 0 {
		t.Fatalf("cpu = %v, want 0 for a reused PID", got[0].CPU)
	}
}

func TestInspector_SortsByMemoryWhenIdle(t *testing.T) {
	// Arrange
	src := &fakeSource{snaps: [][]platform.ProcessInfo{
		{{PID: 1, Name: "small", Private: 1}, {PID: 2, Name: "big", Private: 9}},
	}}
	in := newTest(src, 1)

	// Act
	got, _ := in.List(context.Background())

	// Assert
	if got[0].Name != "big" {
		t.Fatalf("order = %+v", got)
	}
}

func TestInspector_WrapsSourceError(t *testing.T) {
	// Arrange
	cause := errors.New("snapshot failed")
	in := newTest(&fakeSource{err: cause}, 1)

	// Act
	_, err := in.List(context.Background())

	// Assert
	if !errors.Is(err, cause) {
		t.Fatalf("err = %v", err)
	}
}

func TestInspector_AttachesPortsGPUAndHung(t *testing.T) {
	// Arrange
	src := &fakeSource{
		snaps: [][]platform.ProcessInfo{{{PID: 7, Name: "server.exe", Hung: true}}},
		ports: []platform.PortInfo{{PID: 7, Proto: "tcp", Local: "0.0.0.0", Port: 8080, State: "listen"}},
	}
	in := New(Deps{Processes: src, GPU: fakeGPU{7: 42}, CPUs: 1})

	// Act
	got, err := in.List(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	p := got[0]
	if !p.Hung || p.GPU != 42 || len(p.Ports) != 1 || p.Ports[0].Port != 8080 {
		t.Fatalf("process = %+v", p)
	}
}
