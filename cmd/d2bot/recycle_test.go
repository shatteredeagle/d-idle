package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/metrics"
)

func TestSwapSourceReadsThroughTheNewSource(t *testing.T) {
	a, b := mem.NewFake(), mem.NewFake()
	a.Put(0x1000, []byte{1, 2})
	b.Put(0x1000, []byte{9, 8})
	sw := &swapSource{s: a}
	buf := make([]byte, 2)
	if err := sw.ReadAt(buf, 0x1000); err != nil || buf[0] != 1 {
		t.Fatalf("old: %v %v", buf, err)
	}
	sw.Swap(b)
	if err := sw.ReadAt(buf, 0x1000); err != nil || buf[0] != 9 {
		t.Fatalf("new: %v %v", buf, err)
	}
}

func TestRecycleDepReportsReasonAndRestart(t *testing.T) {
	if recycleDep(nil) != nil {
		t.Fatal("attach mode has no recycler")
	}
	var steps []string
	rc := &launch.Recycler{Policy: launch.RecyclePolicy{MaxCycles: 2}}
	rc.Leave = func(context.Context) error { steps = append(steps, "leave"); return nil }
	rc.Stop = func(context.Context) error { steps = append(steps, "stop"); return nil }
	rc.Start = func(context.Context) error { steps = append(steps, "start"); return nil }
	rc.Login = func(context.Context) error { steps = append(steps, "login"); return nil }
	dep := recycleDep(rc)
	if did, _, err := dep(context.Background()); did || err != nil {
		t.Fatalf("not due: %v %v", did, err)
	}
	rc.NoteCycle()
	rc.NoteCycle()
	did, why, err := dep(context.Background())
	if !did || err != nil || !strings.Contains(why, "2 cycles") {
		t.Fatalf("due: %v %q %v", did, why, err)
	}
	if strings.Join(steps, ",") != "leave,stop,start,login" || rc.Recycles() != 1 {
		t.Fatalf("steps %v", steps)
	}
}

func TestGameFDsMetric(t *testing.T) {
	reg := metrics.NewRegistry()
	pid := 0
	registerGameMetrics(reg, func() int { return pid }, nil)
	var sb strings.Builder
	reg.WriteTo(&sb)
	if !strings.Contains(sb.String(), "game_open_fds -1") {
		t.Fatalf("no game: %s", sb.String())
	}
	pid = selfPID()
	sb.Reset()
	reg.WriteTo(&sb)
	if strings.Contains(sb.String(), "game_open_fds -1") || !strings.Contains(sb.String(), "game_open_fds") {
		t.Fatalf("self: %s", sb.String())
	}
}

func selfPID() int { return os.Getpid() }
