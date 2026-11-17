package agents

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cn8001/reforgepulse/internal/domain"
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

func TestConnector_RegisteredTellsThisCopyFromAnother(t *testing.T) {
	// Arrange
	exe := `C:\Program Files\ReforgePulse\reforgepulse.exe`
	answer := func(out string) func(context.Context, string, ...string) (string, error) {
		return func(context.Context, string, ...string) (string, error) { return out, nil }
	}
	this := New(Deps{Exe: exe, LookPath: found, Run: answer("reforgepulse:\n  command: c:\\program files\\reforgepulse\\reforgepulse.exe mcp")})
	other := New(Deps{Exe: exe, LookPath: found, Run: answer(`reforgepulse: C:\Program Files\Old\ReforgePulse\reforgepulse.exe mcp`)})
	none := New(Deps{Exe: exe, LookPath: found, Run: func(context.Context, string, ...string) (string, error) {
		return "No MCP server named reforgepulse", errors.New("exit status 1")
	}})
	missing := New(Deps{Exe: exe, LookPath: func(string) (string, error) { return "", errors.New("not found") }})

	// Act
	got := []domain.AgentStatus{
		this.Registered(context.Background(), "claude"), other.Registered(context.Background(), "codex"),
		none.Registered(context.Background(), "claude"), missing.Registered(context.Background(), "claude"),
	}

	// Assert
	if !got[0].Registered || got[1].Registered || !strings.Contains(got[1].Note, "another copy") ||
		got[2].Registered || !got[2].Installed || got[3].Installed {
		t.Fatalf("registrations = %+v", got)
	}
}
