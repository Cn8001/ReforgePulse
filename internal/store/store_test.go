package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/recorder"
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

func TestAnalyses_BelongToTheNameAndTheDumpTime(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewAnalyses(db)
	first := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)

	// Act
	saveErr := s.Save(ctx, "MEMORY.DMP", first, domain.DumpAnalysis{Name: "MEMORY.DMP", Image: "nvlddmkm.sys"})
	same, err1 := s.Get(ctx, "MEMORY.DMP", first)
	newer, err2 := s.Get(ctx, "MEMORY.DMP", first.Add(time.Hour)) // overwritten by a later crash

	// Assert
	if saveErr != nil || err1 != nil || err2 != nil || same == nil || same.Image != "nvlddmkm.sys" || newer != nil {
		t.Fatalf("save %v, same %+v %v, newer %+v %v", saveErr, same, err1, newer, err2)
	}
}

func TestRecordings_KeepTheNewestAndListWithoutSeries(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewRecordings(db, 2)
	start := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

	// Act
	for i := range 3 {
		if _, err := s.Save(ctx, recorder.Recording{Started: start.Add(time.Duration(i) * time.Hour), CPU: []float64{1, 2}}); err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.All(ctx)
	full, getErr := s.Get(ctx, all[0].ID)

	// Assert
	if err != nil || len(all) != 2 || all[0].ID != "20261117-110000" || all[0].CPU != nil {
		t.Fatalf("all = %+v, %v", all, err)
	}
	if getErr != nil || len(full.CPU) != 2 {
		t.Fatalf("get = %+v, %v", full, getErr)
	}
}

func TestDeleteBefore_RecordingsAndAnalyses(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	recs, an := NewRecordings(db, 10), NewAnalyses(db)
	old, recent := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	for _, at := range []time.Time{old, recent} {
		if _, err := recs.Save(ctx, recorder.Recording{Started: at}); err != nil {
			t.Fatal(err)
		}
		if err := an.Save(ctx, "MEMORY.DMP", at, domain.DumpAnalysis{Name: "MEMORY.DMP"}); err != nil {
			t.Fatal(err)
		}
	}
	cut := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	// Act
	nr, err1 := recs.DeleteBefore(ctx, cut)
	na, err2 := an.DeleteBefore(ctx, cut)

	// Assert
	leftR, _ := recs.All(ctx)
	leftA, _ := an.All(ctx)
	if err1 != nil || err2 != nil || nr != 1 || na != 1 || len(leftR) != 1 || len(leftA) != 1 {
		t.Fatalf("recordings %d %v left %d, analyses %d %v left %d", nr, err1, len(leftR), na, err2, len(leftA))
	}
}

func TestChecks_KeepTheNewest(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewChecks(db, 2)
	at := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)

	// Act
	for i := range 3 {
		r := domain.CheckResult{Name: "sfc", At: at.Add(time.Duration(i) * time.Minute), Summary: "no integrity violations", Log: `C:\Windows\Logs\CBS\CBS.log`, Output: "not kept"}
		if _, err := s.Save(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.All(ctx)
	gone, delErr := s.DeleteBefore(ctx, at.Add(2*time.Minute))

	// Assert
	if err != nil || len(all) != 2 || !all[0].At.Equal(at.Add(2*time.Minute)) || all[0].Summary == "" || all[0].Log == "" || all[0].Output != "" {
		t.Fatalf("all = %+v, %v", all, err)
	}
	if delErr != nil || gone != 1 {
		t.Fatalf("delete before: %d, %v", gone, delErr)
	}
}
