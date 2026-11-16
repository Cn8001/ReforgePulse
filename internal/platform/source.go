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
	Hung                    bool // a window of it stopped answering
}

// PortInfo is an open TCP or UDP endpoint and the process that owns it.
type PortInfo struct {
	PID                  uint32
	Proto, Local, Remote string
	Port                 uint16
	State                string
}

type Processes interface {
	Processes(ctx context.Context) ([]ProcessInfo, error)
	Ports(ctx context.Context) ([]PortInfo, error)
	Kill(ctx context.Context, pid uint32) error
}

// GPUByProcess reads each process's share of the 3D engines, in percent.
type GPUByProcess interface {
	GPUProcesses(ctx context.Context) (map[uint32]float64, error)
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
	ID                                                                      string // the uninstall key, stable across calls
	Name, Version, Publisher, InstallDate, InstallLocation, UninstallString string
	SizeKB                                                                  uint64 // EstimatedSize, as the installer declared it
}

// Machine reads the evidence crash analysis draws on.
type Machine interface {
	Events(ctx context.Context, channel, provider string, ids []uint32, since time.Time, max int) ([]Event, error)
	Microcode(ctx context.Context) (uint32, error)
	Services(ctx context.Context) ([]Service, error)
	Programs(ctx context.Context) ([]Program, error)
	// LastRuns maps an upper-case exe name to when Windows last saw it start,
	// from the Prefetch folder. Missing means no record, not never.
	LastRuns(ctx context.Context) (map[string]time.Time, error)
	Uninstall(ctx context.Context, uninstallString string) error
	DumpDir() string // where Windows writes minidumps
}

// Location is a folder whose contents Windows or apps recreate when needed.
// Only files older than MinAge are touched, so work in progress is spared.
type Location struct {
	ID, Name, Note string
	Paths          []string
	MinAge         time.Duration
}

// StartupEntry is a program Windows starts at sign-in.
type StartupEntry struct {
	ID      string // stable across calls: source and name
	Name    string
	Command string
	Source  string // where it is registered, for the user
	Machine bool   // for every user rather than only this one
	Enabled bool
}

// Startup lists and toggles startup programs the way Task Manager does,
// which leaves the registration itself in place so it can be re-enabled.
type Startup interface {
	StartupEntries(ctx context.Context) ([]StartupEntry, error)
	SetStartupEnabled(ctx context.Context, id string, enabled bool) error
}

// LeftoverRoot is a folder whose subfolders each usually belong to one
// program, such as Program Files or AppData\Roaming.
type LeftoverRoot struct {
	Name, Path string
}

// Files reaches the parts of the file system cleanup needs beyond deleting.
type Files interface {
	LeftoverRoots() []LeftoverRoot
	// Recycle moves paths to the Recycle Bin, so the user can restore them.
	Recycle(ctx context.Context, paths []string) error
}

// PowerPlan is a Windows power plan with its processor frequency caps in
// MHz, 0 meaning no cap. P is efficiency class 1 (performance cores), E is
// class 0 (efficiency cores, or every core on a CPU without classes).
type PowerPlan struct {
	GUID    string `json:"guid"`
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	PCoreAC uint32 `json:"pCoreAc"`
	PCoreDC uint32 `json:"pCoreDc"`
	ECoreAC uint32 `json:"eCoreAc"`
	ECoreDC uint32 `json:"eCoreDc"`
}

// Power reads and writes processor frequency caps on power plans.
type Power interface {
	PowerPlans(ctx context.Context) ([]PowerPlan, error)
	// SetCPUCaps writes all four caps of one plan and reapplies the active plan.
	SetCPUCaps(ctx context.Context, p PowerPlan) error
}

// Feature is a Windows optional feature, as Windows Features lists them.
type Feature struct {
	Name    string
	Enabled bool
}

// Features reads and toggles Windows optional features.
type Features interface {
	OptionalFeatures(ctx context.Context) ([]Feature, error)
	// SetFeature turns a feature on or off; restart says Windows needs one.
	SetFeature(ctx context.Context, name string, enabled bool) (restart bool, err error)
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
