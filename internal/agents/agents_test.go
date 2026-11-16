package agents

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type runs struct{ calls []string }

func (r *runs) run(_ context.Context, name string, args ...string) (string, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	return "ok", nil
}

func found(file string) (string, error) { return `C:\bin\` + file + ".exe", nil }

func TestConnector_ClaudeReplacesRegistration(t *testing.T) {
	// Arrange
	r := &runs{}
	c := New(Deps{Exe: `C:\RP\reforgepulse.exe`, LookPath: found, Run: r.run})

	// Act
	_, err := c.Connect(context.Background(), "claude")

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 2 || !strings.Contains(r.calls[0], "mcp remove") ||
		!strings.HasSuffix(r.calls[1], `mcp add --scope user reforgepulse -- C:\RP\reforgepulse.exe mcp`) {
		t.Fatalf("calls = %q", r.calls)
	}
}

func TestConnector_MissingCLI(t *testing.T) {
	// Arrange
	c := New(Deps{LookPath: func(string) (string, error) { return "", errors.New("no") }, Run: (&runs{}).run})

	// Act
	_, err := c.Connect(context.Background(), "codex")

	// Assert
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestConnector_OpenCodeKeepsOtherSettings(t *testing.T) {
	// Arrange
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "opencode.json")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(`{"theme":"dark","mcp":{"other":{"type":"local"}}}`), 0o644)
	c := New(Deps{Exe: `C:\RP\reforgepulse.exe`, Home: home})

	// Act
	_, err := c.Connect(context.Background(), "opencode")

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	b, _ := os.ReadFile(path)
	_ = json.Unmarshal(b, &cfg)
	mcp := cfg["mcp"].(map[string]any)
	if cfg["theme"] != "dark" || mcp["other"] == nil || mcp["reforgepulse"] == nil {
		t.Fatalf("config = %s", b)
	}
}

func TestConnector_OpenCodeRefusesBrokenConfig(t *testing.T) {
	// Arrange
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "opencode.json")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(`{ broken`), 0o644)
	c := New(Deps{Home: home})

	// Act
	_, err := c.Connect(context.Background(), "opencode")

	// Assert
	b, _ := os.ReadFile(path)
	if err == nil || string(b) != `{ broken` {
		t.Fatalf("err = %v, file = %s", err, b)
	}
}
