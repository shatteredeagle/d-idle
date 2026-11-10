package readiness

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"d2lod-bot/internal/data"
)

// fixtureDB is a tiny stand-in for the extracted tables: a quiet field, a cold-immune cave, a cave where every element
// is immune somewhere. Numbers are made up for the tests (the real values are checked in TestRealAnchors).
func fixtureDB() *data.DB {
	mon := func(key string, boss bool, res [3][6]int) *data.Monster {
		m := &data.Monster{Key: key, Enabled: true, Killable: true, Boss: boss, Res: res}
		for d := 0; d < 3; d++ {
			m.HPMax[d] = 50 * (d + 1)
			m.A1Max[d] = 10 * (d + 1) * (d + 1)
			m.A1TH[d] = 100 * (d + 1)
		}
		return m
	}
	plain := [3][6]int{}
	cold := [3][6]int{{0, 0, 0, 0, 100, 0}, {0, 0, 0, 0, 100, 0}, {0, 0, 0, 0, 100, 0}}
	var all [3][6]int
	for d := range all {
		all[d] = [6]int{100, 0, 100, 100, 100, 100}
	}
	scale := &data.MonLevel{HP: [3]int{100, 100, 100}, DM: [3]int{100, 100, 100}, TH: [3]int{100, 100, 100}}
	ml := make([]*data.MonLevel, 90)
	for i := range ml {
		ml[i] = scale
	}
	return &data.DB{
		Monsters:  []*data.Monster{mon("zombie1", false, plain), mon("frozen1", false, cold), mon("wall1", false, all)},
		MonLevels: ml,
		Levels: []*data.Level{
			{ID: 2, Key: "Act 1 - Wilderness 1", Name: "Blood Moor", Act: 0, MonLvl: [3]int{1, 36, 67}, Mon: []string{"zombie1"}, NMon: []string{"zombie1"}},
			{ID: 9, Key: "Act 1 - Cave 1", Name: "Cold Cave", Act: 0, MonLvl: [3]int{3, 38, 79}, Mon: []string{"zombie1", "frozen1"}, NMon: []string{"zombie1", "frozen1"}},
			{ID: 30, Key: "Act 2 - Wall", Name: "Wall Hall", Act: 1, MonLvl: [3]int{14, 43, 75}, Mon: []string{"wall1"}, NMon: []string{"wall1"}},
			{ID: 1, Key: "Act 1 - Town", Name: "Rogue Encampment", Act: 0},
		},
	}
}

