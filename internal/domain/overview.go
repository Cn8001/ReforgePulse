package domain

// Overview is the Home screen's picture of the PC.
type Overview struct {
	OS          string      `json:"os"`
	CPUModel    string      `json:"cpuModel"`
	GPUModel    string      `json:"gpuModel"`
	LogicalCPUs int         `json:"logicalCpus"`
	Elevated    bool        `json:"elevated"` // false when the user declined UAC
	Uptime      Measurement `json:"uptime"`
	MemoryTotal Measurement `json:"memoryTotal"`
	MemoryUsed  Measurement `json:"memoryUsed"`
	Volumes     []Volume    `json:"volumes"`
}

type Volume struct {
	Root  string      `json:"root"`
	Total Measurement `json:"total"`
	Free  Measurement `json:"free"`
}

// Live is the moment-to-moment load the overview tiles plot.
type Live struct {
	CPU    Measurement `json:"cpu"`
	GPU    Measurement `json:"gpu"`
	Memory Measurement `json:"memory"`
}
