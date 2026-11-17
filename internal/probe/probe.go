// Package probe runs the reads that touch hardware directly (CPUID pinned to
// each processor, per-core counters, GPU sensors, the core test) in a child
// process. Native code that goes wrong, or a processor that computes wrong,
// can take a Go process down in a way no recover can catch; in the child it
// takes only the child, and the window reports it and starts a new one.
//
// The protocol is one JSON object per line: requests on the child's stdin,
// progress and responses on its stdout.
package probe

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

type request struct {
	ID     uint64          `json:"id"`
	Method string          `json:"method"` // "cancel" stops request ID
	Args   json.RawMessage `json:"args,omitempty"`
}

type reply struct {
	ID       uint64          `json:"id"`
	Result   json.RawMessage `json:"result,omitempty"`
	Error    string          `json:"error,omitempty"`
	Progress *progress       `json:"progress,omitempty"` // set on interim lines; the last line has none
}

type progress struct {
	Done float64 `json:"done"`
	Note string  `json:"note"`
}

// Handler answers one method. report sends progress back while it runs.
type Handler func(ctx context.Context, args json.RawMessage, report func(float64, string)) (any, error)

// Serve answers requests until in closes or ctx ends. Each request runs on
// its own goroutine, so a long core test does not hold up the live loads.
func Serve(ctx context.Context, in io.Reader, out io.Writer, handlers map[string]Handler) error {
	var mu sync.Mutex
	enc := json.NewEncoder(out)
	send := func(r reply) {
		mu.Lock()
		defer mu.Unlock()
		_ = enc.Encode(r) // a closed stdout means the window is gone, and so is the reader
	}
	var running sync.Map // request ID to its cancel func
	var wg sync.WaitGroup
	defer wg.Wait()

	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var req request
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			continue
		}
		if req.Method == "cancel" {
			if cancel, ok := running.Load(req.ID); ok {
				cancel.(context.CancelFunc)()
			}
			continue
		}
		h, ok := handlers[req.Method]
		if !ok {
			send(reply{ID: req.ID, Error: "unknown method " + req.Method})
			continue
		}
		rctx, cancel := context.WithCancel(ctx)
		running.Store(req.ID, cancel)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			defer running.Delete(req.ID)
			res, err := h(rctx, req.Args, func(done float64, note string) {
				send(reply{ID: req.ID, Progress: &progress{Done: done, Note: note}})
			})
			r := reply{ID: req.ID}
			if err != nil {
				r.Error = err.Error()
			} else if r.Result, err = json.Marshal(res); err != nil {
				r.Error = fmt.Sprintf("encode result: %v", err)
			}
			send(r)
		}()
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read requests: %w", err)
	}
	return nil
}
