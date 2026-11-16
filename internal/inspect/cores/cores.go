// Package cores maps the processor topology to the hardware errors the
// event log records, so "WHEA APIC 32" becomes "P-core 4, threads 8 and 9".
package cores

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

type Inspector struct {
	cpus platform.CPUs
	m    platform.Machine
	now  func() time.Time
}

func New(cpus platform.CPUs, m platform.Machine) *Inspector {
	return &Inspector{cpus: cpus, m: m, now: time.Now}
}

func (in *Inspector) List(ctx context.Context) ([]domain.CoreInfo, error) {
	lps, err := in.cpus.Topology(ctx)
	if err != nil {
		return nil, fmt.Errorf("read topology: %w", err)
	}
	whea, _ := in.WHEAByAPIC(ctx, in.now().Add(-30*24*time.Hour)) // without the log the counts stay zero
	return Map(lps, whea), nil
}

// WHEAByAPIC counts hardware errors per x2APIC ID since a time.
func (in *Inspector) WHEAByAPIC(ctx context.Context, since time.Time) (map[uint32]int, error) {
	events, err := in.m.Events(ctx, "System", "Microsoft-Windows-WHEA-Logger", nil, since, 500)
	out := map[uint32]int{}
	for _, e := range events {
		if id, perr := strconv.ParseUint(strings.TrimSpace(e.Data["ApicId"]), 10, 32); perr == nil {
			out[uint32(id)]++
		}
	}
	return out, err
}

// Map turns topology and error counts into rows. Kind is only set on hybrid
// CPUs, where efficiency classes differ.
func Map(lps []platform.LogicalCPU, whea map[uint32]int) []domain.CoreInfo {
	hybrid, maxEff := false, uint8(0)
	for _, l := range lps {
		if l.Efficiency != lps[0].Efficiency {
			hybrid = true
		}
		maxEff = max(maxEff, l.Efficiency)
	}
	byCore := map[int][]int{}
	for _, l := range lps {
		byCore[l.Core] = append(byCore[l.Core], l.Index)
	}
	out := make([]domain.CoreInfo, len(lps))
	for i, l := range lps {
		c := domain.CoreInfo{Index: l.Index, Core: l.Core, APIC: l.APIC, Sibling: -1, WHEA: whea[l.APIC]}
		if hybrid {
			c.Kind = "E"
			if l.Efficiency == maxEff {
				c.Kind = "P"
			}
		}
		for _, other := range byCore[l.Core] {
			if other != l.Index {
				c.Sibling = other
			}
		}
		out[i] = c
	}
	return out
}
