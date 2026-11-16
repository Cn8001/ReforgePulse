package domain

// StartupApp is a program that starts at sign-in, with what it costs now.
type StartupApp struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Source  string `json:"source"`
	Machine bool   `json:"machine"`
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
	Memory  uint64 `json:"memory"` // private bytes of its running processes; 0 when not running
}
