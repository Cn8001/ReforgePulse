package domain

import "time"

// Release is a newer ReforgePulse published on GitHub.
type Release struct {
	Version     string    `json:"version"`
	Name        string    `json:"name,omitempty"`
	Notes       string    `json:"notes,omitempty"` // the release text, in Markdown
	PublishedAt time.Time `json:"publishedAt,omitzero"`
	Size        int64     `json:"size,omitempty"`
	URL         string    `json:"url"`
}

// CheckResult is what one of Windows' health checks said. Only the verdict
// is kept: Windows writes the full log itself, and Log says where.
type CheckResult struct {
	ID          string    `json:"id,omitempty"` // set once saved
	Name        string    `json:"name"`
	At          time.Time `json:"at"`
	ExitCode    int       `json:"exitCode"`
	Summary     string    `json:"summary"`
	Log         string    `json:"log,omitempty"`         // Windows' own full log, when the check writes one
	Output      string    `json:"output,omitempty"`      // the full output of a run just finished; never saved
	FromWindows bool      `json:"fromWindows,omitempty"` // read from Windows' event log, so not deleted here
}
