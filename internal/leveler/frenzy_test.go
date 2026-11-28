package leveler

import "testing"

// barb1: the Barbarian Frenzy table is selectable by name, lands its points as planned at clvl 30 and changes no
// default; a second and third class prove the selector registration changed nothing else.
func TestBarbarianFrenzyBuildSelectableAndDefaultsUnchanged(t *testing.T) {
	lb, err := LoadBuild("barbarian_frenzy_guide", "")
	if err != nil {
		t.Fatal(err)
	}
	if lb.Class != "barbarian" || lb.Combat != "frenzy" {
		t.Fatalf("class %q combat %q", lb.Class, lb.Combat)
	}
	if TableCombat("barbarian_frenzy_guide") != "frenzy" || TableCombat("barbarian_melee_normal30") != "barbarian" {
		t.Errorf("combat lines: guide %q, normal30 %q", TableCombat("barbarian_frenzy_guide"), TableCombat("barbarian_melee_normal30"))
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
	// Bash, Howl, Shout and Double Throw are the prerequisite points of Double Swing, Battle Orders and Frenzy.
	want := map[string]int{"Bash": 1, "Axe Mastery": 3, "Double Swing": 13, "Howl": 1, "Shout": 1, "Double Throw": 1, "Increased Stamina": 2, "Iron Skin": 3, "Frenzy": 7, "Battle Orders": 1}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %d points at clvl 30, want %d", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("skills %v, want exactly %v", got, want)
	}
	// Defaults: nothing selects the new table unless named, for the Barbarian and for three other classes.
	if n := DefaultBuildName("barbarian"); n != "barbarian_melee_normal30" {
		t.Errorf("default barbarian build %q", n)
	}
	if n := EndgameBuildName("barbarian"); n != "barbarian_warcry_endgame90" {
		t.Errorf("endgame barbarian build %q", n)
	}
	if n := SelectBuild("barbarian", 1, nil); n == "barbarian_frenzy_guide" {
		t.Errorf("a fresh barbarian got the opt-in table")
	}
	if n := ResolveBuild("barbarian_frenzy_guide", "barbarian"); n != "barbarian_frenzy_guide" {
		t.Errorf("explicit selector resolved to %q", n)
	}
	if n := DefaultBuildName("paladin"); n != "paladin-normal30" {
		t.Errorf("default paladin build %q", n)
	}
	if n := DefaultBuildName("necromancer"); n != "necromancer_summoner_normal30" {
		t.Errorf("default necromancer build %q", n)
	}
	if n := DefaultBuildName("sorceress"); n != "fallback" {
		t.Errorf("default sorceress build %q", n)
	}
	if _, err := LoadBuild("barbarian_frenzy_nope", ""); err == nil {
		t.Error("an unknown table must be an error")
	}
}
