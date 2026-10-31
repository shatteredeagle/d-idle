package bot

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/env"
)

func repeatRig(t *testing.T) (*rig, *bytes.Buffer, string) {
	dir := t.TempDir()
	g := newRig(t, func(c *Config, _ *Deps, _ *env.Tuning) { c.DeathLogPath = filepath.Join(dir, "deaths.json") })
	buf := &bytes.Buffer{}
	g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	return g, buf, dir
}

// die logs a death at the offset from now and runs the policy as the recovery would.
func dieAt(g *rig, off time.Duration, area int) {
	now := g.clock.Now()
	rec := DeathRecord{Time: now.Add(off), Area: area, AreaName: "Blood Moor"}
	g.r.obMu.Lock()
	g.r.dlog.add(rec)
	g.r.obMu.Unlock()
	g.r.deathPolicy(now, &recovery{rec: rec, area: area})
}

func dumps(dir string) []string {
	m, _ := filepath.Glob(filepath.Join(dir, "plannerdump-*.json"))
	return m
}

func TestRepeatSiteTwoDeathsIn25Min(t *testing.T) {
	g, buf, dir := repeatRig(t)
	g.r.poll()
	dieAt(g, -25*time.Minute, 2)
	if g.r.banned(2) || len(dumps(dir)) != 0 {
		t.Fatal("one death must not ban")
	}
	dieAt(g, -1*time.Minute, 2)
	if !g.r.banned(2) || g.r.avoidWhy[2] != whyRepeat {
		t.Fatalf("banned=%v why=%q", g.r.banned(2), g.r.avoidWhy[2])
	}
	if left := g.r.avoid[2].Sub(g.clock.Now()); left < 29*time.Minute || left > 31*time.Minute {
		t.Fatalf("ban length %v", left)
	}
	out := buf.String()
	if !strings.Contains(out, "death: repeat-site ban area=2 name=Blood Moor deaths=2 window=30min planner dump=") {
		t.Fatalf("log text missing:\n%s", out)
	}
	d := dumps(dir)
	if len(d) != 1 {
		t.Fatalf("dumps %v", d)
	}
	b, _ := os.ReadFile(d[0])
	for _, want := range []string{`"decisions"`, `"areas"`, `"kit"`, `"recentDeaths"`, `"runner"`, `repeat-site`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("dump lacks %s", want)
		}
	}
}

func TestRepeatSiteGapRule(t *testing.T) {
	for _, tc := range []struct {
		name string
		prev time.Duration // previous death, in another area so only the gap can trigger
		ban  bool
	}{
		{"120s", -120 * time.Second, true},
		{"180s", -180 * time.Second, true},
		{"266s", -266 * time.Second, false},
		{"40min", -40 * time.Minute, false},
	} {
		g, _, _ := repeatRig(t)
		dieAt(g, tc.prev, 3)
		dieAt(g, 0, 2)
		if g.r.banned(2) != tc.ban {
			t.Errorf("%s: banned=%v want %v", tc.name, g.r.banned(2), tc.ban)
		}
	}
	// same area, 266 s apart: the window rule fires, that is the a8 case the SME wanted caught
	g, _, _ := repeatRig(t)
	dieAt(g, -266*time.Second, 2)
	dieAt(g, 0, 2)
	if !g.r.banned(2) {
		t.Error("two deaths 266 s apart in one area must ban by the window rule")
	}
	g, _, _ = repeatRig(t)
	dieAt(g, -40*time.Minute, 2)
	dieAt(g, 0, 2)
	if g.r.banned(2) {
		t.Error("deaths 40 min apart must not ban")
	}
}

func TestRepeatSiteOffAndTown(t *testing.T) {
	g, _, _ := repeatRig(t)
	g.r.e.Tune.Bot.RepeatSiteBanMin = 0
	dieAt(g, -10*time.Second, 2)
	dieAt(g, 0, 2)
	if g.r.banned(2) {
		t.Fatal("rule off")
	}
}

