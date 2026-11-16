package domain

import "time"

type ChangeStatus string

const (
	Started  ChangeStatus = "started" // written before the system is touched
	Applied  ChangeStatus = "applied"
	Failed   ChangeStatus = "failed"
	Reverted ChangeStatus = "reverted" // a later change undid it
)

// Change is one entry in the change log, kept small like a git commit: a
// short ID, a one-line summary and the values before and after, never file
// contents.
type Change struct {
	ID         string       `json:"id"`
	At         time.Time    `json:"at"`
	Action     string       `json:"action,omitempty"` // what kind of change, for undoing it: "startup", "cleanup", ...
	Summary    string       `json:"summary,omitempty"`
	Target     string       `json:"target,omitempty"` // what was changed: a path, a setting
	Before     string       `json:"before,omitempty"`
	After      string       `json:"after,omitempty"`
	Reversible bool         `json:"reversible,omitempty"`
	Reverts    string       `json:"reverts,omitempty"` // ID of the change this one undoes
	Status     ChangeStatus `json:"status"`
	Error      string       `json:"error,omitempty"`
}
