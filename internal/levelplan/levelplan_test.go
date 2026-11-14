package levelplan

import (
	"strings"
	"testing"

	"d2lod-bot/internal/campaign"
)

func shipped(t *testing.T) (*Plan, Table, *campaign.Spine) {
	t.Helper()
	p, err := Load("normal")
	if err != nil {
		t.Fatal(err)
	}
	tab, err := LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	sp, err := campaign.Shipped(campaign.Validator{})
	if err != nil {
		t.Fatal(err)
	}
	return p, tab, sp
}

func TestShippedPlanIsValidAndCovered(t *testing.T) {
	p, tab, _ := shipped(t)
	for _, class := range []string{"paladin", "barbarian", "sorceress"} {
		c, err := p.For(class)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range c.Farms() {
			for _, a := range s.Areas {
				if _, ok := tab[a]; !ok {
					t.Errorf("%s %s: area %d has no XP table row", class, s.ID, a)
				}
			}
		}
	}
	if _, ok := p.Active("paladin"); !ok {
		t.Error("the Paladin plan must be active")
	}
	for _, class := range []string{"barbarian", "sorceress"} {
		if _, ok := p.Active(class); ok {
			t.Errorf("%s is review-only until its survival model exists", class)
		}
	}
}

func TestUnknownClassIsAnError(t *testing.T) {
	p, _, _ := shipped(t)
	if _, err := p.For("warlock"); err == nil || !strings.Contains(err.Error(), "warlock") {
		t.Fatalf("err %v, want one naming the class", err)
	}
	if _, ok := p.Active("necromancer"); ok {
		t.Fatal("a class without a plan is not active")
	}
}

func TestParseRejects(t *testing.T) {
	refs := Refs{
		Area:    func(id int) bool { return id >= 2 && id <= 40 },
		WP:      func(id int) bool { return id == 3 || id == 4 },
		Chapter: func(id string) bool { return id == "den_of_evil" },
	}
	head := `{"schema":"levelplan/3","difficulty":"normal","classes":{"paladin":{"active":true,"build":"b","slices":[`
	cases := map[string]string{
		"area 99":         `{"id":"a","kind":"farm","areas":[99],"entry":1,"exit":3,"way_in":"walk","why":""}`,
		"town":            `{"id":"a","kind":"farm","areas":[1],"entry":1,"exit":3,"way_in":"walk","why":""}`,
		"below exit":      `{"id":"a","kind":"farm","areas":[2],"entry":3,"exit":3,"way_in":"walk","why":""}`,
		"holds clvl 4":    `{"id":"a","kind":"farm","areas":[2],"entry":1,"exit":3,"way_in":"walk","why":""},{"id":"b","kind":"farm","areas":[3],"entry":5,"exit":8,"way_in":"walk","why":""}`,
		"merc false":      `{"id":"a","kind":"farm","areas":[2],"entry":1,"exit":4,"way_in":"walk","why":""},{"id":"b","kind":"farm","areas":[3],"entry":4,"exit":8,"no_merc_add":2,"way_in":"walk","why":""}`,
		"not monotone":    `{"id":"a","kind":"farm","areas":[2],"entry":3,"exit":6,"way_in":"walk","why":""},{"id":"b","kind":"farm","areas":[3],"entry":1,"exit":8,"way_in":"walk","why":""}`,
		"journey owns":    `{"id":"a","kind":"farm","areas":[2],"entry":1,"exit":6,"way_in":"walk","why":""},{"id":"q","kind":"quest","chapter":"den_of_evil","entry":3,"why":""}`,
		"unknown chapter": `{"id":"a","kind":"farm","areas":[2],"entry":1,"exit":6,"way_in":"walk","why":""},{"id":"q","kind":"quest","chapter":"cows","why":""}`,
		"way_in":          `{"id":"a","kind":"farm","areas":[2],"entry":1,"exit":6,"way_in":"teleport","why":""}`,
		"no waypoint":     `{"id":"a","kind":"farm","areas":[2],"entry":1,"exit":6,"way_in":"waypoint","via_wp":5,"why":""}`,
		"repeated":        `{"id":"a","kind":"farm","areas":[2],"entry":1,"exit":6,"way_in":"walk","why":""},{"id":"a","kind":"farm","areas":[3],"entry":2,"exit":6,"way_in":"walk","why":""}`,
	}
	for want, body := range cases {
		_, err := Parse([]byte(head+body+`]}}}`), refs)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err %v", want, err)
		}
	}
	ok := head + `{"id":"a","kind":"farm","areas":[2],"entry":1,"exit":6,"way_in":"waypoint","via_wp":3,"why":""},{"id":"q","kind":"quest","chapter":"den_of_evil","why":""}]}}}`
	if _, err := Parse([]byte(ok), refs); err != nil {
		t.Fatalf("valid plan: %v", err)
	}
}

