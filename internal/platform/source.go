// Package platform is the boundary between the program and the operating
// system. Each OS gets its own subpackage; everything above depends only on
// these interfaces.
package platform

import (
	"context"
	"errors"
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
	GPU(ctx context.Context) (model string, err error)
	Memory(ctx context.Context) (total, available uint64, err error)
	Uptime(ctx context.Context) (time.Duration, error)
	Volumes(ctx context.Context) ([]Volume, error)
}

// ProcessInfo is one process as the OS reports it, with its total CPU time
// so far; turning that into a percentage needs two samples.
type ProcessInfo struct {
	PID, ParentPID, Threads uint32
	Name, Path              string
	CPUTime                 time.Duration
	WorkingSet, Private     uint64
	Opened                  bool // false when the OS refused access
}

type Processes interface {
	Processes(ctx context.Context) ([]ProcessInfo, error)
}

// Sampler reads load since its previous call, in percent. The first call
// only sets the baseline and may answer an average since boot.
type Sampler interface {
	CPULoad(ctx context.Context) (float64, error)
	GPULoad(ctx context.Context) (float64, error)
}

// ErrDenied means the user refused the elevation prompt.
var ErrDenied = errors.New("elevation denied")

// Privileges answers whether the process runs elevated and can restart it so.
type Privileges interface {
	Elevated() bool
	Relaunch(args []string) error // ErrDenied when the user cancels the prompt
}
