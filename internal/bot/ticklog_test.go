package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/arbiter"
)

func tickRig(t *testing.T, shadow, gap int) (*rig, *bytes.Buffer) {
	t.Helper()
	g := newRig(t)
	var buf bytes.Buffer
	g.r.e.Log = slog.New(slog.NewJSONHandler(&buf, nil))
	g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.TickLogSec = shadow, gap
	g.w.set(func(w *world) { w.area = 5 })
	g.r.poll()
	return g, &buf
}

func tickLines(buf *bytes.Buffer) []map[string]any {
	var out []map[string]any
	for _, l := range strings.Split(buf.String(), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(l), &m) == nil && m["msg"] == "tick" {
			out = append(out, m)
		}
	}
	return out
}

func TestTickLogOffByDefault(t *testing.T) {
	g, buf := tickRig(t, 0, 0)
	if def := newRig(t).r.e.Tune.Bot.TickLogSec; def != 0 {
		t.Fatalf("bot.tickLogSec must default to 0, got %d", def)
	}
	for i := 0; i < 5; i++ {
		g.clock.Sleep(time.Second)
		g.r.runGoalShadowed(context.Background(), g.r.latest(), levelGoalControl())
	}
	if n := len(tickLines(buf)); n != 0 {
		t.Fatalf("%d tick lines with the knob off", n)
	}
}

// The tick line is rate-limited by the bot clock, carries the arbiter's facts, works with the shadow off or on, never
// creates the shadow by itself, and its output replays through internal/arbiter (cmd/d2arbiter -follow).
func TestTickLogRateLimitedAndReplays(t *testing.T) {
	for _, shadow := range []int{0, 1} {
		g, buf := tickRig(t, shadow, 2)
		for i := 0; i < 10; i++ {
			g.clock.Sleep(500 * time.Millisecond)
			g.r.runGoalShadowed(context.Background(), g.r.latest(), levelGoalControl())
		}
		lines := tickLines(buf)
		if len(lines) < 2 || len(lines) > 3 {
			t.Fatalf("shadow %d: %d tick lines in 5 s at a 2 s gap", shadow, len(lines))
		}
		facts, _ := lines[0]["facts"].(map[string]any)
		if _, ok := facts[arbiter.FactPotions]; !ok || lines[0]["area"] == nil || lines[0]["vtMs"] == nil {
			t.Fatalf("shadow %d: tick line lacks facts, area or clock: %v", shadow, lines[0])
		}
		if _, ok := g.r.Arbiter(); ok != (shadow == 1) {
			t.Fatalf("shadow %d: shadow decision present = %v", shadow, ok)
		}
		res, err := arbiter.Replay(bytes.NewReader(buf.Bytes()), arbiter.DefaultConfig(), arbiter.ShadowConfig{LogGapSec: 60}, true)
		if err != nil || res.Ticks != len(lines) {
			t.Fatalf("shadow %d: replay ticks %d of %d, err %v", shadow, res.Ticks, len(lines), err)
		}
	}
}
