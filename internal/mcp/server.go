// Package mcp lets AI agents (Claude Code, Codex, OpenCode or any MCP
// client) read what ReforgePulse sees. It speaks the Model Context Protocol
// over stdio, one JSON-RPC message per line, so there is no port to expose.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const (
	Name            = "reforgepulse"
	ProtocolVersion = "2025-06-18"
)

// Tool is one capability offered to agents.
type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage // JSON Schema of the arguments
	Call        func(ctx context.Context, args json.RawMessage) (any, error)
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // absent on notifications
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// Observer hears what the server does, for the window's Agents panel.
type Observer interface {
	Started(client string)
	Called(tool string, took time.Duration, err error)
	Ended(err error)
}

type silent struct{}

func (silent) Started(string)                      {}
func (silent) Called(string, time.Duration, error) {}
func (silent) Ended(error)                         {}

// Serve answers requests from in on out until in ends or ctx is cancelled.
// version is the app's, from build/config.yml. obs may be nil.
func Serve(ctx context.Context, in io.Reader, out io.Writer, version string, tools []Tool, obs Observer) (err error) {
	if obs == nil {
		obs = silent{}
	}
	defer func() { obs.Ended(err) }()
	byName := make(map[string]Tool, len(tools))
	for _, t := range tools {
		byName[t.Name] = t
	}
	enc := json.NewEncoder(out)
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		var req request
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			if err := enc.Encode(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32700, "parse error"}}); err != nil {
				return fmt.Errorf("write response: %w", err)
			}
			continue
		}
		if len(req.ID) == 0 {
			continue // notifications such as notifications/initialized need no answer
		}
		result, rerr := handle(ctx, req, version, byName, tools, obs)
		if err := enc.Encode(response{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rerr}); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	return nil
}

func handle(ctx context.Context, req request, version string, byName map[string]Tool, tools []Tool, obs Observer) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		var p struct {
			ClientInfo struct {
				Name string `json:"name"`
			} `json:"clientInfo"`
		}
		_ = json.Unmarshal(req.Params, &p) // an agent that names nothing is still served
		obs.Started(p.ClientInfo.Name)
		return map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]string{"name": Name, "version": version},
			"instructions": "ReforgePulse diagnoses a Windows PC. These tools only read; " +
				"any change must be approved by the user in the ReforgePulse window.",
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		list := make([]map[string]any, 0, len(tools))
		for _, t := range tools {
			list = append(list, map[string]any{"name": t.Name, "description": t.Description, "inputSchema": t.Schema})
		}
		return map[string]any{"tools": list}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{-32602, "invalid params"}
		}
		t, ok := byName[p.Name]
		if !ok {
			return nil, &rpcError{-32602, "unknown tool: " + p.Name}
		}
		start := time.Now()
		v, err := t.Call(ctx, p.Arguments)
		obs.Called(p.Name, time.Since(start), err)
		return callResult(v, err), nil
	default:
		return nil, &rpcError{-32601, "method not found: " + req.Method}
	}
}

// callResult wraps a tool's answer as MCP text content. A tool failure is a
// result with isError, not a protocol error, so the agent can read it.
func callResult(v any, err error) map[string]any {
	if err != nil {
		return map[string]any{"isError": true, "content": []map[string]string{{"type": "text", "text": err.Error()}}}
	}
	b, merr := json.MarshalIndent(v, "", " ")
	if merr != nil {
		return map[string]any{"isError": true, "content": []map[string]string{{"type": "text", "text": merr.Error()}}}
	}
	return map[string]any{"content": []map[string]string{{"type": "text", "text": string(b)}}}
}
