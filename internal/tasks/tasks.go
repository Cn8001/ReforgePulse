// Package tasks runs the program's work off the UI thread, a bounded number
// at a time, and reports each task's state so the status bar can show it.
package tasks

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

type State string

const (
	Queued    State = "queued"
	Running   State = "running"
	Done      State = "done"
	Failed    State = "failed"
	Cancelled State = "cancelled"
)

// Status is one task as the status bar shows it.
type Status struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	State    State     `json:"state"`
	Progress float64   `json:"progress"` // 0..1, or -1 while the length is unknown
	Note     string    `json:"note,omitempty"`
	Error    string    `json:"error,omitempty"`
	Started  time.Time `json:"started"`
}

// Report lets a task tell how far it is.
type Report func(progress float64, note string)

// Notifier hears every change of state; the window forwards it as an event.
type Notifier func(Status)

type Runner struct {
	mu     sync.Mutex
	slots  chan struct{}
	notify Notifier
	next   int
	live   map[string]*task
}

type task struct {
	status Status
	cancel context.CancelFunc
}

// New allows at most limit tasks to run at once; the rest wait their turn.
func New(limit int, notify Notifier) *Runner {
	if limit < 1 {
		limit = 1
	}
	return &Runner{slots: make(chan struct{}, limit), notify: notify, live: map[string]*task{}}
}

// Do runs fn as a tracked task and waits for it. Only the caller waits:
// window calls already arrive on their own goroutine, so the UI never does.
func (r *Runner) Do(ctx context.Context, title string, fn func(ctx context.Context, report Report) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	t := r.add(title, cancel)

	select {
	case r.slots <- struct{}{}:
		defer func() { <-r.slots }()
	case <-ctx.Done():
		r.finish(t, ctx.Err())
		return fmt.Errorf("%s: %w", title, ctx.Err())
	}
	r.update(t, func(s *Status) { s.State = Running })

	err := fn(ctx, func(p float64, note string) {
		r.update(t, func(s *Status) { s.Progress, s.Note = p, note })
	})
	r.finish(t, err)
	if err != nil {
		return fmt.Errorf("%s: %w", title, err)
	}
	return nil
}

// Cancel asks a task to stop; it ends once its work notices the context.
func (r *Runner) Cancel(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.live[id]; ok {
		t.cancel()
	}
}

// List returns the unfinished tasks, oldest first.
func (r *Runner) List() []Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Status, 0, len(r.live))
	for _, t := range r.live {
		out = append(out, t.status)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started.Before(out[j].Started) })
	return out
}

func (r *Runner) add(title string, cancel context.CancelFunc) *task {
	r.mu.Lock()
	r.next++
	t := &task{cancel: cancel, status: Status{ID: fmt.Sprint(r.next), Title: title, State: Queued, Progress: -1, Started: time.Now()}}
	r.live[t.status.ID] = t
	s := t.status
	r.mu.Unlock()
	r.notify(s)
	return t
}

func (r *Runner) update(t *task, change func(*Status)) {
	r.mu.Lock()
	change(&t.status)
	s := t.status
	r.mu.Unlock()
	r.notify(s)
}

func (r *Runner) finish(t *task, err error) {
	r.mu.Lock()
	switch {
	case err == nil:
		t.status.State, t.status.Progress = Done, 1
	case errors.Is(err, context.Canceled):
		t.status.State = Cancelled
	default:
		t.status.State, t.status.Error = Failed, err.Error()
	}
	delete(r.live, t.status.ID)
	s := t.status
	r.mu.Unlock()
	r.notify(s)
}
