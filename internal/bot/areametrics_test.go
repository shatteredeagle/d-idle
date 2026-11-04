package bot

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestAreaSummaryAndGoldPickup(t *testing.T) {
	g := newRig(t)
	buf := &lockedBuf{}
	g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	now := g.r.e.Clock.Now()
	s := beltSnap(0, 0)
	p := s.Player
	step := func(area, gold int, exp uint32, dt time.Duration) {
		now = now.Add(dt)
		p.Area, p.Gold, p.Exp = area, gold, exp
		g.r.obMu.Lock()
		g.r.noteAreaMetrics(p, now)
		g.r.obMu.Unlock()
	}
	step(2, 100, 1000, 0)
	step(2, 150, 1200, time.Minute) // +50 gold picked up
	step(2, 140, 1300, time.Minute) // a purchase lowers gold, never counts
	g.r.obMu.Lock()
	g.r.ob.kills += 3
	got := g.r.ob.goldPickupMin
	g.r.obMu.Unlock()
	if got != 50 {
		t.Fatalf("goldPickupMin %d", got)
	}
	step(3, 140, 1300, time.Minute) // leaves area 2
	out := buf.String()
	want := "msg=\"area summary\" area=2 minutes=3 xp=300 kills=3 goldPickup=50"
	if !strings.Contains(out, want) {
		t.Fatalf("missing %q in\n%s", want, out)
	}
	if strings.Count(out, "area summary") != 1 {
		t.Fatalf("one summary per change:\n%s", out)
	}
}

func TestMinuteLineGoldPickupResets(t *testing.T) {
	g := newRig(t)
	buf := &lockedBuf{}
	g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	s := beltSnap(0, 0)
	g.r.obMu.Lock()
	g.r.ob.goldPickupMin = 77
	g.r.minuteLine(s, s.Player, g.r.e.Clock.Now())
	first := lastMinute(buf)
	g.r.minuteLine(s, s.Player, g.r.e.Clock.Now())
	second := lastMinute(buf)
	g.r.obMu.Unlock()
	if !strings.Contains(first, " goldPickup=77") || !strings.Contains(second, " goldPickup=0") {
		t.Fatalf("%s\n%s", first, second)
	}
}
