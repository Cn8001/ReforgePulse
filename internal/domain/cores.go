package domain

// CoreInfo is one logical processor with what the event log says about it.
type CoreInfo struct {
	Index   int    `json:"index"` // logical processor, as Task Manager numbers it
	Core    int    `json:"core"`
	Kind    string `json:"kind"` // "P" or "E" on hybrid CPUs, "" otherwise
	APIC    uint32 `json:"apic"`
	Sibling int    `json:"sibling"` // the other thread of the same core, -1 when none
	WHEA    int    `json:"whea"`    // hardware errors in 30 days that named this processor
}

// CoreTestResult is how one logical processor did in the core test.
type CoreTestResult struct {
	Index   int     `json:"index"`
	Rounds  int     `json:"rounds"`
	Errors  int     `json:"errors"` // rounds whose result differed from the other cores'
	MHz     float64 `json:"mhz"`    // average effective clock while it ran
	Skipped string  `json:"skipped,omitempty"`
}

// CoreTest is a whole core test run.
type CoreTest struct {
	Results []CoreTestResult `json:"results"`
	NewWHEA int              `json:"newWhea"` // hardware errors Windows logged during the test
	Seconds int              `json:"seconds"` // per processor
}
