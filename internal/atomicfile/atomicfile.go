// Package atomicfile replaces a file so that a crash halfway leaves either
// the old contents or the new ones, never a mix.
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Write puts b at path through a synced temporary file in the same folder,
// then renames it over path.
func Write(path string, b []byte) error {
	name := filepath.Base(path)
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", name, err)
	}
	defer os.Remove(tmp.Name()) // a no-op once the rename succeeded
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", name, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", name, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replace %s: %w", name, err)
	}
	return nil
}
