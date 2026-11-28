package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/plandata"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/town/belt"
)

// planBuilds lists every embedded build table with its class.
func planBuilds(t *testing.T) map[string]d2.Class {
	t.Helper()
	classes := map[string]d2.Class{}
	for c := d2.Amazon; c <= d2.Assassin; c++ {
		classes[strings.ToLower(c.String())] = c
	}
	out := map[string]d2.Class{}
	ents, err := plandata.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if !strings.HasPrefix(e.Name(), "build-") {
			continue
		}
		raw, err := plandata.FS.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		var h struct{ Build, Class string }
		if json.Unmarshal(raw, &h) != nil || h.Build == "" {
			continue // the catalog
		}
		c, ok := classes[strings.ToLower(h.Class)]
		if !ok {
			t.Fatalf("%s: unknown class %q", e.Name(), h.Class)
		}
		out[h.Build] = c
	}
	return out
}

// Review C B2: the mana need has one source (resource.ManaNeedFor). For every plan build of all seven classes, in every
// difficulty, the kit gate, the shop rule and the belt columns (the bot's and the town's) agree: a mana minimum in the
// kit means a mana column to carry it in and a shop rule that buys it, and a build that needs no mana gets none of the
// three. Must fail while the kit asks the build and the shop and the belt ask the class (a War Cry Barbarian or a
// Hammerdin in Normal: "mana potions 0/N" for good).
func TestManaNeedKitShopBeltAgree(t *testing.T) {
	builds := planBuilds(t)
	for _, want := range []string{"necromancer_poison_nova_sme", "sorceress_normal30", "paladin_hammerdin_endgame90",
		"barbarian_warcry_endgame90", "barbarian_melee_normal30", "amazon_javelin_normal30", "assassin_sentry_endgame90", "druid_tornado_endgame90"} {
		if _, ok := builds[want]; !ok {
			t.Errorf("plan build %s is gone: name its successor here", want)
		}
	}
	seen := map[d2.Class]bool{}
	for build, class := range builds {
		seen[class] = true
		for _, diff := range []d2.Difficulty{d2.Normal, d2.Nightmare, d2.Hell} {
			for _, level := range []int{10, 20, 40, 70} {
				name := fmt.Sprintf("%s %v clvl %d", build, diff, level)
				g := newRig(t, func(c *Config, _ *Deps, _ *env.Tuning) { c.LevelerBuild = build })
				g.w.set(func(w *world) { w.level, w.diff, w.area, w.gold0 = level, diff, 4, 50000 })
				s, _ := g.w.snapshot()
				s.Player.Class = class
				k, known := g.r.kitRead(s)
				if !known {
					t.Fatalf("%s: kit unknown", name)
				}
				in := g.r.resInput(s)
				rt := resource.FromEnv(*g.e.Tune)
				need := resource.ManaNeedFor(class, build)
				plan := resource.PlanForBuild(class, level, diff, build)
				shop := resource.ManaBuy(need, rt)
				band := belt.PlanForBuild(class, level, diff, build) // the plan the town shops and fills by
				townMana := band.Capacity(belt.MP, 4) > 0
				if plan.HasMana() != (need > 0) || shop.Allow != (need > 0) || townMana != (need > 0) {
					t.Errorf("%s: need %d but belt mana column %v, town mana column %v, shop buys %v", name, need, plan.HasMana(), townMana, shop.Allow)
				}
				if (k.minMP > 0) != (need >= 2) {
					t.Errorf("%s: need %d but kit mana minimum %d", name, need, k.minMP)
				}
				if c := resource.ManaCapacity(in, rt); k.minMP > c {
					t.Errorf("%s: kit asks %d mana potions, the belt plan holds %d: never complete", name, k.minMP, c)
				}
				if k.minMP > 0 && (shop.MinHealing > 0 || shop.KeepKit || shop.MaxPerVisit > 0) {
					t.Errorf("%s: kit asks %d mana potions but the shop rule is the occasional one: %+v", name, k.minMP, shop)
				}
			}
		}
	}
	for c := d2.Amazon; c <= d2.Assassin; c++ {
		if !seen[c] {
			t.Errorf("no plan build for %v", c)
		}
	}
}
