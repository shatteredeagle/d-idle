package main

import (
	"errors"
	"testing"

	"d2lod-bot/internal/launch"
)

func TestInstanceGuardNonBlockingAndOffUnderWrapper(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	id := "i97"
	g := instanceGuard(id)
	if g == nil {
		t.Skip("lock already held on this machine")
	}
	rel, err := g()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g(); !errors.Is(err, launch.ErrLocked) {
		t.Fatalf("second take must fail at once, got %v", err)
	}
	if instanceGuard(id) != nil {
		t.Fatal("a lock held at startup means a wrapper owns it: guard must be off")
	}
	rel()
	if _, err := g(); err != nil {
		t.Fatal(err)
	}
}