// The only farm area carries a repeat-site ban: the no-idle fallback lifts it after the short area wait, not the 30 min.
func TestRepeatSiteBanOnOnlyAreaStillFarmsAfterWait(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 8)
	g.r.poll()
	now := g.clock.Now()
	g.r.skipArea(99, 30*time.Minute, whyRepeat)
	for i := 0; i < 40 && g.r.banned(99); i++ {
		g.r.idleBreak(false)
		g.clock.Sleep(5 * time.Second)
	}
	if g.r.banned(99) {
		t.Fatal("repeat-site ban kept the bot idle")
	}
	if waited := g.clock.Now().Sub(now); waited > 90*time.Second {
		t.Fatalf("waited %v", waited)
	}
}

func TestLowValueCorpseSkipped(t *testing.T) {
	g, buf, _ := repeatRig(t)
	g.r.e.Tune.Bot.CorpseValueMin = 1 // only the new floor decides
	g.w.set(func(w *world) { w.level, w.area = 8, 2 })
	s, _ := g.w.snapshot()
	rec := &recovery{rec: DeathRecord{Time: g.clock.Now(), Area: 2, AreaName: "Blood Moor", CorpseValue: 20}, area: 2}
	ok, why := g.r.retrievalDecision(rec, s)
	if ok || !strings.Contains(why, "skipped") || !strings.Contains(buf.String(), "corpse: skipped (value 20 < 50)") {
		t.Fatalf("ok=%v why=%q log=%s", ok, why, buf.String())
	}
	buf.Reset()
	rec.rec.CorpseValue = 500
	if _, why := g.r.retrievalDecision(rec, s); strings.Contains(why, "skipped") {
		t.Fatalf("valuable corpse skipped: %s", why)
	}
}

func TestAutopsySummaryTagsBug(t *testing.T) {
	au := &Autopsy{BeltHP: 2, Bug: true, Kit: Kit{BeltHP: 2}}
	if s := au.summarize("Fallen", "Blood Moor"); !strings.Contains(s, "BUG: 2 healing potions") {
		t.Fatal(s)
	}
}

// Escalation: the second repeat-site ban of an area within 12 h lasts 4x, the third 12x (cap 6 h).
func TestRepeatSiteEscalates(t *testing.T) {
	g, _, _ := repeatRig(t)
	left := func() time.Duration { return g.r.avoid[2].Sub(g.clock.Now()) }
	near := func(got, want time.Duration) bool { return got > want-time.Minute && got <= want+time.Minute }
	dieAt(g, -200*time.Minute, 2)
	dieAt(g, -190*time.Minute, 2)
	if !near(left(), 30*time.Minute) {
		t.Fatalf("first ban %v", left())
	}
	dieAt(g, -120*time.Minute, 2)
	dieAt(g, -110*time.Minute, 2)
	if !near(left(), 120*time.Minute) {
		t.Fatalf("second ban %v want 120m", left())
	}
	dieAt(g, -60*time.Minute, 2)
	dieAt(g, -50*time.Minute, 2)
	if !near(left(), 360*time.Minute) {
		t.Fatalf("third ban %v want 6h cap (12 x 30m)", left())
	}
	// another area and the knob off stay at the base length
	dieAt(g, -20*time.Minute, 3)
	dieAt(g, -10*time.Minute, 3)
	if l := g.r.avoid[3].Sub(g.clock.Now()); !near(l, 30*time.Minute) {
		t.Fatalf("other area %v", l)
	}
	g2, _, _ := repeatRig(t)
	g2.r.e.Tune.Bot.RepeatSiteEscalate = 0
	for _, off := range []time.Duration{-200, -190, -120, -110} {
		dieAt(g2, off*time.Minute, 2)
	}
	if l := g2.r.avoid[2].Sub(g2.clock.Now()); !near(l, 30*time.Minute) {
		t.Fatalf("escalate off: %v", l)
	}
}
