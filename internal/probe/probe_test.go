package probe

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

// pipeStart runs Serve in-process as the child; its Kill makes it vanish the way
// a crashed process does, mid-call.
func pipeStart(handlers map[string]Handler, starts *atomic.Int32) Start {
	return func() (*Child, error) {
		starts.Add(1)
		reqR, reqW := io.Pipe()
		repR, repW := io.Pipe()
		done := make(chan struct{})
		go func() {
			_ = Serve(context.Background(), reqR, repW, handlers)
			repW.Close()
			close(done)
		}()
		kill := func() { reqR.CloseWithError(io.ErrClosedPipe); repW.CloseWithError(io.EOF) }
		return &Child{In: reqW, Out: repR, Wait: func() error { return nil }, Kill: kill}, nil
	}
}

func TestClient_CallReportsProgressAndDecodes(t *testing.T) {
	// Arrange
	var starts atomic.Int32
	c := NewClient(pipeStart(map[string]Handler{
		"double": func(_ context.Context, args json.RawMessage, report func(float64, string)) (any, error) {
			var n int
			_ = json.Unmarshal(args, &n)
			report(0.5, "halfway")
			return n * 2, nil
		},
	}, &starts), nil)
	defer c.Close()
	var notes []string

	// Act
	var got int
	err := c.Call(context.Background(), "double", 21, &got, func(_ float64, note string) { notes = append(notes, note) })

	// Assert
	if err != nil || got != 42 || len(notes) != 1 || notes[0] != "halfway" {
		t.Fatalf("got %d, err %v, notes %v", got, err, notes)
	}
}

func TestClient_ChildDeathFailsTheCallAndTheNextCallRestarts(t *testing.T) {
	// Arrange
	var starts atomic.Int32
	var c *Client
	c = NewClient(pipeStart(map[string]Handler{
		"crash": func(context.Context, json.RawMessage, func(float64, string)) (any, error) {
			c.mu.Lock()
			ch := c.child
			c.mu.Unlock()
			ch.Kill()
			time.Sleep(time.Second) // the answer never comes
			return nil, nil
		},
		"ok": func(context.Context, json.RawMessage, func(float64, string)) (any, error) { return "fine", nil },
	}, &starts), nil)
	defer c.Close()

	// Act
	crashErr := c.Call(context.Background(), "crash", nil, nil, nil)
	var got string
	okErr := c.Call(context.Background(), "ok", nil, &got, nil)

	// Assert
	if !errors.Is(crashErr, ErrStopped) {
		t.Fatalf("crash call: %v", crashErr)
	}
	if okErr != nil || got != "fine" || starts.Load() != 2 {
		t.Fatalf("after restart: %q, %v, starts %d", got, okErr, starts.Load())
	}
}

func TestClient_CancelReachesTheHandler(t *testing.T) {
	// Arrange
	var starts atomic.Int32
	stopped := make(chan struct{})
	c := NewClient(pipeStart(map[string]Handler{
		"wait": func(ctx context.Context, _ json.RawMessage, _ func(float64, string)) (any, error) {
			<-ctx.Done()
			close(stopped)
			return nil, ctx.Err()
		},
	}, &starts), nil)
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Act
	err := c.Call(ctx, "wait", nil, nil, nil)

	// Assert
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("the handler never heard the cancel")
	}
}

type fakeLocal struct{}

func (fakeLocal) List(context.Context) ([]domain.CoreInfo, error) {
	return []domain.CoreInfo{{Index: 8, APIC: 32, WHEA: 3}}, nil
}
func (fakeLocal) CoreLoads(context.Context) ([]platform.CoreLoad, error) { return nil, nil }
func (fakeLocal) Sensors(context.Context) platform.SensorStatus          { return platform.SensorStatus{} }
func (fakeLocal) IconPNG(path string) ([]byte, error) {
	if path == "none" {
		return nil, errors.New("no icon")
	}
	return []byte{1, 2, 3}, nil
}

func (fakeLocal) Run(_ context.Context, cpus []int, _ time.Duration, report func(float64, string)) (domain.CoreTest, error) {
	report(1, "done")
	return domain.CoreTest{}, nil
}

func TestRemote_CoresComeBackThroughTheChild(t *testing.T) {
	// Arrange
	var starts atomic.Int32
	l := fakeLocal{}
	c := NewClient(pipeStart(Handlers(Local{Cores: l, Loads: l, Sensors: l, Tester: l, Icons: l}), &starts), nil)
	defer c.Close()
	r := NewRemote(c)

	// Act
	cores, err := r.List(context.Background())

	// Assert
	if err != nil || len(cores) != 1 || cores[0].APIC != 32 || cores[0].WHEA != 3 {
		t.Fatalf("cores = %+v, %v", cores, err)
	}
}

func TestRemote_IconsComeBackAsDataURIs(t *testing.T) {
	// Arrange
	var starts atomic.Int32
	l := fakeLocal{}
	c := NewClient(pipeStart(Handlers(Local{Cores: l, Loads: l, Sensors: l, Tester: l, Icons: l}), &starts), nil)
	defer c.Close()

	// Act
	got, err := NewRemote(c).Icons(context.Background(), []string{`C:\a.exe`, "none"})

	// Assert
	if err != nil || got[`C:\a.exe`] != "data:image/png;base64,AQID" || got["none"] != "" || len(got) != 2 {
		t.Fatalf("icons = %v, %v", got, err)
	}
}
