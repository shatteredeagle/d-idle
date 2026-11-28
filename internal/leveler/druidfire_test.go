package leveler

import "testing"

// druid1: the Druid fire table is selectable by name, lands its points as planned at clvl 30 and changes no default;
// three other classes prove the selector registration changed nothing else.
func TestDruidFireBuildSelectableAndDefaultsUnchanged(t *testing.T) {
	lb, err := LoadBuild("druid_fire_guide", "")
	if err != nil {
		t.Fatal(err)
	}
	if lb.Class != "druid" || lb.Combat != "fire" {
		t.Fatalf("class %q combat %q", lb.Class, lb.Combat)
	}
	if TableCombat("druid_fire_guide") != "fire" || TableCombat("druid_fire_normal30") != "druid" {
		t.Errorf("combat lines: guide %q, normal30 %q", TableCombat("druid_fire_guide"), TableCombat("druid_fire_normal30"))
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
	// Molten Boulder is Eruption's prerequisite, Arctic Blast Cyclone Armor's; the 33 points are 29 level points plus 4 quest points.
	want := map[string]int{"Firestorm": 13, "Molten Boulder": 6, "Eruption": 10, "Oak Sage": 1, "Arctic Blast": 1, "Cyclone Armor": 1, "Volcano": 1}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %d points at clvl 30, want %d", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("skills %v, want exactly %v", got, want)
	}
	for _, s := range []string{"Tornado", "Hurricane", "Armageddon", "Wearwolf", "Wearbear"} {
		if got[s] != 0 {
			t.Errorf("unsupported skill %s has %d points", s, got[s])
		}
	}
	// Defaults: nothing selects the new table unless named, for the Druid and for four other classes.
	for class, want := range map[string]string{"druid": "druid_fire_normal30", "barbarian": "barbarian_melee_normal30", "paladin": "paladin-normal30",
		"necromancer": "necromancer_summoner_normal30", "sorceress": "fallback"} {
		if n := DefaultBuildName(class); n != want {
			t.Errorf("default %s build %q, want %q", class, n, want)
		}
	}
	if n := EndgameBuildName("druid"); n != "druid_tornado_endgame90" {
		t.Errorf("endgame druid build %q", n)
	}
	if n := SelectBuild("druid", 1, nil); n == "druid_fire_guide" {
		t.Errorf("a fresh druid got the opt-in table")
	}
	if n := ResolveBuild("druid_fire_guide", "druid"); n != "druid_fire_guide" {
		t.Errorf("explicit selector resolved to %q", n)
	}
	if _, err := LoadBuild("druid_fire_nope", ""); err == nil {
		t.Error("an unknown table must be an error")
	}
}
