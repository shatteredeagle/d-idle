package combat

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

const idFrostNova = 44

// novaHarness is the Sorceress of the fallback build with Charged Bolt on f4 and Frost Nova on f6.
func novaHarness(t testing.TB) *harness {
	h := newHarness(t)
	c := testCatalog()
	c.Skills[idFrostNova] = &data.Skill{ID: idFrostNova, Key: "Frost Nova", Name: "Frost Nova", EType: "cold", Mana: 9, MinMana: 1, ManaShift: 8}
	h.f.SetCatalog(c)
	h.s.skills = []d2.Skill{{ID: 36, Level: 0, Bonus: 1}, {ID: 38, Level: 6}, {ID: idFrostNova, Level: 1}}
	h.f.SetHotkeys(map[int]string{36: "f8", 38: "f4", idFrostNova: "f6"})
	h.s.mana, h.s.maxM = 200, 200
	return h
}

func logTo(h *harness) *bytes.Buffer {
	var buf bytes.Buffer
	h.f.e.Log = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return &buf
}

// INVARIANT: a pack of 15 melee hostiles with life at 60% is fought: casts happen, no walking. With a portal scroll the
// crowd rule (crowd.go, SME decision EXP) now leaves at 70% instead; the invariant holds when there is no scroll.
func TestCastFirstPackAt60(t *testing.T) {
	h := novaHarness(t)
	h.s.noTP = true
	h.s.life = 60
	h.s.stock(itHP1, 0, 4)
	for i := 0; i < 15; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 106+float64(i%5), 100+float64(i/5)*2, 40, 1, 2)
	}
	buf := logTo(h)
	h.f.Clear(context.Background(), ClearOpts{Timeout: 10 * time.Second})
	if h.s.casts < 5 {
		t.Fatalf("casts=%d\n%s", h.s.casts, buf.String())
	}
	if len(h.s.moves) > 2 {
		t.Fatalf("moved %d times against a pack: %v", len(h.s.moves), h.s.moves)
	}
}

func TestCastFirstNovaOnCrowd(t *testing.T) {
	h := novaHarness(t)
	for i := 0; i < 3; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 101+float64(i), 101, 60, 1, 0)
	}
	logTo(h)
	h.f.Clear(context.Background(), ClearOpts{Timeout: 6 * time.Second})
	if len(h.s.castKeys) < 2 || h.s.castKeys[0] != "f6" {
		t.Fatalf("keys %v: Nova first expected", h.s.castKeys)
	}
	nova := 0
	for _, k := range h.s.castKeys {
		if k == "f6" {
			nova++
		}
	}
	if nova > 3 || nova == len(h.s.castKeys) {
		t.Fatalf("keys %v: Nova once, then bolts", h.s.castKeys)
	}
}

func TestCastFirstSingleMeleeStepsBackOnceThenCasts(t *testing.T) {
	h := novaHarness(t)
	h.s.life = 58
	h.s.healAmount = 0 // potions drunk do not help: the monster keeps hurting us
	h.s.stock(itHP1, 0, 6)
	h.s.addMon(10, clsFallen, 102, 100, 200, 4, 3)
	buf := logTo(h)
	h.f.Clear(context.Background(), ClearOpts{Timeout: 8 * time.Second})
	if len(h.s.moves) == 0 || len(h.s.moves) > 2*stepBackSteps {
		t.Fatalf("moves=%v, want a bounded step back\n%s", h.s.moves, buf.String())
	}
	if h.s.casts == 0 {
		t.Fatal("never cast")
	}
}

// The nav-yield loop: the monster was ignored before; a yield call fights it at once, a plain Clear does not.
func TestCastFirstYieldEngagesIgnored(t *testing.T) {
	run := func(yield bool) int {
		h := novaHarness(t)
		m := h.s.addMon(10, clsFallen, 109, 100, 30, 1, 0)
		_ = m
		h.f.ignores = map[uint32]ignoreRec{10: {until: h.s.t.Add(time.Hour)}}
		h.f.Clear(context.Background(), ClearOpts{Timeout: 4 * time.Second, Yield: yield})
		return h.s.casts
	}
	if run(false) != 0 {
		t.Fatal("plain Clear should honour the ignore memory")
	}
	if run(true) == 0 {
		t.Fatal("a yield Clear must cast at the blocking monster")
	}
}

func TestCastFirstWatchdogForcesCast(t *testing.T) {
	h := novaHarness(t)
	h.s.addMon(10, clsFallen, 108, 100, 100, 1, 0)
	buf := logTo(h)
	snap, _ := h.s.snapshot()
	hs := Hostiles(snap, h.f.hostileCtx(snap.Player))
	st := newFightState()
	now := h.f.now()
	st.lastCastAt, st.nearSince, st.lastNoCast = now.Add(-7*time.Second), now.Add(-7*time.Second), "kite: test"
	h.f.lastCast = now.Add(-time.Minute)
	ok, err := h.f.castFirst(context.Background(), snap, hs, st)
	if err != nil || !ok || h.s.casts != 1 || st.forced != 1 {
		t.Fatalf("ok=%v err=%v casts=%d forced=%d", ok, err, h.s.casts, st.forced)
	}
	if !strings.Contains(buf.String(), "combat: forced cast (no cast for 7s, kite: test)") {
		t.Fatalf("log: %s", buf.String())
	}
	// no hostile in range: nothing forced
	h2 := novaHarness(t)
	h2.s.addMon(10, clsFallen, 140, 100, 100, 1, 0)
	snap, _ = h2.s.snapshot()
	st = newFightState()
	st.lastCastAt = h2.f.now().Add(-time.Minute)
	if ok, _ := h2.f.castFirst(context.Background(), snap, Hostiles(snap, h2.f.hostileCtx(snap.Player)), st); ok {
		t.Fatal("forced a cast at a monster out of range")
	}
}

// Evasion never replaces casts for long: 2 s after the last cast a missile dodge is refused.
func TestEvadeGate(t *testing.T) {
	h := novaHarness(t)
	st := newFightState()
	st.lastCastAt = h.f.now()
	if !h.f.evadeOK(st) {
		t.Fatal("evade should be allowed right after a cast")
	}
	h.s.Sleep(3 * time.Second)
	if h.f.evadeOK(st) {
		t.Fatal("evade must be refused 3 s after the last cast")
	}
}

// BUILDV2: Frost Nova on a crowd is not cast below combat.novaManaFloor (9 mana against a 50-60 pool starves the bolts).
func TestCastFirstNovaRespectsManaFloor(t *testing.T) {
	for _, tc := range []struct {
		mana int
		nova bool
	}{{20, false}, {60, true}} {
		h := novaHarness(t)
		h.s.mana, h.s.maxM = float64(tc.mana), 60
		for i := 0; i < 3; i++ {
			h.s.addMon(uint32(10+i), clsFallen, 101+float64(i), 101, 60, 1, 0)
		}
		snap, _ := h.s.snapshot()
		st := newFightState()
		ok, err := h.f.novaTick(snap, st)
		if err != nil || ok != tc.nova {
			t.Errorf("mana %d: nova=%v err=%v, want %v", tc.mana, ok, err, tc.nova)
		}
	}
}
