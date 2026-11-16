package domain

import "time"

// Dump is one crash dump with what its header says.
type Dump struct {
	Name     string    `json:"name"`
	Time     time.Time `json:"time"`
	Size     int64     `json:"size"`
	Code     string    `json:"code"` // stop code, "" when the header is unreadable
	CodeName string    `json:"codeName,omitempty"`
	Params   []string  `json:"params"`
}

// DumpAnalysis is the debugger's verdict on one dump, the fields that matter
// pulled out of !analyze -v.
type DumpAnalysis struct {
	Name    string `json:"name"`
	Module  string `json:"module"` // the driver or module the debugger blames
	Image   string `json:"image"`
	Symbol  string `json:"symbol"`  // the function it stopped in
	Process string `json:"process"` // the process running at the time
	Bucket  string `json:"bucket"`  // Microsoft's name for this kind of crash
	Output  string `json:"output"`  // the tail of the raw output, for experts
}
