// Package domain holds the plain models every layer shares. It depends on
// nothing else in the program.
package domain

import "time"

// Availability tells a real zero apart from a value nobody could read.
type Availability string

const (
	Measured    Availability = "measured"
	NotMeasured Availability = "not_measured"
	Unsupported Availability = "unsupported"
	Unknown     Availability = "unknown"
)

type Unit string

const (
	Bytes   Unit = "bytes"
	Seconds Unit = "seconds"
)

// Measurement is one observed value with where and when it came from, so a
// finding can always point back to its evidence.
type Measurement struct {
	Value  float64      `json:"value"`
	Unit   Unit         `json:"unit"`
	Status Availability `json:"status"`
	Source string       `json:"source"` // the API that produced it
	At     time.Time    `json:"at"`
}

func Measure(v float64, u Unit, source string, at time.Time) Measurement {
	return Measurement{Value: v, Unit: u, Status: Measured, Source: source, At: at}
}

// Missing records that a value could not be read. Value stays zero, so
// readers must check Status before showing it.
func Missing(s Availability, source string) Measurement {
	if s == Measured {
		s = Unknown
	}
	return Measurement{Status: s, Source: source}
}
