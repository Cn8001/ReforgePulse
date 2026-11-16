package domain

import "time"

// EventGroup is one kind of event, counted: Event Viewer's list folded so
// a thousand repeats are one row.
type EventGroup struct {
	Channel  string    `json:"channel"`
	Provider string    `json:"provider"`
	ID       uint32    `json:"id"`
	Level    uint8     `json:"level"` // 1 critical, 2 error, 3 warning
	Count    int       `json:"count"`
	First    time.Time `json:"first"`
	Last     time.Time `json:"last"`
	Message  string    `json:"message"` // the newest one's text
	Explain  string    `json:"explain,omitempty"`
	Noise    bool      `json:"noise"` // known to be harmless
}
