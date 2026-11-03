package combat

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/merc"
)

// semSim is the sim with the semantic Belt verb; every other SemanticInput method stays nil (never called here).
type semSim struct {
	*sim
	input.SemanticInput
	belts []beltCall
	err   error
}

type beltCall struct {
	slot int
	merc bool
}

func (s *semSim) Belt(slot int, m bool) error {
	s.belts = append(s.belts, beltCall{slot, m})
	return s.err
}

func mercHarness(t *testing.T, st merc.MercState) (*harness, *bytes.Buffer, *merc.MercState) {
	t.Helper()
	h := newHarness(t)
	var buf bytes.Buffer
	h.f.e.Log = slog.New(slog.NewTextHandler(&buf, nil))
	cur := st
	h.f.MercState = func() (merc.MercState, error) { return cur, nil }
	return h, &buf, &cur
}

func twoMonsters(h *harness) {
	h.s.addMon(10, clsFallen, 106, 100, 500, 0, 0)
	h.s.addMon(11, clsFallen, 107, 101, 500, 0, 0)
}

func TestMercPotionGivenBelowThreshold(t *testing.T) {
	h, buf, st := mercHarness(t, merc.MercState{Known: true, Owned: true, Alive: true, LifePct: 40})
	ss := &semSim{sim: h.s}
	h.f.e.In = ss
	h.s.stock(itHP1, 2, 3)
	twoMonsters(h)
	snap, _ := h.s.snapshot()
	hs := Hostiles(snap, h.f.hostileCtx(snap.Player))
	if !h.f.mercPotion(snap, hs) {
		t.Fatalf("no potion given\n%s", buf)
	}
	if len(ss.belts) != 1 || ss.belts[0] != (beltCall{2, true}) {
		t.Errorf("belt verb calls %v, want column 2 with the merc flag", ss.belts)
	}
	if !strings.Contains(buf.String(), "merc: gave healing potion col=3") {
		t.Errorf("log:\n%s", buf)
	}
	if h.f.mercPotion(snap, hs) || len(ss.belts) != 1 {
		t.Error("a second potion inside the gap")
	}
	h.s.t = h.s.t.Add(mercPotionGap)
	if !h.f.mercPotion(snap, hs) || len(ss.belts) != 2 {
		t.Error("no potion after the gap")
	}
	st.LifePct = 50
	h.s.t = h.s.t.Add(mercPotionGap)
	if h.f.mercPotion(snap, hs) {
		t.Error("given at the threshold itself (rule is below)")
	}
}

func TestMercPotionGuards(t *testing.T) {
	h, _, st := mercHarness(t, merc.MercState{Known: true, Owned: true, Alive: true, LifePct: 40})
	ss := &semSim{sim: h.s}
	h.f.e.In = ss
	h.s.stock(itHP1, 0, 3)
	snap, _ := h.s.snapshot()
	hs := Hostiles(snap, h.f.hostileCtx(snap.Player))
	if h.f.mercPotion(snap, hs) {
		t.Error("trivial fight (no hostiles)")
	}
	twoMonsters(h)
	snap, _ = h.s.snapshot()
	hs = Hostiles(snap, h.f.hostileCtx(snap.Player))
	for name, s := range map[string]merc.MercState{
		"dead":    {Known: true, Owned: true, LifePct: 0},
		"unknown": {LifePct: 40},
		"not own": {Known: true, Alive: true, LifePct: 40},
	} {
		*st = s
		if h.f.mercPotion(snap, hs) {
			t.Errorf("%s merc got a potion", name)
		}
	}
	*st = merc.MercState{Known: true, Owned: true, Alive: true, LifePct: 40}
	h.f.MercState = nil
	if h.f.mercPotion(snap, hs) {
		t.Error("rule on without the wiring")
	}
	h.f.MercState = func() (merc.MercState, error) { return merc.MercState{}, errors.New("x") }
	if h.f.mercPotion(snap, hs) {
		t.Error("read error")
	}
	if len(ss.belts) != 0 {
		t.Errorf("verb called %v", ss.belts)
	}
}

func TestMercPotionNoVerbLogsOncePerMinute(t *testing.T) {
	h, buf, _ := mercHarness(t, merc.MercState{Known: true, Owned: true, Alive: true, LifePct: 40}) // sim has no semantic input
	h.s.stock(itHP1, 0, 3)
	twoMonsters(h)
	snap, _ := h.s.snapshot()
	hs := Hostiles(snap, h.f.hostileCtx(snap.Player))
	for i := 0; i < 3; i++ {
		if h.f.mercPotion(snap, hs) {
			t.Fatal("a potion went without a verb")
		}
	}
	if n := strings.Count(buf.String(), "merc: potion wanted (no verb)"); n != 1 {
		t.Errorf("%d lines in a minute, want 1\n%s", n, buf)
	}
	h.s.t = h.s.t.Add(61 * time.Second)
	h.f.mercPotion(snap, hs)
	if n := strings.Count(buf.String(), "merc: potion wanted (no verb)"); n != 2 {
		t.Errorf("%d lines after a minute, want 2", n)
	}
}

