package combat

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
)

// upkeepBuild is one hero of the two-build tests: a class whose default profile keeps one self buff up, and the
// Skills.txt row of that buff as internal/data resolves it (StateID from aurastate). The ids and state numbers are
// fixtures of the test, not facts the driver knows: it reads them from the row.
type upkeepBuild struct {
	class d2.Class
	row   *data.Skill
}

func upkeepBuilds() []upkeepBuild {
	hs := &data.Skill{ID: 117, Key: "Holy Shield", Name: "Holy Shield", Mana: 35, ManaShift: 8, MinMana: 35, StateID: 101}
	fa := frozenArmorRow(true)
	fa.StateID = d2.StateFrozenArmor
	return []upkeepBuild{{d2.Paladin, hs}, {d2.Sorceress, fa}}
}

// upkeepSetup is a hero of the build in the necro test world, with the buff on f3 and knob combat.deliveryUpkeep at mode.
func upkeepSetup(t *testing.T, b upkeepBuild, mode int) (*harness, *nworld) {
	t.Helper()
	h, w := newNecro(t)
	cat := necroCat()
	cat.Skills[b.row.ID] = b.row
	h.f.SetCatalog(cat)
	h.f.e.Tune.Combat.DeliveryUpkeep = mode
	p := w.snap.Player
	p.Class, p.Level, p.Mana, p.MaxMana, p.RightSkill = b.class, 30, 400, 400, b.row.ID
	p.Skills = []d2.Skill{{ID: b.row.ID, Level: 1}}
	h.f.SetHotkeys(map[int]string{b.row.ID: "f3"})
	return h, w
}

// editProfile changes the loaded profile in place for one test and restores it after (no combat test runs in parallel).
func editProfile(t *testing.T, p *Profile, edit func(*Profile)) {
	t.Helper()
	saved := *p
	t.Cleanup(func() { *p = saved })
	edit(p)
}

func setState(p *d2.Player, st int, on bool) {
	if on {
		p.States[st/32] |= 1 << (st % 32)
	} else {
		p.States[st/32] &^= 1 << (st % 32)
	}
}

// The driver reads the state id from the table row: up = not due, gone after being seen = recast after the grace.
func TestUpkeepStateFromTableTwoBuilds(t *testing.T) {
	for _, b := range upkeepBuilds() {
		h, w := upkeepSetup(t, b, 2)
		buffNow(t, h)
		if len(w.casts) != 1 {
			t.Fatalf("%v: opening casts %d, want 1", b.class, len(w.casts))
		}
		setState(w.snap.Player, b.row.StateID, true)
		h.s.Sleep(10 * time.Second)
		buffNow(t, h)
		if len(w.casts) != 1 {
			t.Fatalf("%v: state up, casts %d, want 1", b.class, len(w.casts))
		}
		setState(w.snap.Player, b.row.StateID, false)
		buffNow(t, h)
		if len(w.casts) != 2 {
			t.Fatalf("%v: learned state gone: casts %d, want 2", b.class, len(w.casts))
		}
	}
}

// A wrong state id (never seen present) leaves the clock in charge: no recast spam (the card's named risk).
func TestUpkeepWrongStateFallsBackToClock(t *testing.T) {
	for _, b := range upkeepBuilds() {
		h, w := upkeepSetup(t, b, 2)
		b.row.StateID = 200 // never set on the player
		for range 60 {
			buffNow(t, h)
			h.s.Sleep(time.Second)
		}
		if len(w.casts) != 1 {
			t.Errorf("%v: %d casts in 60 s with an unseen state, want 1", b.class, len(w.casts))
		}
	}
}

