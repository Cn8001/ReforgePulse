package consent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cn8001/reforgepulse/internal/store"
)

func TestStore_AcceptHoldsUntilTheTextsChange(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db, err := store.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)
	v1, v2 := Version("terms", "mit"), Version("terms, changed", "mit")

	// Act
	before, _ := s.Accepted(ctx, v1)
	acceptErr := s.Accept(ctx, v1)
	after, _ := s.Accepted(ctx, v1)
	changed, _ := s.Accepted(ctx, v2)

	// Assert
	if before || acceptErr != nil || !after || changed {
		t.Fatalf("before %v, accept %v, after %v, changed texts %v", before, acceptErr, after, changed)
	}
}
