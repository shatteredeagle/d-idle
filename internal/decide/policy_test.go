package decide

import (
	"strings"
	"testing"
)

func TestShippedPolicyTable(t *testing.T) {
	cat := mustCat(t)
	tbl, err := ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.Rows) < 10 {
		t.Fatalf("rows = %d", len(tbl.Rows))
	}
	sub := 0
	for _, r := range tbl.Rows {
		for _, o := range r.Options {
			if o.Suboptimal {
				sub++
			}
		}
	}
	if sub < 8 {
		t.Fatalf("suboptimal options = %d: the sheet must allow bad choices", sub)
	}
}

func TestPolicyTableErrors(t *testing.T) {
	cat := mustCat(t)
	row := func(extra string) string {
		return `{"version":1,"rows":[{"id":"r","label":"L","help":"H","default":"d",` + extra + `}]}`
	}
	cases := map[string]struct{ doc, want string }{
		"unknown point":  {row(`"options":[{"id":"d","label":"D"},{"id":"x","label":"X","rules":[{"point":"t9.nope","answer":"a"}]}]`), "t9.nope"},
		"unknown answer": {row(`"options":[{"id":"d","label":"D"},{"id":"x","label":"X","rules":[{"point":"t1.repair","answer":"melt"}]}]`), `"melt"`},
		"unknown field":  {row(`"options":[{"id":"d","label":"D"},{"id":"x","label":"X","rules":[{"point":"t1.repair","when":{"zzz":"1"},"answer":"skip"}]}]`), `"zzz"`},
		"unknown class":  {row(`"classes":["druid"],"options":[{"id":"d","label":"D"}]`), `"druid"`},
		"no default":     {row(`"options":[{"id":"x","label":"X"}]`), `no option "d"`},
		"default acts":   {row(`"options":[{"id":"d","label":"D","rules":[{"point":"t1.repair","answer":"skip"}]}]`), "change nothing"},
		"no risk":        {row(`"options":[{"id":"d","label":"D"},{"id":"x","label":"X","suboptimal":true}]`), "risk line"},
		"record no note": {row(`"options":[{"id":"d","label":"D"},{"id":"x","label":"X","applies":"record"}]`), "needs a note"},
	}
	for name, c := range cases {
		_, err := ParsePolicy([]byte(c.doc), cat)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want it to name %s", name, err, c.want)
		}
	}
	two := `{"version":1,"rows":[
	 {"id":"a","label":"L","help":"H","default":"d","options":[{"id":"d","label":"D"},{"id":"x","label":"X","rules":[{"point":"t1.repair","answer":"skip"}]}]},
	 {"id":"b","label":"L","help":"H","default":"d","options":[{"id":"d","label":"D"},{"id":"x","label":"X","rules":[{"point":"t1.repair","answer":"repair"}]}]}]}`
	if _, err := ParsePolicy([]byte(two), cat); err == nil || !strings.Contains(err.Error(), `"t1.repair"`) {
		t.Errorf("two rows on one point: %v", err)
	}
}

func TestRowsForClass(t *testing.T) {
	cat := mustCat(t)
	tbl, _ := ShippedPolicy(cat)
	for _, c := range []string{"sorceress", "paladin"} {
		rows, err := tbl.RowsFor(cat, c)
		if err != nil || len(rows) == 0 {
			t.Fatalf("%s: %v %d", c, err, len(rows))
		}
	}
	if _, err := tbl.RowsFor(cat, "necromancer"); err == nil || !strings.Contains(err.Error(), `"necromancer"`) {
		t.Fatalf("unknown class: %v", err)
	}
	if _, err := tbl.Row("nope"); err == nil || !strings.Contains(err.Error(), `"nope"`) {
		t.Fatalf("unknown row: %v", err)
	}
}

// A policy answers before any card, for both builds, survives a reload and a later option replaces it.
func TestPolicyAnswersAndPersists(t *testing.T) {
	cat := mustCat(t)
	tbl, _ := ShippedPolicy(cat)
	dir := t.TempDir()
	s, err := Open(dir, "pol", cat, &ManualClock{})
	if err != nil {
		t.Fatal(err)
	}
	repair, _ := tbl.Row("repair")
	never, _ := repair.Option("never")
	if err := s.SetPolicy("repair", "never", never.Rules); err != nil {
		t.Fatal(err)
	}
	for _, class := range []string{"sorceress", "paladin"} {
		o := baseObs(cat, class).with(nil, map[string]float64{"durability": 5})
		ctx := mustBuild(t, cat, "t1.repair", o)
		d, err := s.AskWithDefault(ctx, "repair")
		if err != nil {
			t.Fatal(err)
		}
		if d.Answer != "skip" || d.Source != "policy" || d.Thought != nil {
			t.Fatalf("%s: %+v, want skip from policy and no card", class, d)
		}
	}
	if n := len(s.OpenThoughts()); n != 0 {
		t.Fatalf("open thoughts = %d", n)
	}
	// reload: same answer
	s2, err := Open(dir, "pol", cat, &ManualClock{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := mustBuild(t, cat, "t1.repair", baseObs(cat, "paladin"))
	if d, _ := s2.AskWithDefault(ctx, "repair"); d.Answer != "skip" || d.Source != "policy" {
		t.Fatalf("after reload: %+v", d)
	}
	if s2.PolicyOption("repair") != "never" || len(s2.PolicyLog()) != 1 {
		t.Fatalf("option %q log %d", s2.PolicyOption("repair"), len(s2.PolicyLog()))
	}
	// back to the default: no rules, the legacy answer acts again (and a card may open)
	if err := s2.SetPolicy("repair", "default", nil); err != nil {
		t.Fatal(err)
	}
	if d, _ := s2.AskWithDefault(ctx, "repair"); d.Answer != "repair" || d.Source == "policy" {
		t.Fatalf("after default: %+v", d)
	}
	if got := s2.Policies(); len(got) != 1 || got[0].Option != "default" {
		t.Fatalf("policies = %+v", got)
	}
	// a rule naming nothing real is refused and logs nothing
	n := len(s2.Log())
	if err := s2.SetPolicy("repair", "x", []PolicyRule{{Point: "t1.repair", Answer: "melt"}}); err == nil {
		t.Fatal("bad rule accepted")
	}
	if len(s2.Log()) != n {
		t.Fatal("a refused policy was logged")
	}
}

// The "low" option is a two-rule list: the first match wins.
func TestPolicyFirstRuleWins(t *testing.T) {
	cat := mustCat(t)
	tbl, _ := ShippedPolicy(cat)
	s, _ := New(cat, "pol", &ManualClock{})
	row, _ := tbl.Row("repair")
	low, _ := row.Option("low")
	if err := s.SetPolicy("repair", "low", low.Rules); err != nil {
		t.Fatal(err)
	}
	for dur, want := range map[float64]string{10: "repair", 25: "skip", 90: "skip"} {
		ctx := mustBuild(t, cat, "t1.repair", baseObs(cat, "sorceress").with(nil, map[string]float64{"durability": dur}))
		if d, _ := s.Ask(ctx); d.Answer != want {
			t.Errorf("durability %v: %s, want %s", dur, d.Answer, want)
		}
	}
}
