package plandata

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Card classapply (gap list of the SME review 2026-11-27): every class that has a normal30 plan table has a row in the
// level plan and in the Act 2 plan, so neither errors by class name. The new rows are copies; their build tables exist.
func TestEveryPlayableClassHasPlanRows(t *testing.T) {
	for _, f := range []string{"levelplan-normal.json", "act2plan.json"} {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var p struct {
			Classes map[string]struct {
				Build string `json:"build"`
				Tag   string `json:"tag"`
			} `json:"classes"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatal(err)
		}
		for _, c := range []string{"paladin", "sorceress", "necromancer", "barbarian", "druid", "amazon", "assassin"} {
			row, ok := p.Classes[c]
			if !ok {
				t.Errorf("%s has no %s row", f, c)
				continue
			}
			switch c {
			case "barbarian", "druid", "amazon", "assassin":
				if _, err := os.Stat("build-" + row.Build + ".json"); err != nil {
					t.Errorf("%s %s: build %q has no table: %v", f, c, row.Build, err)
				}
			}
		}
	}
}

// Card cls2 (review 8 C5): the archetype of an Act 2 plan row is the caster trait of the class in
// internal/d2/classtraits.json, the trait the admission view derives its archetype from. Two sources that disagree
// would plan with one set of floors and enforce the other. The level plan has no archetype: there a class must not
// carry the copied row set of a class with the other trait.
func TestPlanArchetypeFollowsClassTraits(t *testing.T) {
	raw, err := os.ReadFile("../d2/classtraits.json")
	if err != nil {
		t.Fatal(err)
	}
	var traits map[string]struct {
		Caster bool `json:"caster"`
	}
	if err := json.Unmarshal(raw, &traits); err != nil {
		t.Fatal(err)
	}
	casterOf := map[string]bool{}
	for name, tr := range traits {
		casterOf[strings.ToLower(name)] = tr.Caster
	}
	p, err := LoadAct2Plan()
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range p.Classes {
		caster, ok := casterOf[name]
		if !ok {
			t.Errorf("act2plan class %q has no classtraits row", name)
			continue
		}
		want := "melee"
		if caster {
			want = "caster"
		}
		if c.Archetype != want {
			t.Errorf("act2plan %s: archetype %q, classtraits caster=%v wants %q", name, c.Archetype, caster, want)
		}
	}

	raw, err = os.ReadFile("levelplan-normal.json")
	if err != nil {
		t.Fatal(err)
	}
	var lp struct {
		Classes map[string]struct {
			Slices []struct {
				ID    string `json:"id"`
				Entry int    `json:"entry"`
				Exit  int    `json:"exit"`
			} `json:"slices"`
		} `json:"classes"`
	}
	if err := json.Unmarshal(raw, &lp); err != nil {
		t.Fatal(err)
	}
	same := func(a, b string) bool {
		x, y := lp.Classes[a].Slices, lp.Classes[b].Slices
		if len(x) != len(y) {
			return false
		}
		for i := range x {
			if x[i] != y[i] {
				return false
			}
		}
		return true
	}
	for a := range lp.Classes {
		for b := range lp.Classes {
			if casterOf[a] != casterOf[b] && same(a, b) {
				t.Errorf("levelplan: %s (caster=%v) carries the row set of %s (caster=%v)", a, casterOf[a], b, casterOf[b])
			}
		}
	}
}
