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
	if b.Class != "sorceress" || b.Combat != "lightning" || b.Respec != 0 {
		t.Errorf("class %q combat %q respec %d", b.Class, b.Combat, b.Respec)
	}
	if first, last := b.Rows[0].Clvl, b.Rows[len(b.Rows)-1].Clvl; first != 2 || last != 30 {
		t.Errorf("rows %d..%d, want 2..30", first, last)
	}
	if got := leveler.TableCombat(sorcNormal30); got != "lightning" {
		t.Errorf("combat profile %q", got)
	}
}

// V3 stays the approved table; the new table must spend the same points for clvl 1..30 (no difference to document).
func TestSorceressNormal30EqualsV3(t *testing.T) {
	n, err := leveler.LoadBuild(sorcNormal30, "")
	if err != nil {
		t.Fatal(err)
	}
	v3, err := leveler.LoadBuild(leveler.FallbackV3, "")
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range n.Rows {
		if !reflect.DeepEqual(r, v3.Rows[i]) {
			t.Errorf("clvl %d differs from v3: %+v vs %+v", r.Clvl, r, v3.Rows[i])
		}
	}
	if len(n.Rows) != 29 {
		t.Errorf("%d rows", len(n.Rows))
	}
}

// The default and the pinned tables are untouched: a fresh Sorceress still gets the v2 table, never normal30.
func TestDefaultSorceressTableUnchanged(t *testing.T) {
	if got := leveler.SelectBuild("sorceress", 1, nil); got != leveler.FallbackV2 {
		t.Errorf("fresh sorceress table %q, want %q", got, leveler.FallbackV2)
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
