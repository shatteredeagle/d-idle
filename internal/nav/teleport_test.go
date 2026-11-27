package nav

import (
	"context"
	"math"
	"testing"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

// wallMaze: a 300x300 level split by a solid wall from y=2000 to y=2290 at x=2100; the only walking gap is at
// the bottom, so walking to the east side is a long detour while a hop crosses the wall.
func wallMaze() *d2map.Grid {
	g := openField(2000, 2000, 300, 300)
	blockRect(g, d2map.Rect{X: 2100, Y: 2000, W: 6, H: 290})
	return g
}

func TestTeleportExploreCrossesWallsAndLandsOnWalkableCells(t *testing.T) {
	g := wallMaze()
	s := newSim(g, input.Point{X: 2010, Y: 2010})
	n := s.navigator()
	n.TeleportKey = "f2"
	oldHopRules(n)
	tuningOf(n).Nav.ExploreTimeoutSec = 2000
	target := d2map.Point{X: 2290, Y: 2010}
	done := func() bool { return s.Current().Grid.Known(target.X, target.Y) }
	if err := n.Explore(ctxT(t), done); err != nil {
		t.Fatal(err)
	}
	if len(s.casts) == 0 {
		t.Fatal("no teleport casts")
	}
	for _, c := range s.casts {
		if !g.Walkable(c.X, c.Y) {
			t.Fatalf("cast at non-walkable cell %v", c)
		}
	}
	if s.travelled > 150 {
		t.Fatalf("walked %.0f subtiles, hopping should do most of it", s.travelled)
	}
}

func TestTeleportExploreTowardHintHeadsThatWay(t *testing.T) {
	g := openField(2000, 2000, 400, 400)
	s := newSim(g, input.Point{X: 2200, Y: 2200})
	n := s.navigator()
	n.TeleportKey = "f2"
	oldHopRules(n)
	tuningOf(n).Nav.ExploreTimeoutSec = 2000
	hint := input.Point{X: 2390, Y: 2200}
	done := func() bool { return s.pos().X > 2330 }
	if err := n.ExploreToward(ctxT(t), hint, done); err != nil {
		t.Fatal(err)
	}
	if len(s.casts) > 12 {
		t.Fatalf("%d casts to cross 130 subtiles toward the hint", len(s.casts))
	}
}

func TestTeleportFallsBackToWalkingWhenCastsDoNothing(t *testing.T) {
	g := openField(2000, 2000, 200, 200)
	s := newSim(g, input.Point{X: 2010, Y: 2010})
	s.noTeleport = true
	n := s.navigator()
	n.TeleportKey = "f2"
	oldHopRules(n)
	tuningOf(n).Nav.ExploreTimeoutSec = 5000
	if err := n.Explore(ctxT(t), nil); err != ErrExplored {
		t.Fatalf("err = %v, want ErrExplored", err)
	}
	if len(s.casts) > 3*teleportBadHops+3 || len(s.clicks) == 0 {
		t.Fatalf("%d casts, %d walk clicks: the walk fallback did not take over", len(s.casts), len(s.clicks))
	}
	if got := s.Current().Grid.KnownCount(); got != 200*200 {
		t.Fatalf("known %d", got)
	}
}

func TestTeleportWithoutManaWalksInsteadOfWaiting(t *testing.T) {
	g := openField(2000, 2000, 400, 100)
	s := newSim(g, input.Point{X: 2010, Y: 2050})
	s.maxMana, s.mana, s.castCost, s.manaRegen = 100, 0, 24, 0.1
	n := s.navigator()
	n.TeleportKey = "f2"
	oldHopRules(n)
	refills := 0
	n.Refill = func(context.Context, int, int) error { refills++; return nil }
	tuningOf(n).Nav.ExploreTimeoutSec = 2000
	start := s.now
	done := func() bool { return s.pos().X > 2060 }
	if err := n.ExploreToward(ctxT(t), input.Point{X: 2390, Y: 2050}, done); err != nil {
		t.Fatal(err)
	}
	if len(s.casts) != 0 || refills != 0 {
		t.Fatalf("%d casts, %d refills: a hop without mana must walk, not wait", len(s.casts), refills)
	}
	if w := s.now.Sub(start); w > 20*time.Second {
		t.Fatalf("took %v: idle wait for mana", w)
	}
}

func TestTeleportTowardStopsNearTarget(t *testing.T) {
	g := wallMaze()
	s := newSim(g, input.Point{X: 2010, Y: 2010})
	n := s.navigator()
	n.TeleportKey = "f2"
	oldHopRules(n)
	// the east side must be known to hop there: reveal it first
	s.px, s.py = 2200, 2010
	_, _ = s.Update()
	s.px, s.py = 2010, 2010
	_, _ = s.Update()
	if err := n.teleportToward(ctxT(t), input.Point{X: 2200, Y: 2010}, 10); err != nil {
		t.Fatal(err)
	}
	if d := dist(s.pos(), input.Point{X: 2200, Y: 2010}); d > 10 {
		t.Fatalf("ended %.0f away", d)
	}
}

// oldHopRules switches the break-even rules off (any leg, no mana reserve): the pre-teletune behaviour.
func oldHopRules(n *Navigator) {
	tuningOf(n).Nav.TeleMinLeg = 0
	tuningOf(n).Nav.TeleReservePct = 0
}

func TestTeleportHopsAcrossWallWhenTheWalkIsLong(t *testing.T) {
	g := wallMaze()
	s := newSim(g, input.Point{X: 2090, Y: 2100})
	n := s.navigator()
	n.TeleportKey = "f2" // default break-even rules
	tuningOf(n).Nav.ExploreTimeoutSec = 2000
	done := func() bool { return s.pos().X > 2115 }
	if err := n.ExploreToward(ctxT(t), input.Point{X: 2290, Y: 2100}, done); err != nil {
		t.Fatal(err)
	}
	if len(s.casts) == 0 || s.travelled > 80 {
		t.Fatalf("%d casts, walked %.0f: the walk around the wall is over the break-even, so it must hop", len(s.casts), s.travelled)
	}
}

func TestHopWorth(t *testing.T) {
	inf := math.Inf(1)
	cases := []struct {
		name string
		h    hopFacts
		want bool
		why  string
	}{
		{"long walk, full pool", hopFacts{WalkLen: 80, Mana: 150, MaxMana: 150, Cost: 24, ReservePct: 40, MinLeg: 36}, true, ""},
		{"short leg 9 subtiles", hopFacts{WalkLen: 9, Mana: 150, MaxMana: 150, Cost: 24, ReservePct: 40, MinLeg: 13}, false, "leg"},
		{"straight 17 subtiles (DRILLS24 hop)", hopFacts{WalkLen: 17, Mana: 150, MaxMana: 150, Cost: 24, ReservePct: 40, MinLeg: 13}, true, ""},
		{"escape: short leg allowed", hopFacts{WalkLen: 3, Mana: 150, MaxMana: 150, Cost: 24, ReservePct: 40, MinLeg: 13, Escape: true, EscapePct: 15}, true, ""},
		{"escape: 60 - 24 = 36 above the 23 floor", hopFacts{WalkLen: 3, Mana: 60, MaxMana: 150, Cost: 24, ReservePct: 40, MinLeg: 13, Escape: true, EscapePct: 15}, true, ""},
		{"escape: 40 - 24 = 16 below the 23 floor", hopFacts{WalkLen: 3, Mana: 40, MaxMana: 150, Cost: 24, ReservePct: 40, MinLeg: 13, Escape: true, EscapePct: 15}, false, "mana"},
		{"escape reserve never above the normal one", hopFacts{WalkLen: 3, Mana: 60, MaxMana: 150, Cost: 24, ReservePct: 10, MinLeg: 13, Escape: true, EscapePct: 50}, true, ""},
		{"no walk at all (island)", hopFacts{WalkLen: inf, Mana: 150, MaxMana: 150, Cost: 24, ReservePct: 40, MinLeg: 36}, true, ""},
		{"reserve: 150 - 24 = 126 ok, 81 - 24 = 57 below 60", hopFacts{WalkLen: 80, Mana: 81, MaxMana: 150, Cost: 24, ReservePct: 40, MinLeg: 36}, false, "mana"},
		{"reserve exactly met", hopFacts{WalkLen: 80, Mana: 84, MaxMana: 150, Cost: 24, ReservePct: 40, MinLeg: 36}, true, ""},
		{"mana stat unknown: leg rule only", hopFacts{WalkLen: 80, Mana: 0, MaxMana: 0, Cost: 24, ReservePct: 40, MinLeg: 36}, true, ""},
		{"knobs at 0: old behaviour", hopFacts{WalkLen: 5, Mana: 24, MaxMana: 150, Cost: 24}, true, ""},
	}
	for _, c := range cases {
		got, why := hopWorth(c.h)
		if got != c.want || why != c.why {
			t.Errorf("%s: got %v %q, want %v %q", c.name, got, why, c.want, c.why)
		}
	}
}

// DRILLS22 geometry (docs/teletune.md): Rocky Waste to Stony Tomb walks in 254 ticks (143 subtiles at 0.5625). Seven
// hops of 17 subtiles on the 150 mana pool lost to the walk in the model as on the live d2d slot.
func TestBreakEvenDrills22Legs(t *testing.T) {
	const regen, fps = 1.17, 25.0 // 150 max mana: 12 256ths a frame (internal/resource.ManaPerSec)
	for _, c := range []struct {
		name  string
		ticks float64 // walking control
		hops  int
	}{{"Stony Tomb", 254, 7}, {"Halls of the Dead 1", 578, 5}} {
		walkSub := c.ticks * RunSubPerTick
		hopTicks := 64.0 // 448 ticks / 7 hops, DRILLS22; about twice the nominal teleHopTicksNominal
		withHops := TeleLegTicks(walkSub, 17, hopTicks, c.hops, 150, 24, regen, fps)
		if c.name == "Stony Tomb" && withHops <= c.ticks {
			t.Errorf("%s: %d hops model %.0f ticks, not slower than walking %.0f", c.name, c.hops, withHops, c.ticks)
		}
		// With the default knobs every one of these straight hops is declined, so the leg is the walk.
		taken := 0
		for left := walkSub; left > 0; left -= 17 {
			if ok, _ := hopWorth(hopFacts{WalkLen: 17, Mana: 150, MaxMana: 150, Cost: 24, ReservePct: 40, MinLeg: BreakEvenLeg(RunSubPerTick, 64)}); ok {
				taken++
			}
		}
		if taken != 0 {
			t.Errorf("%s: %d straight hops taken", c.name, taken)
		}
	}
}

// TELETUNE2: the default minimum leg is v x T (T = 18 ticks, V DRILLS24) plus a margin of 3, rounded.
func TestMinLegDefaultMatchesBreakEven(t *testing.T) {
	want := math.Round(BreakEvenLeg(RunSubPerTick, teleHopTicks) + 3)
	if d := env.DefaultNavTuning().TeleMinLeg; d != want {
		t.Fatalf("default %v, want %v", d, want)
	}
}

// The move driver's answer replaces the key, range, min hop and mana constants; a refusal means no Teleport.
func TestTeleportHopFnReplacesConstants(t *testing.T) {
	n := &Navigator{}
	if n.hasTeleport() || n.teleRange() != defaultTeleportRange || n.teleMinHop() != teleportMinHop {
		t.Fatal("defaults changed")
	}
	h := TeleportHop{Key: "f4", Range: 12, MinHop: 6, Cost: 30}
	n.TeleportHopFn = func() TeleportHop { return h }
	if n.teleKey() != "f4" || n.teleRange() != 12 || n.teleMinHop() != 6 || n.teleMana() != 30 {
		t.Fatalf("key %q range %v min %v mana %d", n.teleKey(), n.teleRange(), n.teleMinHop(), n.teleMana())
	}
	n.TeleportRange = 9 // an explicit field still wins over the row
	if n.teleRange() != 9 {
		t.Fatalf("range %v", n.teleRange())
	}
	h.Refused = "mana"
	if n.hasTeleport() || n.teleMinHop() != teleportMinHop {
		t.Fatal("a refusal still hops")
	}
	n.TeleportKey = "f2" // a fixed key (rigs) wins
	if n.teleKey() != "f2" {
		t.Fatal("fixed key lost")
	}
}
