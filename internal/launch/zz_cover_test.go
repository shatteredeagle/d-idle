package launch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestCoverValidIDAndSortInts(t *testing.T) {
	for id, want := range map[string]bool{"i0": true, "i9": true, "i10": false, "": false, "x1": false, "i": false} {
		if ValidID(id) != want {
			t.Errorf("ValidID(%q) = %v", id, !want)
		}
	}
	a := []int{5, 3, 9, 1, 3}
	sortInts(a)
	for i := 1; i < len(a); i++ {
		if a[i] < a[i-1] {
			t.Fatalf("unsorted %v", a)
		}
	}
}

// StopGame is the orphan clean-up: it must leave the Xvfb (and so the d2input helper) of the instance up.
func TestCoverStopGameKeepsXvfbAndOtherInstances(t *testing.T) {
	l, f, out := testLife(t)
	ctx := context.Background()
	for _, id := range []string{"i1", "i2"} {
		cfg, _ := l.config(id)
		f.exists[filepath.Join(cfg.PrefixDir, "pfx")] = true
		if err := l.Start(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	out.Reset()
	if err := l.StopGame(ctx, "i1"); err != nil {
		t.Fatal(err)
	}
	if len(l.GamePIDs("i1")) != 0 {
		t.Fatal("i1 still running")
	}
	if len(l.GamePIDs("i2")) != 1 {
		t.Fatal("StopGame i1 killed i2")
	}
	if strings.Contains(out.String(), "Xvfb") {
		t.Fatalf("StopGame touched the display: %q", out.String())
	}
	for _, p := range f.procs {
		if p.Argv[0] == "Xvfb" && p.Argv[1] == ":101" {
			return
		}
	}
	t.Fatal("Xvfb :101 gone after StopGame")
}

type sigFailSys struct {
	*fakeSys
	err error
}

func (s sigFailSys) Signal(int, syscall.Signal) error { return s.err }

func TestCoverStopSurfacesSignalError(t *testing.T) {
	l, f, _ := testLife(t)
	cfg, _ := l.config("i1")
	f.exists[filepath.Join(cfg.PrefixDir, "pfx")] = true
	if err := l.Start(context.Background(), "i1"); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("operation not permitted")
	l.Sys = sigFailSys{f, boom}
	if err := l.Stop(context.Background(), "i1"); !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
}

func TestCoverStartMissingLogDirAndSpawnFailure(t *testing.T) {
	l, f, _ := testLife(t)
	cfg, _ := l.config("i1")
	f.exists[filepath.Join(cfg.PrefixDir, "pfx")] = true
	l.Sys = spawnFailSys{f}
	err := l.Start(context.Background(), "i1")
	if err == nil || !strings.Contains(err.Error(), "spawn") {
		t.Fatalf("got %v", err)
	}
	if _, serr := os.Stat(l.Base.Root); serr != nil {
		t.Log("root not created:", serr)
	}
}

type spawnFailSys struct{ *fakeSys }

func (s spawnFailSys) Spawn(env []string, dir, log, name string, args ...string) error {
	if name == "Xvfb" {
		return s.fakeSys.Spawn(env, dir, log, name, args...)
	}
	return errors.New("spawn denied")
}

func TestCoverEnsureHelperNeverListens(t *testing.T) {
	l, f, _ := testLife(t)
	cfg, _ := l.config("i1")
	f.exists[filepath.Join(cfg.PrefixDir, "pfx")] = true
	l.Sys = noListenSys{f}
	err := l.EnsureHelper(context.Background(), "i1")
	if err == nil || !strings.Contains(err.Error(), "did not start") {
		t.Fatalf("got %v", err)
	}
	if err := l.EnsureHelper(context.Background(), "i0"); err == nil {
		t.Fatal("i0 helper must be refused")
	}
}

type noListenSys struct{ *fakeSys }

func (noListenSys) HelperUp(int) bool { return false }
