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
	Seconds int            `json:"seconds"`
	CPU     []float64      `json:"cpu"` // one value per sample, in percent
	GPU     []float64      `json:"gpu"`
	Memory  []float64      `json:"memory"`
	Top     []ProcessTrend `json:"top"` // by peak memory
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
}

func New(live LiveReader, procs ProcessLister) *Recorder {
	return &Recorder{live: live, procs: procs, tick: time.After}
}

// Record samples until the duration passes or ctx ends; ending early is how
// the user stops it, so it still returns what was recorded.
func (r *Recorder) Record(ctx context.Context, d time.Duration, report func(float64, string)) Recording {
	var rec Recording
	type series struct {
		name string
		mem  []point
		cpu  float64
	}
	byPID := map[uint32]*series{}
	n := int(d / every)
	for i := 0; i < n; i++ {
		if l, err := r.live.Live(ctx); err == nil {
			rec.CPU = append(rec.CPU, l.CPU.Value)
			rec.GPU = append(rec.GPU, l.GPU.Value)
			rec.Memory = append(rec.Memory, l.Memory.Value)
		}
		if ps, err := r.procs.List(ctx); err == nil {
			for _, p := range ps {
				s := byPID[p.PID]
				if s == nil || s.name != p.Name {
					s = &series{name: p.Name}
					byPID[p.PID] = s
				}
				s.mem = append(s.mem, point{x: float64(i) * every.Minutes(), y: float64(p.Private) / (1 << 20)})
				s.cpu += p.CPU
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
	samples := len(rec.CPU)
	for pid, s := range byPID {
		if len(s.mem) < 2 {
			continue
		}
		t := ProcessTrend{Name: s.name, PID: pid, StartMB: s.mem[0].y, EndMB: s.mem[len(s.mem)-1].y, AvgCPU: s.cpu / float64(len(s.mem))}
		for _, p := range s.mem {
			t.PeakMB = math.Max(t.PeakMB, p.y)
		}
		t.MBPerMinute, t.Fit = fit(s.mem)
		minutes := s.mem[len(s.mem)-1].x - s.mem[0].x
		t.Leak = minutes >= 3 && len(s.mem)*5 >= samples*4 && t.EndMB-t.StartMB >= 100 && t.MBPerMinute >= 5 && t.Fit >= 0.8
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

type point struct{ x, y float64 }

// fit is a least-squares line: its slope and how well it explains the points.
func fit(ps []point) (slope, r2 float64) {
	n := float64(len(ps))
	var sx, sy, sxx, sxy float64
	for _, p := range ps {
		sx, sy, sxx, sxy = sx+p.x, sy+p.y, sxx+p.x*p.x, sxy+p.x*p.y
	}
	den := n*sxx - sx*sx
	if den == 0 {
		return 0, 0
	}
	slope = (n*sxy - sx*sy) / den
	mean := sy / n
	var ssTot, ssRes float64
	for _, p := range ps {
		pred := mean + slope*(p.x-sx/n)
		ssTot += (p.y - mean) * (p.y - mean)
		ssRes += (p.y - pred) * (p.y - pred)
	}
	if ssTot == 0 {
		return slope, 0
	}
	return slope, 1 - ssRes/ssTot
}
