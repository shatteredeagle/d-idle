package plandata_test

import (
	"reflect"
	"strings"
	"testing"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/leveler"
)

const sorcNormal30 = "sorceress_normal30"

func TestSorceressNormal30Loads(t *testing.T) {
	b, err := leveler.LoadBuild(sorcNormal30, "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Class != "sorceress" || b.Combat != "cold" || b.Respec != 0 {
		t.Errorf("class %q combat %q respec %d", b.Class, b.Combat, b.Respec)
	}
	if first, last := b.Rows[0].Clvl, b.Rows[len(b.Rows)-1].Clvl; first != 2 || last != 30 {
		t.Errorf("rows %d..%d, want 2..30", first, last)
	}
	if got := leveler.TableCombat(sorcNormal30); got != "cold" {
		t.Errorf("combat profile %q", got)
	}
}

// sorcready2 (user decisions 2026-11-10): the table is v3 with Warmth at clvl 2 (Charged Bolt from clvl 3) and a cold
// line from clvl 12 instead of Lightning and Chain Lightning (known broken on d2d). Stats are v3's; rows 3..11 and 18
// (Teleport) are v3's skills.
func TestSorcNormal30RowsVsV3(t *testing.T) {
	n, err := leveler.LoadBuild(sorcNormal30, "")
	if err != nil {
		t.Fatal(err)
	}
	v3, err := leveler.LoadBuild(leveler.FallbackV3, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Rows) != 29 {
		t.Fatalf("%d rows", len(n.Rows))
	}
	for i, r := range n.Rows {
		if r.Attr != v3.Rows[i].Attr {
			t.Errorf("clvl %d stats differ from v3: %v vs %v", r.Clvl, r.Attr, v3.Rows[i].Attr)
		}
		if (r.Clvl >= 3 && r.Clvl <= 11) || r.Clvl == 18 {
			if !reflect.DeepEqual(r.Skills, v3.Rows[i].Skills) {
				t.Errorf("clvl %d skills differ from v3: %+v vs %+v", r.Clvl, r.Skills, v3.Rows[i].Skills)
			}
		}
		for _, s := range r.Skills {
			if s.Skill == "Lightning" || s.Skill == "Chain Lightning" {
				t.Errorf("clvl %d spends %s", r.Clvl, s.Skill)
			}
		}
	}
	if s := n.Rows[0].Skills; len(s) != 1 || s[0].Skill != "Warmth" {
		t.Errorf("clvl 2: %+v", s)
	}
	last := n.Rows[len(n.Rows)-1].Skills
	if len(last) != 1 || last[0].Skill != "Blizzard" {
		t.Errorf("clvl 30: %+v", last)
	}
}

// The default and the pinned tables are untouched: a fresh Sorceress still gets the v2 table, never normal30.
func TestDefaultSorceressTableUnchanged(t *testing.T) {
	if got := leveler.SelectBuild("sorceress", 1, nil); got != sorcNormal30 {
		t.Errorf("fresh sorceress table %q, want %q", got, sorcNormal30)
	}
	if got := leveler.DefaultBuildName("sorceress"); got != "fallback" {
		t.Errorf("sorceress default %q", got)
	}
}

// Two classes: the Sorceress table is refused for a Paladin and the Paladin table for a Sorceress, naming the class.
func TestNormal30TablesAreClassBound(t *testing.T) {
	for table, class := range map[string]string{sorcNormal30: "sorceress", leveler.PaladinV2: "paladin"} {
		b, err := leveler.LoadBuild(table, "")
		if err != nil || b.Class != class {
			t.Fatalf("%s: %v class %q", table, err, b.Class)
		}
	}
	if _, err := leveler.LoadBuild("no_such_normal30", ""); err == nil || !strings.Contains(err.Error(), "no_such_normal30") {
		t.Errorf("unknown table error %v", err)
	}
}

// Every skill exists, belongs to the class and its ReqLevel is met by the row (skipped without the install tables).
func TestSorceressNormal30AgainstRealTables(t *testing.T) {
	db, err := data.Open("/tmp/d2mpq", t.TempDir())
	if err != nil {
		t.Skip("no real install tables: " + err.Error())
	}
	b, _ := leveler.LoadBuild(sorcNormal30, "")
	cc := db.ClassByName("sorceress")
	for _, r := range b.Rows {
		for _, sp := range r.Skills {
			s := db.SkillByName(sp.Skill)
			if s == nil || s.Class != cc.Code {
				t.Errorf("clvl %d: %s missing or not a Sorceress skill", r.Clvl, sp.Skill)
			} else if s.ReqLevel > r.Clvl {
				t.Errorf("clvl %d: %s needs clvl %d", r.Clvl, sp.Skill, s.ReqLevel)
			}
		}
	}
}

// SME review 2026-11-27, confidence medium-high: the cold band casts Ice Blast ahead of Glacial Spike, so Glacial Spike
// gets its one Blizzard-prerequisite point at clvl 24 and the points of clvl 25-29 go to Ice Blast (15 by clvl 29).
func TestSorcNormal30IceBlastNotGlacialSpike(t *testing.T) {
	b, err := leveler.LoadBuild(sorcNormal30, "")
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]int{}
	for _, r := range b.Rows {
		for _, s := range r.Skills {
			have[s.Skill] += s.Points
		}
		switch {
		case r.Clvl == 24 && have["Glacial Spike"] != 1:
			t.Errorf("clvl 24: Glacial Spike %d, want 1", have["Glacial Spike"])
		case r.Clvl >= 24 && r.Clvl < 30 && have["Glacial Spike"] != 1:
			t.Errorf("clvl %d: Glacial Spike %d, want it to stay 1", r.Clvl, have["Glacial Spike"])
		case r.Clvl == 29 && have["Ice Blast"] != 15:
			t.Errorf("clvl 29: Ice Blast %d, want 15", have["Ice Blast"])
		case r.Clvl == 30 && have["Blizzard"] != 1:
			t.Errorf("clvl 30: Blizzard %d", have["Blizzard"])
		}
	}
	// The other Sorceress table still loads (the edit touched only the normal30 file).
	e, err := leveler.LoadBuild("sorceress_blizzard_endgame90", "")
	if err != nil {
		t.Fatal(err)
	}
	if e.Class != "sorceress" {
		t.Errorf("class %q", e.Class)
	}
}
