package points

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/leveler"
)

// ladderRunewords are the 23 words that are ladder-only in 1.14d and cannot be made in an offline game (design review
// 2026-11-19, 03 section 1; V for Spirit and Insight from docs/sme/builds-gear.md, the rest S).
var ladderRunewords = map[string]bool{}

func init() {
	for _, n := range strings.Split("Spirit,Insight,Fortitude,Grief,Infinity,Faith,Harmony,Ice,Dream,Dragon,Phoenix,Pride,Edge,Obedience,Oath,Death,Destruction,Brand,Last Wish,Lawbringer,Rift,Wrath,Voice of Reason", ",") {
		ladderRunewords[n] = true
	}
}

func shippedTables(t *testing.T) map[string]*leveler.Build {
	t.Helper()
	files, err := filepath.Glob("../plandata/build-*.json")
	if err != nil || len(files) < 10 {
		t.Fatalf("plan tables not found: %v %d", err, len(files))
	}
	out := map[string]*leveler.Build{}
	for _, f := range files {
		if filepath.Base(f) == "build-catalog.json" {
			continue // the build text catalogue, not a plan table
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		tb, err := leveler.ParseBuild(b)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		out[filepath.Base(f)] = tb
	}
	return out
}

// TestLadderTagsOnEndgameTables: the five endgame tables that set stat targets by Spirit, Insight or Fortitude say so in
// data (`ladder_only`), every tagged word is a known ladder word, and a table that names a ladder word in its text is tagged.
func TestLadderTagsOnEndgameTables(t *testing.T) {
	want := map[string][]string{
		"build-barbarian_warcry_endgame90.json":     {"Spirit", "Insight", "Fortitude"},
		"build-druid_tornado_endgame90.json":        {"Spirit"},
		"build-necromancer_summoner_endgame90.json": {"Spirit"},
		"build-paladin_hammerdin_endgame90.json":    {"Spirit", "Fortitude"},
		"build-sorceress_blizzard_endgame90.json":   {"Spirit"},
	}
	tables := shippedTables(t)
	for file, words := range want {
		tb := tables[file]
		if tb == nil {
			t.Fatalf("%s missing", file)
		}
		if strings.Join(tb.LadderOnly, ",") != strings.Join(words, ",") {
			t.Errorf("%s: ladder_only %v, want %v", file, tb.LadderOnly, words)
		}
		for _, w := range tb.LadderOnly {
			if !ladderRunewords[w] {
				t.Errorf("%s tags %q, which is not a known ladder word", file, w)
			}
		}
	}
	for file, tb := range tables {
		if _, tagged := want[file]; !tagged && len(tb.LadderOnly) > 0 {
			t.Errorf("%s has an unexpected ladder_only %v", file, tb.LadderOnly)
		}
	}
}

// TestHammerdinTableBreaksThePerLevelCap pins the one fact the build-file prototype found: the hammerdin table banks its
// hammer points and spends 11 of them at clvl 18, where a skill that needs level 18 takes one (kolbot AutoSkill rule). The
// required level 18 is V (the table's own tags: "reqlevel 18, V skills.txt"). d2d does not enforce the rule (V: fork
// components/rules/rules.hpp can_learn), so the table stays as it is; the real-data sweep of every table is
// TestPerLevelCapOnShippedTablesReal. The rule is no longer a knob (retired); only the table checker keeps it.
func TestHammerdinTableBreaksThePerLevelCap(t *testing.T) {
	tb := shippedTables(t)["build-paladin_hammerdin_endgame90.json"]
	var rows []TableRow
	for _, r := range tb.Rows {
		tr := TableRow{Clvl: r.Clvl, Skills: map[string]int{}}
		for _, s := range r.Skills {
			tr.Skills[s.Skill] += s.Points
		}
		rows = append(rows, tr)
	}
	v := TableViolations(rows, func(s string) (int, bool) { return 18, s == "Blessed Hammer" })
	if len(v) != 1 || !strings.Contains(v[0], "clvl 18: Blessed Hammer has 11 points, the cap is 1") {
		t.Errorf("%v", v)
	}
}

// TestPerLevelCapOnShippedTablesReal sweeps every shipped table against the real skills.txt (skipped without the install
// tables at /tmp/d2mpq, run on the worker host) and logs each first break; it fails only when a skill has no required level.
func TestPerLevelCapOnShippedTablesReal(t *testing.T) {
	if _, err := os.Stat("/tmp/d2mpq/d2data.mpq"); err != nil {
		t.Skip("no real install at /tmp/d2mpq (run on the worker host)")
	}
	db, err := data.Open("/tmp/d2mpq", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	breaks := 0
	for file, tb := range shippedTables(t) {
		var rows []TableRow
		for _, r := range tb.Rows {
			tr := TableRow{Clvl: r.Clvl, Skills: map[string]int{}}
			for _, s := range r.Skills {
				tr.Skills[s.Skill] += s.Points
			}
			rows = append(rows, tr)
		}
		for _, line := range TableViolations(rows, func(n string) (int, bool) {
			if s := db.SkillByName(n); s != nil {
				return s.ReqLevel, true
			}
			t.Errorf("%s: unknown skill %q", file, n)
			return 0, false
		}) {
			breaks++
			t.Logf("%s: %s", file, line)
		}
	}
	t.Logf("%d first breaks of the per-level cap over the shipped tables", breaks)
}
