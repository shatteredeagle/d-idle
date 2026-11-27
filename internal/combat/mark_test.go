package combat

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

// areaWorld applies a mark's state to every standing monster within r of the click, as an area curse does.
type areaWorld struct {
	*nworld
	r float64
}

func (a areaWorld) CastAt(key string, at input.Point) error {
	a.casts, a.castAt = append(a.casts, key), append(a.castAt, at)
	a.snap.Player.Mana -= a.cost
	for i := range a.snap.Monsters {
		u := &a.snap.Monsters[i]
		if a.curseSt != 0 && u.Mode == d2.MonModeStand && dist(u.X, u.Y, at.X, at.Y) <= a.r {
			u.Monster.States = append(u.Monster.States, uint8(a.curseSt))
		}
	}
	return nil
}

func (a areaWorld) clear() {
	for i := range a.snap.Monsters {
		a.snap.Monsters[i].Monster.States = nil
	}
}

// markSetup is the Necromancer test world with the Amplify Damage row resolved to its state (as internal/data does from
// aurastate) and knob combat.deliveryUpkeep at mode.
func markSetup(t *testing.T, mode int) (*harness, areaWorld) {
	t.Helper()
	h, w := newNecro(t)
	cat := necroCat()
	cat.Skills[idAmplify].StateID, cat.Skills[idAmplify].Curse = stateAmplify, true
	h.f.SetCatalog(cat)
	h.f.e.Tune.Combat.DeliveryUpkeep = mode
	a := areaWorld{w, 5}
	h.f.e.In = a
	w.snap.Player.Mana, w.snap.Player.MaxMana = 200, 200
	return h, a
}

func pack(w *nworld) {
	w.snap.Monsters = []d2.Unit{mon(2, 1, 112, 100), mon(3, 1, 113, 101), mon(4, 1, 114, 100), mon(5, 1, 113, 99)}
}

func markNow(t *testing.T, h *harness, w *nworld) bool {
	t.Helper()
	done, err := h.f.markTick(context.Background(), w.snap)
	if err != nil {
		t.Fatal(err)
	}
	return done
}

// Scenario combat.curse-covers-pack (offline form; the scenario runner is not built yet): one cast covers a pack, no
// recast while the state is on the units, one recast when it wears off.
func TestMarkCoversPack(t *testing.T) {
	h, a := markSetup(t, 2)
	a.curseSt = stateAmplify
	pack(a.nworld)
	for range 10 {
		markNow(t, h, a.nworld)
		h.s.Sleep(time.Second)
	}
	if len(a.casts) != 1 || a.casts[0] != "f4" {
		t.Fatalf("casts %v, want one Amplify Damage (f4)", a.casts)
	}
	if !h.f.mark.learned[stateAmplify] {
		t.Fatal("state not learned after the cast")
	}
	a.clear()
	for range 3 {
		markNow(t, h, a.nworld)
	}
	if len(a.casts) != 2 {
		t.Fatalf("casts %v after the curse wore off, want 2", a.casts)
	}
}

// With no readable state the area memory stands in: one cast per pack until the memory runs out.
func TestMarkCoverByMemory(t *testing.T) {
	h, a := markSetup(t, 2)
	pack(a.nworld)
	for range 20 {
		markNow(t, h, a.nworld)
		h.s.Sleep(time.Second)
	}
	if len(a.casts) != 1 {
		t.Fatalf("casts %d in 20 s with the memory, want 1", len(a.casts))
	}
	h.s.Sleep(15 * time.Second)
	markNow(t, h, a.nworld)
	if len(a.casts) != 2 {
		t.Fatalf("casts %d after the memory ran out, want 2", len(a.casts))
	}
}

// Target choice by when: a boss first; a lone trash monster is not worth a cast; one group per target (Weaken is not
// put on a boss that carries Amplify Damage).
func TestMarkTargetAndGroup(t *testing.T) {
	h, a := markSetup(t, 2)
	a.curseSt = stateAmplify
	a.snap.Monsters = []d2.Unit{mon(2, 1, 110, 100)}
	if markNow(t, h, a.nworld) {
		t.Fatal("marked a lone trash monster")
	}
	a.snap.Monsters = []d2.Unit{mon(2, 1, 110, 100), mon(3, 1, 111, 100), mon(6, 6, 118, 104)}
	markNow(t, h, a.nworld)
	if len(a.castAt) != 1 || a.castAt[0] != (input.Point{X: 118, Y: 104}) {
		t.Fatalf("cast at %v, want the boss", a.castAt)
	}
	// Amplify gone from the known skills: the boss still carries it, Weaken (same group) must not replace it.
	a.snap.Player.Skills = []d2.Skill{{ID: idWeaken, Level: 1}, {ID: idAmplify, Level: 1}}
	h.s.Sleep(3 * time.Second)
	markNow(t, h, a.nworld)
	for _, k := range a.casts[1:] {
		if k == "f6" {
			t.Fatalf("Weaken put on a marked boss: %v at %v", a.casts, a.castAt)
		}
	}
}

