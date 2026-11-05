package bot

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// Gate 2 g2c seed 13 death: the guard dropped DrinkLife and DrinkRejuv in state clearing, and the nav stuck ladder
// walked the character from 65 % to 0 with potions in the belt. The guard now carries them out in every state that
// can stand near hostiles, but not in town; Chicken, town portal and crowd abandon are unchanged.
func TestGuardDrinksInEveryFieldState(t *testing.T) {
	// the fake fighter stands in for combat.Decide on the belt: heal 55 %, rejuv at or below 25 %, no potion left: chicken
	decide := func(heal, rejuv bool) func(*d2.Snapshot) SafetyAction {
		return func(s *d2.Snapshot) SafetyAction {
			lp := s.Player.Life * 100 / s.Player.MaxLife
			switch {
			case lp <= 25 && rejuv:
				return SafetyDrinkRejuv
			case lp <= 25:
				return SafetyChicken
			case lp < 55 && heal:
				return SafetyDrinkLife
			}
			return SafetyNone
		}
	}
	cases := []struct {
		name        string
		class       d2.Class
		st          State
		life        int
		heal, rejuv bool
		town        bool
		want        SafetyAction
	}{
		{"sorc clearing heal", d2.Sorceress, StateClearing, 40, true, false, false, SafetyDrinkLife},
		{"barb clearing heal", d2.Barbarian, StateClearing, 40, true, false, false, SafetyDrinkLife},
		{"sorc clearing rejuv", d2.Sorceress, StateClearing, 20, true, true, false, SafetyDrinkRejuv},
		{"barb clearing rejuv", d2.Barbarian, StateClearing, 20, false, true, false, SafetyDrinkRejuv},
		{"sorc traveling heal", d2.Sorceress, StateTraveling, 40, true, false, false, SafetyDrinkLife},
		{"barb traveling rejuv", d2.Barbarian, StateTraveling, 20, true, true, false, SafetyDrinkRejuv},
		{"sorc clearing nothing usable", d2.Sorceress, StateClearing, 20, false, false, false, SafetyChicken},
		{"barb traveling nothing usable", d2.Barbarian, StateTraveling, 20, false, false, false, SafetyChicken},
		{"healthy", d2.Sorceress, StateClearing, 90, true, true, false, SafetyNone},
		{"town", d2.Barbarian, StateClearing, 40, true, true, true, SafetyNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := newRig(t)
			area := 2
			if c.town {
				area = 1
			}
			g.w.set(func(w *world) {
				w.state = d2.StateInGame
				w.area = area
				w.life, w.maxL = c.life, 100
				w.class = c.class
			})
			g.fight.safety = decide(c.heal, c.rejuv)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			fired := g.r.guardOnce(ctx, cancel, c.st)
			var si safetyInterrupt
			got := SafetyNone
			if errors.As(context.Cause(ctx), &si) {
				got = si.Action
			}
			if got != c.want || fired != (c.want != SafetyNone) {
				t.Fatalf("fired %v action %v, want %v", fired, got, c.want)
			}
		})
	}
}

// A second drink inside the guard's gap is not forwarded (the fighter's own potion cooldown keeps its Safety quiet
// as well, combat.Fighter lastDrink); after the gap it is.
func TestGuardDrinkIsRateLimited(t *testing.T) {
	// the reaction sleep would advance the virtual clock past the gap
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Bot.ReactionMs = 0 })
	g.w.set(func(w *world) { w.state = d2.StateInGame; w.area = 2; w.life, w.maxL = 40, 100 })
	g.fight.safety = func(*d2.Snapshot) SafetyAction { return SafetyDrinkLife }
	try := func() bool {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		g.r.guardOnce(ctx, cancel, StateClearing)
		var si safetyInterrupt
		return errors.As(context.Cause(ctx), &si)
	}
	if !try() {
		t.Fatal("first drink not forwarded")
	}
	if try() {
		t.Fatal("second drink inside the gap forwarded")
	}
	g.clock.Sleep(time.Second)
	if !try() {
		t.Fatal("drink after the gap not forwarded")
	}
}
