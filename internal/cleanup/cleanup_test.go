package cleanup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

type fakeJournal struct {
	begun    []domain.Change
	finished []string
}

func (f *fakeJournal) Begin(_ context.Context, c domain.Change) (string, error) {
	f.begun = append(f.begun, c)
	return "abc1234", nil
}

func (f *fakeJournal) Finish(_ context.Context, _, after string, _ error) error {
	f.finished = append(f.finished, after)
	return nil
}

// write creates a file of size bytes and backdates it by age.
func write(t *testing.T, path string, size int, age time.Duration) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-age)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
}

func TestEngine_AnalyzeCountsOnlyOldFiles(t *testing.T) {
	// Arrange
	root := t.TempDir()
	write(t, filepath.Join(root, "old.tmp"), 100, 48*time.Hour)
	write(t, filepath.Join(root, "sub", "old2.tmp"), 50, 48*time.Hour)
	write(t, filepath.Join(root, "fresh.tmp"), 999, time.Minute)
	e := New([]platform.Location{{ID: "temp", Paths: []string{root}, MinAge: 24 * time.Hour}}, &fakeJournal{})

	// Act
	areas, err := e.Analyze(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if areas[0].Files != 2 || areas[0].Bytes != 150 {
		t.Fatalf("area = %+v, want 2 files, 150 bytes", areas[0])
	}
}

func TestEngine_CleanDeletesLogsAndKeepsRoot(t *testing.T) {
	// Arrange
	root := t.TempDir()
	write(t, filepath.Join(root, "a", "b", "old.tmp"), 100, 48*time.Hour)
	write(t, filepath.Join(root, "fresh.tmp"), 10, time.Minute)
	j := &fakeJournal{}
	e := New([]platform.Location{{ID: "temp", Name: "Temp", Paths: []string{root}, MinAge: 24 * time.Hour}}, j)

	// Act
	r, err := e.Clean(context.Background(), []string{"temp"})

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if r.Deleted != 1 || r.Freed != 100 || r.ChangeID != "abc1234" {
		t.Fatalf("result = %+v", r)
	}
	if _, err := os.Stat(filepath.Join(root, "a")); !os.IsNotExist(err) {
		t.Fatal("emptied folders were left behind")
	}
	if _, err := os.Stat(filepath.Join(root, "fresh.tmp")); err != nil {
		t.Fatal("a fresh file was deleted")
	}
	if len(j.begun) != 1 || j.begun[0].Action != "cleanup" || len(j.finished) != 1 {
		t.Fatalf("journal = %+v / %v", j.begun, j.finished)
	}
}

func TestEngine_DoesNotFollowLinks(t *testing.T) {
	// Arrange
	root, outside := t.TempDir(), t.TempDir()
	write(t, filepath.Join(outside, "precious.doc"), 100, 48*time.Hour)
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	e := New([]platform.Location{{ID: "temp", Paths: []string{root}, MinAge: time.Hour}}, &fakeJournal{})

	// Act
	_, err := e.Clean(context.Background(), []string{"temp"})

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outside, "precious.doc")); err != nil {
		t.Fatal("cleanup followed a link out of its location")
	}
}

func TestEngine_CleanNothingSelected(t *testing.T) {
	// Arrange
	e := New([]platform.Location{{ID: "temp", Paths: []string{t.TempDir()}}}, &fakeJournal{})

	// Act
	_, err := e.Clean(context.Background(), nil)

	// Assert
	if err == nil {
		t.Fatal("want an error for an empty selection")
	}
}

func TestEngine_IgnoresRelativePaths(t *testing.T) {
	// Arrange
	e := New([]platform.Location{{ID: "bad", Paths: []string{"relative/dir"}}}, &fakeJournal{})

	// Act
	areas, err := e.Analyze(context.Background())

	// Assert
	if err != nil || areas[0].Files != 0 {
		t.Fatalf("areas = %+v, err = %v", areas, err)
	}
}
