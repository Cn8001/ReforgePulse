package consent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStore_AcceptHoldsUntilTheTextsChange(t *testing.T) {
	// Arrange
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
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

func TestStore_DamagedFileAsksAgain(t *testing.T) {
	// Arrange
	ctx := context.Background()
	dir := t.TempDir()
	s, _ := Open(dir)
	if err := os.WriteFile(filepath.Join(dir, "accepted.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Act
	ok, err := s.Accepted(ctx, Version("x"))

	// Assert
	if ok || err != nil {
		t.Fatalf("accepted = %v, err = %v", ok, err)
	}
}
