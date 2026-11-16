package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type AgentConnector interface {
	Connect(ctx context.Context, agent string) (string, error)
}

// AgentService registers the MCP server with an agent at a click, and
// tells the panel the command for agents it cannot reach (such as one in WSL).
type AgentService struct {
	exe     string
	conn    AgentConnector
	journal Recorder
}

func NewAgentService(exe string, conn AgentConnector, j Recorder) *AgentService {
	return &AgentService{exe: exe, conn: conn, journal: j}
}

// Executable is the path agents run with the "mcp" argument.
func (s *AgentService) Executable() string {
	return s.exe
}

// Connect changes the agent's own config, so it goes in the change log too.
func (s *AgentService) Connect(ctx context.Context, agent string) (string, error) {
	id, err := s.journal.Begin(ctx, domain.Change{
		Action: "agent-connect", Summary: "Connect " + agent + " to the ReforgePulse MCP server", Target: agent,
	})
	if err != nil {
		return "", fmt.Errorf("log agent connect: %w", err)
	}
	out, err := s.conn.Connect(ctx, agent)
	if ferr := s.journal.Finish(ctx, id, out, err); ferr != nil && err == nil {
		err = ferr
	}
	if err != nil {
		return out, fmt.Errorf("connect %s: %w", agent, err)
	}
	return out, nil
}
