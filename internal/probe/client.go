package probe

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// ErrStopped is what calls get when the child dies under them. The next call
// starts a new child.
var ErrStopped = errors.New("the hardware reader stopped unexpectedly; it starts again on the next try")

// Child is a running probe process.
type Child struct {
	In   io.WriteCloser
	Out  io.Reader
	Wait func() error // after Out reached its end
	Kill func()
}

// Start launches a child.
type Start func() (*Child, error)

// Command starts exe with args as the child; its stderr, where a crash
// report lands, goes to stderr.
func Command(exe string, stderr io.Writer, args ...string) Start {
	return func() (*Child, error) {
		cmd := exec.Command(exe, args...)
		cmd.Stderr = stderr
		in, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("probe stdin: %w", err)
		}
		out, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("probe stdout: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("start probe: %w", err)
		}
		return &Child{In: in, Out: out, Wait: cmd.Wait, Kill: func() { _ = cmd.Process.Kill() }}, nil
	}
}

type call struct {
	child   *Child
	replies chan reply // closed when the child dies
}

// Client sends calls to the child, starting it when needed.
type Client struct {
	start  Start
	onExit func(error) // told why a child ended, for the log

	mu      sync.Mutex
	child   *Child
	enc     *json.Encoder
	next    uint64
	pending map[uint64]*call
}

func NewClient(start Start, onExit func(error)) *Client {
	return &Client{start: start, onExit: onExit, pending: map[uint64]*call{}}
}

// Call runs method in the child and decodes its result into out. report,
// when set, hears the child's progress.
func (c *Client) Call(ctx context.Context, method string, args, out any, report func(float64, string)) error {
	raw, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("encode %s arguments: %w", method, err)
	}
	c.mu.Lock()
	if c.child == nil {
		ch, err := c.start()
		if err != nil {
			c.mu.Unlock()
			return err
		}
		c.child, c.enc = ch, json.NewEncoder(ch.In)
		go c.read(ch)
	}
	c.next++
	id := c.next
	cl := &call{child: c.child, replies: make(chan reply, 64)}
	c.pending[id] = cl
	err = c.enc.Encode(request{ID: id, Method: method, Args: raw})
	c.mu.Unlock()
	if err != nil {
		c.forget(id)
		return fmt.Errorf("send %s: %w", method, ErrStopped)
	}

	for {
		select {
		case r, ok := <-cl.replies:
			if !ok {
				return fmt.Errorf("%s: %w", method, ErrStopped)
			}
			if r.Progress != nil {
				if report != nil {
					report(r.Progress.Done, r.Progress.Note)
				}
				continue
			}
			c.forget(id)
			if r.Error != "" {
				return errors.New(r.Error)
			}
			if out == nil {
				return nil
			}
			if err := json.Unmarshal(r.Result, out); err != nil {
				return fmt.Errorf("decode %s result: %w", method, err)
			}
			return nil
		case <-ctx.Done():
			c.forget(id)
			c.mu.Lock()
			if c.child == cl.child {
				_ = c.enc.Encode(request{ID: id, Method: "cancel"})
			}
			c.mu.Unlock()
			return ctx.Err()
		}
	}
}

func (c *Client) forget(id uint64) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// read hands each reply to its call until the child's output ends, then
// fails the calls that were waiting on that child.
func (c *Client) read(ch *Child) {
	sc := bufio.NewScanner(ch.Out)
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var r reply
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			continue
		}
		c.mu.Lock()
		cl := c.pending[r.ID]
		c.mu.Unlock()
		if cl == nil {
			continue
		}
		select {
		case cl.replies <- r:
		default: // a caller this far behind only misses progress lines
		}
	}
	err := ch.Wait()
	if err == nil {
		err = sc.Err() // a broken pipe here also means the child is gone
	}
	c.mu.Lock()
	if c.child == ch {
		c.child, c.enc = nil, nil
	}
	for id, cl := range c.pending {
		if cl.child == ch {
			close(cl.replies)
			delete(c.pending, id)
		}
	}
	c.mu.Unlock()
	if c.onExit != nil {
		c.onExit(err)
	}
}

// Close stops the child.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.child != nil {
		_ = c.child.In.Close()
		c.child.Kill()
	}
}
