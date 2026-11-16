package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// serve runs the server over the given request lines and decodes each answer.
func serve(t *testing.T, tools []Tool, lines ...string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := Serve(context.Background(), strings.NewReader(strings.Join(lines, "\n")+"\n"), &out, tools); err != nil {
		t.Fatal(err)
	}
	var answers []map[string]any
	dec := json.NewDecoder(&out)
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			t.Fatal(err)
		}
		answers = append(answers, m)
	}
	return answers
}

var testTools = []Tool{
	{Name: "ok", Schema: noArgs, Call: func(context.Context, json.RawMessage) (any, error) { return map[string]int{"answer": 42}, nil }},
	{Name: "fails", Schema: noArgs, Call: func(context.Context, json.RawMessage) (any, error) { return nil, errors.New("no access") }},
}

func TestServe_InitializeAndIgnoresNotifications(t *testing.T) {
	// Act
	got := serve(t, testTools,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`)

	// Assert
	if len(got) != 1 {
		t.Fatalf("got %d answers, want 1", len(got))
	}
	result := got[0]["result"].(map[string]any)
	if result["protocolVersion"] != ProtocolVersion || result["serverInfo"].(map[string]any)["name"] != Name {
		t.Fatalf("initialize = %v", result)
	}
}

func TestServe_ListsTools(t *testing.T) {
	// Act
	got := serve(t, testTools, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)

	// Assert
	tools := got[0]["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 2 || tools[0].(map[string]any)["name"] != "ok" {
		t.Fatalf("tools = %v", tools)
	}
}

func TestServe_CallReturnsJSONText(t *testing.T) {
	// Act
	got := serve(t, testTools, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ok"}}`)

	// Assert
	result := got[0]["result"].(map[string]any)
	text := result["content"].([]any)[0].(map[string]any)["text"].(string)
	if result["isError"] != nil || !strings.Contains(text, `"answer": 42`) {
		t.Fatalf("result = %v", result)
	}
}

func TestServe_ToolErrorIsResult(t *testing.T) {
	// Act
	got := serve(t, testTools, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"fails"}}`)

	// Assert
	result := got[0]["result"].(map[string]any)
	if result["isError"] != true || got[0]["error"] != nil {
		t.Fatalf("answer = %v", got[0])
	}
}

func TestServe_UnknownMethod(t *testing.T) {
	// Act
	got := serve(t, testTools, `{"jsonrpc":"2.0","id":5,"method":"resources/list"}`)

	// Assert
	if code := got[0]["error"].(map[string]any)["code"]; code != float64(-32601) {
		t.Fatalf("code = %v, want -32601", code)
	}
}

func TestServe_ParseError(t *testing.T) {
	// Act
	got := serve(t, testTools, `{not json`)

	// Assert
	if code := got[0]["error"].(map[string]any)["code"]; code != float64(-32700) {
		t.Fatalf("code = %v, want -32700", code)
	}
}
