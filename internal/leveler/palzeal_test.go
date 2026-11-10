package leveler

import (
	"testing"
	"testing/fstest"
)

// PALZEAL: a data table that names itself is selectable with no Go entry, and the default and endgame choice of
// every class is exactly what it was before the seam (pinned below).
func TestZealotLiteSelectableByName(t *testing.T) {
	b, err := LoadBuild("paladin-zealot-lite", "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Class != "paladin" || b.Name != "paladin_zealot_lite" || len(b.Rows) != 29 {
		t.Errorf("%s class %s rows %d", b.Name, b.Class, len(b.Rows))
	}
	for _, c := range []string{"paladin", "sorceress", "amazon", "necromancer", "druid", "assassin", "barbarian"} {
		if DefaultBuildName(c) == "paladin-zealot-lite" || EndgameBuildName(c) == "paladin-zealot-lite" || SelectBuild(c, 1, nil) == "paladin-zealot-lite" {
			t.Errorf("%s: zealot-lite became a default", c)
		}
	}
}

func TestSelectionUnchangedForAllClasses(t *testing.T) {
	cases := []struct{ class, def, endgame, fresh, existing string }{
		{"sorceress", "fallback", "sorceress_blizzard_endgame90", SorceressNormal30, "fallback"},
		{"paladin", "paladin-normal30", "paladin_hammerdin_endgame90", PaladinV2, "paladin-normal30"},
		{"necromancer", "necromancer_summoner_normal30", "necromancer_summoner_endgame90", "necromancer_summoner_normal30", "necromancer_summoner_normal30"},
		{"amazon", "amazon_bow_normal30", "amazon_multishot_endgame90", "amazon_bow_normal30", "amazon_bow_normal30"},
		{"druid", "druid_fire_normal30", "druid_tornado_endgame90", "druid_fire_normal30", "druid_fire_normal30"},
		{"assassin", "assassin_fireblast_normal30", "assassin_sentry_endgame90", "assassin_fireblast_normal30", "assassin_fireblast_normal30"},
		{"barbarian", "barbarian_melee_normal30", "barbarian_warcry_endgame90", "barbarian_melee_normal30", "barbarian_melee_normal30"},
	}
	for _, c := range cases {
		if got := DefaultBuildName(c.class); got != c.def {
			t.Errorf("%s default %q want %q", c.class, got, c.def)
		}
		if got := EndgameBuildName(c.class); got != c.endgame {
			t.Errorf("%s endgame %q want %q", c.class, got, c.endgame)
		}
		if got := ResolveBuild(EndgameAlias, c.class); got != c.endgame {
			t.Errorf("%s endgame alias %q", c.class, got)
		}
		if got := SelectBuild(c.class, 1, nil); got != c.fresh {
			t.Errorf("%s fresh %q want %q", c.class, got, c.fresh)
		}
		if got := SelectBuild(c.class, 20, map[string]int{"Static Field": 1, "Holy Fire": 1}); got != c.existing {
			t.Errorf("%s existing %q want %q", c.class, got, c.existing)
		}
		if _, err := LoadBuild(c.def, ""); err != nil {
			t.Errorf("%s: %v", c.def, err)
		}
	}
}

// The metadata fills gaps only: a taken selector, a class with a default or an endgame table keeps its entry.
func TestRegisterDataTablesFillsGapsOnly(t *testing.T) {
	tbl := func(sel, extra string) *fstest.MapFile {
		return &fstest.MapFile{Data: []byte(`{"build":"x","selector":"` + sel + `"` + extra + `}`)}
	}
	fsys := fstest.MapFS{
		"build-a.json": tbl("zz-new", `,"endgame_for":"Paladin","default_for_class":"sorceress"`),
		"build-b.json": tbl("fallback", ""),
		"other.json":   tbl("zz-ignored", ""),
	}
	registerDataTables(fsys)
	defer delete(buildFiles, "zz-new")
	if buildFiles["zz-new"] != "build-a.json" || buildFiles["fallback"] != "build-primary-blizzard.json" && buildFiles["fallback"] != "build-fallback-lightning.json" {
		t.Errorf("buildFiles: %v %v", buildFiles["zz-new"], buildFiles["fallback"])
	}
	if _, ok := buildFiles["zz-ignored"]; ok {
		t.Error("non build- file registered")
	}
	if EndgameBuildName("paladin") != "paladin_hammerdin_endgame90" || DefaultBuildName("sorceress") != "fallback" {
		t.Error("metadata overrode an existing class choice")
	}
}

// PALZEAL: the Zealot-lite table is legal under the skills.txt rules of the probe fixture (checked against the real
// skills.txt by TestPalProbeRulesMatchInstall where an install exists), with and without the Den point.
func TestZealotLiteLegal(t *testing.T) {
	for _, den := range []int{0, 5} {
		r := simPaladin(t, "paladin-zealot-lite", den)
		if len(r.illegal) != 0 {
			t.Errorf("den %d: illegal picks %v", den, r.illegal)
		}
		if r.skills["Zeal"] != 4 || r.skills["Sacrifice"] != 1 || r.skills["Smite"] != 1 || r.skills["Prayer"] != 1 || r.skills["Might"] < 9 || r.skills["Holy Fire"] != 2 || r.skills["Defiance"] != 1 || r.skills["Vigor"] != 0 {
			t.Errorf("den %d: skills %v", den, r.skills)
		}
		if z := r.firstAt["Zeal"]; z < 12 || z > 13-den/5 || r.firstAt["Sacrifice"] >= r.firstAt["Zeal"] {
			t.Errorf("den %d: Sacrifice %d Zeal %d", den, r.firstAt["Sacrifice"], r.firstAt["Zeal"])
		}
		t.Logf("den %d: spent %d points, skills %v, stuck %v", den, r.spent, r.skills, r.stuck)
	}
	sk := skillsAt(t, "paladin-zealot-lite")
	if sk[30]["Holy Fire"] == 0 || sk[30]["Blessed Hammer"] == 0 {
		t.Errorf("clvl 30 skills %v", sk[30])
	}
}
