// Package processes turns process snapshots into per-process load. CPU use
// is the CPU time a process gained since the previous snapshot, spread over
// every logical processor, the way Task Manager shows it.
package processes

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

// Deps are what the inspector reads from.
type Deps struct {
	Processes platform.Processes
	GPU       platform.GPUByProcess
	CPUs      int // logical processors, to spread CPU time over
}

type Inspector struct {
	src  platform.Processes
	gpu  platform.GPUByProcess
	cpus int
	now  func() time.Time

	mu   sync.Mutex
	prev map[uint32]time.Duration
	at   time.Time
}

func New(d Deps) *Inspector {
	return &Inspector{src: d.Processes, gpu: d.GPU, cpus: max(1, d.CPUs), now: time.Now}
}

// List returns every process, busiest first. The first call has no earlier
// snapshot, so every CPU figure starts at zero.
func (in *Inspector) List(ctx context.Context) ([]domain.Process, error) {
	infos, err := in.src.Processes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}
	at := in.now()
	// Ports and GPU shares only add detail; the list stands without them.
	ports, _ := in.src.Ports(ctx)
	byPID := map[uint32][]domain.Port{}
	for _, p := range ports {
		byPID[p.PID] = append(byPID[p.PID], domain.Port{Proto: p.Proto, Local: p.Local, Port: p.Port, Remote: p.Remote, State: p.State})
	}
	var gpu map[uint32]float64
	if in.gpu != nil {
		gpu, _ = in.gpu.GPUProcesses(ctx)
	}

	in.mu.Lock()
	wall := at.Sub(in.at)
	prev := in.prev
	in.prev, in.at = make(map[uint32]time.Duration, len(infos)), at
	out := make([]domain.Process, 0, len(infos))
	for _, p := range infos {
		in.prev[p.PID] = p.CPUTime
		var cpu float64
		// A PID reused by a new process shows less CPU time than before; skip it.
		if before, ok := prev[p.PID]; ok && wall > 0 && p.CPUTime >= before {
			cpu = 100 * float64(p.CPUTime-before) / float64(wall) / float64(in.cpus)
		}
		out = append(out, domain.Process{
			PID: p.PID, ParentPID: p.ParentPID, Name: p.Name, Path: p.Path, Threads: p.Threads,
			CPU: min(cpu, 100), WorkingSet: p.WorkingSet, Private: p.Private, Access: p.Opened,
			GPU: min(gpu[p.PID], 100), Hung: p.Hung, Ports: byPID[p.PID],
		})
	}
	in.mu.Unlock()

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CPU != out[j].CPU {
			return out[i].CPU > out[j].CPU
		}
		return out[i].Private > out[j].Private
	})
	return out, nil
}

// Kill ends a process.
func (in *Inspector) Kill(ctx context.Context, pid uint32) error {
	return in.src.Kill(ctx, pid)
}
