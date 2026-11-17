package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

// McpLog is what agents' MCP servers did. Each server writes its own lines;
// the window reads them to say whether an agent is connected.
type McpLog struct {
	db   *sql.DB
	keep int
}

func NewMcpLog(db *sql.DB, keep int) *McpLog { return &McpLog{db: db, keep: keep} }

// Add records an event and drops the oldest past keep.
func (l *McpLog) Add(ctx context.Context, e domain.McpEvent) error {
	if _, err := l.db.ExecContext(ctx, `INSERT INTO mcp_log (at, pid, client, event, tool, millis, error) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.At.UTC().Format(Stamp), e.PID, e.Client, e.Event, e.Tool, e.Millis, e.Error); err != nil {
		return fmt.Errorf("record mcp event: %w", err)
	}
	if _, err := l.db.ExecContext(ctx, `DELETE FROM mcp_log WHERE seq <= (SELECT MAX(seq) FROM mcp_log) - ?`, l.keep); err != nil {
		return fmt.Errorf("trim mcp log: %w", err)
	}
	return nil
}

// Recent returns up to limit events, newest first.
func (l *McpLog) Recent(ctx context.Context, limit int) ([]domain.McpEvent, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT at, pid, client, event, tool, millis, error FROM mcp_log ORDER BY seq DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("read mcp log: %w", err)
	}
	defer rows.Close()
	out := []domain.McpEvent{}
	for rows.Next() {
		var e domain.McpEvent
		var at string
		if err := rows.Scan(&at, &e.PID, &e.Client, &e.Event, &e.Tool, &e.Millis, &e.Error); err != nil {
			return nil, fmt.Errorf("read mcp event: %w", err)
		}
		e.At, _ = time.Parse(Stamp, at)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read mcp log: %w", err)
	}
	return out, nil
}
