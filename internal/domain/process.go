package domain

// Process is one running program as the Processes page lists it. Access is
// false when Windows refused to open it (protected and system processes);
// its numbers are then unknown rather than zero.
type Process struct {
	PID        uint32  `json:"pid"`
	ParentPID  uint32  `json:"parentPid"`
	Name       string  `json:"name"`
	Path       string  `json:"path,omitempty"`
	Threads    uint32  `json:"threads"`
	CPU        float64 `json:"cpu"` // percent of the whole machine since the previous sample
	WorkingSet uint64  `json:"workingSet"`
	Private    uint64  `json:"private"`
	Access     bool    `json:"access"`
}
