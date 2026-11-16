package oldfiles

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInspector_ListsOnlyOldAndBig(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	put := func(name string, size int, age time.Duration) {
		p := filepath.Join(dir, name)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, make([]byte, size), 0o644)
		at := time.Now().Add(-age)
		_ = os.Chtimes(p, at, at)
	}
	year := 365 * 24 * time.Hour
	put("setup.exe", 2<<20, year)
	put("small.txt", 10, year)
	put("new.zip", 3<<20, time.Hour)
	put("project/a.bin", 2<<20, year)
	put("project/b.bin", 10, time.Hour) // one fresh file keeps the folder
	in := New([]Folder{{Name: "Downloads", Path: dir}})

	// Act
	got, err := in.List(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(got[0].Path) != "setup.exe" || got[0].Root != "Downloads" {
		t.Fatalf("got %+v", got)
	}
}
