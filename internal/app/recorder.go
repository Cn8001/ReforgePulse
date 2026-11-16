package app

import (
	"context"
	"fmt"
	"time"

	"github.com/cn8001/reforgepulse/internal/recorder"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type SessionRecorder interface {
	Record(ctx context.Context, d time.Duration, report func(float64, string)) recorder.Recording
}

// RecorderService records a session as a task; cancelling it from the
// status bar ends the recording early and keeps what it has.
type RecorderService struct {
	rec    SessionRecorder
	runner TaskRunner
}

func NewRecorderService(r SessionRecorder, runner TaskRunner) *RecorderService {
	return &RecorderService{rec: r, runner: runner}
}

func (s *RecorderService) Record(ctx context.Context, minutes int) (recorder.Recording, error) {
	if minutes < 1 || minutes > 240 {
		return recorder.Recording{}, fmt.Errorf("%d minutes is outside 1 to 240", minutes)
	}
	var rec recorder.Recording
	_ = s.runner.Do(ctx, "Recording", func(ctx context.Context, report tasks.Report) error {
		rec = s.rec.Record(ctx, time.Duration(minutes)*time.Minute, report)
		return nil // stopping early is how a recording ends, not a failure
	})
	return rec, nil
}
