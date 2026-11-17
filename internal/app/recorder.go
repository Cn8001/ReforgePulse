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

// RecordingStore keeps finished sessions.
type RecordingStore interface {
	Save(ctx context.Context, r recorder.Recording) (recorder.Recording, error)
	All(ctx context.Context) ([]recorder.Recording, error)
	Get(ctx context.Context, id string) (recorder.Recording, error)
	Delete(ctx context.Context, id string) error
}

// RecorderService records a session as a task; cancelling it from the
// status bar ends the recording early and keeps what it has. Every session
// is saved for Reports.
type RecorderService struct {
	rec    SessionRecorder
	runner TaskRunner
	saved  RecordingStore
}

func NewRecorderService(r SessionRecorder, runner TaskRunner, saved RecordingStore) *RecorderService {
	return &RecorderService{rec: r, runner: runner, saved: saved}
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
	if rec.Seconds == 0 {
		return rec, nil // stopped before the first sample: nothing to keep
	}
	// Saved without the caller's ctx: a recording stopped by cancelling it
	// is still one to keep.
	saved, err := s.saved.Save(context.WithoutCancel(ctx), rec)
	if err != nil {
		return rec, fmt.Errorf("save recording: %w", err)
	}
	return saved, nil
}

// Recordings lists saved sessions, newest first, without their load series.
func (s *RecorderService) Recordings(ctx context.Context) ([]recorder.Recording, error) {
	return s.saved.All(ctx)
}

func (s *RecorderService) Recording(ctx context.Context, id string) (recorder.Recording, error) {
	return s.saved.Get(ctx, id)
}

func (s *RecorderService) DeleteRecording(ctx context.Context, id string) error {
	return s.saved.Delete(ctx, id)
}
