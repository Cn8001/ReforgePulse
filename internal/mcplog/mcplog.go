// Package mcplog writes what an agent's MCP server does to the app's log,
// so the window can say which agent is connected and what it asked for.
package mcplog

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type Writer interface {
	Add(ctx context.Context, e domain.McpEvent) error
}

// Recorder is the MCP server's observer for one process.
type Recorder struct {
	w      Writer
	pid    uint32
	now    func() time.Time
	mu     sync.Mutex
	client string
}

func New(w Writer, pid uint32) *Recorder { return &Recorder{w: w, pid: pid, now: time.Now} }

func (r *Recorder) Started(client string) {
	r.mu.Lock()
	r.client = client
	r.mu.Unlock()
	r.add(domain.McpEvent{Event: "start"})
}

func (r *Recorder) Called(tool string, took time.Duration, err error) {
	r.add(domain.McpEvent{Event: "call", Tool: tool, Millis: took.Milliseconds(), Error: text(err)})
}

func (r *Recorder) Ended(err error) {
	r.add(domain.McpEvent{Event: "end", Error: text(err)})
}

// add never fails the agent's request: a log that cannot be written is
// reported on stderr, which the agent keeps, and the work goes on.
func (r *Recorder) add(e domain.McpEvent) {
	r.mu.Lock()
	e.Client = r.client
	r.mu.Unlock()
	e.At, e.PID = r.now(), r.pid
	if err := r.w.Add(context.Background(), e); err != nil {
		log.Printf("mcp log: %v", err)
	}
}

func text(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
