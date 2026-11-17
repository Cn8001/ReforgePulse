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
	GPU        float64 `json:"gpu"`  // percent of the 3D engines
	Hung       bool    `json:"hung"` // a window has stopped answering, as "Not responding" in Task Manager
	Ports      []Port  `json:"ports"`
	Access     bool    `json:"access"`
	Ours       bool    `json:"ours,omitempty"` // one of ReforgePulse's own: the window, its probe, an agent's MCP server
}

// Port is a TCP or UDP endpoint a process holds open.
type Port struct {
	Proto  string `json:"proto"` // tcp, udp, tcp6, udp6
	Local  string `json:"local"`
	Port   uint16 `json:"port"`
	Remote string `json:"remote,omitempty"`
	State  string `json:"state,omitempty"` // listen, established, ...; empty for udp
}
