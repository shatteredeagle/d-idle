package leveler

import "testing"

// hammer1: the Paladin Hammerdin table is selectable by name, lands its points as planned at clvl 30 and changes no
// default; a second class proves the selector registration changed nothing else.
func TestPaladinHammerdinBuildSelectableAndDefaultsUnchanged(t *testing.T) {
	lb, err := LoadBuild("paladin_hammerdin_guide", "")
	if err != nil {
		t.Fatal(err)
	}
	if lb.Class != "paladin" || lb.Combat != "hammerdin" {
		t.Fatalf("class %q combat %q", lb.Class, lb.Combat)
	}
	if TableCombat("paladin_hammerdin_guide") != "hammerdin" || TableCombat("paladin-normal30") != "paladin" {
		t.Errorf("combat lines: guide %q, normal30 %q", TableCombat("paladin_hammerdin_guide"), TableCombat("paladin-normal30"))
	}
	got := map[string]int{}
	for _, r := range lb.Rows {
		if r.Clvl > 30 {
			break
		}
		for _, s := range r.Skills {
			got[s.Skill] += s.Points
		}
	}
	// Holy Bolt, Charge and Blessed Aim are the prerequisite points of Blessed Hammer, Holy Shield and Concentration.
	want := map[string]int{"Might": 1, "Smite": 6, "Holy Bolt": 1, "Blessed Hammer": 14, "Blessed Aim": 1, "Concentration": 8, "Charge": 1, "Holy Shield": 1}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %d points at clvl 30, want %d", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("skills %v, want exactly %v", got, want)
	}
	// Defaults: nothing selects the new table unless named, for the Paladin and for two other classes.
	if n := DefaultBuildName("paladin"); n != "paladin-normal30" {
		t.Errorf("default paladin build %q", n)
	}
	if n := EndgameBuildName("paladin"); n != "paladin_hammerdin_endgame90" {
		t.Errorf("endgame paladin build %q", n)
	}
	if n := SelectBuild("paladin", 1, nil); n == "paladin_hammerdin_guide" {
		t.Errorf("a fresh paladin got the opt-in table")
	}
	if n := ResolveBuild("paladin_hammerdin_guide", "paladin"); n != "paladin_hammerdin_guide" {
		t.Errorf("explicit selector resolved to %q", n)
	}
	if n := DefaultBuildName("necromancer"); n != "necromancer_summoner_normal30" {
		t.Errorf("default necromancer build %q", n)
	}
	if n := DefaultBuildName("sorceress"); n != "fallback" {
		t.Errorf("default sorceress build %q", n)
	}
	if _, err := LoadBuild("paladin_hammerdin_nope", ""); err == nil {
		t.Error("an unknown table must be an error")
	}
}
