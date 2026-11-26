// Package recorder records load and per-process memory over a session, so
// a slowdown or a leak can be seen after the fact instead of watched live.
package recorder

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

const every = 2 * time.Second

type LiveReader interface {
	Live(ctx context.Context) (domain.Live, error)
}

type ProcessLister interface {
	List(ctx context.Context) ([]domain.Process, error)
}

// Recording is one session: the load over time and what each process did.
type Recording struct {
	ID            string         `json:"id,omitempty"` // set once saved
	Started       time.Time      `json:"started"`
	Seconds       int            `json:"seconds"`
	SampleSeconds int            `json:"sampleSeconds"` // how much time each point below covers
	CPU           []float64      `json:"cpu"`           // in percent, at most maxPoints values
	GPU           []float64      `json:"gpu"`
	Memory        []float64      `json:"memory"`
	Top           []ProcessTrend `json:"top"` // by peak memory
}

// ProcessTrend is one process across the session. A leak is steady growth
// that a straight line explains well, not one jump.
type ProcessTrend struct {
	Name        string  `json:"name"`
	PID         uint32  `json:"pid"`
	StartMB     float64 `json:"startMb"`
	EndMB       float64 `json:"endMb"`
	PeakMB      float64 `json:"peakMb"`
	AvgCPU      float64 `json:"avgCpu"`
	MBPerMinute float64 `json:"mbPerMinute"`
	Fit         float64 `json:"fit"` // R², 1 is a perfect straight line
	Leak        bool    `json:"leak"`
}

type Recorder struct {
	live  LiveReader
	procs ProcessLister
	tick  func(time.Duration) <-chan time.Time
	now   func() time.Time
}

func New(live LiveReader, procs ProcessLister) *Recorder {
	return &Recorder{live: live, procs: procs, tick: time.After, now: time.Now}
}

// maxPoints caps each load series, so a day long session costs what a
// twenty minute one does: when full, neighbours are averaged and each point
// covers twice the time.
const maxPoints = 600

// series is one process in running sums: enough for its start, end, peak and
// a least-squares line without keeping a point per sample, so memory stays
// flat however long the recording runs.
type series struct {
	name                    string
	n                       float64
	sx, sy, sxx, sxy, syy   float64
	first, last, peak, xMin float64
	xMax, cpu               float64
}

func (s *series) add(x, y, cpu float64) {
	if s.n == 0 {
		s.first, s.xMin = y, x
	}
	s.n++
	s.sx, s.sy, s.sxx, s.sxy, s.syy = s.sx+x, s.sy+y, s.sxx+x*x, s.sxy+x*y, s.syy+y*y
	s.last, s.xMax, s.peak = y, x, math.Max(s.peak, y)
	s.cpu += cpu
}

// fit is the least-squares slope and how well the line explains the points.
func (s *series) fit() (slope, r2 float64) {
	den := s.n*s.sxx - s.sx*s.sx
	if den == 0 {
		return 0, 0
	}
	slope = (s.n*s.sxy - s.sx*s.sy) / den
	ssTot := s.syy - s.sy*s.sy/s.n
	if ssTot <= 0 {
		return slope, 0
	}
	ssRes := ssTot - slope*(s.sxy-s.sx*s.sy/s.n)
	return slope, 1 - ssRes/ssTot
}

// squeeze halves a full series by averaging neighbours.
func squeeze(v []float64) []float64 {
	out := v[:0]
	for i := 0; i+1 < len(v); i += 2 {
		out = append(out, (v[i]+v[i+1])/2)
	}
	return out
}

// Record samples until the duration passes or ctx ends; ending early is how
// the user stops it, so it still returns what was recorded.
func (r *Recorder) Record(ctx context.Context, d time.Duration, report func(float64, string)) Recording {
	rec := Recording{Started: r.now()}
	byPID := map[uint32]*series{}
	n := int(d / every)
	samples, per, acc := 0, 1, [3]float64{}
	for i := 0; i < n; i++ {
		if l, err := r.live.Live(ctx); err == nil {
			acc[0], acc[1], acc[2] = acc[0]+l.CPU.Value, acc[1]+l.GPU.Value, acc[2]+l.Memory.Value
			samples++
			if samples%per == 0 {
				f := float64(per)
				rec.CPU, rec.GPU, rec.Memory = append(rec.CPU, acc[0]/f), append(rec.GPU, acc[1]/f), append(rec.Memory, acc[2]/f)
				acc = [3]float64{}
				if len(rec.CPU) >= maxPoints {
					rec.CPU, rec.GPU, rec.Memory = squeeze(rec.CPU), squeeze(rec.GPU), squeeze(rec.Memory)
					per *= 2
				}
			}
		}
		if ps, err := r.procs.List(ctx); err == nil {
			for _, p := range ps {
				s := byPID[p.PID]
				if s == nil || s.name != p.Name {
					s = &series{name: p.Name}
					byPID[p.PID] = s
				}
				s.add(float64(i)*every.Minutes(), float64(p.Private)/(1<<20), p.CPU)
			}
		}
		rec.Seconds = int(float64(i+1) * every.Seconds())
		report(float64(i+1)/float64(n), "")
		select {
		case <-ctx.Done():
			i = n
		case <-r.tick(every):
		}
	}
	rec.SampleSeconds = int(every.Seconds()) * per
	for pid, s := range byPID {
		if s.n < 2 {
			continue
		}
		t := ProcessTrend{Name: s.name, PID: pid, StartMB: s.first, EndMB: s.last, PeakMB: s.peak, AvgCPU: s.cpu / s.n}
		t.MBPerMinute, t.Fit = s.fit()
		minutes := s.xMax - s.xMin
		t.Leak = minutes >= 3 && s.n*5 >= float64(samples)*4 && t.EndMB-t.StartMB >= 100 && t.MBPerMinute >= 5 && t.Fit >= 0.8
		rec.Top = append(rec.Top, t)
	}
	sort.Slice(rec.Top, func(i, j int) bool {
		if rec.Top[i].Leak != rec.Top[j].Leak {
			return rec.Top[i].Leak
		}
		return rec.Top[i].PeakMB > rec.Top[j].PeakMB
	})
	if len(rec.Top) > 25 {
		rec.Top = rec.Top[:25]
	}
	return rec
}