// Two builds: a Barbarian war cry row (Howl: a skill the generated skills.json carries) is cast at the own feet when a pack stands within its radius and never as a
// buff; a Paladin profile with no marks rows casts nothing (the negative).
func TestMarkTwoBuilds(t *testing.T) {
	const idCry, stCry, idHoly = 900, 77, 117 // fixtures: the driver reads both from the catalog row, not from a name
	h, a := markSetup(t, 2)
	cat := necroCat()
	cat.Skills[idCry] = &data.Skill{ID: idCry, Key: "Howl", Name: "Howl", Mana: 5, ManaShift: 8, MinMana: 5, StateID: stCry}
	h.f.SetCatalog(cat)
	p := a.snap.Player
	p.Class, p.Skills = d2.Barbarian, []d2.Skill{{ID: idCry, Level: 1}}
	h.f.SetHotkeys(map[int]string{idCry: "f2"})
	editProfile(t, h.f.profileFor(p), func(pr *Profile) {
		pr.Marks = []MarkRule{{Name: "Howl", When: "pack", Self: true, Radius: 8, Source: "U test row"}}
		pr.Buffs = [][]string{{"Howl"}}
	})
	a.curseSt, a.r = stCry, 8
	a.snap.Monsters = []d2.Unit{mon(2, 1, 120, 100)}
	if markNow(t, h, a.nworld) {
		t.Fatal("cried at a monster outside the radius")
	}
	a.snap.Monsters = []d2.Unit{mon(2, 1, 104, 100), mon(3, 1, 105, 102)}
	markNow(t, h, a.nworld)
	markNow(t, h, a.nworld)
	if len(a.casts) != 1 || a.castAt[0] != (input.Point{X: p.X, Y: p.Y}) {
		t.Fatalf("barbarian casts %v at %v, want one at the own feet", a.casts, a.castAt)
	}
	buffNow(t, h)
	if len(a.casts) != 1 {
		t.Fatalf("mark row cast as a buff: %v", a.casts)
	}

	// The Paladin has Amplify Damage bound too (a charge, say): the rows are profile data, so it is never cast.
	h2, a2 := markSetup(t, 2)
	cat2 := necroCat()
	cat2.Skills[idHoly] = &data.Skill{ID: idHoly, Key: "Holy Shield", Name: "Holy Shield", Mana: 35, ManaShift: 8, MinMana: 35}
	h2.f.SetCatalog(cat2)
	a2.snap.Player.Class = d2.Paladin
	a2.snap.Player.Skills = []d2.Skill{{ID: idHoly, Level: 1}, {ID: idAmplify, Level: 1}}
	pack(a2.nworld)
	if rows := h2.f.profileFor(a2.snap.Player).markRows(); len(rows) != 0 {
		t.Fatalf("paladin profile has mark rows %v", rows)
	}
	if markNow(t, h2, a2.nworld) || len(a2.casts) != 0 {
		t.Fatalf("paladin marked: %v", a2.casts)
	}
}

// Mode 0 never consults the driver; mode 1 logs its choice once and casts nothing (the old curse path acts); mode 2
// stands the old path down.
func TestMarkModes(t *testing.T) {
	for _, mode := range []int{0, 1, 2} {
		h, a := markSetup(t, mode)
		pack(a.nworld)
		for range 3 {
			markNow(t, h, a.nworld)
		}
		if mode < 2 && len(a.casts) != 0 {
			t.Fatalf("mode %d: driver casts %v", mode, a.casts)
		}
		want := 0
		if mode == 1 {
			want = 1
		}
		if got := h.f.MarkShadowChoices(); got != want {
			t.Errorf("mode %d: shadow choices %d, want %d", mode, got, want)
		}
		done, _ := h.f.Curse(context.Background(), a.snap, Hostiles(a.snap, h.f.hostileCtx(a.snap.Player)))
		if done != (mode < 2) {
			t.Errorf("mode %d: old curse path acted=%v", mode, done)
		}
	}
}

// Every Necromancer profile reads its ten curse rules as rows of one group; the default profile set validates.
func TestMarkRowsFromNecroProfile(t *testing.T) {
	rows := ProfileFor(d2.Necromancer, "").markRows()
	if len(rows) != 10 {
		t.Fatalf("%d curse rows, want 10", len(rows))
	}
	for _, r := range rows {
		if r.group() != "curse" {
			t.Fatalf("row %s group %q", r.Name, r.group())
		}
	}
	if err := validateMarks(Profiles()); err != nil {
		t.Fatal(err)
	}
	bad := []struct {
		r    MarkRule
		want string
	}{
		{MarkRule{When: "any"}, "no name"},
		{MarkRule{Name: "x", When: "sometimes"}, "unknown when"},
		{MarkRule{Name: "x", When: "any", Radius: 6}, "source tag"},
		{MarkRule{Name: "x", When: "any", Range: -1, Source: "U"}, "negative"},
	}
	for _, b := range bad {
		err := validateMarks([]Profile{{Class: "barbarian", Build: "t", Marks: []MarkRule{b.r}}})
		if err == nil || !strings.Contains(err.Error(), b.want) {
			t.Errorf("%+v: err %v, want %q", b.r, err, b.want)
		}
	}
}
