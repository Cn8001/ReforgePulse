package domain

import "time"

// InstalledProgram is an installed program and the evidence of its use.
type InstalledProgram struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Publisher   string    `json:"publisher"`
	Version     string    `json:"version"`
	Size        uint64    `json:"size"` // as the installer declared it; 0 when unknown
	Location    string    `json:"location"`
	LastRun     time.Time `json:"lastRun"`  // zero when Windows has no record
	DaysIdle    int       `json:"daysIdle"` // -1 when unknown
	Forgotten   bool      `json:"forgotten"`
	Basis       string    `json:"basis"` // why it is or is not flagged, in plain words
	Uninstaller bool      `json:"uninstaller"`
}
