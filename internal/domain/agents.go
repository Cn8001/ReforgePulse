package domain

import "time"

// McpEvent is one thing an agent's MCP server did: started for a client,
// answered a tool call, or ended.
type McpEvent struct {
	At     time.Time `json:"at"`
	PID    uint32    `json:"pid"`
	Client string    `json:"client"` // the name the agent gave when it connected, such as claude-code
	Event  string    `json:"event"`  // start, call or end
	Tool   string    `json:"tool,omitempty"`
	Millis int64     `json:"millis,omitempty"`
	Error  string    `json:"error,omitempty"`
}

// AgentStatus is one agent as the Agents panel shows it.
type AgentStatus struct {
	Agent      string     `json:"agent"` // claude, codex or opencode
	Installed  bool       `json:"installed"`
	Registered bool       `json:"registered"`     // its config starts this copy of ReforgePulse
	Note       string     `json:"note,omitempty"` // why not, in a sentence
	Live       bool       `json:"live"`           // its server is running now
	Events     []McpEvent `json:"events"`         // newest first
}
