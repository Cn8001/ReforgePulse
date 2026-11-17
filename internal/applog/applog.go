// Package applog is the app's one log file. The window runs elevated and
// without a console, so a crash would otherwise leave nothing to read.
package applog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Open appends to dir/reforgepulse.log. Past maxBytes the file becomes
// reforgepulse.old.log, replacing the one before, so at most two stay.
func Open(dir string, maxBytes int64) (*os.File, error) {
	path := filepath.Join(dir, "reforgepulse.log")
	if st, err := os.Stat(path); err == nil && st.Size() > maxBytes {
		if err := os.Rename(path, filepath.Join(dir, "reforgepulse.old.log")); err != nil {
			return nil, fmt.Errorf("rotate log: %w", err)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("check log: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log: %w", err)
	}
	return f, nil
}
