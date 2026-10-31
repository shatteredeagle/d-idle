package resource

import (
	"encoding/json"
	"os"
	"testing"

	"d2lod-bot/internal/d2"
)

func TestPlanForBuildManaHeavy(t *testing.T) {
	const b = "barbarian_warcry_endgame90"
	hp, mp, rv := PotionLife, PotionMana, PotionRejuv
	cases := []struct {
		d     d2.Difficulty
		lvl   int
		build string
		want  [4]Potion
	}{
		{d2.Normal, 40, b, PlanFor(d2.Barbarian, 40).Cols},
		{d2.Nightmare, 20, b, [4]Potion{hp, mp, mp, hp}},
		{d2.Nightmare, 40, b, [4]Potion{hp, mp, mp, rv}},
		{d2.Hell, 70, b, [4]Potion{rv, mp, mp, rv}},
		{d2.Hell, 70, "paladin_hammerdin_endgame90", PlanForDiff(d2.Barbarian, 70, d2.Hell).Cols},
	}
	for _, c := range cases {
		if got := PlanForBuild(d2.Barbarian, c.lvl, c.d, c.build).Cols; got != c.want {
			t.Errorf("%v lvl %d %s: %v want %v", c.d, c.lvl, c.build, got, c.want)
		}
	}
}

func TestManaHeavyThreshold(t *testing.T) {
	in := Input{Life: 100, MaxLife: 100, Mana: 30, MaxMana: 100, Class: d2.Barbarian, Diff: d2.Nightmare, Hostiles: 3,
		Belt: Belt{MP: 4, Rows: 1}, Build: "barbarian_warcry_endgame90", Gold: 100000}
	t0 := DefaultTuning()
	if d := PotionPolicy(in, t0); d.Drink != PotionMana {
		t.Errorf("mana-heavy at 30%%: %+v", d)
	}
	in.Build = ""
	if d := PotionPolicy(in, t0); d.Drink != PotionNone {
		t.Errorf("barbarian default never drinks mana: %+v", d)
	}
}

// The Go table and the data file must agree.
func TestManaHeavyMatchesJSON(t *testing.T) {
	raw, err := os.ReadFile("../../docs/plan/data/potions-belt.json")
	if err != nil {
		t.Skip(err)
	}
	var f struct {
		M struct {
			Builds []string `json:"builds"`
			By     []struct {
				Clvl [2]int   `json:"clvl"`
				Cols []string `json:"belt_cols"`
			} `json:"by_clvl"`
		} `json:"mana_heavy_builds"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.M.Builds) != len(manaHeavyBuilds) {
		t.Errorf("builds %v vs %v", f.M.Builds, manaHeavyBuilds)
	}
	for _, b := range f.M.Builds {
		if !ManaHeavyBuild(b) {
			t.Errorf("%s missing", b)
		}
	}
	name := map[string]Potion{"hp": PotionLife, "mp": PotionMana, "rejuv": PotionRejuv}
	for _, r := range f.M.By {
		for _, lvl := range []int{r.Clvl[0], r.Clvl[1]} {
			got := manaHeavyCols(lvl)
			for i, c := range r.Cols {
				if got[i] != name[c] {
					t.Errorf("clvl %d col %d: %v vs %s", lvl, i, got[i], c)
				}
			}
		}
	}
}
