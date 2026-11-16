// Package system inspects this PC locally and turns raw platform readings
// into the system overview. Nothing it reads leaves the machine.
package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

type Inspector struct {
	src  platform.Source
	priv platform.Privileges
	now  func() time.Time
}

func New(src platform.Source, priv platform.Privileges) *Inspector {
	return &Inspector{src: src, priv: priv, now: time.Now}
}

// Overview reads every fact it can. A failed reading is marked missing
// rather than failing the whole overview; only cancellation is an error.
func (c *Inspector) Overview(ctx context.Context) (domain.Overview, error) {
	at := c.now()
	o := domain.Overview{Elevated: c.priv.Elevated()}

	o.OS, _ = c.src.OSName(ctx)
	o.CPUModel, o.LogicalCPUs, _ = c.src.CPU(ctx)

	if up, err := c.src.Uptime(ctx); err != nil {
		o.Uptime = missing(err, "GetTickCount64")
	} else {
		o.Uptime = domain.Measure(up.Seconds(), domain.Seconds, "GetTickCount64", at)
	}

	const memSrc = "GlobalMemoryStatusEx"
	if total, avail, err := c.src.Memory(ctx); err != nil {
		o.MemoryTotal, o.MemoryUsed = missing(err, memSrc), missing(err, memSrc)
	} else {
		o.MemoryTotal = domain.Measure(float64(total), domain.Bytes, memSrc, at)
		o.MemoryUsed = domain.Measure(float64(total-avail), domain.Bytes, memSrc, at)
	}

	vols, _ := c.src.Volumes(ctx)
	for _, v := range vols {
		o.Volumes = append(o.Volumes, domain.Volume{
			Root:  v.Root,
			Total: domain.Measure(float64(v.Total), domain.Bytes, "GetDiskFreeSpaceEx", at),
			Free:  domain.Measure(float64(v.Free), domain.Bytes, "GetDiskFreeSpaceEx", at),
		})
	}

	if err := ctx.Err(); err != nil {
		return domain.Overview{}, fmt.Errorf("read overview: %w", err)
	}
	return o, nil
}

func missing(err error, source string) domain.Measurement {
	if errors.Is(err, errors.ErrUnsupported) {
		return domain.Missing(domain.Unsupported, source)
	}
	return domain.Missing(domain.Unknown, source)
}
