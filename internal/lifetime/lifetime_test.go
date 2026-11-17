package lifetime

import (
	"context"
	"testing"
	"time"
)

func TestFollow_EndsWhenAnOpenWindowCloses(t *testing.T) {
	// Arrange
	states := []bool{false, true, true, false}
	i := 0
	open := func() bool { v := states[min(i, len(states)-1)]; i++; return v }
	ended := make(chan struct{})

	// Act
	go Follow(context.Background(), open, time.Millisecond, func() { close(ended) })

	// Assert
	select {
	case <-ended:
	case <-time.After(time.Second):
		t.Fatal("did not end after the window closed")
	}
}

func TestFollow_KeepsServingWhenNoWindowWasEverOpen(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	ended := false

	// Act
	Follow(ctx, func() bool { return false }, time.Millisecond, func() { ended = true })

	// Assert
	if ended {
		t.Fatal("ended although no window was ever open")
	}
}
