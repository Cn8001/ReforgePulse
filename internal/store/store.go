// Package store is the app's one SQLite database: the change log, saved
// scans, resolved findings and settings. The window and the MCP server open
// it at the same time, so it runs in WAL mode with a busy timeout.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure Go: the Windows build has no C toolchain
)

// Stamp is how times are stored: fixed width, so as text they sort and
// compare in time order, which RFC 3339 with trimmed fractions does not.
const Stamp = "2006-01-02T15:04:05.000000000Z"

// migrations run in order, once each; the database's user_version says how
// many already have. Append, never edit.
var migrations = []string{
	`CREATE TABLE changes (
		seq        INTEGER PRIMARY KEY AUTOINCREMENT,
		id         TEXT NOT NULL UNIQUE,
		at         TEXT NOT NULL,
		action     TEXT NOT NULL DEFAULT '',
		summary    TEXT NOT NULL DEFAULT '',
		target     TEXT NOT NULL DEFAULT '',
		before     TEXT NOT NULL DEFAULT '',
		after      TEXT NOT NULL DEFAULT '',
		reversible INTEGER NOT NULL DEFAULT 0,
		reverts    TEXT NOT NULL DEFAULT '',
		status     TEXT NOT NULL,
		error      TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE reports (
		id   TEXT PRIMARY KEY,
		at   TEXT NOT NULL,
		body TEXT NOT NULL
	);
	CREATE TABLE resolutions (
		key   TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		at    TEXT NOT NULL
	);
	CREATE TABLE settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,
	// A dump file name comes back (MEMORY.DMP is overwritten), so an
	// analysis belongs to the name and the dump's own time together.
	`CREATE TABLE dump_analyses (
		name   TEXT NOT NULL,
		dumped TEXT NOT NULL,
		body   TEXT NOT NULL,
		PRIMARY KEY (name, dumped)
	);
	CREATE TABLE recordings (
		id   TEXT PRIMARY KEY,
		body TEXT NOT NULL
	);`,
	`CREATE TABLE check_results (
		id        TEXT PRIMARY KEY,
		name      TEXT NOT NULL,
		exit_code INTEGER NOT NULL,
		summary   TEXT NOT NULL,
		log       TEXT NOT NULL DEFAULT ''
	);`,
	`CREATE TABLE mcp_log (
		seq    INTEGER PRIMARY KEY AUTOINCREMENT,
		at     TEXT NOT NULL,
		pid    INTEGER NOT NULL,
		client TEXT NOT NULL,
		event  TEXT NOT NULL,
		tool   TEXT NOT NULL DEFAULT '',
		millis INTEGER NOT NULL DEFAULT 0,
		error  TEXT NOT NULL DEFAULT ''
	);`,
}

// Open opens path, creating it and its folder when needed, and brings the
// schema up to date.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create data folder: %w", err)
	}
	dsn := "file:" + filepath.ToSlash(path) +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=synchronous(FULL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer tx.Rollback()
	var version int
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version > len(migrations) {
		return fmt.Errorf("the database is from a newer ReforgePulse (schema %d, this one knows %d)", version, len(migrations))
	}
	for i := version; i < len(migrations); i++ {
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			return fmt.Errorf("migrate to schema %d: %w", i+1, err)
		}
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", len(migrations))); err != nil {
		return fmt.Errorf("set schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	return nil
}
