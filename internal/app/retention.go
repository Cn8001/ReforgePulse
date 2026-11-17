package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Pruner deletes what is older than a time and says how many went.
type Pruner func(ctx context.Context, before time.Time) (int, error)

type RetentionDeps struct {
	Settings SettingStore
	Prune    []Pruner // the change log, saved scans, recordings, dump analyses
	Now      func() time.Time
}

const retentionKey = "retention.days"

// maxRetention is ten years: past it the setting means nothing anyway.
const maxRetention = 3650

// RetentionService deletes history and reports older than the number of
// days the user chose. It is off unless turned on, because a deleted change
// can no longer be reverted.
type RetentionService struct {
	d RetentionDeps
}

func NewRetentionService(d RetentionDeps) *RetentionService { return &RetentionService{d: d} }

// Days is how many days are kept; 0 means everything is kept.
func (s *RetentionService) Days(ctx context.Context) (int, error) {
	v, err := s.d.Settings.Get(ctx, retentionKey, "0")
	if err != nil {
		return 0, fmt.Errorf("read retention: %w", err)
	}
	days, err := strconv.Atoi(v)
	if err != nil || days < 0 {
		return 0, nil // a setting this app did not write counts as off
	}
	return days, nil
}

// SetDays saves the choice and applies it at once, returning how many
// entries went.
func (s *RetentionService) SetDays(ctx context.Context, days int) (int, error) {
	if days < 0 || days > maxRetention {
		return 0, fmt.Errorf("%d days is outside 0 to %d", days, maxRetention)
	}
	if err := s.d.Settings.Set(ctx, retentionKey, strconv.Itoa(days)); err != nil {
		return 0, fmt.Errorf("save retention: %w", err)
	}
	return s.Apply(ctx)
}

// Apply deletes everything older than the chosen days. The window runs it
// at start; each place is tried even when another fails.
func (s *RetentionService) Apply(ctx context.Context) (int, error) {
	days, err := s.Days(ctx)
	if err != nil || days == 0 {
		return 0, err
	}
	before := s.d.Now().AddDate(0, 0, -days)
	total := 0
	var errs []error
	for _, p := range s.d.Prune {
		n, err := p(ctx, before)
		total += n
		errs = append(errs, err)
	}
	if err := errors.Join(errs...); err != nil {
		return total, fmt.Errorf("delete old entries: %w", err)
	}
	return total, nil
}
