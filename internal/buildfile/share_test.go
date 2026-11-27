package buildfile

import (
	"reflect"
	"testing"

	"d2lod-bot/internal/leveler"
)

// Property over the shipped tables: export -> JSON -> parse is the identity on the file, and export -> import -> export
// is a fixed point (same goals, same stat goals, same quest points), so a shared file loses nothing on the way through
// a plan table.
func TestExportImportIdentity(t *testing.T) {
	for _, name := range PlanNames() {
		rules := looseRulesFor(t, name)
		b1, _, err := ExportPlan(name, rules)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := Marshal(b1)
		if err != nil {
			t.Fatal(err)
		}
		back, probs, err := ReadShared(raw, rules)
		if err != nil || len(probs) > 0 {
			t.Fatalf("%s: shared file: %v %v", name, err, probs)
		}
		if !reflect.DeepEqual(b1, back) {
			t.Errorf("%s: parse(marshal(file)) != file", name)
		}
		tab, err := ImportPlan(b1, rules)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		lb, err := leveler.ParseBuild(tab)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		b2, _ := FromRows(lb.Rows, lb.Class, rules)
		if !reflect.DeepEqual(b1.Skills, b2.Skills) || !reflect.DeepEqual(b1.Stats, b2.Stats) || !reflect.DeepEqual(b1.Quest, b2.Quest) {
			t.Errorf("%s: export(import(export(table))) differs from export(table)", name)
		}
	}
}

func looseRulesFor(t *testing.T, name string) Rules {
	t.Helper()
	b, _, err := ExportPlan(name, looseRules{})
	if err != nil {
		t.Fatal(err)
	}
	return looseRules{b.Class}
}

// Two classes, one list each, through the importer and the validator; wrong-class and bad lists are reported.
func TestImportKolbotBuildTwoClasses(t *testing.T) {
	pal, probs, err := ImportKolbotBuild("hammerdin-list", "pal",
		[]byte(`[["Holy Bolt",1],["Blessed Hammer",20,true],["Concentration",20]]`),
		[]byte(`[["strength",60],["dexterity",70],["vitality","all"]]`), fixtureRules{})
	if err != nil || len(probs) > 0 || pal.Class != "paladin" || pal.Source != "kolbot-list" {
		t.Fatalf("paladin: %v %v %+v", err, probs, pal)
	}
	sor, probs, err := ImportKolbotBuild("fire-list", "sorceress",
		[]byte(`[["Fire Bolt",5],["Fire Ball",20],["Fire Bolt",20,false]]`), nil, fixtureRules{})
	if err != nil || len(probs) > 0 || sor.Class != "sorceress" {
		t.Fatalf("sorceress: %v %v", err, probs)
	}
	for _, b := range []*Build{pal, sor} {
		if _, err := ImportPlan(b, fixtureRules{}); err != nil {
			t.Errorf("%s: %v", b.Name, err)
		}
	}
	// A Sorceress skill in a Paladin list is a problem, not a silent drop.
	_, probs, err = ImportKolbotBuild("x", "paladin", []byte(`[["Fire Ball",20]]`), nil, fixtureRules{})
	if err != nil || !onlyCode(probs, "skill_wrong_class") {
		t.Errorf("wrong class: %v %v", err, probs)
	}
	if _, _, err = ImportKolbotBuild("x", "wizard", nil, nil, fixtureRules{}); err == nil {
		t.Error("unknown class accepted")
	}
	if _, _, err = ImportKolbotBuild("x", "pal", []byte(`[["Might"]]`), nil, fixtureRules{}); err == nil {
		t.Error("malformed entry accepted")
	}
	if _, _, err = ImportKolbotBuild("x", "pal", []byte(`[[36,20]]`), nil, fixtureRules{}); err == nil {
		t.Error("numeric skill id accepted")
	}
}
