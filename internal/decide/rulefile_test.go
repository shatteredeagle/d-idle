package decide

import (
	"strings"
	"testing"
)

// The T0 to T2 rule file expresses today's knob behaviour (spec 5.3). The test proves every row is valid, that
// rows marked "match" agree with the untrained default and rows marked "differs" really differ somewhere, and that
// every point the file cannot fully cover is named in its gap list.
func TestRuleFileExpressesTodaysDefaults(t *testing.T) {
	cat := mustCat(t)
	f, err := ShippedRuleFile()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Validate(cat); err != nil {
		t.Fatal(err)
	}
	checks, err := f.Check(cat)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range f.Rules {
		pc := checks[r.Point]
		switch r.Vs {
		case "match":
			if pc.RowMismatch[i] {
				t.Errorf("row %d (%s %v) is marked match but disagrees with the default", i+1, r.Point, r.When)
			}
		case "differs":
			if !pc.RowMismatch[i] {
				t.Errorf("row %d (%s %v) is marked differs but agrees everywhere", i+1, r.Point, r.When)
			}
		default:
			t.Errorf("row %d: vs must be match or differs", i+1)
		}
	}
	gaps := map[string]string{}
	for _, g := range f.Gaps {
		gaps[g.Point] = g.Gap
	}
	var full, partial, none []string
	for _, p := range cat.Points {
		pc := checks[p.ID]
		switch {
		case pc.Covered == pc.Total:
			full = append(full, p.ID)
		case pc.Covered == 0:
			none = append(none, p.ID)
		default:
			partial = append(partial, p.ID)
		}
		if pc.Covered < pc.Total && gaps[p.ID] == "" {
			t.Errorf("%s is not fully covered and has no gap entry", p.ID)
		}
	}
	for id, g := range gaps {
		if _, err := cat.Point(id); err != nil || g == "" {
			t.Errorf("bad gap entry %q", id)
		}
	}
	t.Logf("fully covered: %s", strings.Join(full, " "))
	t.Logf("partly covered: %s", strings.Join(partial, " "))
	t.Logf("not covered: %s", strings.Join(none, " "))
	if len(full) < 5 || len(none) == 0 {
		t.Fatalf("coverage moved: full %d none %d", len(full), len(none))
	}
	// Verdict (spec 5.3): the file cannot express all of today's defaults; the gate stays open with the gaps above.
}

func TestRuleFileErrorsNameTheCulprit(t *testing.T) {
	cat := mustCat(t)
	for name, tc := range map[string]struct{ body, want string }{
		"point":  {`{"version":1,"rules":[{"point":"t9.zz","when":{},"answer":"a"}]}`, "t9.zz"},
		"answer": {`{"version":1,"rules":[{"point":"t1.restock","when":{},"answer":"maybe"}]}`, "maybe"},
		"field":  {`{"version":1,"rules":[{"point":"t1.restock","when":{"mood":"x"},"answer":"yes"}]}`, "mood"},
		"value":  {`{"version":1,"rules":[{"point":"t1.restock","when":{"belt":"x"},"answer":"yes"}]}`, `"x"`},
		"gap":    {`{"version":1,"rules":[],"gaps":[{"point":"t7.q","gap":"g"}]}`, "t7.q"},
	} {
		f, err := ParseRuleFile([]byte(tc.body))
		if err == nil {
			err = f.Validate(cat)
		}
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := ParseRuleFile([]byte(`{"version":9}`)); err == nil {
		t.Error("newer rule file accepted")
	}
}
