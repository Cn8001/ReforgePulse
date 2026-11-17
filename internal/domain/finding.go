package domain

import "time"

// Severity says how urgent a finding is; Confidence says how sure we are.
// They are separate on purpose: a likely-but-unproven hardware fault is
// urgent, a confirmed but harmless fact is not.
type (
	Severity   string
	Confidence string
)

const (
	Info     Severity = "info"
	Low      Severity = "low"
	Medium   Severity = "medium"
	High     Severity = "high"
	Critical Severity = "critical"

	Confirmed Confidence = "confirmed"
	Likely    Confidence = "likely"
	Possible  Confidence = "possible"
)

// Finding is one problem in plain words, with the evidence behind it.
type Finding struct {
	Rule         string        `json:"rule"`
	Key          string        `json:"key"` // the rule and its subject, the same from scan to scan
	Title        string        `json:"title"`
	Summary      string        `json:"summary"`
	Severity     Severity      `json:"severity"`
	Confidence   Confidence    `json:"confidence"`
	Evidence     []string      `json:"evidence"`
	Alternatives string        `json:"alternatives,omitempty"` // other explanations the evidence allows
	Action       *Action       `json:"action,omitempty"`
	Latest       time.Time     `json:"latest,omitzero"` // newest event behind it; zero when it describes a state
	Status       FindingStatus `json:"status"`
	ResolvedAt   time.Time     `json:"resolvedAt,omitzero"`
	New          bool          `json:"new,omitempty"` // not in the scan before this one
}

// FindingStatus is where the user left a finding. Returned means it was marked
// resolved but evidence newer than that has turned up since.
type FindingStatus string

const (
	Open     FindingStatus = "open"
	Resolved FindingStatus = "resolved"
	Returned FindingStatus = "returned"
)

// Resolution is the user saying a finding is dealt with.
type Resolution struct {
	Key   string    `json:"key"`
	Title string    `json:"title"`
	At    time.Time `json:"at"`
}

// Action is the next step a finding offers. Kind tells the window what to
// open; nothing is changed without the user confirming there.
type Action struct {
	Label string `json:"label"`
	Kind  string `json:"kind"` // open-processes, open-cleanup, open-startup, cpu-cap
	Arg   string `json:"arg,omitempty"`
}

// ScanResult is one Scan PC run.
type ScanResult struct {
	ID       string    `json:"id"`
	At       time.Time `json:"at"`
	Findings []Finding `json:"findings"`
	Suspects []Suspect `json:"suspects"` // only when there is instability to explain
	Elevated bool      `json:"elevated"`
}

// ReportSummary is one saved scan in the list of recent scans.
type ReportSummary struct {
	ID       string    `json:"id"`
	At       time.Time `json:"at"`
	Open     int       `json:"open"` // findings still open or returned
	Resolved int       `json:"resolved"`
	Worst    Severity  `json:"worst,omitempty"` // of the open findings
}

// CrashEvidence is what Windows recorded about crashes and hardware errors
// in the window the scan looked at.
type CrashEvidence struct {
	Since          time.Time    `json:"since"`
	Bugchecks      []CrashEvent `json:"bugchecks"`      // blue screens, from Windows Error Reporting
	Restarts       []CrashEvent `json:"restarts"`       // Kernel-Power 41: restarts without a clean shutdown
	WHEA           []CrashEvent `json:"whea"`           // hardware errors the CPU, memory or PCIe reported
	GPUResets      []CrashEvent `json:"gpuResets"`      // the display driver stopped responding and recovered
	AppCrashes     []CrashEvent `json:"appCrashes"`     // programs that crashed; Detail is "app.exe in module.dll"
	AppHangs       []CrashEvent `json:"appHangs"`       // programs Windows closed after they stopped responding
	DiskErrors     []CrashEvent `json:"diskErrors"`     // bad blocks, paging and file system errors
	Services       []CrashEvent `json:"services"`       // services that crashed or failed to start
	Throttling     []CrashEvent `json:"throttling"`     // firmware limited processor speed: heat or power limits
	DriverInstalls []CrashEvent `json:"driverInstalls"` // drivers installed, to line up with when crashes began
	Updates        []CrashEvent `json:"updates"`        // Windows updates installed
	MemoryTests    []CrashEvent `json:"memoryTests"`    // Windows Memory Diagnostic results; Detail is "no errors" or "errors found"
	Minidumps      []string     `json:"minidumps"`      // newest first
	Microcode      uint32       `json:"microcode"`      // 0 when unknown
	Tuning         []string     `json:"tuning"`         // overclocking and RGB tool drivers that touch hardware
	EventsError    string       `json:"eventsError,omitempty"`
}

// Suspect is one place a crash can come from, weighed against the evidence.
// Verdict is strong, some, none or unmeasured; unmeasured means Windows
// cannot see it, not that it is fine.
type Suspect struct {
	Area    string   `json:"area"` // cpu, memory, storage, heat, power, drivers, software
	Title   string   `json:"title"`
	Verdict string   `json:"verdict"`
	For     []string `json:"for"`
	Against []string `json:"against"`
	Checks  []string `json:"checks"` // what to try next to confirm or rule it out
}

// CrashEvent is one event reduced to what the rules need.
type CrashEvent struct {
	Time   time.Time `json:"time"`
	ID     uint32    `json:"id"`
	Detail string    `json:"detail"` // bugcheck code, APIC ID, driver name
}