func slice(t *testing.T, c *ClassPlan, id string) Slice {
	t.Helper()
	for _, s := range c.Slices {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("%s has no slice %q", c.Class, id)
	return Slice{}
}

// TestGates: the merc rule (entry + no_merc_add without a living merc) and the kit rule, on two classes.
func TestGates(t *testing.T) {
	p, _, _ := shipped(t)
	pal, _ := p.For("paladin")
	tamoe := slice(t, pal, "tamoe")
	if ok, _ := Gate(tamoe, Hero{Level: tamoe.Entry, Merc: true}); !ok {
		t.Error("Tamoe at its entry with a merc")
	}
	if ok, why := Gate(tamoe, Hero{Level: tamoe.Entry + 1}); ok || !strings.Contains(why, "without a merc") {
		t.Errorf("Tamoe one above entry without a merc: %v %q", ok, why)
	}
	if ok, _ := Gate(tamoe, Hero{Level: tamoe.Entry + tamoe.NoMercAdd}); !ok {
		t.Error("Tamoe at entry+no_merc_add without a merc")
	}
	stony := slice(t, pal, "stony_field")
	if ok, why := Gate(stony, Hero{Level: stony.Entry}); ok || !strings.Contains(why, "full kit") {
		t.Errorf("Stony Field at entry without a kit: %v %q", ok, why)
	}
	if ok, _ := Gate(stony, Hero{Level: stony.Entry, FullKit: true}); !ok {
		t.Error("Stony Field at entry with a full kit")
	}
	sorc, _ := p.For("sorceress")
	cp := slice(t, sorc, "cold_plains")
	if ok, _ := Gate(cp, Hero{Level: slice(t, pal, "cold_plains").Entry, FullKit: true}); ok {
		t.Error("the Sorceress enters Cold Plains later than the Paladin")
	}
	if ok, _ := Gate(slice(t, pal, "den"), Hero{Level: 30}); ok {
		t.Error("a quest slice is never a farm")
	}
}

// TestChooseAdvancesBeforeReclearing: with the clvl 5-6 fields cleared, a clvl 7 Paladin goes to the fresh Black Marsh
// instead of re-clearing; with that cleared too, the best in-band re-clear wins, at half rate.
func TestChooseAdvancesBeforeReclearing(t *testing.T) {
	p, tab, _ := shipped(t)
	pal, _ := p.For("paladin")
	done := map[int]bool{2: true, 3: true, 4: true, 9: true, 13: true}
	fresh := func(a int) bool { return !done[a] }
	h := Hero{Level: 7, FullKit: true}
	ch, ok := pal.Choose(h, fresh, tab, 0.5)
	if !ok || ch.Slice != "black_marsh" || !ch.Fresh {
		t.Fatalf("clvl 7: %+v %v, want fresh black_marsh", ch, ok)
	}
	done[6] = true
	ch, ok = pal.Choose(h, fresh, tab, 0.5)
	if !ok || ch.Fresh || ch.Slice != "black_marsh" {
		t.Fatalf("clvl 7 all cleared: %+v, want the Black Marsh re-clear (best half rate in band)", ch)
	}
	if want := tab[6].PerMin(7) / 2; ch.PerMin != want {
		t.Errorf("re-clear rate %v, want %v", ch.PerMin, want)
	}
	// Route order: Cave 2 opens only after Cave 1.
	done = map[int]bool{2: true, 3: true, 4: true}
	ch, _ = pal.Choose(Hero{Level: 6, FullKit: true}, fresh, tab, 0.5)
	if ch.Area != 9 {
		t.Errorf("clvl 6: area %d, want Cave 1 before Cave 2", ch.Area)
	}
}

// TestSimulateOldPlanMatchesAreamon calibrates the minute model: plan v2's rows, first fresh row wins, give the
// AREAMON model's 13,117 XP at 60 minutes (I) and stay within 15 percent of the logged hour 4 (13,883, V).
func TestSimulateOldPlanMatchesAreamon(t *testing.T) {
	_, tab, _ := shipped(t)
	r, err := Simulate(oldPlanRows(), tab, SimOpts{Minutes: 60, Rule: RuleFirst})
	if err != nil {
		t.Fatal(err)
	}
	xp, clvl := r.At(60)
	if xp < 13117*0.9 || xp > 13117*1.1 || clvl != 5 {
		t.Errorf("old plan at 60 min: %.0f XP clvl %d, want 13,117 +-10 percent, clvl 5", xp, clvl)
	}
	if xp < 13883*0.85 || xp > 13883*1.15 {
		t.Errorf("old plan at 60 min %.0f, logged hour 4 13,883", xp)
	}
}

// TestPaladinPrediction pins the numbers docs/levelplan3.md publishes (I) and checks the plan beats plan v2.
func TestPaladinPrediction(t *testing.T) {
	p, tab, sp := shipped(t)
	pal, _ := p.For("paladin")
	r, err := Simulate(pal, tab, SimOpts{Minutes: 240, Class: "paladin", Spine: sp})
	if err != nil {
		t.Fatal(err)
	}
	old, _ := Simulate(oldPlanRows(), tab, SimOpts{Minutes: 240, Rule: RuleFirst})
	want := map[float64][2]int{60: {20376, 6}, 120: {63772, 10}, 240: {170551, 14}}
	for m, w := range want {
		xp, clvl := r.At(m)
		if int(xp+0.5) < w[0]-200 || int(xp+0.5) > w[0]+200 || clvl != w[1] {
			t.Errorf("v3 at %v min: %.0f XP clvl %d, the doc says %d clvl %d", m, xp, clvl, w[0], w[1])
		}
		if ox, _ := old.At(m); xp <= ox {
			t.Errorf("v3 at %v min %.0f <= plan v2 %.0f", m, xp, ox)
		}
	}
	for _, q := range []string{"den_of_evil", "blood_raven"} {
		if _, ok := r.Done[q]; !ok {
			t.Errorf("quest %s not done in 240 minutes", q)
		}
	}
	prev := 0
	for _, pt := range r.Trace {
		if pt.Clvl < prev {
			t.Fatalf("clvl falls at %v", pt)
		}
		prev = pt.Clvl
	}
}

// TestOtherClassesSimulate: the review-only rows play through on the same model, the Sorceress takes the journey's
// caster Den row (recommended 6), not the Paladin's.
func TestOtherClassesSimulate(t *testing.T) {
	p, tab, sp := shipped(t)
	for _, class := range []string{"barbarian", "sorceress"} {
		c, _ := p.For(class)
		r, err := Simulate(c, tab, SimOpts{Minutes: 240, Class: class, Spine: sp})
		if err != nil {
			t.Fatalf("%s: %v", class, err)
		}
		if _, l := r.At(240); l < 8 {
			t.Errorf("%s at 240 min clvl %d", class, l)
		}
		for _, pt := range r.Trace {
			if pt.What == "quest den_of_evil" && class == "sorceress" && pt.Clvl < 6 {
				t.Errorf("sorceress did the Den at clvl %d", pt.Clvl)
			}
		}
	}
	c, _ := p.For("paladin")
	if _, err := Simulate(c, tab, SimOpts{Minutes: 240, Class: "warlock", Spine: sp}); err == nil || !strings.Contains(err.Error(), "warlock") {
		t.Errorf("warlock: %v", err)
	}
}

func TestLevelOf(t *testing.T) {
	for xp, want := range map[float64]int{0: 1, 499: 1, 500: 2, 7875: 5, 14174: 5, 14175: 6, 112725: 13} {
		if got := LevelOf(xp); got != want {
			t.Errorf("LevelOf(%v) = %d, want %d", xp, got, want)
		}
	}
}
