package combat

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/ladder"
)

// installTaught installs a ladder whose scripted player answers every ask of point with answer; the rule is learned
// by repeating run (a fresh scenario each time) with the clock moving between runs.
func installTaught(t *testing.T, point, answer string, run func()) *ladder.Ladder {
	t.Helper()
	src, err := ladder.ParseScript([]byte(`{"rows":[{"point":"` + point + `","answer":"` + answer + `","confirm":"yes","delaySec":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	clk := &decide.ManualClock{}
	l, err := ladder.Open("", "combat", clk, nil, src, nil)
	if err != nil {
		t.Fatal(err)
	}
	ladder.Install(l)
	t.Cleanup(func() { ladder.Install(nil) })
	for i := 0; i < 10 && l.Stats().ByRule == 0; i++ {
		run()
		clk.Advance(5 * time.Second)
		l.Pump()
	}
	return l
}

// fleeScenario reaches the abort step: no potion, no portal, life under the flee line, a monster on top of the hero.
func fleeScenario(t *testing.T) (*harness, error) {
	h := newHarness(t, func(e *env.Tuning) { e.Combat.KiteEnabled = 0 })
	h.s.class = d2.Sorceress
	h.s.noTP = true
	h.s.life = 45
	h.s.mana, h.s.manaRegen = 0, 0
	h.s.addMon(10, clsFallen, 103, 100, 500, 0, 0)
	_, err := h.f.Clear(context.Background(), ClearOpts{Timeout: 20 * time.Second})
	return h, err
}

// t0.retreat.drink at the abort step: with the ladder installed and nothing taught the legacy break-away stands; a
// learned "leave" replaces it with the chicken.
func TestLadderRetreatDrink(t *testing.T) {
	ladder.Install(nil)
	if h, err := fleeScenario(t); h.s.left || errors.Is(err, ErrChicken) {
		t.Fatalf("legacy must flee, not leave: %v", err)
	}
	l, err := ladder.Open("", "combat", &decide.ManualClock{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ladder.Install(l)
	if h, err := fleeScenario(t); h.s.left || errors.Is(err, ErrChicken) {
		t.Fatalf("ladder on, untaught: %v", err)
	}
	if st := l.Stats(); st.Asked == 0 || st.ByLegacy != st.Asked || st.Faults != 0 {
		t.Fatalf("expected legacy answers only: %+v", st)
	}
	ladder.Install(nil)

	l = installTaught(t, ladder.PointDrink, "leave", func() { fleeScenario(t) })
	if l.Stats().ByRule == 0 {
		t.Fatalf("no rule learned: %+v", l.Stats())
	}
	if h, err := fleeScenario(t); !h.s.left || !errors.Is(err, ErrChicken) {
		t.Fatalf("learned leave: left %v err %v", h.s.left, err)
	}
}

func uniqueScenario(t *testing.T) (*harness, ClearResult) {
	h := newHarness(t)
	h.s.class = d2.Sorceress
	h.s.life = 68
	h.s.addMon(10, clsBoss, 130, 100, 40, 0, 0) // 30 away: seen, not yet in the fight radius
	res, err := h.f.Clear(context.Background(), ClearOpts{Timeout: 20 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return h, res
}

// t0.unique.fight at pre-engage: untaught the pack is fought as before; a learned "avoid" skips the pack.
func TestLadderUniqueFight(t *testing.T) {
	ladder.Install(nil)
	if h, res := uniqueScenario(t); res.Kills == 0 && h.s.casts == 0 {
		t.Fatalf("legacy must fight the pack: %+v", res)
	}
	l, err := ladder.Open("", "combat", &decide.ManualClock{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ladder.Install(l)
	if h, res := uniqueScenario(t); res.Kills == 0 && h.s.casts == 0 || l.Stats().Asked == 0 {
		t.Fatalf("ladder on, untaught must fight and ask: %+v %+v", res, l.Stats())
	}
	ladder.Install(nil)

	l = installTaught(t, ladder.PointUnique, "avoid", func() { uniqueScenario(t) })
	if l.Stats().ByRule == 0 {
		t.Fatalf("no rule learned: %+v", l.Stats())
	}
	if h, res := uniqueScenario(t); res.Kills != 0 || h.s.casts != 0 {
		t.Fatalf("learned avoid must skip the pack: %+v casts %d", res, h.s.casts)
	}
}
