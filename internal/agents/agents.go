// Package agents registers ReforgePulse's MCP server with AI coding agents,
// so the user never has to type a command: Claude Code and Codex through
// their own CLIs, OpenCode by merging its JSON config.
package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotFound means the agent is not installed where Windows can find it.
var ErrNotFound = errors.New("agent not found")

// Deps are the outside world the connector touches.
type Deps struct {
	Exe      string                                                                 // reforgepulse.exe, started with "mcp"
	Home     string                                                                 // the user's profile folder
	LookPath func(file string) (string, error)                                      // exec.LookPath
	Run      func(ctx context.Context, name string, args ...string) (string, error) // runs a program, returns its output
}

type Connector struct{ d Deps }

func New(d Deps) *Connector { return &Connector{d: d} }

const name = "reforgepulse"

// Connect registers the server with one agent: "claude", "codex" or
// "opencode". It replaces an earlier registration, so it is safe to repeat.
func (c *Connector) Connect(ctx context.Context, agent string) (string, error) {
	switch agent {
	case "claude":
		return c.viaCLI(ctx, "claude", []string{"mcp", "remove", "--scope", "user", name},
			[]string{"mcp", "add", "--scope", "user", name, "--", c.d.Exe, "mcp"})
	case "codex":
		return c.viaCLI(ctx, "codex", []string{"mcp", "remove", name},
			[]string{"mcp", "add", name, "--", c.d.Exe, "mcp"})
	case "opencode":
		return c.openCode()
	}
	return "", fmt.Errorf("unknown agent %q", agent)
}

func (c *Connector) viaCLI(ctx context.Context, cli string, remove, add []string) (string, error) {
	path, err := c.d.LookPath(cli)
	if err != nil {
		return "", fmt.Errorf("%s: %w", cli, ErrNotFound)
	}
	_, _ = c.d.Run(ctx, path, remove...) // nothing to remove the first time
	out, err := c.d.Run(ctx, path, add...)
	if err != nil {
		return out, fmt.Errorf("%s mcp add: %w", cli, err)
	}
	return out, nil
}

// openCode merges the server into ~/.config/opencode/opencode.json and
// keeps every other setting as it was.
func (c *Connector) openCode() (string, error) {
	path := filepath.Join(c.d.Home, ".config", "opencode", "opencode.json")
	cfg := map[string]any{"$schema": "https://opencode.ai/config.json"}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return "", fmt.Errorf("read %s: %w", path, err) // never overwrite a config we cannot read
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	servers, _ := cfg["mcp"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers[name] = map[string]any{"type": "local", "command": []string{c.d.Exe, "mcp"}, "enabled": true}
	cfg["mcp"] = servers

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode opencode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create opencode config dir: %w", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return "added to " + path, nil
}