// Inside a fight a non-vital upkeep casts at most once in ShareEvery casts; a vital slot ignores the share.
func TestUpkeepShareInFight(t *testing.T) {
	for _, b := range upkeepBuilds() {
		for _, vital := range []bool{false, true} {
			h, w := upkeepSetup(t, b, 2)
			if vital {
				editProfile(t, h.f.profileFor(w.snap.Player), func(p *Profile) { p.Upkeep = &UpkeepParams{Vital: []string{b.row.Name}} })
			}
			h.f.upk.fightStart()
			buffNow(t, h) // upkeep cast 1 of fight cast 1
			setState(w.snap.Player, b.row.StateID, true)
			buffNow(t, h) // learned
			setState(w.snap.Player, b.row.StateID, false)
			h.s.Sleep(5 * time.Second)
			buffNow(t, h)
			want := 1
			if vital {
				want = 2
			}
			if len(w.casts) != want {
				t.Fatalf("%v vital=%v: casts %d before the share, want %d", b.class, vital, len(w.casts), want)
			}
			if vital {
				continue
			}
			h.f.upk.fightCasts += 3 // three attacks
			buffNow(t, h)
			if len(w.casts) != 2 {
				t.Fatalf("%v: casts %d after three attacks, want 2", b.class, len(w.casts))
			}
		}
	}
}

// Inside a fight the cast must leave the reserve (10 percent of the pool by default); outside a fight it need not.
func TestUpkeepManaReserve(t *testing.T) {
	for _, b := range upkeepBuilds() {
		for _, inFight := range []bool{true, false} {
			h, w := upkeepSetup(t, b, 2)
			cost := KnownSkills(w.snap.Player, h.f.cat, h.f.keys)[b.row.Name].Cost
			w.snap.Player.Mana = cost + 39 // 39 left, reserve 40 of 400
			if inFight {
				h.f.upk.fightStart()
			}
			buffNow(t, h)
			want := 1
			if inFight {
				want = 0
			}
			if len(w.casts) != want {
				t.Errorf("%v inFight=%v: casts %d, want %d", b.class, inFight, len(w.casts), want)
			}
		}
	}
}

// A buff slot whose table row says curse is never cast at the own feet (mode 2), with no name list; the Necromancer
// keeps Bone Armor up beside it.
func TestUpkeepCurseIsNotUpkeep(t *testing.T) {
	h, w := newNecro(t)
	cat := necroCat()
	cat.Skills[idAmplify].Curse = true
	cat.Skills[idBoneArmor].StateID = stateBoneArmor
	h.f.SetCatalog(cat)
	h.f.e.Tune.Combat.DeliveryUpkeep = 2
	editProfile(t, h.f.profileFor(w.snap.Player), func(p *Profile) { p.Buffs = [][]string{{"Amplify Damage"}, {"Bone Armor"}} })
	w.snap.Player.Mana, w.snap.Player.MaxMana = 200, 200
	buffNow(t, h)
	buffNow(t, h)
	if len(w.casts) != 1 || w.casts[0] != "f3" {
		t.Fatalf("casts %v, want only Bone Armor (f3)", w.casts)
	}
}

// Mode 1: the old pass acts; a lasting disagreement is logged once. Mode 0 never consults the driver.
func TestUpkeepShadow(t *testing.T) {
	b := upkeepBuilds()[0] // Holy Shield: the old pass has no state for it, the driver has
	for _, mode := range []int{0, 1} {
		h, w := upkeepSetup(t, b, mode)
		buffNow(t, h)
		setState(w.snap.Player, b.row.StateID, true)
		buffNow(t, h) // the driver learns the state
		setState(w.snap.Player, b.row.StateID, false)
		h.s.Sleep(5 * time.Second)
		for range 5 {
			buffNow(t, h)
		}
		if len(w.casts) != 1 {
			t.Fatalf("mode %d: casts %d, want 1 (the old clock acts)", mode, len(w.casts))
		}
		want := 0
		if mode == 1 {
			want = 1
		}
		if got := h.f.UpkeepShadowDiffs(); got != want {
			t.Errorf("mode %d: shadow diffs %d, want %d", mode, got, want)
		}
	}
}

func TestUpkeepParamsDefaults(t *testing.T) {
	if u := (*Profile)(nil).upkeepParams(); u.ShareEvery != 4 || u.ManaReservePct != 10 {
		t.Fatalf("defaults %+v", u)
	}
	p := &Profile{Upkeep: &UpkeepParams{ShareEvery: 2, ManaReservePct: 25}}
	if u := p.upkeepParams(); u.ShareEvery != 2 || u.ManaReservePct != 25 {
		t.Fatalf("profile %+v", u)
	}
	if env.DefaultTuning().Combat.DeliveryUpkeep != 0 {
		t.Fatal("combat.deliveryUpkeep must ship at 0 (dark launch)")
	}
}
