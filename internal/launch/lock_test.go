package launch

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLockInstanceHoldsNewAndLegacy(t *testing.T) {
	t.Setenv("D2BOT_DATA", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	id := "i7"
	defer os.Remove(LockPaths(id)[1])
	ps := LockPaths(id)
	if len(ps) != 2 || filepath.Base(ps[0]) != "game-i7.lock" || ps[1] != "/tmp/d2lod-game-i7.lock" {
		t.Fatalf("paths %v", ps)
	}
	rel, err := LockInstance(id, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !LockHeld(id) {
		t.Error("LockHeld false while held")
	}
	if _, err := LockInstance(id, 300*time.Millisecond); !errors.Is(err, ErrLocked) {
		t.Errorf("second lock: %v", err)
	}
	rel()
	if LockHeld(id) {
		t.Error("still held after release")
	}
	// A holder of only the legacy lock (old tools) still blocks a new run.
	old, _ := os.OpenFile(ps[1], os.O_RDWR|os.O_CREATE, 0o666)
	defer old.Close()
	if err := flockEx(old); err != nil {
		t.Fatal(err)
	}
	if _, err := LockInstance(id, 300*time.Millisecond); !errors.Is(err, ErrLocked) {
		t.Errorf("legacy holder ignored: %v", err)
	}
}
