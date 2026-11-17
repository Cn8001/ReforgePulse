package mcplog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type memWriter struct{ got []domain.McpEvent }

func (m *memWriter) Add(_ context.Context, e domain.McpEvent) error {
	m.got = append(m.got, e)
	return nil
}

func TestRecorder_WritesStartCallsAndEndUnderTheClientName(t *testing.T) {
	// Arrange
	w := &memWriter{}
	r := New(w, 42)

	// Act
	r.Started("codex-mcp-client")
	r.Called("scan_pc", 1500*time.Millisecond, nil)
	r.Called("get_hardware", 0, errors.New("no access"))
	r.Ended(nil)

	// Assert
	if len(w.got) != 4 || w.got[1].Tool != "scan_pc" || w.got[1].Millis != 1500 || w.got[2].Error != "no access" {
		t.Fatalf("events = %+v", w.got)
	}
	for _, e := range w.got {
		if e.Client != "codex-mcp-client" || e.PID != 42 {
			t.Fatalf("event %+v lost its client or pid", e)
		}
	}
}
