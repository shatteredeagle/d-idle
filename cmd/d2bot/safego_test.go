package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestGoSafeRestartsAfterPanic(t *testing.T) {
	goBackoff = time.Millisecond
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	n := 0
	goSafe(context.Background(), log, "t", func() {
		n++
		if n < 3 {
			panic("boom")
		}
	})
	if n != 3 {
		t.Fatalf("ran %d times, want 3", n)
	}
}

func TestGoSafeGivesUp(t *testing.T) {
	goBackoff = time.Microsecond
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	n := 0
	goSafe(context.Background(), log, "t", func() { n++; panic("always") })
	if n != maxGoPanics+1 {
		t.Fatalf("ran %d times, want %d", n, maxGoPanics+1)
	}
}
