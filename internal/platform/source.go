// Package platform is the boundary between the program and the operating
// system. Each OS gets its own subpackage; everything above depends only on
// these interfaces.
package platform

import (
	"context"
	"time"
)

// Volume is a fixed drive as the OS reports it, before any interpretation.
type Volume struct {
	Root  string
	Total uint64
	Free  uint64
}

// Source reads the raw facts behind the system overview. A method returns
// errors.ErrUnsupported when the OS cannot answer it at all.
type Source interface {
	OSName(ctx context.Context) (string, error)
	CPU(ctx context.Context) (model string, logical int, err error)
	Memory(ctx context.Context) (total, available uint64, err error)
	Uptime(ctx context.Context) (time.Duration, error)
	Volumes(ctx context.Context) ([]Volume, error)
}
