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

// Event is one Windows event log record with its structured EventData.
type Event struct {
	Provider string
	ID       uint32
	Time     time.Time
	Data     map[string]string
}

// Service is a driver or service registration, running or not.
type Service struct {
	Name, DisplayName, ImagePath string
	Start                        uint32 // 0 boot, 1 system, 2 automatic, 3 manual, 4 disabled
	Driver                       bool
}

// Program is an installed program as its uninstall entry describes it.
type Program struct {
	Name, Version, Publisher, InstallDate, InstallLocation, UninstallString string
}

// Machine reads the evidence crash analysis draws on.
type Machine interface {
	Events(ctx context.Context, channel, provider string, ids []uint32, since time.Time, max int) ([]Event, error)
	Microcode(ctx context.Context) (uint32, error)
	Services(ctx context.Context) ([]Service, error)
	Programs(ctx context.Context) ([]Program, error)
	DumpDir() string // where Windows writes minidumps
}

// Location is a folder whose contents Windows or apps recreate when needed.
// Only files older than MinAge are touched, so work in progress is spared.
type Location struct {
	ID, Name, Note string
	Paths          []string
	MinAge         time.Duration
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
