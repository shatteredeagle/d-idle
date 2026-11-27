package decide

import "testing"

// Card necroepic: the shipped tables know every class the bot plays. The checks run over the Necromancer and the
// Paladin (a caster and a melee class), the same data rule for both.
func TestShippedTablesKnowNecromancer(t *testing.T) {
	cat := mustCat(t)
	pol, err := ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	pz, err := ShippedProposeTable()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{"necromancer", "paladin"} {
		if _, ok := cat.Classes[c]; !ok {
			t.Errorf("catalogue has no class %q", c)
		}
		if rows, err := pol.RowsFor(cat, c); err != nil || len(rows) == 0 {
			t.Errorf("%s: policy rows %v (%d)", c, err, len(rows))
		}
		if _, ok := pz.Classes[c]; !ok {
			t.Errorf("proposezone has no lead for %q", c)
		}
	}
	// A caster gets caster stance numbers: the Necromancer's equal the Sorceress's.
	row, err := pol.Row("stance")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range row.Options {
		if len(o.ClassKnobs) == 0 {
			continue
		}
		n, err := o.KnobsFor("necromancer")
		if err != nil {
			t.Fatal(err)
		}
		s, _ := o.KnobsFor("sorceress")
		for k, v := range s {
			if n[k] != v {
				t.Errorf("stance %s: necromancer %s=%v, caster table says %v", o.ID, k, n[k], v)
			}
		}
	}
}
