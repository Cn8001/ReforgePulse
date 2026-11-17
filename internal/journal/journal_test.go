package journal

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
)

func newTest(t *testing.T) *Journal {
	t.Helper()
	j, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tick := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	j.now = func() time.Time { mu.Lock(); defer mu.Unlock(); tick = tick.Add(time.Second); return tick }
	return j
}

func TestJournal_RoundTrip(t *testing.T) {
	// Arrange
	j, ctx := newTest(t), context.Background()

	// Act
	id, err := j.Begin(ctx, domain.Change{Summary: "Cap P-core to 5500 MHz", Target: "PROCFREQMAX1", Before: "0", After: "5500", Reversible: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Finish(ctx, id, "", nil); err != nil {
		t.Fatal(err)
	}
	log, err := j.Log(ctx)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if len(log) != 1 || len(id) != 7 {
		t.Fatalf("log = %+v, id = %q", log, id)
	}
	if c := log[0]; c.Status != domain.Applied || c.Before != "0" || c.After != "5500" || c.Summary == "" {
		t.Fatalf("change = %+v", c)
	}
}

func TestJournal_FailedAndNewestFirst(t *testing.T) {
	// Arrange
	j, ctx := newTest(t), context.Background()
	first, _ := j.Begin(ctx, domain.Change{Summary: "first"})
	second, _ := j.Begin(ctx, domain.Change{Summary: "second"})

	// Act
	_ = j.Finish(ctx, first, "", errors.New("access denied"))
	log, _ := j.Log(ctx)

	// Assert
	if len(log) != 2 || log[0].ID != second || log[1].ID != first {
		t.Fatalf("order = %+v", log)
	}
	if log[1].Status != domain.Failed || log[1].Error != "access denied" {
		t.Fatalf("first = %+v", log[1])
	}
	if log[0].Status != domain.Started {
		t.Fatalf("unfinished change = %+v, want started", log[0])
	}
}

func TestJournal_RevertMarksOriginal(t *testing.T) {
	// Arrange
	j, ctx := newTest(t), context.Background()
	orig, _ := j.Begin(ctx, domain.Change{Summary: "cap", Reversible: true})
	_ = j.Finish(ctx, orig, "", nil)

	// Act
	undo, _ := j.Begin(ctx, domain.Change{Summary: "remove cap", Reverts: orig})
	_ = j.Finish(ctx, undo, "", nil)
	log, _ := j.Log(ctx)

	// Assert
	if log[1].ID != orig || log[1].Status != domain.Reverted || log[0].Status != domain.Applied {
		t.Fatalf("log = %+v", log)
	}
}

func TestJournal_SkipsTornLine(t *testing.T) {
	// Arrange
	j, ctx := newTest(t), context.Background()
	id, _ := j.Begin(ctx, domain.Change{Summary: "kept"})
	f, _ := os.OpenFile(j.path, os.O_APPEND|os.O_WRONLY, 0)
	_, _ = f.WriteString(`{"id":"abc","summ`)
	f.Close()

	// Act
	log, err := j.Log(ctx)

	// Assert
	if err != nil || len(log) != 1 || log[0].ID != id {
		t.Fatalf("log = %+v, err = %v", log, err)
	}
}

func TestJournal_EmptyLog(t *testing.T) {
	// Arrange
	j := newTest(t)

	// Act
	log, err := j.Log(context.Background())

	// Assert
	if err != nil || log == nil || len(log) != 0 {
		t.Fatalf("log = %#v, err = %v", log, err)
	}
}

func TestJournal_ConcurrentWriters(t *testing.T) {
	// Arrange
	j, ctx := newTest(t), context.Background()
	var wg sync.WaitGroup

	// Act
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := j.Begin(ctx, domain.Change{Summary: "change", Target: string(rune('a' + i))})
			if err == nil {
				_ = j.Finish(ctx, id, "", nil)
			}
		}()
	}
	wg.Wait()
	log, _ := j.Log(ctx)

	// Assert
	if len(log) != 20 {
		t.Fatalf("got %d changes, want 20", len(log))
	}
	for _, c := range log {
		if c.Status != domain.Applied {
			t.Fatalf("change %+v not applied", c)
		}
	}
}

func TestJournal_FinishRecordsAfter(t *testing.T) {
	// Arrange
	j, ctx := newTest(t), context.Background()
	id, _ := j.Begin(ctx, domain.Change{Summary: "clean temp", Before: "120 files"})

	// Act
	_ = j.Finish(ctx, id, "freed 1.2 GB, skipped 3", nil)
	log, _ := j.Log(ctx)

	// Assert
	if log[0].After != "freed 1.2 GB, skipped 3" || log[0].Before != "120 files" {
		t.Fatalf("change = %+v", log[0])
	}
}

// record begins and finishes a change and returns its ID.
func record(t *testing.T, j *Journal, c domain.Change) string {
	t.Helper()
	id, err := j.Begin(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Finish(context.Background(), id, "", nil); err != nil {
		t.Fatal(err)
	}
	return id
}

func ids(t *testing.T, j *Journal) []string {
	t.Helper()
	log, err := j.Log(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, c := range log {
		out = append(out, c.ID)
	}
	return out
}

func TestJournal_DeleteTakesRevertsAlongButNotAlone(t *testing.T) {
	// Arrange
	j, ctx := newTest(t), context.Background()
	a := record(t, j, domain.Change{Summary: "a", Reversible: true})
	undo := record(t, j, domain.Change{Summary: "undo a", Reverts: a})
	b := record(t, j, domain.Change{Summary: "b"})

	// Act
	_, aloneErr := j.Delete(ctx, undo)
	n, err := j.Delete(ctx, a)

	// Assert
	if !errors.Is(aloneErr, ErrRevertAlone) {
		t.Fatalf("deleting a revert alone: %v", aloneErr)
	}
	if got := ids(t, j); err != nil || n != 2 || len(got) != 1 || got[0] != b {
		t.Fatalf("n = %d, err = %v, left = %v", n, err, got)
	}
}

func TestJournal_DeleteBeforeAndPrune(t *testing.T) {
	// Arrange
	j, ctx := newTest(t), context.Background()
	var all []string
	for i := range 5 {
		all = append(all, record(t, j, domain.Change{Summary: string(rune('a' + i))}))
	}
	log, _ := j.Log(ctx)
	third := log[2].At // log is newest first: the third of five

	// Act
	before, err := j.DeleteBefore(ctx, third)
	pruned, pruneErr := j.Prune(ctx, 2)

	// Assert
	if err != nil || before != 2 {
		t.Fatalf("delete before: %d, %v", before, err)
	}
	if got := ids(t, j); pruneErr != nil || pruned != 1 || len(got) != 2 || got[0] != all[4] || got[1] != all[3] {
		t.Fatalf("pruned %d, %v, left %v", pruned, pruneErr, got)
	}
	if log, _ := j.Log(ctx); log[0].Status != domain.Applied {
		t.Fatalf("status lost in the rewrite: %+v", log[0])
	}
}
