package combat

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

func deadMon(id uint32, class, x, y int) d2.Unit {
	u := mon(id, class, x, y)
	u.Mode = d2.MonModeDead
	return u
}

// The shipped Necromancer rows are the old pick: Revive the highest level, Corpse Explosion the pack.
func TestCorpseRowsShippedForNecromancer(t *testing.T) {
	h, w := reviveWorld(t, map[int]int{idRevive: 3}, map[int]string{idRevive: "f6"})
	p := w.snap.Player
	for _, name := range []string{"Corpse Explosion", "Revive"} {
		if h.f.corpseRow(p, name) == nil {
			t.Fatalf("no corpse row for %s", name)
		}
	}
	weak, big := deadMon(20, 1, 102, 100), deadMon(21, 2, 103, 100)
	h.f.e.Tune.Combat.DeliveryUpkeep = 2
	c, ok := h.f.corpseFor(p, "Revive", corpseRowNearest, []d2.Unit{weak, big}, nil)
	if !ok || c.ID != 21 {
		t.Fatalf("row corpseBest: %v %v", c.ID, ok)
	}
}

// Modes: 0 the caller's default row, 1 the default and a shadow count, 2 the profile's row (and a refusal).
func TestCorpseAimModes(t *testing.T) {
	h, w := reviveWorld(t, map[int]int{idRevive: 3}, map[int]string{idRevive: "f6"})
	p := w.snap.Player
	row := h.f.corpseRow(p, "Revive")
	old := row.Rule
	t.Cleanup(func() { row.Rule = old })
	row.Rule = AimCorpseNearest // the data now disagrees with the default
	cs := []d2.Unit{deadMon(20, 1, 102, 100), deadMon(21, 2, 103, 100)}
	for mode, want := range map[int]uint32{0: 21, 1: 21, 2: 20} {
		h.f.e.Tune.Combat.DeliveryUpkeep = mode
		if c, ok := h.f.corpseFor(p, "Revive", corpseRowBest, cs, nil); !ok || c.ID != want {
			t.Fatalf("mode %d: corpse %d, want %d", mode, c.ID, want)
		}
	}
	if n := h.f.CorpseCounts(); n.ShadowMoved != 1 || n.Picked != 1 {
		t.Fatalf("counts %+v", n)
	}
	// a pack rule that no corpse meets refuses at mode 2 only
	row.Rule, row.MinHit = AimCorpseNearPack, 3
	t.Cleanup(func() { row.MinHit = 0 })
	h.f.e.Tune.Combat.DeliveryUpkeep = 2
	if _, ok := h.f.corpseFor(p, "Revive", corpseRowBest, cs, nil); ok {
		t.Fatal("mode 2 did not refuse")
	}
	h.f.e.Tune.Combat.DeliveryUpkeep = 1
	if _, ok := h.f.corpseFor(p, "Revive", corpseRowBest, cs, nil); !ok {
		t.Fatal("shadow must act as the default")
	}
}

// A build with no corpse rows (the Sorceress) always gets the caller's default, whatever the knob.
func TestCorpseAimNoRowKeepsDefault(t *testing.T) {
	h := newHarness(t)
	snap, _ := h.s.snapshot()
	cs := []d2.Unit{deadMon(20, 1, 102, 100), deadMon(21, 1, 103, 100)}
	for mode := 0; mode <= 2; mode++ {
		h.f.e.Tune.Combat.DeliveryUpkeep = mode
		if c, ok := h.f.corpseFor(snap.Player, "Corpse Explosion", corpseRowNearest, cs, nil); !ok || c.ID != 20 {
			t.Fatalf("mode %d: %v %v", mode, c.ID, ok)
		}
	}
}

// One reservation: a corpse taken by any corpse spell is gone for the next (the Death Sentry list reads the same map).
func TestCorpseReservationShared(t *testing.T) {
	h, w := reviveWorld(t, nil, nil)
	w.corpse(20, 102, 100)
	w.corpse(21, 103, 100)
	me := input.Point{X: 100, Y: 100}
	if cs := Corpses(w.snap, h.f.cat, me, 14, h.f.takenCorpses(w.snap.Player.Area)); len(cs) != 2 {
		t.Fatalf("free corpses: %d", len(cs))
	}
	h.f.takeCorpse(20)
	if cs := Corpses(w.snap, h.f.cat, me, 14, h.f.takenCorpses(w.snap.Player.Area)); len(cs) != 1 || cs[0].ID != 21 {
		t.Fatalf("taken corpse still offered: %v", cs)
	}
	if corpseAt(w.snap.Monsters, input.Point{X: 103, Y: 100}) != 21 {
		t.Fatal("corpseAt")
	}
}

func TestValidateCorpseAims(t *testing.T) {
	sk := map[string]skillMeta{"Revive": {Kind: KindGround}}
	ok := CorpseAim{Name: "Revive", Rule: AimCorpseBest, Source: "U test"}
	if err := validateCorpseAims([]Profile{{Class: "necromancer", CorpseAims: []CorpseAim{ok}}}, sk); err != nil {
		t.Fatal(err)
	}
	bad := map[string]CorpseAim{
		"unknown":  {Name: "Revive", Rule: "x", Source: "U"},
		"names no": {Name: "Nope", Rule: AimCorpseBest, Source: "U"},
		"source":   {Name: "Revive", Rule: AimCorpseBest},
		"must be":  {Name: "Revive", Rule: AimCorpseBest, MinHit: -1, Source: "U"},
	}
	for want, r := range bad {
		err := validateCorpseAims([]Profile{{Class: "necromancer", CorpseAims: []CorpseAim{r}}}, sk)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: %v", want, err)
		}
	}
	if err := validateCorpseAims([]Profile{{Class: "necromancer", CorpseAims: []CorpseAim{ok, ok}}}, sk); err == nil {
		t.Fatal("twice")
	}
}

// Card fix3 (review 3 finding 6): the reservation was taken on aim and never released, so one cut cast removed a corpse
// for raise, revive, explode and Death Sentry alike. A corpse still standing after corpseHold is offered again, another
// area drops the set (unit ids repeat), and id 0 (a lay point with no corpse) reserves nothing.
func TestCorpseReservationReleased(t *testing.T) {
	h, w := reviveWorld(t, nil, nil)
	w.corpse(7, 102, 100)
	me := input.Point{X: 100, Y: 100}
	area := w.snap.Player.Area
	free := func(area int) int { return len(Corpses(w.snap, h.f.cat, me, 14, h.f.takenCorpses(area))) }
	if free(area) != 1 {
		t.Fatal("corpse 7 not offered")
	}
	h.f.takeCorpse(7)
	h.s.Sleep(corpseHold / 2)
	if free(area) != 0 {
		t.Fatal("the reservation lapsed inside the hold")
	}
	h.s.Sleep(corpseHold)
	if free(area) != 1 {
		t.Fatal("a cut cast removed corpse 7 for good: it still stands after the hold and is not offered")
	}
	h.f.takeCorpse(7)
	if free(area+1) != 1 {
		t.Fatal("another area kept the old reservations")
	}
	h.f.takeCorpse(0)
	if len(h.f.takenCorpses(area+1)) != 0 {
		t.Fatalf("id 0 reserved: %v", h.f.necro.tried)
	}
}