func TestMercPotionVerbFailureBacksOff(t *testing.T) {
	h, buf, _ := mercHarness(t, merc.MercState{Known: true, Owned: true, Alive: true, LifePct: 40})
	ss := &semSim{sim: h.s, err: errors.New("busy")}
	h.f.e.In = ss
	h.s.stock(itHP1, 0, 3)
	twoMonsters(h)
	snap, _ := h.s.snapshot()
	hs := Hostiles(snap, h.f.hostileCtx(snap.Player))
	if h.f.mercPotion(snap, hs) || h.f.mercPotion(snap, hs) || len(ss.belts) != 1 {
		t.Errorf("calls %v: a failed verb must wait out the gap", ss.belts)
	}
	if !strings.Contains(buf.String(), "merc: potion failed") {
		t.Errorf("log:\n%s", buf)
	}
}

// The crowd of TestSafetyCrowdFromSnapshot at full life: the ordinary rule says nothing, with the merc down the
// character leaves by portal.
func TestSafetyCrowdWithMercDownLeaves(t *testing.T) {
	h, buf, st := mercHarness(t, merc.MercState{Known: true, Owned: true, Alive: true, LifePct: 100})
	h.f.e.Tune.Combat.CrowdAdjacent = 3
	h.s.stock(itHP1, 0, 3)
	for i := 0; i < 9; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 105+float64(i%3), 98+float64(i%5), 30, 5, 5)
	}
	snap, _ := h.s.snapshot()
	if a := h.f.Safety(snap); a != SafetyNone {
		t.Fatalf("merc alive, full life: %v", a)
	}
	st.Alive, st.LifePct = false, 0
	if a := h.f.Safety(snap); a != TownPortal {
		t.Fatalf("merc down: %v", a)
	}
	h.f.Safety(snap)
	if n := strings.Count(buf.String(), "merc: down in a crowd, leaving"); n != 1 {
		t.Errorf("%d log lines for two ticks, want 1\n%s", n, buf)
	}
	h.f.MercState = nil
	if a := h.f.Safety(snap); a != SafetyNone {
		t.Errorf("rule off: %v", a)
	}
	*st = merc.MercState{Known: true} // no merc owned: nothing to leave behind
	h.f.MercState = func() (merc.MercState, error) { return *st, nil }
	if a := h.f.Safety(snap); a != SafetyNone {
		t.Errorf("no merc owned: %v", a)
	}
}

// RunnerGuard interplay: a retreat is a walk away, which a close runner makes pointless, so the fight goes on.
func TestMercDownLeaveRetreatRespectsRunnerGuard(t *testing.T) {
	cat := MapCatalog{Monsters: map[int]*data.Monster{3: {Key: "zombie1", AI: "Zombie", Killable: true}}}
	snap := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: 1, X: 100, Y: 100, Area: 5},
		Monsters: []d2.Unit{{ID: 11, Class: 3, Mode: 1, X: 102, Y: 100}}}
	down := merc.MercState{Known: true, Owned: true}
	for _, c := range []struct {
		guard int
		want  SafetyAction
	}{{1, SafetyNone}, {0, CrowdRetreat}} {
		h, _, _ := mercHarness(t, down)
		h.f.e.Tune.Combat.RunnerGuard = c.guard
		h.f.SetCatalog(cat)
		in := crowdIn(100, 8, 0, func(i *SafetyInput) { i.HasPortal = false })
		if got := h.f.mercDownLeave(snap, in); got != c.want {
			t.Errorf("runner guard %d: %v want %v", c.guard, got, c.want)
		}
	}
	// a portal is not a walk: it is taken even with a runner close
	h, _, _ := mercHarness(t, down)
	h.f.SetCatalog(cat)
	if got := h.f.mercDownLeave(snap, crowdIn(100, 8, 0, nil)); got != TownPortal {
		t.Errorf("portal with a runner close: %v", got)
	}
	// surrounded: the cast would fail, so no portal (retreat or fight)
	if got := h.f.mercDownLeave(snap, crowdIn(100, 8, 3, func(i *SafetyInput) { i.CanRetreat = false })); got != SafetyNone {
		t.Errorf("surrounded: %v", got)
	}
}
