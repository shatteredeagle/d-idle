package decide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBands(t *testing.T) {
	c := mustCat(t)
	cases := []struct {
		field string
		v     float64
		want  string
	}{
		{"life", 24.9, "lt25"}, {"life", 25, "25-35"}, {"life", 35, "35-55"}, {"life", 55, "55-70"}, {"life", 70, "gt70"}, {"life", 100, "gt70"},
		{"gold", 99, "lt100"}, {"gold", 100, "100-500"}, {"gold", 1500, "gt1500"},
		{"freerows", 0, "0"}, {"freerows", 1, "1"}, {"freerows", 5, "2+"},
		{"timeratio", 0.69, "fast"}, {"timeratio", 1, "normal"}, {"timeratio", 1.3, "slow"},
		{"durability", 19, "lt20"}, {"durability", 30, "gt30"},
	}
	for _, tc := range cases {
		got, err := c.BandLabel("sorceress", tc.field, tc.v)
		if err != nil || got != tc.want {
			t.Errorf("%s %v = %q, %v; want %q", tc.field, tc.v, got, err, tc.want)
		}
	}
}

func TestContextKey(t *testing.T) {
	c := mustCat(t)
	o := baseObs(c, "sorceress").with(map[string]string{"hc": "hardcore", "kit": "THIN"}, map[string]float64{"life": 40})
	ctx, err := c.Build("t0.retreat.drink", o, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "t0.retreat.drink|act=1,champbonus=none,diff=normal,drinklife=26-49,hc=hardcore,held=none,kit=THIN"
	if ctx.Key != want {
		t.Fatalf("key = %s", ctx.Key)
	}
	back, err := ParseKey(ctx.Key)
	if err != nil || back.Plain() != ctx.Plain() {
		t.Fatalf("parse: %v %v", back, err)
	}
	// raw life 26 and 49 share the key (bands, never raw values)
	o2 := o.with(nil, map[string]float64{"life": 49})
	if c2, _ := c.Build("t0.retreat.drink", o2, 1, nil); c2.Key != want {
		t.Fatal("same band must give the same key")
	}
	// hc is always part of the key: softcore and hardcore never share a streak
	o3 := o.with(map[string]string{"hc": "normal"}, nil)
	if c3, _ := c.Build("t0.retreat.drink", o3, 1, nil); c3.Key == want {
		t.Fatal("hc missing from the key")
	}
}

func TestBuildErrorsNameTheCulprit(t *testing.T) {
	c := mustCat(t)
	good := baseObs(c, "sorceress")
	for _, tc := range []struct {
		name, point string
		o           Obs
		want        string
	}{
		{"unknown point", "t9.nothing", good, "t9.nothing"},
		{"unknown class", "t1.restock", baseObs(c, "necromancer"), "necromancer"},
		{"missing field", "t1.restock", Obs{Class: "sorceress", Cat: good.Cat}, "gold"},
		{"bad enum", "t1.restock", good.with(map[string]string{"belt": "overflowing"}, nil), "overflowing"},
	} {
		if _, err := c.Build(tc.point, tc.o, 1, nil); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v", tc.name, err)
		}
	}
	if _, err := c.BandLabel("sorceress", "nofield", 1); err == nil || !strings.Contains(err.Error(), "nofield") {
		t.Error("unknown field")
	}
	if _, err := c.BandLabel("monk", "mana", 1); err == nil || !strings.Contains(err.Error(), "monk") {
		t.Error("unknown class in classband")
	}
}

type stubUrgency struct{ band string }

func (s stubUrgency) Urgency(Obs) (string, error) { return s.band, nil }

func TestUrgencyStageHook(t *testing.T) {
	c := mustCat(t)
	o := baseObs(c, "sorceress")
	delete(o.Cat, "urgency")
	s1, err := c.Build("t0.retreat.drink", o, 1, nil)
	if err != nil || s1.Fields["drinklife"] == "" || s1.Fields["urgency"] != "" {
		t.Fatalf("stage 1 keys on life: %v %v", s1, err)
	}
	if _, err := c.Build("t0.retreat.drink", o, 2, nil); err == nil || !strings.Contains(err.Error(), "urgency") {
		t.Fatalf("stage 2 without a provider: %v", err)
	}
	s2, err := c.Build("t0.retreat.drink", o, 2, stubUrgency{"urgent"})
	if err != nil || s2.Fields["urgency"] != "urgent" || s2.Fields["drinklife"] != "" {
		t.Fatalf("stage 2: %v %v", s2, err)
	}
	if _, err := c.Build("t0.retreat.drink", o, 2, stubUrgency{"frantic"}); err == nil {
		t.Fatal("bad urgency band accepted")
	}
	if s1.Key == s2.Key {
		t.Fatal("stage 1 and stage 2 rules must have different keys")
	}
	// a non-urgency point is unchanged at stage 2
	r1, _ := c.Build("t1.restock", o, 1, nil)
	r2, _ := c.Build("t1.restock", o, 2, nil)
	if r1.Key != r2.Key {
		t.Fatal("t1 point changed at stage 2")
	}
}

