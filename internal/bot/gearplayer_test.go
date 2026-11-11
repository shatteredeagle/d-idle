package bot

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// fakeAdvisor is a fakeGear that also answers the pure-read advice.
type fakeAdvisor struct{ *fakeGear }

func (fakeAdvisor) GearHints(*d2.Snapshot) []GearHint {
	return []GearHint{{ID: 77, Text: "Hand Axe: better than your Short Sword in the weapon slot"}}
}

// gear.decisions = player: a town visit with gold and a better item in the pack equips nothing and buys nothing; the
// advice and the onboarding prompt appear instead (docs/gear-policy.md). Paladin and Sorceress.
func TestGearStepPlayerNeverEquipsOrBuys(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(class.String(), func(t *testing.T) {
			g := newRig(t, func(_ *Config, _ *Deps, tn *env.Tuning) { tn.Resource.GearDecisions = 0 })
			gr := &fakeGear{w: g.w}
			gr.swaps.Store(1)
			g.r.d.Gear = fakeAdvisor{gr}
			g.w.set(func(w *world) { w.class = class })
			g.mover.exploreFn = blockExplore
			g.w.gold(1000)
			g.run()
			defer g.finish()
			g.r.Start(reach(10))
			g.waitFor("advice", func() bool { return g.r.GearAdviceFor(77) != "" })
			if gr.shopped.Load() != 0 || g.w.count("equip") != 0 || g.w.count("shop") != 0 {
				t.Fatalf("the bot made an equipment decision: %s", g.w.log())
			}
			if !strings.Contains(g.r.GearAdviceFor(77), "Hand Axe") {
				t.Fatalf("advice %q", g.r.GearAdviceFor(77))
			}
			if p := g.r.GearPrompt(); p != Onboarding {
				t.Fatalf("prompt %q", p)
			}
		})
	}
}

// No weapon worn on a class that needs one: the prompt names it; a caster keeps the onboarding line.
func TestGearPromptNoWeapon(t *testing.T) {
	g := newRig(t, func(_ *Config, _ *Deps, tn *env.Tuning) { tn.Resource.GearDecisions = 0 })
	g.r.d.Gear = fakeAdvisor{&fakeGear{w: g.w}}
	s := &d2.Snapshot{Player: &d2.Player{Class: d2.Paladin}, Inventory: &d2.Inventory{}}
	g.r.adviseGear(s)
	if p := g.r.GearPrompt(); p != NoWeaponPrompt {
		t.Fatalf("paladin prompt %q", p)
	}
	s.Player.Class = d2.Sorceress
	g.r.adviseGear(s)
	if p := g.r.GearPrompt(); p != Onboarding {
		t.Fatalf("sorceress prompt %q", p)
	}
}
