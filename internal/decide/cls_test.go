package decide

import "testing"

// Card cls1: the shipped tables know the Druid and the Barbarian. The same checks run over a caster and a melee class,
// each against the class whose says-text it shares (the stance sentence names one radius per archetype).
func TestShippedTablesKnowDruidAndBarbarian(t *testing.T) {
	cat := mustCat(t)
	pol, err := ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	pz, err := ShippedProposeTable()
	if err != nil {
		t.Fatal(err)
	}
	row, err := pol.Row("stance")
	if err != nil {
		t.Fatal(err)
	}
	for class, like := range map[string]string{"druid": "sorceress", "barbarian": "paladin"} {
		cl, ok := cat.Classes[class]
		if !ok || len(cl.Bands["mana"]) != 2 {
			t.Errorf("catalogue: class %q has no mana band", class)
		}
		if rows, err := pol.RowsFor(cat, class); err != nil || len(rows) == 0 {
			t.Errorf("%s: policy rows %v (%d)", class, err, len(rows))
		}
		if _, ok := pz.Classes[class]; !ok {
			t.Errorf("proposezone has no lead for %q", class)
		}
		for _, o := range row.Options {
			if len(o.ClassKnobs) == 0 {
				continue
			}
			got, err := o.KnobsFor(class)
			if err != nil {
				t.Fatal(err)
			}
			want, _ := o.KnobsFor(like)
			if len(got) != len(want) {
				t.Errorf("stance %s: %s sets %d knobs, %s %d", o.ID, class, len(got), like, len(want))
			}
			// The option's sentence promises these per archetype; the rest is the class's own judgment.
			for _, k := range []string{"bot.clearRadius", "combat.drinkLifePct", "combat.chickenPct", "combat.townPotionMin"} {
				if got[k] != want[k] {
					t.Errorf("stance %s: %s %s=%v, the sentence says %v", o.ID, class, k, got[k], want[k])
				}
			}
		}
	}
	// A class without rows is still an error that names it, never another class's numbers.
	for _, o := range row.Options {
		if len(o.ClassKnobs) == 0 {
			continue
		}
		if _, err := o.KnobsFor("mystic"); err == nil {
			t.Errorf("stance %s: mystic has no rows and got knobs", o.ID)
		}
	}
}
