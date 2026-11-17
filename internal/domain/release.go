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
