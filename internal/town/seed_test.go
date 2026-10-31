package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/belt"
	"d2lod-bot/internal/town/sim"
)

// The same chores must work wherever the seed puts the town.
func TestChoresAtOtherSeedOrigins(t *testing.T) {
	geartest.Need(t)
	for _, o := range [][2]int{{0, 0}, {1300, -400}, {-900, 700}} {
		r := newRigAt(t, o[0], o[1])
		w := r.W
		w.Player.Life = 10
		w.Add("r07", d2.ContInventory, 0, 0)
		rep, err := r.T.Chores(context.Background(), town.FullPlan())
		if err != nil || rep.Errors != 0 || !rep.Healed || rep.Stashed != 1 || count(w, d2.ContBelt) != 4 {
			t.Fatalf("origin %v: %v %+v", o, err, rep.Steps)
		}
	}
}

// lazyMover spawns the NPC when the walker arrives at the preset position and fails anywhere else, so the
// only coordinates that can work are the ones the presets gave.
type lazyMover struct {
	*sim.Mover
	spawn map[input.Point]func()
	t     *testing.T
}

func (m lazyMover) GoTo(ctx context.Context, p input.Point) error {
	f, ok := m.spawn[p]
	if !ok {
		m.t.Errorf("walked to %v: not a preset position", p)
		return town.ErrNoNPC
	}
	f()
	return m.Mover.GoTo(ctx, p)
}

func TestFallbackUsesPresetsNotConstants(t *testing.T) {
	geartest.Need(t)
	const dx, dy = 2222, -777
	r := newRigAt(t, dx, dy)
	w := r.W
	w.Mons = nil // nobody loaded yet
	w.Player.Life = 10
	akara := input.Point{X: 6082 + dx, Y: 4911 + dy}
	mv := lazyMover{Mover: r.Mv, t: t, spawn: map[input.Point]func(){akara: func() {
		a := w.AddNPC(148, akara.X, akara.Y)
		w.Stock(a, "hp1")
	}}}
	tw := town.New(r.Env, mv)
	tw.UseSource(town.Source{Snapshot: w.Snapshot, Items: w.AllItems, Owner: w.Owner, Stats: w.UnitStats,
		Presets: func() []d2map.Preset {
			return []d2map.Preset{{Kind: d2map.PresetNPC, ID: 148, Pos: d2map.Point{X: akara.X, Y: akara.Y}}}
		}})
	rep, err := tw.Chores(context.Background(), town.ChorePlan{Heal: true})
	if err != nil || rep.Errors != 0 || !rep.Healed {
		t.Fatalf("%v %+v", err, rep.Steps)
	}
}

func TestBrokeIsACleanOutcome(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Gold = 0
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Scrolls: true})
	if err != nil || rep.Errors != 0 || !rep.Broke || rep.Spent != 0 {
		t.Fatalf("%v %+v", err, rep)
	}
	if r.W.UI.AnyPanelOpen() {
		t.Error("panel left open")
	}
}

func TestNoManaAdviceForBarbarianAndAfterFutileVisit(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Class = d2.Barbarian
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	for i := 0; i < 4; i++ {
		r.W.Add("hp1", d2.ContBelt, i, 0)
	}
	s, _ := r.W.Snapshot()
	if ok, why := r.T.NeedsTrip(s); ok {
		t.Fatalf("barbarian: %q", why)
	}
	// Broke: nothing in the shop is affordable, so the advice stays quiet instead of looping trips.
	r1 := newRig(t)
	r1.W.Player.Gold = 0
	r1.W.Add("tsc", d2.ContInventory, 9, 3)
	for i := 0; i < 3; i++ {
		r1.W.Add("hp1", d2.ContBelt, i, 0)
	}
	s, _ = r1.W.Snapshot()
	if ok, why := r1.T.NeedsTrip(s); ok {
		t.Fatalf("0 gold: %q", why)
	}
	// A sorceress with a mana column and gold, but a shop without mana potions: one visit, then quiet, even
	// when gold grows, for FutileMinutes; after that only once gold grew by the step.
	r2 := newRig(t)
	r2.W.Player.Gold = 200
	r2.W.Stock(r2.Akara) // nothing on sale
	r2.W.Add("tsc", d2.ContInventory, 9, 3)
	for i := 0; i < 3; i++ {
		r2.W.Add("hp1", d2.ContBelt, i, 0)
	}
	s, _ = r2.W.Snapshot()
	ok, why := r2.T.NeedsTrip(s)
	if !ok || why != "low on mana potions" {
		t.Fatalf("%v %q", ok, why)
	}
	if _, err := r2.T.Chores(context.Background(), town.FullPlan()); err != nil {
		t.Fatal(err)
	}
	s, _ = r2.W.Snapshot()
	if ok, why := r2.T.NeedsTrip(s); ok {
		t.Fatalf("still asking after a futile visit: %q", why)
	}
	r2.W.Player.Gold = 500
	s, _ = r2.W.Snapshot()
	if ok, why := r2.T.NeedsTrip(s); ok {
		t.Fatalf("asked again inside the quiet time: %q", why)
	}
	r2.Env.Clock.Sleep(11 * time.Minute)
	if ok, _ := r2.T.NeedsTrip(s); !ok {
		t.Error("gold grew and the quiet time passed but the advice stayed silent")
	}
	// A custom plan without mana columns silences it too.
	r3 := newRig(t)
	r3.T.SetPlan(func(c d2.Class, l int) belt.Band { b := belt.PlanFor(l); b.Cols[2] = belt.HP; return b })
	r3.W.Add("tsc", d2.ContInventory, 9, 3)
	for i := 0; i < 4; i++ {
		r3.W.Add("hp1", d2.ContBelt, i, 0)
	}
	s, _ = r3.W.Snapshot()
	if ok, why := r3.T.NeedsTrip(s); ok {
		t.Fatalf("custom plan: %q", why)
	}
}

func TestPartialGoldDeposit(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Gold = 5000 // level 1: keeps 300 + 50 = 350, deposits 4650 by typing
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Gold: true})
	if err != nil || rep.Errors != 0 || r.W.Player.StashGold != 4650 || r.W.Player.Gold != 350 {
		t.Fatalf("%v %+v gold %d bank %d", err, rep.Steps, r.W.Player.Gold, r.W.Player.StashGold)
	}
}

// LIVE G1 attempt 12: 1125 gold at clvl 11 was all lost on death; that much must now be banked above the keep line.
func TestGoldBankedAtClvl11(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Level = 11
	r.W.Player.Gold = 1125 // keep 300 + 11*50 = 850
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Gold: true})
	if err != nil || rep.Errors != 0 || r.W.Player.StashGold != 275 || r.W.Player.Gold != 850 {
		t.Fatalf("%v %+v gold %d bank %d", err, rep.Steps, r.W.Player.Gold, r.W.Player.StashGold)
	}
}
