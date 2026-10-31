package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHeapGuardFiresAndKeepsThreeProfiles(t *testing.T) {
	t.Setenv("GOMEMLIMIT", "1GiB") // the real soft limit stays out of the test process
	var buf bytes.Buffer
	dir := t.TempDir()
	var code int
	used := uint64(100 << 20)
	clock := time.Date(2026, 10, 30, 4, 0, 0, 0, time.UTC)
	g := newHeapGuard(slog.New(slog.NewTextHandler(&buf, nil)), func() int { return 512 }, dir)
	g.heap = func() uint64 { return used }
	g.exit = func(c int) { code = c }
	g.soft = func(int64) {}
	g.now = func() time.Time { clock = clock.Add(time.Second); return clock }
	if g.check() || code != 0 {
		t.Fatal("fired below the limit")
	}
	used = 600 << 20
	for i := 0; i < 5; i++ {
		if !g.check() || code != exitHeap {
			t.Fatalf("round %d: fired=%v code=%d", i, false, code)
		}
	}
	if code != 4 {
		t.Fatalf("exit code %d must be one the supervisor restarts", code)
	}
	if !strings.Contains(buf.String(), "watchdog: heap 600 MB > limit: writing heap profile and exiting for a restart") {
		t.Fatalf("log: %s", buf.String())
	}
	for _, pat := range []string{"heap-*.pprof", "goroutines-*.txt"} {
		m, _ := filepath.Glob(filepath.Join(dir, pat))
		if len(m) != heapGuardKeep {
			t.Fatalf("%s: %d files, want %d", pat, len(m), heapGuardKeep)
		}
	}
}

func TestHeapGuardOffAndSoftLimit(t *testing.T) {
	t.Setenv("GOMEMLIMIT", "1GiB")
	os.Unsetenv("GOMEMLIMIT") // restored by t.Setenv at the end
	limit := 0
	var soft int64
	g := newHeapGuard(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), func() int { return limit }, t.TempDir())
	g.heap = func() uint64 { return 1 << 40 }
	g.exit = func(int) { t.Fatal("exit with the knob at 0") }
	g.soft = func(b int64) { soft = b }
	if g.check() || soft != 0 {
		t.Fatal("knob 0 must be off")
	}
	limit = 1000
	g.exit = func(int) {}
	g.check()
	if want := int64(1250) << 20; soft != want {
		t.Fatalf("soft limit %d, want %d", soft, want)
	}
}
