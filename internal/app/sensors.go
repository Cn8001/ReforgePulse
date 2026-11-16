package app

import (
	"context"

	"github.com/cn8001/reforgepulse/internal/platform"
)

type SensorReader interface {
	Sensors(ctx context.Context) platform.SensorStatus
}

// SensorService feeds the Sensors tool; it is polled, outside the runner.
type SensorService struct{ r SensorReader }

func NewSensorService(r SensorReader) *SensorService { return &SensorService{r: r} }

func (s *SensorService) Read(ctx context.Context) platform.SensorStatus { return s.r.Sensors(ctx) }