func TestCatalogueShape(t *testing.T) {
	c := mustCat(t)
	if len(c.Points) != 25 {
		t.Fatalf("%d points, spec says 25", len(c.Points))
	}
	tiers := map[int]int{}
	for _, p := range c.Points {
		tiers[p.Tier]++
		if len(p.Answers) < 2 || len(p.Answers) > 5 {
			t.Errorf("%s answers", p.ID)
		}
	}
	if tiers[0] != 7 || tiers[2] == 0 {
		t.Fatalf("tiers %v", tiers)
	}
	if c.Config.NByRisk[Safe] != 2 || c.Config.NByRisk[Normal] != 3 || c.Config.NByRisk[Risky] != 5 {
		t.Fatal("N by risk")
	}
	// life edges are the five bands of the decision 2026-11-05
	if e := c.Fields["life"].Edges; len(e) != 4 || e[0] != 25 || e[3] != 70 {
		t.Fatalf("life edges %v", e)
	}
	// class is a total function of the key: every context of every point has a class and a default
	for _, p := range c.Points {
		ctxs, err := c.Contexts(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, ctx := range ctxs {
			if _, err := c.RiskClass(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := c.Default(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestCatalogueRejectsBadData(t *testing.T) {
	good := string(shippedCatalogue)
	for name, bad := range map[string]string{
		"no catch-all risk row": strings.Replace(good, `{"match": {}, "class": "safe"}]`, `{"match": {"kit": "OK"}, "class": "safe"}]`, 1),
		"unknown risk class":    strings.Replace(good, `"class": "safe"`, `"class": "meh"`, 1),
		"unknown field":         strings.Replace(good, `"contextKeys": ["tripgold", "belt"]`, `"contextKeys": ["tripgold", "mood"]`, 1),
		"default not an answer": strings.Replace(good, `"answer": "repair"`, `"answer": "x"`, 1),
		"newer catalogue":       strings.Replace(good, `"version": 4,`, `"version": 5,`, 1),
	} {
		if bad == good {
			t.Fatalf("%s: test edit did not apply", name)
		}
		if _, err := ParseCatalogue([]byte(bad)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// Two builds: the same catalogue and flow for the default Sorceress and a second class; no class name in the data
// or in the Go code; class numbers come from the class table.
func TestTwoBuilds(t *testing.T) {
	cat := mustCat(t)
	manaBands := map[string]string{}
	for _, class := range []string{"sorceress", "paladin"} {
		t.Run(class, func(t *testing.T) {
			clk := &ManualClock{}
			s, _ := New(cat, "p-"+class, clk)
			o := baseObs(cat, class)
			ctx, err := cat.Build("t1.restock", o, 1, nil)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				d, _ := s.Ask(ctx)
				if err := s.Answer(d.Thought.ID, "no"); err != nil {
					t.Fatal(err)
				}
			}
			if d, _ := s.Ask(ctx); d.Source != "rule" || d.Answer != "no" {
				t.Fatalf("%+v", d)
			}
			for _, p := range cat.Points {
				if _, err := cat.Build(p.ID, o, 1, nil); err != nil {
					t.Errorf("%s: %v", p.ID, err)
				}
			}
			l, err := cat.BandLabel(class, "mana", 45)
			if err != nil {
				t.Fatal(err)
			}
			manaBands[class] = l
		})
	}
	if manaBands["sorceress"] == manaBands["paladin"] {
		t.Fatalf("class mana bands should differ: %v", manaBands)
	}
	var names []string
	for n := range cat.Classes {
		names = append(names, n)
	}
	pts, _ := json.Marshal(cat.Points)
	flds, _ := json.Marshal(cat.Fields)
	files, _ := filepath.Glob("*.go")
	for _, n := range names {
		for _, hay := range []string{string(pts), string(flds)} {
			if strings.Contains(strings.ToLower(hay), n) {
				t.Errorf("class %q named in points or fields", n)
			}
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, _ := os.ReadFile(f)
			if strings.Contains(strings.ToLower(string(b)), n) {
				t.Errorf("class %q named in %s", n, f)
			}
		}
	}
}
