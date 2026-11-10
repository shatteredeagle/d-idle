package combat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
)

// castMs is the length of one cast in ms on the slower backend: ticks * 40 ms.
func castMs(name string, fcr int) int { return castTicks(name, fcr, false) * tickMs }

// The oracle values of docs/research/cast-trace-gameexe.md (V, cast_rates.py): the Sorceress plain SC cast at FCR 0 /
// 9 / 20 / 37 / 63 / 105 / 200 and the Lightning sequence at FCR 0, 7 and 194.
func TestCastTicksMatchOracle(t *testing.T) {
	for fcr, want := range map[int]int{0: 13, 9: 12, 20: 11, 37: 10, 63: 9, 105: 8, 200: 7} {
		if g := castTicks("Fire Bolt", fcr, false); g != want {
			t.Errorf("Fire Bolt at FCR %d: %d ticks, want %d", fcr, g, want)
		}
	}
	for fcr, want := range map[int]int{0: 19, 7: 18, 194: 11} {
		if g := castTicks("Lightning", fcr, false); g != want {
			t.Errorf("Lightning at FCR %d: %d ticks, want %d", fcr, g, want)
		}
	}
	if castTicks("Hydra", 0, false) != 0 {
		t.Error("an unknown skill must have no table entry")
	}
}

// Per skill: the gap is at least the cast length (13 ticks on game.exe, 14 on d2d, 19 for the sequence) at the
// assumed FCR, for every plan skill and every FCR the oracle lists.
func TestCastGapCoversEveryPlanSkill(t *testing.T) {
	d := env.DefaultCombatTuning()
	for _, name := range []string{"Fire Bolt", "Charged Bolt", "Ice Bolt", "Frost Nova", "Lightning", "Chain Lightning", "Teleport", "Frozen Armor"} {
		for _, fcr := range []int{0, 9, 20, 37, 63, 105, 200} {
			g := castGapFor(name, d.CastGapMs, d.CastMarginMs, fcr, true, false)
			if g < time.Duration(castMs(name, fcr)+tickMs)*time.Millisecond {
				t.Errorf("%s at FCR %d: gap %v below the cast %d ms", name, fcr, g, castMs(name, fcr))
			}
		}
	}
	g := castGapFor("Chain Lightning", d.CastGapMs, d.CastMarginMs, 0, true, false)
	if g != 890*time.Millisecond {
		t.Errorf("Chain Lightning at FCR 0: %v, want 890ms", g)
	}
	if g := castGapFor("Fire Bolt", d.CastGapMs, d.CastMarginMs, 0, true, false); g != 650*time.Millisecond {
		t.Errorf("Fire Bolt at FCR 0: %v, want the old 650ms", g)
	}
}

func TestCastGapFallbackAndOverride(t *testing.T) {
	if g := castGapFor("Hydra", 650, 90, 0, true, false); g != 650*time.Millisecond {
		t.Errorf("unknown skill: %v, want CastGapMs", g)
	}
	if g := castGapFor("Lightning", 650, 90, 0, false, false); g != 650*time.Millisecond {
		t.Errorf("per-skill off: %v, want CastGapMs", g)
	}
	if g := castGapFor("Fire Bolt", 1200, 90, 0, true, false); g != 1200*time.Millisecond {
		t.Errorf("a raised CastGapMs is a floor: %v", g)
	}
	if g := castGapFor("Lightning", 650, 90, 200, true, false); g != 650*time.Millisecond {
		t.Errorf("a fast caster is floored at CastGapMs: %v", g)
	}
}

// lightningHarness is a level 30 Lightning Sorceress whose only attack is Lightning on f9; the cast takes no clock time
// so a command is the start of a cast.
func lightningHarness(t testing.TB, mods ...func(*env.Tuning)) *harness {
	h := newHarness(t, mods...)
	// the shipped profile avoids Lightning (known broken on d2d); this test is about the cast gap, so lift it here
	pr := ProfileFor(d2.Sorceress, BuildLightning)
	saved := pr.Avoid
	pr.Avoid = nil
	t.Cleanup(func() { pr.Avoid = saved })
	c := testCatalog()
	c.Skills[49] = &data.Skill{ID: 49, Key: "Lightning", Name: "Lightning", EType: "ltng", Mana: 8, MinMana: 1}
	h.f.SetCatalog(c)
	h.f.Build = BuildLightning
	h.s.skills = []d2.Skill{{ID: 49, Level: 5}}
	h.f.SetHotkeys(map[int]string{49: "f9"})
	h.s.clvl, h.s.instantCast = 30, true
	h.s.mana, h.s.maxM, h.s.castDmg = 500, 500, 4
	return h
}

// earlyReissues runs a fight of back-to-back Lightning casts and counts commands issued before the previous cast (19
// ticks, 760 ms, game.exe) ended. Commands are timed from the fake clock at the moment CastAt is called.
func earlyReissues(t *testing.T, mods ...func(*env.Tuning)) (early, cmds int) {
	h := lightningHarness(t, mods...)
	var last time.Time
	h.s.onCast = func() {
		now := h.s.t
		if !last.IsZero() && now.Sub(last) < 760*time.Millisecond {
			early++
		}
		last, cmds = now, cmds+1
	}
	h.s.addMon(10, clsFallen, 112, 100, 400, 0, 0)
	h.f.Clear(context.Background(), ClearOpts{Timeout: 20 * time.Second})
	return early, cmds
}

// The old flat gap (650 ms) re-issues Lightning 650 + a poll tick after the last one; at the 760 ms cast that is
// sometimes early. The per-skill gap never is.
func TestFightLoopLightningNeverReissuedEarly(t *testing.T) {
	earlyOld, n := earlyReissues(t, func(tu *env.Tuning) { tu.Combat.CastPerSkill = 0 })
	if n < 5 || earlyOld == 0 {
		t.Fatalf("old flat gap: %d early of %d commands, want some early ones (the test must see the bug)", earlyOld, n)
	}
	early, n := earlyReissues(t)
	if n < 5 || early != 0 {
		t.Fatalf("per-skill gap: %d early of %d commands", early, n)
	}
}

// The Lightning gap per backend: 890 ms for the 19-tick game.exe sequence, 650 ms on d2d where it is a plain SC cast
// of 14 ticks (V-d2d for the SC length).
func TestLightningGapPerBackend(t *testing.T) {
	d := env.DefaultCombatTuning()
	for _, name := range []string{"Lightning", "Chain Lightning"} {
		if g := castGapFor(name, d.CastGapMs, d.CastMarginMs, 0, true, false); g != 890*time.Millisecond {
			t.Errorf("%s game.exe: %v, want 890ms", name, g)
		}
		if g := castGapFor(name, d.CastGapMs, d.CastMarginMs, 0, true, true); g != 650*time.Millisecond {
			t.Errorf("%s d2d: %v, want 650ms", name, g)
		}
	}
	h := lightningHarness(t)
	h.f.D2D = true
	h.f.lastCastName = "Lightning"
	if g := h.f.castGap(); g != 650*time.Millisecond {
		t.Errorf("fighter on d2d: %v", g)
	}
}
