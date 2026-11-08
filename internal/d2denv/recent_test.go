package d2denv

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dsrc"
)

func TestReaderRecent(t *testing.T) {
	r := &Reader{}
	if _, ok := r.Recent(time.Second); ok {
		t.Fatal("no snapshot read yet: Recent must say no")
	}
	snap := &d2.Snapshot{State: d2.StateInGame}
	r.last, r.lastAt = &d2dsrc.Result{Snap: snap}, time.Now()
	if got, ok := r.Recent(time.Second); !ok || got != snap {
		t.Fatalf("a snapshot just read must be returned: %v %v", got, ok)
	}
	r.lastAt = time.Now().Add(-2 * time.Second)
	if _, ok := r.Recent(time.Second); ok {
		t.Fatal("a snapshot older than maxAge must not be returned")
	}
}
