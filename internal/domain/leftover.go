package domain

import "time"

// Leftover is a folder no installed program claims, which an uninstalled
// program may have left behind. Matching is by name, so it is never certain.
type Leftover struct {
	Path     string    `json:"path"`
	Root     string    `json:"root"` // Program Files, AppData (Roaming), ...
	Size     uint64    `json:"size"`
	Files    int       `json:"files"`
	Modified time.Time `json:"modified"` // newest file inside
}
