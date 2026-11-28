package decide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildClasses reads the class key of every shipped plan table of the two classes, so the checks below run per build
// (bow, javelin, multishot; fire blast, sentry) and a new table of either class is covered without touching this file.
func buildClasses(t *testing.T, classes ...string) map[string]string {
	t.Helper()
	files, err := filepath.Glob("../plandata/build-*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no plan tables: %v", err)
	}
	out := map[string]string{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var b struct {
			Build string `json:"build"`
			Class string `json:"class"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, c := range classes {
			if strings.EqualFold(b.Class, c) {
				out[b.Build] = strings.ToLower(b.Class)
			}
		}
	}
	return out
}

// Card cls2 (review 8 section 4, column D): every decide entry point answers for the Amazon and the Assassin instead of
// "unknown class". Values are S or U from docs/sme/class-amazon.md, class-assassin.md and the SME review of 2026-11-27
// (the notes in proposezone.json say which). Both classes shoot or lay from range in every shipped table, so the stance
// option carries the ranged (caster) numbers, not the melee ones.
func TestDecideAnswersForAmazonAndAssassinBuilds(t *testing.T) {
	cat := mustCat(t)
	pol, err := ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	pz, err := ShippedProposeTable()
	if err != nil {
		t.Fatal(err)
	}
	builds := buildClasses(t, "amazon", "assassin")
	perClass := map[string]int{}
	for build, class := range builds {
		perClass[class]++
		if _, err := cat.BandLabel(class, "mana", 10); err != nil {
			t.Errorf("%s: mana band: %v", build, err)
		}
		rows, err := pol.RowsFor(cat, class)
		if err != nil || len(rows) == 0 {
			t.Errorf("%s: policy rows %v (%d)", build, err, len(rows))
		}
		for _, r := range rows {
			for i := range r.Options {
				if _, err := r.Options[i].KnobsFor(class); err != nil {
					t.Errorf("%s: %s/%s: %v", build, r.ID, r.Options[i].ID, err)
				}
			}
		}
		if l, ok := pz.Classes[class]; !ok || l.Lead < 0 {
			t.Errorf("%s: proposezone lead missing or negative", build)
		}
		ft, err := ShippedFirsts()
		if err != nil {
			t.Fatal(err)
		}
		fc, err := cat.WithFirsts(ft)
		if err != nil {
			t.Fatal(err)
		}
		clk := &ManualClock{}
		st, err := New(fc, "p1", clk)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := NewTracker(ft, fc, st, NewCodex("p1"), clk, Who{Class: class, Diff: "normal"}); err != nil {
			t.Errorf("%s: firsts tracker: %v", build, err)
		}
	}
	for _, c := range []string{"amazon", "assassin"} {
		if perClass[c] < 2 {
			t.Errorf("%s: %d plan tables, the check needs two builds", c, perClass[c])
		}
	}

	row, err := pol.Row("stance")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range row.Options {
		if len(o.ClassKnobs) == 0 {
			continue
		}
		ranged, _ := o.KnobsFor("sorceress")
		for _, c := range []string{"amazon", "assassin"} {
			got, err := o.KnobsFor(c)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range ranged {
				if got[k] != v {
					t.Errorf("stance %s: %s %s=%v, the ranged row says %v", o.ID, c, k, got[k], v)
				}
			}
		}
	}

	if _, err := cat.BandLabel("amazoon", "mana", 10); err == nil {
		t.Error("a misspelt class got a mana band")
	}
	if _, err := pol.RowsFor(cat, "assasin"); err == nil {
		t.Error("a misspelt class got policy rows")
	}
}
