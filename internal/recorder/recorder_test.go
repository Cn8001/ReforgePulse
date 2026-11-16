package recorder

import (
	"context"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

type fakeLive struct{}

func (fakeLive) Live(context.Context) (domain.Live, error) {
	return domain.Live{CPU: domain.Measurement{Value: 10}}, nil
}

// procs grows one process by 30 MB per sample (900 MB a minute) and keeps
// another flat with one spike.
type procs struct{ i int }

func (p *procs) List(context.Context) ([]domain.Process, error) {
	p.i++
	spike := uint64(0)
	if p.i == 10 {
		spike = 500 << 20
	}
	return []domain.Process{
		{PID: 1, Name: "leaky.exe", Private: uint64(200+30*p.i) << 20},
		{PID: 2, Name: "steady.exe", Private: 400<<20 + spike},
	}, nil
}

func instant(time.Duration) <-chan time.Time {
	c := make(chan time.Time, 1)
	c <- time.Time{}
	return c
}

func TestRecorder_FindsTheSteadyGrowerNotTheSpike(t *testing.T) {
	// Arrange
	r := New(fakeLive{}, &procs{})
	r.tick = instant

	// Act
	rec := r.Record(context.Background(), 5*time.Minute, func(float64, string) {})

	// Assert
	if rec.Seconds != 300 || len(rec.CPU) != 150 {
		t.Fatalf("seconds = %d, samples = %d", rec.Seconds, len(rec.CPU))
	}
	if !rec.Top[0].Leak || rec.Top[0].Name != "leaky.exe" || rec.Top[0].Fit < 0.99 {
		t.Fatalf("top = %+v", rec.Top[0])
	}
	for _, tr := range rec.Top {
		if tr.Name == "steady.exe" && tr.Leak {
			t.Fatal("a one-off spike was called a leak")
		}
	}
}

func TestRecorder_StopsEarlyAndKeepsData(t *testing.T) {
	// Arrange
	r := New(fakeLive{}, &procs{})
	ctx, cancel := context.WithCancel(context.Background())
	r.tick = func(time.Duration) <-chan time.Time { cancel(); return make(chan time.Time) }

	// Act
	rec := r.Record(ctx, time.Hour, func(float64, string) {})

	// Assert
	if len(rec.CPU) != 1 || rec.Seconds != 2 {
		t.Fatalf("rec = %+v", rec)
	}
}
