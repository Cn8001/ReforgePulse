package applog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_RotatesPastTheLimit(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "reforgepulse.log"), make([]byte, 100), 0o600); err != nil {
		t.Fatal(err)
	}

	// Act
	f, err := Open(dir, 50)
	if err == nil {
		f.Close()
	}

	// Assert
	old, oldErr := os.Stat(filepath.Join(dir, "reforgepulse.old.log"))
	cur, curErr := os.Stat(filepath.Join(dir, "reforgepulse.log"))
	if err != nil || oldErr != nil || old.Size() != 100 || curErr != nil || cur.Size() != 0 {
		t.Fatalf("err %v, old %v %v, current %v %v", err, old, oldErr, cur, curErr)
	}
}
