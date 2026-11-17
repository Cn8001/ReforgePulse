package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type AgentConnector interface {
	Connect(ctx context.Context, agent string) (string, error)
	Registered(ctx context.Context, agent string) domain.AgentStatus // Installed, Registered, Note
}

// McpLogReader is what agents' MCP servers recorded, newest first.
type McpLogReader interface {
	Recent(ctx context.Context, limit int) ([]domain.McpEvent, error)
}

type AgentDeps struct {
	Exe     string
	Conn    AgentConnector
	Journal Recorder
	Log     McpLogReader
	Alive   func(pid uint32) bool
	Procs   ProcessLister // to see servers that keep no log, such as an older copy
}

// AgentService registers the MCP server with an agent at a click, and says
// whether each agent is set up and connected, with what its server did.
type AgentService struct {
	d AgentDeps
}

func NewAgentService(d AgentDeps) *AgentService { return &AgentService{d: d} }

// Agents are the agents the panel knows, in its order.
var Agents = []string{"claude", "codex", "opencode"}

// Executable is the path agents run with the "mcp" argument.
func (s *AgentService) Executable() string {
	return s.d.Exe
}

// Connect changes the agent's own config, so it goes in the change log too.
func (s *AgentService) Connect(ctx context.Context, agent string) (string, error) {
	id, err := s.d.Journal.Begin(ctx, domain.Change{
		Action: "agent-connect", Summary: "Connect " + agent + " to the ReforgePulse MCP server", Target: agent,
	})
	if err != nil {
		return "", fmt.Errorf("log agent connect: %w", err)
	}
	out, err := s.d.Conn.Connect(ctx, agent)
	if ferr := s.d.Journal.Finish(ctx, id, out, err); ferr != nil && err == nil {
		err = ferr
	}
	if err != nil {
		return out, fmt.Errorf("connect %s: %w", agent, err)
	}
	return out, nil
}

// statusWait bounds how long one agent's CLI may take to answer.
const statusWait = 15 * time.Second

// shownEvents is how much of each agent's log the panel lists.
const shownEvents = 30

// Status is each agent: installed, set up for this copy, connected now, and
// its server's recent activity. The CLIs are slow, so they are asked at once.
func (s *AgentService) Status(ctx context.Context) ([]domain.AgentStatus, error) {
	events, err := s.d.Log.Recent(ctx, 500)
	if err != nil {
		return nil, fmt.Errorf("read mcp log: %w", err)
	}
	out := make([]domain.AgentStatus, len(Agents))
	var wg sync.WaitGroup
	for i, a := range Agents {
		wg.Go(func() {
			cctx, cancel := context.WithTimeout(ctx, statusWait)
			defer cancel()
			st := s.d.Conn.Registered(cctx, a)
			st.Agent = a
			out[i] = st
		})
	}
	wg.Wait()
	for i := range out {
		out[i].Events = []domain.McpEvent{}
		latest := map[uint32]string{} // the newest event of each server process
		for _, e := range events {
			if agentOf(e.Client) != out[i].Agent {
				continue
			}
			if _, seen := latest[e.PID]; !seen {
				latest[e.PID] = e.Event
			}
			if len(out[i].Events) < shownEvents {
				out[i].Events = append(out[i].Events, e)
			}
		}
		for pid, last := range latest {
			if last != "end" && s.d.Alive(pid) {
				out[i].Live = true
			}
		}
	}
	// A server the agent started is its child, whatever copy of ReforgePulse
	// it runs and whether that copy keeps the log.
	if ps, err := s.d.Procs.List(ctx); err == nil {
		byPID := make(map[uint32]domain.Process, len(ps))
		for _, p := range ps {
			byPID[p.PID] = p
		}
		for _, p := range ps {
			parent, ok := byPID[p.ParentPID]
			if !ok || !strings.EqualFold(p.Name, "reforgepulse.exe") {
				continue
			}
			for i := range out {
				if agentOf(parent.Name) == out[i].Agent {
					out[i].Live = true
				}
			}
		}
	}
	return out, nil
}

// agentOf maps the name an agent gives when it connects to the panel's.
func agentOf(client string) string {
	c := strings.ToLower(client)
	for _, a := range Agents {
		if strings.Contains(c, a) {
			return a
		}
	}
	return ""
}