func testEval(t *testing.T) *Evaluator {
	t.Helper()
	at, err := BuildAreaTable(fixtureDB())
	if err != nil {
		t.Fatal(err)
	}
	ct, err := ShippedChecks()
	if err != nil {
		t.Fatal(err)
	}
	ev, err := New(ct, at)
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func state(r Report, id string) string {
	for _, c := range r.Checks {
		if c.ID == id {
			return c.State
		}
	}
	return "missing"
}

func hero(class string, diff string, lvl int) Hero {
	return Hero{Class: class, Level: lvl, Difficulty: diff, Life: 400, MaxLife: 400, Potions: 6, RepairPct: 100,
		Resist: Resists{Fire: 75, Cold: 75, Lightning: 75, Poison: 75}}
}

func TestAreaTableBuild(t *testing.T) {
	at, err := BuildAreaTable(fixtureDB())
	if err != nil {
		t.Fatal(err)
	}
	if len(at.Areas) != 3 {
		t.Fatalf("areas %d, want 3 (town has no spawns)", len(at.Areas))
	}
	cave, err := at.Find("cold cave")
	if err != nil {
		t.Fatal(err)
	}
	if cave.Act != 1 || cave.Diff[2].MLvl != 79 || strings.Join(cave.Diff[0].Immune, ",") != "cold" {
		t.Fatalf("cave %+v", cave)
	}
	if got := cave.Diff[0].Affixes; slicesContains(got, "magic_resistant") || !slicesContains(cave.Diff[1].Affixes, "magic_resistant") {
		t.Fatalf("magic resistant must be Nightmare and Hell only: %v / %v", got, cave.Diff[1].Affixes)
	}
	if a, _ := at.Find("9"); a == nil || a.Name != "Cold Cave" {
		t.Fatal("find by id")
	}
	if _, err := at.Find("Nowhere"); err == nil || !strings.Contains(err.Error(), "Nowhere") {
		t.Fatalf("unknown area must name it: %v", err)
	}
	b, err := at.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseAreaTable(b)
	if err != nil || len(back.Areas) != 3 {
		t.Fatalf("round trip: %v", err)
	}
	if _, err := ParseAreaTable([]byte(`{"schema":"x"}`)); err == nil {
		t.Fatal("wrong schema must fail")
	}
	if _, err := BuildAreaTable(nil); err == nil {
		t.Fatal("nil db must fail")
	}
}

func slicesContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func TestShippedChecksParse(t *testing.T) {
	if _, err := ShippedChecks(); err != nil {
		t.Fatal(err)
	}
	bad := map[string]string{
		"unknown kind":  strings.Replace(string(shippedChecks), `"level_gap"`, `"nope"`, 1),
		"missing param": strings.Replace(string(shippedChecks), `"warnGap": 2, `, ``, 1),
		"unknown class": strings.Replace(string(shippedChecks), `"sorceress": {"nightmare"`, `"wizard": {"nightmare"`, 1),
		"bad damage":    strings.Replace(string(shippedChecks), `"fire", "cold", "lightning"`, `"fire", "fro"`, 1),
		"unknown field": strings.Replace(string(shippedChecks), `"doc":`, `"dox":`, 1),
	}
	for name, src := range bad {
		if _, err := ParseChecks([]byte(src)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestUnknownClassDifficultyArea(t *testing.T) {
	ev := testEval(t)
	for _, tc := range []struct {
		h          Hero
		area, want string
	}{
		{hero("necromancer", "normal", 5), "Blood Moor", "necromancer"},
		{hero("paladin", "easy", 5), "Blood Moor", "easy"},
		{hero("paladin", "normal", 5), "Atlantis", "Atlantis"},
	} {
		if _, err := ev.Evaluate(tc.h, tc.area); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%+v: error %v must name %q", tc.h, err, tc.want)
		}
	}
}

func TestTwoClassesLevelAndLight(t *testing.T) {
	ev := testEval(t)
	// Nightmare Blood Moor mlvl 36: the Sorceress warns from a gap of 5 (override 4), the Paladin from 6.
	for _, tc := range []struct {
		class string
		lvl   int
		want  string
	}{
		{"sorceress", 32, Pass}, {"sorceress", 31, Warn},
		{"paladin", 31, Pass}, {"paladin", 30, Warn},
		{"paladin", 25, Fail}, {"sorceress", 25, Fail},
	} {
		r, err := ev.Evaluate(hero(tc.class, "nightmare", tc.lvl), "Blood Moor")
		if err != nil {
			t.Fatal(err)
		}
		if got := state(r, "level"); got != tc.want {
			t.Errorf("%s clvl %d: level %s, want %s (%s)", tc.class, tc.lvl, got, tc.want, r.Checks[0].Reason)
		}
	}
	good, _ := ev.Evaluate(hero("paladin", "normal", 5), "Blood Moor")
	if good.Light != Green || len(good.Reasons) != 0 {
		t.Fatalf("light %s reasons %v", good.Light, good.Reasons)
	}
	if pc, _ := ev.Evaluate(hero("paladin", "normal", 5), "Cold Cave"); state(pc, "immunity") != Pass {
		t.Fatalf("cold immunity does not touch the Paladin's damage types: %v", pc.Reasons)
	}
	amber, _ := ev.Evaluate(hero("sorceress", "normal", 5), "Cold Cave")
	if amber.Light != Amber {
		t.Fatalf("a cold-immune cave is amber for a class with other types: %s %v", amber.Light, amber.Reasons)
	}
	red, _ := ev.Evaluate(hero("sorceress", "normal", 5), "Wall Hall")
	if red.Light != Red || state(red, "immunity") != Fail || !strings.Contains(strings.Join(red.Reasons, "|"), "every damage type") {
		t.Fatalf("every element immune must be red: %s %v", red.Light, red.Reasons)
	}
}

// Resist: the same raw numbers are fine or not depending on whether the penalty is already in them. Both cases are
// covered until TASK-216 settles which one d2.Player.Resist is (U).
func TestResistBothBasisCases(t *testing.T) {
	ev := testEval(t)
	for _, class := range []string{"sorceress", "paladin"} {
		h := hero(class, "hell", 90)
		h.Resist = Resists{Fire: 75, Cold: 75, Lightning: 75, Poison: 75}
		// numbers read as raw: 75 - 100 = -25 effective -> fail (below 0)
		h.Basis = ResistRaw
		r, _ := ev.Evaluate(h, "Blood Moor")
		if state(r, "resist") != Fail {
			t.Errorf("%s raw 75 in Hell must fail, got %s: %s", class, state(r, "resist"), r.Checks[1].Reason)
		}
		// the same numbers read as already shown: 75 stays 75 -> pass
		h.Basis = ResistWithPenalty
		r, _ = ev.Evaluate(h, "Blood Moor")
		if state(r, "resist") != Pass {
			t.Errorf("%s shown 75 in Hell must pass, got %s", class, state(r, "resist"))
		}
		// raw 120 caps at 75 before the penalty
		h.Basis, h.Resist = ResistRaw, Resists{Fire: 120, Cold: 120, Lightning: 120, Poison: 120}
		r, _ = ev.Evaluate(h, "Blood Moor")
		if state(r, "resist") != Fail {
			t.Errorf("%s raw 120 caps at 75 then -100: fail, got %s", class, state(r, "resist"))
		}
	}
	h := hero("paladin", "nightmare", 60)
	h.Resist = Resists{Fire: 50, Cold: 30, Lightning: 60, Poison: 0}
	r, _ := ev.Evaluate(h, "Blood Moor")
	if state(r, "resist") != Pass { // poison is not counted in Nightmare; cold 30-40 = -10 is above the -20 fail line, below warn 10
		if state(r, "resist") != Warn {
			t.Fatalf("nightmare cold -10: %s", state(r, "resist"))
		}
	}
	if DefaultResistBasis != ResistRaw {
		t.Fatal("the default must match combat.PlayerResists.Effective (raw) until TASK-216")
	}
}

func TestSkipsWhatIsNotMeasured(t *testing.T) {
	ev := testEval(t)
	h := hero("paladin", "normal", 5)
	h.RepairPct = -1
	r, _ := ev.Evaluate(h, "Blood Moor")
	for _, id := range []string{"defence", "damage", "repair"} {
		if state(r, id) != Skip {
			t.Errorf("%s must skip when unmeasured, got %s", id, state(r, id))
		}
	}
	if r.Light != Green {
		t.Fatalf("skips do not colour the light: %s", r.Light)
	}
	h.Defence, h.DamagePerSec, h.RepairPct = 20, 100, 5
	h.MaxLife = 30
	r, _ = ev.Evaluate(h, "Blood Moor")
	if state(r, "repair") != Fail || state(r, "life") == Pass {
		t.Fatalf("repair 5%% fails and 30 life against elite hits does not pass: %+v", r.Checks)
	}
	if state(r, "damage") != Pass {
		t.Fatalf("a 50 life zombie dies fast: %+v", r.Checks)
	}
	h.DamagePerSec = 0.5
	r, _ = ev.Evaluate(h, "Blood Moor")
	if state(r, "damage") == Pass {
		t.Fatalf("0.5 dps against 150 elite life is slow: %+v", r.Checks)
	}
}

func TestHandler(t *testing.T) {
	ev := testEval(t)
	src := func() (Hero, string, bool) { return hero("paladin", "normal", 5), "Blood Moor", true }
	h := Handler(ev, src)
	get := func(url string) View {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		var v View
		if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if v := get(Route); v.Report == nil || v.Report.Area != "Blood Moor" {
		t.Fatalf("default area is where the hero stands: %+v", v)
	}
	if v := get(Route + "?area=Cold+Cave&difficulty=hell"); v.Report == nil || v.Report.Difficulty != "hell" || v.Report.Area != "Cold Cave" {
		t.Fatalf("%+v", v)
	}
	if v := get(Route + "?area=%3Cscript%3E"); v.Error == "" || strings.Contains(v.Error, "<") {
		t.Fatalf("hostile area must be cleaned: %+v", v)
	}
	w := httptest.NewRecorder()
	Handler(ev, func() (Hero, string, bool) { return Hero{}, "", false }).ServeHTTP(w, httptest.NewRequest("GET", Route, nil))
	if !strings.Contains(w.Body.String(), "not read yet") {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", Route, nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
}
