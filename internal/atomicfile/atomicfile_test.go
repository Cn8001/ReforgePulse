package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWrite_ReplacesAndLeavesNoTempFiles(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "a.json")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Act
	err := Write(path, []byte("new"))

	// Assert
	got, _ := os.ReadFile(path)
	entries, _ := os.ReadDir(dir)
	if err != nil || string(got) != "new" || len(entries) != 1 {
		t.Fatalf("err = %v, contents = %q, files = %d", err, got, len(entries))
	}
}
