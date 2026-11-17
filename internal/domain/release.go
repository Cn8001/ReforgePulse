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

// CheckResult is what one of Windows' health checks (SFC, DISM, CHKDSK)
// said, kept for Reports.
type CheckResult struct {
	ID       string    `json:"id,omitempty"` // set once saved
	Name     string    `json:"name"`
	At       time.Time `json:"at"`
	ExitCode int       `json:"exitCode"`
	Output   string    `json:"output"`
}
