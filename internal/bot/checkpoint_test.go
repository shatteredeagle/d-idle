package bot

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

func cpOn(t env.BotTuning) cpTune {
	t.CheckpointEnable = 1
	return cpTune{t.CheckpointEnable, t.CheckpointMinLifePct, t.CheckpointRetreatPct, t.CheckpointGapSec, t.CheckpointMinScrolls,
		t.CheckpointSafeRadius, t.CheckpointEntryDist, t.CheckpointReach, t.CheckpointArrivalHostiles}
}

func TestCheckpointDecideTable(t *testing.T) {
	tune := cpOn(env.DefaultTuning().Bot)
	base := cpIn{Now: time.Unix(1000, 0), Area: 37, X: 10, Y: 10, LifePct: 100, ArrX: 5, ArrY: 5, Scrolls: 5, DangerArea: true}
	rec := cpRec{Area: 37, X: 10, Y: 10}
	cases := []struct {
		name string
		mut  func(*cpIn)
		want cpAction
	}{
		{"healthy at the door", func(*cpIn) {}, cpCast},
		{"not a danger area", func(i *cpIn) { i.DangerArea = false }, cpNothing},
		{"life under the cast line", func(i *cpIn) { i.LifePct = 79 }, cpNothing},
		{"hostile near", func(i *cpIn) { i.HostilesNear = 1 }, cpNothing},
		{"scrolls at the reserve", func(i *cpIn) { i.Scrolls = 2 }, cpNothing},
		{"one above the reserve", func(i *cpIn) { i.Scrolls = 3 }, cpCast},
		{"far from the entry", func(i *cpIn) { i.X, i.Y = 100, 100 }, cpNothing},
		{"gap not over", func(i *cpIn) { i.LastCast = i.Now.Add(-59 * time.Second) }, cpNothing},
		{"gap over", func(i *cpIn) { i.LastCast = i.Now.Add(-61 * time.Second) }, cpCast},
		{"record already here", func(i *cpIn) { i.State, i.Rec = cpSet, rec }, cpNothing},
		{"a step forbids portals", func(i *cpIn) { i.NoTP = true }, cpNothing},
		{"retreat at low life", func(i *cpIn) { i.State, i.Rec, i.LifePct = cpSet, rec, 40 }, cpRetreat},
		{"no retreat above the line", func(i *cpIn) { i.State, i.Rec, i.LifePct = cpSet, rec, 41 }, cpNothing},
		{"no retreat when surrounded", func(i *cpIn) { i.State, i.Rec, i.LifePct, i.Adjacent = cpSet, rec, 30, 3 }, cpNothing},
		{"no retreat out of reach", func(i *cpIn) { i.State, i.Rec, i.LifePct, i.X = cpSet, cpRec{Area: 37, X: 500, Y: 10}, 30, 10 }, cpNothing},
		{"no retreat to another area", func(i *cpIn) { i.State, i.Rec, i.LifePct = cpSet, cpRec{Area: 36}, 30 }, cpNothing},
	}
	for _, c := range cases {
		in := base
		c.mut(&in)
		if got := cpDecide(in, tune); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	off := tune
	off.Enable = 0
	if cpDecide(base, off) != cpNothing {
		t.Error("knob off: a checkpoint was due")
	}
}

// cpTestArea is the Crypt: a danger area the route guard leaves open for the rig hero (Catacombs 4 is gated by the kit).
const cpTestArea = 17

// cpRig stands a hero of class c at the door of a danger area (Catacombs Level 4) with n portal scrolls.
func cpRig(t *testing.T, c d2.Class, scrolls int) *rig {
	g := newRig(t)
	g.e.Tune.Bot.CheckpointEnable = 1
	g.w.set(func(w *world) {
		w.class, w.area, w.level, w.life, w.maxL = c, cpTestArea, 20, 100, 100
		for i := 0; i < scrolls; i++ {
			w.items = append(w.items, d2.Unit{ID: uint32(950 + i), Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
		}
	})
	g.r.poll()
	return g
}

func TestCheckpointCastAtTheDoor(t *testing.T) {
	for _, c := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		g := cpRig(t, c, 4)
		s := g.r.poll()
		if got := g.r.checkpointDue(s, g.clock.Now()); got != cpCast {
			t.Fatalf("%v: due %v, want cast", c, got)
		}
		if !g.r.checkpointStep(context.Background(), s, g.r.control()) {
			t.Fatalf("%v: the step did not act", c)
		}
		if !strings.Contains(g.w.log(), "act town-portal") {
			t.Errorf("%v: no portal cast: %s", c, g.w.log())
		}
		if g.r.cp.st != cpSet || g.r.cp.rec.Area != cpTestArea || g.r.cp.rec.Scrolls != 3 {
			t.Errorf("%v: record %+v state %v", c, g.r.cp.rec, g.r.cp.st)
		}
		// the rate limit and the record: nothing more is due
		if got := g.r.checkpointDue(g.r.poll(), g.clock.Now()); got != cpNothing {
			t.Errorf("%v: second ask %v", c, got)
		}
	}
}

func TestCheckpointOffByDefault(t *testing.T) {
	g := cpRig(t, d2.Sorceress, 4)
	g.e.Tune.Bot.CheckpointEnable = 0
	if got := g.r.checkpointDue(g.r.poll(), g.clock.Now()); got != cpNothing {
		t.Fatalf("knob 0: %v", got)
	}
	if env.DefaultTuning().Bot.CheckpointEnable != 0 {
		t.Fatal("the default must be off")
	}
}

func TestCheckpointKeepsTheScrollReserve(t *testing.T) {
	g := cpRig(t, d2.Sorceress, 2)
	if got := g.r.checkpointDue(g.r.poll(), g.clock.Now()); got != cpNothing {
		t.Fatalf("two scrolls: %v, the reserve must stay", got)
	}
}

func TestCheckpointNotWithHostilesNear(t *testing.T) {
	g := cpRig(t, d2.Barbarian, 4)
	g.w.addMonster(500, 10, 5, 0)
	if got := g.r.checkpointDue(g.r.poll(), g.clock.Now()); got != cpNothing {
		t.Fatalf("hostile near: %v", got)
	}
}

func TestCheckpointRetreatWalksAndTripsHome(t *testing.T) {
	for _, c := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		g := cpRig(t, c, 4)
		s := g.r.poll()
		g.r.checkpointStep(context.Background(), s, g.r.control()) // cast
		g.w.set(func(w *world) { w.x += 20; w.life = 35 })
		s = g.r.poll()
		if got := g.r.checkpointDue(s, g.clock.Now()); got != cpRetreat {
			t.Fatalf("%v: due %v, want retreat", c, got)
		}
		g.r.checkpointStep(context.Background(), s, g.r.control())
		if !strings.Contains(g.w.log(), "goto") {
			t.Errorf("%v: no walk to the portal: %s", c, g.w.log())
		}
		if g.r.loop.trip == nil || g.r.loop.trip.returnTo != cpTestArea || !g.r.loop.trip.emergency {
			t.Errorf("%v: trip %+v", c, g.r.loop.trip)
		}
		if g.r.cp.st != cpUse {
			t.Errorf("%v: state %v", c, g.r.cp.st)
		}
	}
}

func TestCheckpointInvalidations(t *testing.T) {
	g := cpRig(t, d2.Sorceress, 4)
	cast := func() {
		g.r.cp.lastCast = time.Time{}
		g.r.checkpointStep(context.Background(), g.r.poll(), g.r.control())
		if g.r.cp.st != cpSet {
			t.Fatalf("no checkpoint: %v", g.r.cp.st)
		}
	}
	cast()
	g.r.cpNewCast(SafetyTownPortal) // a safety portal replaces it
	if g.r.cp.st != cpNone {
		t.Errorf("safety portal: %v", g.r.cp.st)
	}
	cast()
	g.r.forgetGame() // a new game
	if g.r.cp.st != cpNone {
		t.Errorf("new game: %v", g.r.cp.st)
	}
	cast()
	// home by another way (far from the portal): the record goes
	g.w.set(func(w *world) { w.x += 300 })
	g.r.poll()
	g.w.set(func(w *world) { w.area = 1 })
	g.r.observe(mustSnap(t, g.w), g.clock.Now())
	if g.r.cp.st != cpNone {
		t.Errorf("home another way: %v", g.r.cp.st)
	}
}

func mustSnap(t *testing.T, w *world) *d2.Snapshot {
	s, err := w.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCheckpointDeathRoute(t *testing.T) {
	g := cpRig(t, d2.Sorceress, 4)
	g.r.checkpointStep(context.Background(), g.r.poll(), g.r.control())
	g.r.cpDied()
	if g.r.cp.st != cpDead {
		t.Fatalf("after death: %v", g.r.cp.st)
	}
	g.w.set(func(w *world) { w.area = 1 })
	s := g.r.poll()
	if !g.r.cpRoute(s, cpTestArea) {
		t.Fatalf("the checkpoint is not the route (allowed: %q, stale %d)", g.r.areaAllowed(s, cpTestArea), g.r.stalePackArea(s))
	}
	if g.r.ob.portalFrom != cpTestArea || g.r.cp.st != cpReturn {
		t.Errorf("portalFrom %d state %v", g.r.ob.portalFrom, g.r.cp.st)
	}
	// a stale pack at the exit: waypoint and walk, the record is dropped
	g2 := cpRig(t, d2.Barbarian, 4)
	g2.r.checkpointStep(context.Background(), g2.r.poll(), g2.r.control())
	g2.r.cpDied()
	for i := 0; i < 6; i++ {
		g2.w.addMonster(uint32(600+i), 10, 2+i, 1)
	}
	g2.r.earlyExit(g2.r.poll(), SafetyChicken) // notes the stale pack in 37
	g2.w.set(func(w *world) { w.area = 1 })
	if g2.r.cpRoute(g2.r.poll(), cpTestArea) || g2.r.cp.st != cpNone || g2.r.ob.portalFrom != 0 {
		t.Errorf("stale pack: route kept (state %v, portalFrom %d)", g2.r.cp.st, g2.r.ob.portalFrom)
	}
}

func TestCheckpointArrivalCrowd(t *testing.T) {
	g := cpRig(t, d2.Sorceress, 4)
	g.r.cp.st = cpReturn
	g.r.cp.rec = cpRec{Area: cpTestArea}
	for i := 0; i < 5; i++ {
		g.w.addMonster(uint32(700+i), 10, 3+i, 0)
	}
	if err := g.r.cpArrival(context.Background()); err == nil {
		t.Fatal("a crowd at the exit was accepted")
	}
	if !strings.Contains(g.w.log(), "totown") || g.r.cp.st != cpNone {
		t.Errorf("no way home: log %q state %v", g.w.log(), g.r.cp.st)
	}
	// a quiet exit: the checkpoint is live again
	g2 := cpRig(t, d2.Barbarian, 4)
	g2.r.cp.st = cpReturn
	if err := g2.r.cpArrival(context.Background()); err != nil || g2.r.cp.st != cpSet {
		t.Errorf("quiet exit: err %v state %v", err, g2.r.cp.st)
	}
}

// The enabled rule that casts nothing says why, once per area and reason; off, it writes nothing (two classes).
func TestCheckpointIdleEvidence(t *testing.T) {
	for _, c := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		g := cpRig(t, c, 2) // two scrolls: the reserve holds the cast
		buf := &bytes.Buffer{}
		g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
		for range 3 {
			g.r.checkpointDue(g.r.poll(), g.clock.Now())
		}
		if n := strings.Count(buf.String(), `msg="checkpoint idle" why="scroll reserve"`); n != 1 {
			t.Fatalf("%v: %d idle lines, want 1: %s", c, n, buf.String())
		}
		g.clock.Sleep(cpIdleGap + time.Second)
		g.r.checkpointDue(g.r.poll(), g.clock.Now())
		if n := strings.Count(buf.String(), "checkpoint idle"); n != 2 {
			t.Fatalf("%v: %d idle lines after the gap, want 2", c, n)
		}
		buf.Reset()
		g.e.Tune.Bot.CheckpointEnable = 0
		g.r.checkpointDue(g.r.poll(), g.clock.Now())
		if buf.Len() != 0 {
			t.Fatalf("%v: knob 0 wrote %s", c, buf.String())
		}
	}
}
