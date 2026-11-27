package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

const (
	idTeleport = 54
	idCharge   = 107
	idLeap     = 150
)

// moveWorld is a character of class with the move skills the test names, bound to f2, standing at (100,100).
func moveWorld(t *testing.T, class d2.Class, build string, mode int, ids ...int) (*harness, *nworld) {
	cat := classCat()
	for _, s := range []struct {
		id   int
		name string
	}{{idTeleport, "Teleport"}, {idCharge, "Charge"}, {idLeap, "Leap Attack"}} {
		cat.Skills[s.id] = &data.Skill{ID: s.id, Key: s.name, Name: s.name, Mana: 24, ManaShift: 8, MinMana: 24}
	}
	skills, keys := map[int]int{}, map[int]string{}
	for _, id := range ids {
		skills[id], keys[id] = 5, "f2"
	}
	h, w := newClassWorld(t, class, skills, keys, nil)
	h.f.SetCatalog(cat)
	h.f.Build = Build(build)
	h.f.e.Tune.Combat.DeliveryUpkeep = mode
	w.snap.Player.X, w.snap.Player.Y = 100, 100
	w.snap.Player.MaxLife, w.snap.Player.Life = 400, 400
	return h, w
}

func surround(w *nworld, n int) {
	for i := 0; i < n; i++ {
		w.snap.Monsters = append(w.snap.Monsters, mon(uint32(500+i), 1, 101+i%2, 99+i%3))
	}
}

// HopInfo is what navigation asks: key, range, cost from the row and the skill state; a refusal names its reason.
func TestHopInfo(t *testing.T) {
	h, w := moveWorld(t, d2.Sorceress, "fire", 2, idTeleport)
	hi := h.f.HopInfo(w.snap.Player)
	if hi.Refused != "" || hi.Key != "f2" || hi.Range != 18 || hi.MinHop != 4 || hi.Cost != 24 {
		t.Fatalf("hop info %+v", hi)
	}
	if nh := h.f.NavHop(w.snap.Player); nh.Key != "f2" || nh.Range != 18 || nh.Cost != 24 || nh.Refused != "" {
		t.Fatalf("nav hop %+v", nh)
	}
	w.snap.Player.Mana = 5
	if hi := h.f.HopInfo(w.snap.Player); hi.Refused != MoveNoMana || hi.Key != "" {
		t.Fatalf("no mana: %+v", hi)
	}
	h, w = moveWorld(t, d2.Sorceress, "fire", 2) // Teleport not learned
	if hi := h.f.HopInfo(w.snap.Player); hi.Refused != MoveUnknown {
		t.Fatalf("unknown: %+v", hi)
	}
	if len(ProfileFor(d2.Paladin, "").Moves) != 0 { // a class whose profile has no move row
		t.Fatal("paladin default profile has a move row")
	}
}

// Scenario combat.teleport-reposition (offline part): out of a surround the Sorceress hops to a cell away from the
// pack at 2, only logs at 1, nothing at 0; a Paladin and a character with healthy life in the same crowd do nothing.
func TestEscapeHop(t *testing.T) {
	for _, mode := range []int{0, 1, 2} {
		h, w := moveWorld(t, d2.Sorceress, "fire", mode, idTeleport)
		surround(w, 5)
		w.snap.Player.Life = 200
		acted, err := h.f.moveTick(w.snap, newFightState())
		if err != nil {
			t.Fatal(err)
		}
		if acted != (mode == 2) || (len(w.casts) > 0) != (mode == 2) {
			t.Fatalf("mode %d: acted %v casts %v", mode, acted, w.casts)
		}
		if mode == 2 {
			at := w.castAt[0]
			d := pdist(input.Point{X: 100, Y: 100}, at)
			if d < 4 || d > 18 || pdist(at, input.Point{X: 101, Y: 100}) < 4 {
				t.Fatalf("hop to %v (%v away)", at, d)
			}
			if h.f.MoveHops() != 1 {
				t.Fatalf("hops %d", h.f.MoveHops())
			}
		}
		if mode == 1 && h.f.MoveShadowCasts() != 1 {
			t.Fatalf("shadow %d", h.f.MoveShadowCasts())
		}
	}
	h, w := moveWorld(t, d2.Sorceress, "fire", 2, idTeleport)
	surround(w, 5) // full life: no escape
	if acted, _ := h.f.moveTick(w.snap, newFightState()); acted {
		t.Fatal("hopped at full life")
	}
	h, w = moveWorld(t, d2.Sorceress, "fire", 2, idTeleport)
	surround(w, 5)
	w.snap.Player.Life, w.snap.Player.Mana = 100, 3
	if acted, _ := h.f.moveTick(w.snap, newFightState()); acted {
		t.Fatal("hopped with no mana")
	}
}

func TestValidateMoves(t *testing.T) {
	ok := MoveRule{Name: "Teleport", Uses: []string{MoveTravel, MoveEscape}, Range: 18, MinHop: 4, EscapeSurround: 4, EscapeLifePct: 65, Source: "U test"}
	if err := validateMoves([]Profile{{Class: "x", Moves: []MoveRule{ok}}}); err != nil {
		t.Fatal(err)
	}
	for _, mod := range []func(*MoveRule){
		func(r *MoveRule) { r.Name = "" },
		func(r *MoveRule) { r.Uses = nil },
		func(r *MoveRule) { r.Uses = []string{"fly"} },
		func(r *MoveRule) { r.Range = 1 },
		func(r *MoveRule) { r.MinHop = 18 },
		func(r *MoveRule) { r.EscapeLifePct = 0 },
		func(r *MoveRule) { r.Uses = []string{MoveGap} },
		func(r *MoveRule) { r.Source = "" },
	} {
		r := ok
		mod(&r)
		if validateMoves([]Profile{{Class: "x", Moves: []MoveRule{r}}}) == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
}
