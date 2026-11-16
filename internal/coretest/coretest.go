// Package coretest stresses one logical processor at a time, so each core
// runs at its highest boost, where a degraded core makes mistakes first.
// Every core computes the same deterministic rounds; a round whose result
// differs from what most cores got is a calculation error.
package coretest

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

// Deps are what the test needs from the machine.
type Deps struct {
	CPUs platform.CPUs
	// WHEA counts hardware errors per APIC ID since a time.
	WHEA func(ctx context.Context, since time.Time) (map[uint32]int, error)
}

type Engine struct {
	d    Deps
	work func(seed uint64) [32]byte
	now  func() time.Time
}

func New(d Deps) *Engine { return &Engine{d: d, work: work, now: time.Now} }

// Run tests each processor for perCore, one after another.
func (e *Engine) Run(ctx context.Context, cpus []int, perCore time.Duration, report func(float64, string)) (domain.CoreTest, error) {
	if len(cpus) == 0 {
		return domain.CoreTest{}, fmt.Errorf("no processors selected")
	}
	start := e.now()
	results := make([]domain.CoreTestResult, len(cpus))
	digests := make([]map[uint64][32]byte, len(cpus))
	for i, cpu := range cpus {
		if err := ctx.Err(); err != nil {
			return domain.CoreTest{}, err
		}
		report(float64(i)/float64(len(cpus)), fmt.Sprintf("processor %d", cpu))
		results[i].Index, digests[i] = cpu, map[uint64][32]byte{}
		_, _ = e.d.CPUs.CoreLoads(ctx) // starts this processor's clock window
		deadline := e.now().Add(perCore)
		err := e.d.CPUs.RunPinned(cpu, func() {
			for seed := uint64(1); e.now().Before(deadline) && ctx.Err() == nil; seed++ {
				digests[i][seed] = e.work(seed)
				results[i].Rounds++
			}
		})
		if err != nil {
			results[i].Skipped = err.Error()
			continue
		}
		if loads, err := e.d.CPUs.CoreLoads(ctx); err == nil {
			for _, l := range loads {
				if l.Index == cpu {
					results[i].MHz = l.MHz
				}
			}
		}
	}
	countErrors(results, digests)
	t := domain.CoreTest{Results: results, Seconds: int(perCore.Seconds())}
	if whea, err := e.d.WHEA(ctx, start); err == nil {
		for _, n := range whea {
			t.NewWHEA += n
		}
	}
	return t, ctx.Err()
}

// countErrors takes, for each round, the result most processors got as the
// right one; a lone faulty core cannot become the reference.
func countErrors(results []domain.CoreTestResult, digests []map[uint64][32]byte) {
	votes := map[uint64]map[[32]byte]int{}
	for _, d := range digests {
		for seed, h := range d {
			if votes[seed] == nil {
				votes[seed] = map[[32]byte]int{}
			}
			votes[seed][h]++
		}
	}
	for i, d := range digests {
		for seed, h := range d {
			best, bestN := h, 0
			for cand, n := range votes[seed] {
				if n > bestN {
					best, bestN = cand, n
				}
			}
			if bestN > 1 && h != best {
				results[i].Errors++
			}
		}
	}
}

// work is one round: a SHA-256 chain over a seeded buffer, which keeps the
// vector units busy, then a floating-point loop. IEEE math gives every
// healthy core the same bits.
func work(seed uint64) [32]byte {
	buf := make([]byte, 256<<10)
	x := seed*0x9E3779B97F4A7C15 + 1
	for i := 0; i+8 <= len(buf); i += 8 {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		binary.LittleEndian.PutUint64(buf[i:], x)
	}
	h := sha256.Sum256(buf)
	for i := 0; i < 32; i++ {
		copy(buf, h[:])
		h = sha256.Sum256(buf)
	}
	f := float64(seed)
	for i := 0; i < 200_000; i++ {
		f = math.Sqrt(f*1.000001+float64(i)) + math.Sin(f)
	}
	binary.LittleEndian.PutUint64(h[:8], binary.LittleEndian.Uint64(h[:8])^math.Float64bits(f))
	return h
}
