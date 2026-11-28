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
	idWhirl    = 151
	idZeal     = 123
)

// moveWorld is a character of class with the move skills the test names, bound to f2, standing at (100,100).
func moveWorld(t *testing.T, class d2.Class, build string, mode int, ids ...int) (*harness, *nworld) {
	cat := classCat()
	for _, s := range []struct {
		id   int
		name string
	}{{idTeleport, "Teleport"}, {idCharge, "Charge"}, {idLeap, "Leap Attack"}, {idWhirl, "Whirlwind"}, {idZeal, "Zeal"}} {
		cat.Skills[s.id] = &data.Skill{ID: s.id, Key: s.name, Name: s.name, Mana: 24, ManaShift: 8, MinMana: 24}
	}
	skills, keys := map[int]int{}, map[int]string{}
	for _, id := range ids {
		skills[id], keys[id] = 5, "f2"
	}
	h, w := newClassWorld(t, class, skills, keys, nil)
	h.f.SetCatalog(cat)
	h.f.e.In = &moveIn{Controller: h.f.e.In, w: w}
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
	if len(ProfileFor(d2.Necromancer, "").Moves) != 0 { // a class whose profile has no move row
		t.Fatal("necromancer default profile has a move row")
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

// Card dlv-move-melee: a Barbarian's Leap Attack (through the target) and a Paladin's Charge (at the target) close a
// gap at 2, are logged at 1 and stay silent at 0; a target inside the gap or beyond the range gets no cast; the
// Barbarian leaps out of a surround. Row added to combat.teleport-reposition for both classes.
func TestGapClose(t *testing.T) {
	for _, c := range []struct {
		class   d2.Class
		skill   int
		through bool
	}{{d2.Barbarian, idLeap, true}, {d2.Paladin, idCharge, false}} {
		for _, mode := range []int{0, 1, 2} {
			h, w := moveWorld(t, c.class, "", mode, c.skill)
			if pr := h.f.profileFor(w.snap.Player); len(pr.Moves) == 0 {
				t.Fatalf("profile %s has no move row", pr.ProfileID())
			}
			tg := Target{Unit: mon(300, 1, 110, 100), Dist: 10, Name: "Fallen"}
			aimed, err := h.f.gapClose(w.snap, tg, nil)
			if err != nil {
				t.Fatal(err)
			}
			if aimed != (mode == 2) || (len(w.casts) > 0) != (mode == 2) {
				t.Fatalf("%v mode %d: aimed %v casts %v", c.class, mode, aimed, w.casts)
			}
			if mode == 2 {
				want := 110
				if c.through {
					want = 111
				}
				if w.castAt[0].X != want || h.f.MoveGaps() != 1 {
					t.Fatalf("%v cast at %v, gaps %d", c.class, w.castAt[0], h.f.MoveGaps())
				}
			}
			if mode == 1 && h.f.MoveShadowCasts() != 1 {
				t.Fatalf("%v shadow %d", c.class, h.f.MoveShadowCasts())
			}
		}
		h, w := moveWorld(t, c.class, "", 2, c.skill)
		for _, d := range []float64{3, 40} { // inside minGap, beyond the range
			if aimed, _ := h.f.gapClose(w.snap, Target{Unit: mon(300, 1, 100+int(d), 100), Dist: d}, nil); aimed || len(w.casts) > 0 {
				t.Fatalf("%v cast at distance %v", c.class, d)
			}
		}
		h, w = moveWorld(t, c.class, "", 2) // skill not learned: refused, counted
		if aimed, _ := h.f.gapClose(w.snap, Target{Unit: mon(300, 1, 110, 100), Dist: 10}, nil); aimed {
			t.Fatalf("%v cast an unknown skill", c.class)
		}
	}
	h, w := moveWorld(t, d2.Barbarian, "", 2, idLeap)
	surround(w, 5)
	w.snap.Player.Life = 150
	if acted, _ := h.f.moveTick(w.snap, newFightState()); !acted || len(w.casts) != 1 {
		t.Fatalf("barbarian did not leap out of the surround: %v", w.casts)
	}
}

// moveIn is the d2d-like backend of the move tests: the left hand casts by id (Swing), recorded as a cast.
type moveIn struct {
	input.Controller
	w *nworld
}

func (m *moveIn) Swing(skill int, at input.Point) error {
	m.w.casts, m.w.castAt = append(m.w.casts, "swing"), append(m.w.castAt, at)
	return nil
}

func (m *moveIn) CastAt(key string, at input.Point) error { return m.w.CastAt(key, at) }

// mouthAt is a mover whose cave-mouth guard covers the points within r subtiles of c.
type mouthAt struct {
	Mover
	c input.Point
	r float64
}

func (m *mouthAt) NearMouth(p input.Point) bool { return pdist(p, m.c) <= m.r }

// Fable review 3 finding 5: a gap landing (Leap Attack, Charge) is a movement and must not end in a cave mouth's
// trigger area; a gap cast and a whirl leg count as casts of the fight. Two classes for the gap, one whirl build.
func TestGapAndWhirlAskTheMouthGateAndCountAsCasts(t *testing.T) {
	for _, c := range []struct {
		class d2.Class
		skill int
	}{{d2.Barbarian, idLeap}, {d2.Paladin, idCharge}} {
		h, w := moveWorld(t, c.class, "", 2, c.skill)
		h.f.mv = &mouthAt{Mover: h.f.mv, c: input.Point{X: 110, Y: 100}, r: 3}
		tg := Target{Unit: mon(300, 1, 110, 100), Dist: 10, Name: "Fallen"}
		st := newFightState()
		if aimed, err := h.f.gapClose(w.snap, tg, st); aimed || err != nil || len(w.casts) > 0 {
			t.Fatalf("%v: gap cast into a cave mouth: aimed %v err %v casts %v", c.class, aimed, err, w.casts)
		}
		h.f.mv = &mouthAt{Mover: h.f.mv, c: input.Point{X: 300, Y: 300}, r: 3}
		if aimed, err := h.f.gapClose(w.snap, tg, st); !aimed || err != nil || st.casts != 1 {
			t.Fatalf("%v: gap away from the mouth: aimed %v err %v fight casts %d, want one counted cast", c.class, aimed, err, st.casts)
		}
	}
	h, w := moveWorld(t, d2.Barbarian, "", 2, idWhirl)
	packAt(w, 107, 100, 5)
	st := newFightState()
	if acted, err := h.f.whirlTick(w.snap, st); !acted || err != nil || st.casts != 1 {
		t.Fatalf("whirl leg: acted %v err %v fight casts %d, want one counted cast", acted, err, st.casts)
	}
}
