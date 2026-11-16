package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

type PowerPlans interface {
	PowerPlans(ctx context.Context) ([]platform.PowerPlan, error)
	SetCPUCaps(ctx context.Context, p platform.PowerPlan) error
}

// CPUCapService caps processor frequency on every power plan, the job
// cap-pcore.ps1 did, so switching plans cannot drop the cap. The old values
// of every plan go into the change log, which is what makes it reversible.
type CPUCapService struct {
	power   PowerPlans
	journal Recorder
}

func NewCPUCapService(p PowerPlans, j Recorder) *CPUCapService {
	return &CPUCapService{power: p, journal: j}
}

func (s *CPUCapService) Plans(ctx context.Context) ([]platform.PowerPlan, error) {
	plans, err := s.power.PowerPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("read power plans: %w", err)
	}
	return plans, nil
}

// Apply sets the P-core and E-core caps in MHz, plugged in and on battery;
// 0 removes a cap.
func (s *CPUCapService) Apply(ctx context.Context, pMHz, eMHz uint32) error {
	for _, v := range []uint32{pMHz, eMHz} {
		if v != 0 && (v < 800 || v > 7000) {
			return fmt.Errorf("%d MHz is outside 800 to 7000", v)
		}
	}
	plans, err := s.Plans(ctx)
	if err != nil {
		return err
	}
	target := make([]platform.PowerPlan, len(plans))
	for i, p := range plans {
		p.PCoreAC, p.PCoreDC, p.ECoreAC, p.ECoreDC = pMHz, pMHz, eMHz, eMHz
		target[i] = p
	}
	return s.write(ctx, domain.Change{
		Action: "cpu-cap", Reversible: true,
		Summary: fmt.Sprintf("Cap CPU frequency: P-cores %s, E-cores %s on %d power plans", mhz(pMHz), mhz(eMHz), len(plans)),
		Target:  "power plans: PROCFREQMAX1, PROCFREQMAX",
	}, plans, target)
}

// Revert writes back the values the change recorded before it applied.
func (s *CPUCapService) Revert(ctx context.Context, c domain.Change) error {
	var old []platform.PowerPlan
	if err := json.Unmarshal([]byte(c.Before), &old); err != nil {
		return fmt.Errorf("read the values before %s: %w", c.ID, err)
	}
	current, err := s.Plans(ctx)
	if err != nil {
		return err
	}
	return s.write(ctx, domain.Change{
		Action: "cpu-cap", Reversible: true, Reverts: c.ID, Summary: "Revert: " + c.Summary, Target: c.Target,
	}, current, old)
}

func (s *CPUCapService) write(ctx context.Context, c domain.Change, before, after []platform.PowerPlan) error {
	b, err := json.Marshal(before)
	if err != nil {
		return fmt.Errorf("record power plans: %w", err)
	}
	c.Before = string(b)
	id, err := s.journal.Begin(ctx, c)
	if err != nil {
		return fmt.Errorf("log cpu cap: %w", err)
	}
	var werr error
	for _, p := range after {
		if werr = s.power.SetCPUCaps(ctx, p); werr != nil {
			break
		}
	}
	a, _ := json.Marshal(after)
	if ferr := s.journal.Finish(ctx, id, string(a), werr); ferr != nil && werr == nil {
		werr = ferr
	}
	if werr != nil {
		return fmt.Errorf("set cpu cap: %w", werr)
	}
	return nil
}

func mhz(v uint32) string {
	if v == 0 {
		return "uncapped"
	}
	return fmt.Sprintf("%d MHz", v)
}
