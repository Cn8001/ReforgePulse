package tasks

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recorder keeps every notified status, safe for concurrent tasks.
type recorder struct {
	mu   sync.Mutex
	seen []Status
}

func (r *recorder) notify(s Status) { r.mu.Lock(); r.seen = append(r.seen, s); r.mu.Unlock() }

func (r *recorder) states() []State {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]State, len(r.seen))
	for i, s := range r.seen {
		out[i] = s.State
	}
	return out
}

func TestRunner_ReportsLifecycle(t *testing.T) {
	// Arrange
	rec := &recorder{}
	r := New(2, rec.notify)

	// Act
	err := r.Do(context.Background(), "Read overview", func(ctx context.Context, report Report) error {
		report(0.5, "halfway")
		return nil
	})

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	got := rec.states()
	want := []State{Queued, Running, Running, Done}
	if len(got) != len(want) {
		t.Fatalf("states = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("states = %v, want %v", got, want)
		}
	}
	if len(r.List()) != 0 {
		t.Fatal("finished task still listed")
	}
}

func TestRunner_FailureIsWrapped(t *testing.T) {
	// Arrange
	rec := &recorder{}
	r := New(1, rec.notify)
	cause := errors.New("access denied")

	// Act
	err := r.Do(context.Background(), "Read events", func(context.Context, Report) error { return cause })

	// Assert
	if !errors.Is(err, cause) {
		t.Fatalf("err = %v", err)
	}
	if last := rec.seen[len(rec.seen)-1]; last.State != Failed || last.Error == "" {
		t.Fatalf("last = %+v", last)
	}
}

func TestRunner_LimitsConcurrency(t *testing.T) {
	// Arrange
	r := New(2, func(Status) {})
	var running, peak atomic.Int32
	var wg sync.WaitGroup

	// Act
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Do(context.Background(), "work", func(context.Context, Report) error {
				n := running.Add(1)
				for {
					p := peak.Load()
					if n <= p || peak.CompareAndSwap(p, n) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				running.Add(-1)
				return nil
			})
		}()
	}
	wg.Wait()

	// Assert
	if peak.Load() > 2 {
		t.Fatalf("peak = %d, want at most 2", peak.Load())
	}
}

func TestRunner_Cancel(t *testing.T) {
	// Arrange
	rec := &recorder{}
	r := New(1, rec.notify)
	started := make(chan struct{})
	done := make(chan error)

	// Act
	go func() {
		done <- r.Do(context.Background(), "long scan", func(ctx context.Context, _ Report) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		})
	}()
	<-started
	r.Cancel(r.List()[0].ID)
	err := <-done

	// Assert
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if got := rec.states(); got[len(got)-1] != Cancelled {
		t.Fatalf("states = %v", got)
	}
}
