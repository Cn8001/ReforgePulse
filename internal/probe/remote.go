package probe

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

// Local is what the child runs: the in-process readers.
type Local struct {
	Cores interface {
		List(context.Context) ([]domain.CoreInfo, error)
	}
	Loads interface {
		CoreLoads(context.Context) ([]platform.CoreLoad, error)
	}
	Sensors interface {
		Sensors(context.Context) platform.SensorStatus
	}
	Tester interface {
		Run(ctx context.Context, cpus []int, perCore time.Duration, report func(float64, string)) (domain.CoreTest, error)
	}
}

type testArgs struct {
	CPUs    []int         `json:"cpus"`
	PerCore time.Duration `json:"perCore"`
}

// Handlers serves l to the window.
func Handlers(l Local) map[string]Handler {
	return map[string]Handler{
		"cores": func(ctx context.Context, _ json.RawMessage, _ func(float64, string)) (any, error) {
			return l.Cores.List(ctx)
		},
		"loads": func(ctx context.Context, _ json.RawMessage, _ func(float64, string)) (any, error) {
			return l.Loads.CoreLoads(ctx)
		},
		"sensors": func(ctx context.Context, _ json.RawMessage, _ func(float64, string)) (any, error) {
			return l.Sensors.Sensors(ctx), nil
		},
		"coretest": func(ctx context.Context, raw json.RawMessage, report func(float64, string)) (any, error) {
			var a testArgs
			if err := json.Unmarshal(raw, &a); err != nil {
				return nil, err
			}
			return l.Tester.Run(ctx, a.CPUs, a.PerCore, report)
		},
	}
}

// Remote is the child seen from the window. It satisfies the same
// interfaces as the in-process readers, so the services do not know the
// difference.
type Remote struct{ c *Client }

func NewRemote(c *Client) Remote { return Remote{c: c} }

func (r Remote) List(ctx context.Context) ([]domain.CoreInfo, error) {
	var out []domain.CoreInfo
	err := r.c.Call(ctx, "cores", nil, &out, nil)
	return out, err
}

func (r Remote) CoreLoads(ctx context.Context) ([]platform.CoreLoad, error) {
	var out []platform.CoreLoad
	err := r.c.Call(ctx, "loads", nil, &out, nil)
	return out, err
}

// Sensors reports a failed child the way it reports a missing sensor: as
// the reason next to each source.
func (r Remote) Sensors(ctx context.Context) platform.SensorStatus {
	var out platform.SensorStatus
	if err := r.c.Call(ctx, "sensors", nil, &out, nil); err != nil {
		return platform.SensorStatus{NVML: err.Error(), HWiNFO: err.Error()}
	}
	return out
}

func (r Remote) Run(ctx context.Context, cpus []int, perCore time.Duration, report func(float64, string)) (domain.CoreTest, error) {
	var out domain.CoreTest
	err := r.c.Call(ctx, "coretest", testArgs{CPUs: cpus, PerCore: perCore}, &out, report)
	return out, err
}
