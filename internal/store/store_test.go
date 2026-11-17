package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_CreatesTheSchemaOnceAndReopens(t *testing.T) {
	// Arrange
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "sub", "app.db")

	// Act
	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	_, insertErr := db.ExecContext(ctx, `INSERT INTO settings (key, value) VALUES ('a', 'b')`)
	db.Close()
	again, reopenErr := Open(ctx, path)
	var value string
	var version int
	if reopenErr == nil {
		defer again.Close()
		_ = again.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = 'a'`).Scan(&value)
		_ = again.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version)
	}

	// Assert
	if insertErr != nil || reopenErr != nil || value != "b" || version != len(migrations) {
		t.Fatalf("insert %v, reopen %v, value %q, version %d", insertErr, reopenErr, value, version)
	}
}

func TestImportFiles_MovesOldFilesInOnce(t *testing.T) {
	// Arrange
	ctx := context.Background()
	dir := t.TempDir()
	journal := `{"id":"aaaaaaa","at":"2026-11-16T10:00:00Z","summary":"Disable at startup: Warp","reversible":true,"status":"started"}
{"id":"aaaaaaa","at":"2026-11-16T10:00:01Z","status":"applied","after":"off"}
{"id":"bbb","summ`
	mustWrite(t, filepath.Join(dir, "journal.jsonl"), journal)
	mustWrite(t, filepath.Join(dir, "reports", "20261117-052000-000.json"), `{"id":"20261117-052000-000","findings":[]}`)
	mustWrite(t, filepath.Join(dir, "resolved.json"), `[{"key":"whea","title":"WHEA","at":"2026-11-17T05:00:00Z"}]`)
	mustWrite(t, filepath.Join(dir, "accepted.json"), `{"version":"x"}`)
	db, err := Open(ctx, filepath.Join(dir, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Act
	err = ImportFiles(ctx, db, dir)
	again := ImportFiles(ctx, db, dir)

	// Assert
	var status, after string
	var reports, resolutions int
	_ = db.QueryRowContext(ctx, `SELECT status, after FROM changes WHERE id = 'aaaaaaa'`).Scan(&status, &after)
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM reports`).Scan(&reports)
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM resolutions`).Scan(&resolutions)
	if err != nil || again != nil || status != "applied" || after != "off" || reports != 1 || resolutions != 1 {
		t.Fatalf("err %v/%v, change %s %s, reports %d, resolutions %d", err, again, status, after, reports, resolutions)
	}
	for _, name := range []string{"journal.jsonl", "reports", "resolved.json", "accepted.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s was left behind", name)
		}
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSettings_DefaultUntilSet(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewSettings(db)

	// Act
	before, _ := s.Get(ctx, "updates.on_start", "on")
	setErr := s.Set(ctx, "updates.on_start", "off")
	after, _ := s.Get(ctx, "updates.on_start", "on")

	// Assert
	if before != "on" || setErr != nil || after != "off" {
		t.Fatalf("before %q, set %v, after %q", before, setErr, after)
	}
}
