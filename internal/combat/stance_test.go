package combat

import (
	"context"
	"testing"

	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// minionAt is one standing minion of the hero (the Necromancer's child state).
func minionAt(id uint32, class, x, y int) d2.Unit {
	u := mon(id, class, x, y)
	u.Monster.States = []uint8{d2.StateMinion}
	return u
}

// summonerWorld is the Necromancer summoner build (profile data, f.Build) with an army of n skeletons and a golem
// around (104,100), the hero at (106,100) and a fallen pack at (112,100): the hero stands in front of its army.
func summonerWorld(t *testing.T, mode, skeletons int) (*harness, *nworld) {
	h, w := newNecro(t)
	h.f.SetCatalog(withSummonKeys(necroCat()))
	h.f.Build = "summoner"
	h.f.e.Tune.Combat.DeliveryUpkeep = mode
	w.snap.Player.X, w.snap.Player.Y = 106, 100
	w.snap.Monsters = append(w.snap.Monsters, minionAt(200, clsGolem, 104, 101))
	for i := 0; i < skeletons; i++ {
		w.snap.Monsters = append(w.snap.Monsters, minionAt(uint32(201+i), clsSkel, 104, 99+i%3))
	}
	w.snap.Monsters = append(w.snap.Monsters, mon(300, 1, 112, 100), mon(301, 1, 112, 101))
	return h, w
}

func castJustNow(h *harness) *fightState {
	st := newFightState()
	st.lastCastAt = h.f.now()
	return st
}

func TestBehindPoint(t *testing.T) {
	me := input.Point{X: 106, Y: 100}
	if got := behindPoint(me, input.Point{X: 104, Y: 100}, input.Point{X: 112, Y: 100}, 4); got != (input.Point{X: 100, Y: 100}) {
		t.Fatalf("behind the army: %v", got)
	}
	// army standing on the threat: the hero keeps its own side
	if got := behindPoint(me, input.Point{X: 112, Y: 100}, input.Point{X: 112, Y: 100}, 4); got.X >= 112 {
		t.Fatalf("army on the threat: %v should stay on the hero's side", got)
	}
}

// Army facts for the arbiter: the summoner reports count, want, strength, front and upkeep due from the census.
func TestArmyFactsSummoner(t *testing.T) {
	h, w := summonerWorld(t, 2, 2)
	w.corpse(50, 105, 100)
	f := h.f.ArmyFacts(w.snap)
	if f[arbiter.FactArmy] != 3 || f[arbiter.FactArmyStrength] != 300 {
		t.Fatalf("army facts %v", f)
	}
	if f[arbiter.FactArmyWant] != 4 || f[arbiter.FactArmyBelowWant] != 1 || f[arbiter.FactUpkeepDue] != 1 {
		t.Fatalf("golem 1 + skeleton level 3 want 4, a corpse at the feet: %v", f)
	}
	if f[arbiter.FactArmyFront] != 2 { // front: the minion nearest the pack, (104,100) from the hero at (106,100)
		t.Fatalf("front %d", f[arbiter.FactArmyFront])
	}
	// at 0 the facts stay dark
	h.f.e.Tune.Combat.DeliveryUpkeep = 0
	if f := h.f.ArmyFacts(w.snap); f != nil {
		t.Fatalf("knob 0 reports facts %v", f)
	}
}

// The stance steps the hero behind its army at 2, only logs at 1, does nothing at 0, and never replaces a cast.
func TestStanceBehindArmyModes(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []int{0, 1, 2} {
		h, w := summonerWorld(t, mode, 3)
		acted, err := h.f.stanceTick(ctx, w.snap, castJustNow(h))
		if err != nil {
			t.Fatal(err)
		}
		moved := len(w.walkedTo) > 0
		if acted != (mode == 2) || moved != (mode == 2) {
			t.Fatalf("mode %d: acted %v walked %v", mode, acted, w.walkedTo)
		}
		if mode == 2 && w.walkedTo[0].X >= 104 {
			t.Fatalf("stepped to %v, not behind the army at x 104", w.walkedTo[0])
		}
		if mode == 1 && h.f.StanceShadowSteps() != 1 {
			t.Fatalf("shadow steps %d", h.f.StanceShadowSteps())
		}
	}
	// no cast lately: the step may not take the tick (cast-first)
	h, w := summonerWorld(t, 2, 3)
	if acted, _ := h.f.stanceTick(ctx, w.snap, newFightState()); acted || len(w.walkedTo) > 0 {
		t.Fatal("stance step replaced a cast")
	}
	// already behind the army: hold
	h, w = summonerWorld(t, 2, 3)
	w.snap.Player.X = 100
	if acted, _ := h.f.stanceTick(ctx, w.snap, castJustNow(h)); acted {
		t.Fatalf("stepped while on its spot: %v", w.walkedTo)
	}
}

// Scenario combat.army-holds, boss row: in a boss fight a census under the stance minimum turns the stance into a
// retreat from the boss; at or above it the hero keeps behind its army.
func TestScenarioArmyHoldsBoss(t *testing.T) {
	ctx := context.Background()
	boss := func(w *nworld) {
		b := mon(400, 6, 112, 100)
		b.Monster.Name = "Andariel"
		w.snap.Monsters = append(w.snap.Monsters, b)
	}
	h, w := summonerWorld(t, 2, 0) // golem only: one minion, under bossMin 2
	boss(w)
	if act, _, why := h.f.stancePlan(w.snap); act != "retreat" {
		t.Fatalf("army of one before a boss: act %q %s", act, why)
	}
	if acted, _ := h.f.stanceTick(ctx, w.snap, castJustNow(h)); !acted || len(w.walkedTo) != 1 {
		t.Fatalf("no retreat step: %v", w.walkedTo)
	}
	bp := input.Point{X: 112, Y: 100}
	if pdist(w.walkedTo[0], bp) <= pdist(input.Point{X: 106, Y: 100}, bp) {
		t.Fatalf("retreat to %v does not leave the boss", w.walkedTo[0])
	}
	h, w = summonerWorld(t, 2, 3)
	boss(w)
	if act, _, why := h.f.stancePlan(w.snap); act != "behind" {
		t.Fatalf("full army before a boss: act %q %s", act, why)
	}
}

// Negative builds: a class without an army (melee Barbarian) and a Necromancer profile without a stance block take
// no stance step; the Barbarian reports no army facts at all.
func TestStanceNoArmyBuilds(t *testing.T) {
	ctx := context.Background()
	h, w := newClassWorld(t, d2.Barbarian, nil, nil, nil)
	h.f.e.Tune.Combat.DeliveryUpkeep = 2
	w.snap.Monsters = append(w.snap.Monsters, mon(300, 1, 112, 100), mon(301, 1, 112, 101))
	if pr := h.f.profileFor(w.snap.Player); pr.Class != "barbarian" {
		t.Fatalf("profile %s", pr.ProfileID())
	}
	if f := h.f.ArmyFacts(w.snap); f != nil {
		t.Fatalf("barbarian army facts %v", f)
	}
	if acted, _ := h.f.stanceTick(ctx, w.snap, castJustNow(h)); acted || len(w.walkedTo) > 0 {
		t.Fatal("barbarian took a stance step")
	}
	h, w = summonerWorld(t, 2, 3)
	h.f.Build = ""
	if acted, _ := h.f.stanceTick(ctx, w.snap, castJustNow(h)); acted {
		t.Fatal("necromancer profile without a stance block stepped")
	}
	if f := h.f.ArmyFacts(w.snap); f[arbiter.FactArmy] != 4 {
		t.Fatalf("necromancer default profile army facts %v", f)
	}
}

func TestValidateStance(t *testing.T) {
	ok := Profile{Class: "necromancer", Necro: &NecroParams{Golems: []string{"Clay Golem"}},
		Stance: &ArmyStance{Kind: StanceBehindArmy, Behind: 4, Source: "U test"}}
	if err := validateStance([]Profile{ok}); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Profile{
		{Class: "necromancer", Necro: ok.Necro, Stance: &ArmyStance{Kind: "inFront", Behind: 4, Source: "U"}},
		{Class: "necromancer", Necro: ok.Necro, Stance: &ArmyStance{Kind: StanceBehindArmy, Behind: 0, Source: "U"}},
		{Class: "necromancer", Necro: ok.Necro, Stance: &ArmyStance{Kind: StanceBehindArmy, Behind: 4}},
		{Class: "barbarian", Stance: &ArmyStance{Kind: StanceBehindArmy, Behind: 4, Source: "U"}},
	} {
		if validateStance([]Profile{bad}) == nil {
			t.Fatalf("accepted %+v", bad.Stance)
		}
	}
}

// packWorld is a hero at (100,100) with n hostiles around (112,100) on a class and build whose profile has an
// insidePack stance.
func packWorld(t *testing.T, class d2.Class, build string, mode, n int) (*harness, *nworld) {
	h, w := newClassWorld(t, class, nil, nil, nil)
	h.f.Build = Build(build)
	h.f.e.Tune.Combat.DeliveryUpkeep = mode
	w.snap.Player.X, w.snap.Player.Y = 100, 100
	for i := 0; i < n; i++ {
		w.snap.Monsters = append(w.snap.Monsters, mon(uint32(300+i), 1, 112, 98+i))
	}
	return h, w
}

// Scenario combat.hammer-in-pack (offline part): a Hammerdin and a Poison Nova Necromancer walk to the pack's edge at
// 2, only log at 1, stay put at 0, hold on the spot, and back out of a surround or at low life. Hits per cast, mana and
// the aura uptime of the scenario need the live look H1 and the survival model (R6).
func TestStanceInsidePack(t *testing.T) {
	ctx := context.Background()
	for _, b := range []struct {
		class d2.Class
		build string
		off   int
	}{{d2.Paladin, "hammerdin", 2}, {d2.Necromancer, "poisonnova", 3}} {
		for _, mode := range []int{0, 1, 2} {
			h, w := packWorld(t, b.class, b.build, mode, 4)
			acted, err := h.f.stanceTick(ctx, w.snap, castJustNow(h))
			if err != nil {
				t.Fatal(err)
			}
			if acted != (mode == 2) || (len(w.walkedTo) > 0) != (mode == 2) {
				t.Fatalf("%s mode %d: acted %v walked %v", b.build, mode, acted, w.walkedTo)
			}
			if mode == 2 {
				if d := pdist(w.walkedTo[0], input.Point{X: 112, Y: 100}); d < float64(b.off)-1 || d > float64(b.off)+1 {
					t.Fatalf("%s stepped to %v, %v from the pack", b.build, w.walkedTo[0], d)
				}
			}
			if mode == 1 && h.f.StanceShadowSteps() != 1 {
				t.Fatalf("%s shadow steps %d", b.build, h.f.StanceShadowSteps())
			}
		}
		// in the pack's edge: cast in place
		h, w := packWorld(t, b.class, b.build, 2, 4)
		w.snap.Player.X = 112 - b.off
		if act, _, why := h.f.stancePlan(w.snap); act != "" {
			t.Fatalf("%s on its spot: %s %s", b.build, act, why)
		}
		// a lone hostile is no pack
		h, w = packWorld(t, b.class, b.build, 2, 1)
		if act, _, _ := h.f.stancePlan(w.snap); act != "" {
			t.Fatalf("%s one hostile: %s", b.build, act)
		}
		// surrounded: leave, away from the pack
		h, w = packWorld(t, b.class, b.build, 2, 0)
		for i := 0; i < 5; i++ {
			w.snap.Monsters = append(w.snap.Monsters, mon(uint32(400+i), 1, 101+i%2, 99+i%3))
		}
		act, to, _ := h.f.stancePlan(w.snap)
		if act != "leave" || pdist(to, input.Point{X: 101, Y: 100}) <= 1 {
			t.Fatalf("%s surrounded: %s to %v", b.build, act, to)
		}
		// low life with one in contact: leave; full life with one in contact: hold
		h, w = packWorld(t, b.class, b.build, 2, 0)
		w.snap.Monsters = append(w.snap.Monsters, mon(401, 1, 101, 100), mon(402, 1, 108, 100), mon(403, 1, 108, 101))
		w.snap.Player.MaxLife, w.snap.Player.Life = 400, 100
		if act, _, _ := h.f.stancePlan(w.snap); act != "leave" {
			t.Fatalf("%s low life: %q", b.build, act)
		}
		w.snap.Player.Life = 400
		if act, _, _ := h.f.stancePlan(w.snap); act == "leave" {
			t.Fatalf("%s full life left", b.build)
		}
	}
	// no cast lately: the step may not take the tick
	h, w := packWorld(t, d2.Paladin, "hammerdin", 2, 4)
	if acted, _ := h.f.stanceTick(ctx, w.snap, newFightState()); acted || len(w.walkedTo) > 0 {
		t.Fatal("pack step replaced a cast")
	}
	// builds without a stance block stay put
	h, w = packWorld(t, d2.Sorceress, "fire", 2, 4)
	if act, _, _ := h.f.stancePlan(w.snap); act != "" {
		t.Fatalf("fire sorceress: %q", act)
	}
}

func TestValidateStanceInsidePack(t *testing.T) {
	ok := Profile{Class: "paladin", Stance: &ArmyStance{Kind: StanceInsidePack, StandOff: 2, LeaveSurround: 5, LeaveLifePct: 50, Source: "U test"}}
	if err := validateStance([]Profile{ok}); err != nil {
		t.Fatal(err)
	}
	for _, mod := range []func(*ArmyStance){
		func(s *ArmyStance) { s.StandOff = 0 },
		func(s *ArmyStance) { s.LeaveSurround = 0 },
		func(s *ArmyStance) { s.LeaveLifePct = 0 },
		func(s *ArmyStance) { s.LeaveLifePct = 101 },
		func(s *ArmyStance) { s.Source = "" },
	} {
		st := *ok.Stance
		mod(&st)
		if validateStance([]Profile{{Class: "paladin", Stance: &st}}) == nil {
			t.Fatalf("accepted %+v", st)
		}
	}
}
