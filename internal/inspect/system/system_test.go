package system

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
)

type fakeSource struct {
	memErr error
	err    error // returned by every method when set
}

func (f fakeSource) OSName(context.Context) (string, error) { return "Windows 11 (build 26100)", f.err }
func (f fakeSource) CPU(context.Context) (string, int, error) {
	return "Test CPU", 16, f.err
}
func (f fakeSource) Memory(context.Context) (uint64, uint64, error) {
	if f.memErr != nil {
		return 0, 0, f.memErr
	}
	return 32 << 30, 8 << 30, f.err
}
func (f fakeSource) Uptime(context.Context) (time.Duration, error) { return time.Hour, f.err }
func (f fakeSource) Volumes(context.Context) ([]platform.Volume, error) {
	return []platform.Volume{{Root: `C:\`, Total: 1000, Free: 400}}, f.err
}

type fakePriv bool

func (p fakePriv) Elevated() bool          { return bool(p) }
func (p fakePriv) Relaunch([]string) error { return nil }

var fixed = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func newTest(src platform.Source) *Inspector {
	c := New(src, fakePriv(true))
	c.now = func() time.Time { return fixed }
	return c
}

func TestInspector_FullReading(t *testing.T) {
	// Arrange
	c := newTest(fakeSource{})

	// Act
	o, err := c.Overview(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if o.MemoryUsed.Value != 24<<30 || o.MemoryUsed.Status != domain.Measured {
		t.Fatalf("memory used = %+v, want 24 GiB measured", o.MemoryUsed)
	}
	if o.Uptime.Value != 3600 || !o.Uptime.At.Equal(fixed) {
		t.Fatalf("uptime = %+v", o.Uptime)
	}
	if len(o.Volumes) != 1 || o.Volumes[0].Free.Value != 400 || !o.Elevated {
		t.Fatalf("overview = %+v", o)
	}
}

func TestInspector_MemoryErrorIsUnknown(t *testing.T) {
	// Arrange
	c := newTest(fakeSource{memErr: errors.New("boom")})

	// Act
	o, err := c.Overview(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if o.MemoryTotal.Status != domain.Unknown || o.MemoryUsed.Status != domain.Unknown {
		t.Fatalf("memory = %+v / %+v, want unknown", o.MemoryTotal, o.MemoryUsed)
	}
	if o.Uptime.Status != domain.Measured {
		t.Fatal("one failed reading spoiled the others")
	}
}

func TestInspector_UnsupportedSource(t *testing.T) {
	// Arrange
	c := newTest(fakeSource{err: errors.ErrUnsupported})

	// Act
	o, err := c.Overview(context.Background())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if o.Uptime.Status != domain.Unsupported || o.MemoryTotal.Status != domain.Unsupported {
		t.Fatalf("got %+v, want unsupported", o)
	}
}

func TestInspector_CancelledContext(t *testing.T) {
	// Arrange
	c := newTest(fakeSource{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	_, err := c.Overview(ctx)

	// Assert
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
