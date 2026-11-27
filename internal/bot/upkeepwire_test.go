package bot

import (
	"reflect"
	"testing"

	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/supply"
)

func sourceNames(ss []arbiter.Source) []string {
	var out []string
	for _, s := range ss {
		out = append(out, s.Name())
	}
	return out
}

// The chores family under control swaps trip and resupply for the upkeep source and merges the plan facts into the View;
// every other setting (the defaults) leaves the sources and the facts exactly as they were. Two builds.
func TestUpkeepWiredBehindChoresBit(t *testing.T) {
	pl := supply.Plan{Lines: []supply.PlanLine{{Line: supply.Healing, Have: 1, Min: 3, Buy: 2}}, Verdict: supply.ShortGold, Open: "healing", Short: 20}
	std := sourceNames(arbiter.StdSources())
	upk := sourceNames(arbiter.UpkeepSources())
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		cases := []struct {
			name        string
			shadow, ctl int
			mask        int
			wantUpkeep  bool
		}{
			{"defaults", 0, 0, 3, false},
			{"shadow only", 1, 0, 7, false},
			{"control, chores bit unset", 1, 1, 3, false},
			{"control, chores bit set", 1, 1, 7, true},
		}
		var base map[string]int
		for _, c := range cases {
			g := newRig(t)
			g.w.set(func(w *world) { w.class = class })
			g.r.d.Town = fixedPlanner{Townsman: g.r.d.Town, pl: pl}
			g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = c.shadow, c.ctl, c.mask
			g.r.poll()
			got := sourceNames(g.r.arbSources())
			want := std
			if c.wantUpkeep {
				want = upk
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%v %s: sources %v, want %v", class, c.name, got, want)
			}
			v, ok := g.r.arbView(g.r.latest(), control{})
			if !ok {
				t.Fatalf("%v %s: no view", class, c.name)
			}
			_, has := v.Facts[arbiter.FactPlanBuy]
			if has != c.wantUpkeep {
				t.Fatalf("%v %s: plan facts present %v, want %v", class, c.name, has, c.wantUpkeep)
			}
			if c.wantUpkeep && v.Facts[arbiter.FactPlanVerdict] != arbiter.PlanShort {
				t.Fatalf("%v %s: verdict fact %d", class, c.name, v.Facts[arbiter.FactPlanVerdict])
			}
			if !c.wantUpkeep {
				if base == nil {
					base = v.Facts
				} else if !reflect.DeepEqual(base, v.Facts) {
					t.Fatalf("%v %s: default facts changed: %v vs %v", class, c.name, v.Facts, base)
				}
			}
		}
	}
}
