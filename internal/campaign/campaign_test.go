package campaign_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/campaign"
	"d2lod-bot/internal/readiness"
	"d2lod-bot/internal/task"
)

func shipped(t *testing.T) *campaign.Spine {
	t.Helper()
	ct, err := readiness.ShippedChecks()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, c := range ct.Checks {
		ids[c.ID] = true
	}
	s, err := campaign.Shipped(campaign.Validator{
		Quest: func(q string) bool { return task.Handler(q) != nil },
		Check: func(id string) bool { return ids[id] },
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

var t0 = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

func prog(class string, lvl int, merc bool, done ...string) campaign.Progress {
	p := campaign.Progress{Hero: campaign.Hero{Class: class, Level: lvl, Merc: merc}, Quests: map[string]bool{}, Waypoints: map[int]bool{3: lvl >= 3}}
	for _, q := range done {
		p.Quests[q] = true
	}
	return p
}

func next(t *testing.T, s *campaign.Spine, p campaign.Progress, st *campaign.HeroState, o campaign.Opts) campaign.Decision {
	t.Helper()
	if o.Now.IsZero() {
		o.Now = t0
	}
	d, err := s.Next(p, st, o)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestShippedChapters(t *testing.T) {
	s := shipped(t)
	want := []string{"start", "cold_plains_wp", "den_of_evil", "blood_raven", "find_cain", "countess", "tools_of_the_trade", "andariel"}
	if len(s.Chapters) != len(want) {
		t.Fatalf("chapters %d, want %d", len(s.Chapters), len(want))
	}
	for i, id := range want {
		if s.Chapters[i].ID != id {
			t.Errorf("chapter %d = %q, want %q", i, s.Chapters[i].ID, id)
		}
	}
	for _, cl := range []string{"paladin", "sorceress", "amazon", "necromancer", "barbarian", "druid", "assassin"} {
		for i := range s.Chapters {
			if _, err := s.Chapters[i].LevelsFor(cl); err != nil {
				t.Errorf("%s: %v", cl, err)
			}
		}
	}
}

func TestParseErrorsNameTheChapter(t *testing.T) {
	base := `{"schema":"campaign/1","difficulty":"normal","farm":[],"chapters":[{"id":"a","title":"A","act":1,"area":"x","areaId":1,"kind":"quest","done":{"level":2},"reward":"r","why":"w","levels":{"default":{"floor":1,"recommended":2}}}%s]}`
	cases := map[string]string{
		`,{"id":"b","title":"B","act":1,"area":"x","areaId":1,"kind":"quest","requires":["zz"],"done":{"level":2},"reward":"r","why":"w","levels":{"default":{"floor":1,"recommended":2}}}`: `chapter "b": requires "zz"`,
		`,{"id":"b","title":"B","act":1,"area":"x","areaId":1,"kind":"quest","done":{"quest":"nope"},"reward":"r","why":"w","levels":{"default":{"floor":1,"recommended":2}}}`:              `chapter "b": unknown quest "nope"`,
		`,{"id":"b","title":"B","act":1,"area":"x","areaId":1,"kind":"quest","done":{"level":2},"reward":"r","why":"w","levelsFrom":"plandata:mephisto"}`:                                   `chapter "b": unknown levelsFrom`,
		`,{"id":"b","title":"B","act":1,"area":"x","areaId":1,"kind":"quest","done":{"level":2},"reward":"r","why":"w","checks":["luck"],"levels":{"default":{"floor":1,"recommended":2}}}`: `chapter "b": unknown readiness check "luck"`,
		`,{"id":"a","title":"B","act":1,"area":"x","areaId":1,"kind":"quest","done":{"level":2},"reward":"r","why":"w","levels":{"default":{"floor":1,"recommended":2}}}`:                   `chapter "a": duplicate id`,
		`,{"id":"b","title":"B","act":1,"area":"x","areaId":1,"kind":"walk","done":{"level":2},"reward":"r","why":"w","levels":{"default":{"floor":1,"recommended":2}}}`:                    `chapter "b": unknown kind`,
		`,{"id":"b","title":"B","act":1,"area":"x","areaId":1,"kind":"quest","done":{"level":2},"reward":"r","why":"w","levels":{"paladin":{"floor":1,"recommended":2}}}`:                   `chapter "b": levels has no default row`,
	}
	v := campaign.Validator{Quest: func(q string) bool { return task.Handler(q) != nil }, Check: func(id string) bool { return id == "level" }}
	if _, err := campaign.Parse([]byte(strings.Replace(base, "%s", "", 1)), v); err != nil {
		t.Fatalf("base: %v", err)
	}
	for extra, want := range cases {
		_, err := campaign.Parse([]byte(strings.Replace(base, "%s", extra, 1)), v)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("err %v, want %q", err, want)
		}
	}
}

func TestBands(t *testing.T) {
	s := shipped(t)
	andy := s.Chapter("andariel")
	cases := []struct {
		class string
		lvl   int
		merc  bool
		want  campaign.Band
	}{
		{"paladin", 15, false, campaign.TooEarly}, // floor 16 without a merc
		{"paladin", 16, false, campaign.Ready},
		{"paladin", 17, false, campaign.Comfortable},
		{"paladin", 14, true, campaign.Ready}, // floor 14, recommended 15 with the Rogue
		{"sorceress", 16, false, campaign.TooEarly},
		{"sorceress", 17, false, campaign.Ready},
		{"sorceress", 16, true, campaign.Comfortable},
	}
	for _, c := range cases {
		r, err := campaign.Rate(andy, campaign.Hero{Class: c.class, Level: c.lvl, Merc: c.merc}, nil)
		if err != nil || r.Band != c.want {
			t.Errorf("%s clvl %d merc %v: %v %v, want %v", c.class, c.lvl, c.merc, r.Band, err, c.want)
		}
	}
	// A failing listed check caps Comfortable at Ready; an unlisted one is ignored.
	r, _ := campaign.Rate(andy, campaign.Hero{Class: "paladin", Level: 18}, []campaign.Check{{ID: "potions", State: "fail", Reason: "2 < 8"}, {ID: "damage", State: "fail"}})
	if r.Band != campaign.Ready || len(r.Reasons) != 1 {
		t.Errorf("capped: %+v", r)
	}
	if _, err := campaign.Rate(andy, campaign.Hero{Class: "monk", Level: 18}, nil); err == nil || !strings.Contains(err.Error(), "monk") {
		t.Errorf("unknown class: %v", err)
	}
}

// The ladder of chapters a hero walks when it levels and finishes each chapter it is sent to (both test classes).
func TestLadderPaladinAndSorceress(t *testing.T) {
	s := shipped(t)
	for _, tc := range []struct {
		class string
		want  []string // chapter ids in the order the decision sends the hero
	}{
		{"paladin", []string{"start", "cold_plains_wp", "den_of_evil", "blood_raven", "find_cain", "tools_of_the_trade", "countess", "andariel"}},
		{"sorceress", []string{"start", "cold_plains_wp", "den_of_evil", "blood_raven", "find_cain", "countess", "andariel"}},
	} {
		st := campaign.NewHeroState()
		p := prog(tc.class, 1, false)
		var got []string
		for steps := 0; steps < 100; steps++ {
			p.Waypoints[3] = p.Waypoints[3] || contains(got, "cold_plains_wp")
			d := next(t, s, p, st, campaign.Opts{})
			if d.Action == campaign.ActDone {
				break
			}
			switch d.Action {
			case campaign.ActGo:
				got = append(got, d.Chapter)
				c := s.Chapter(d.Chapter)
				switch {
				case c.Done.Quest != "":
					p.Quests[c.Done.Quest] = true
				case c.Done.Waypoint > 0:
					p.Waypoints[c.Done.Waypoint] = true
				default:
					p.Level = max(p.Level, c.Done.Level)
				}
				if d.Chapter == "blood_raven" {
					p.Merc = true
				}
			case campaign.ActFarm:
				if d.Level <= p.Level || d.Farm == nil {
					t.Fatalf("%s: farm step %+v at clvl %d", tc.class, d, p.Level)
				}
				p.Level = d.Level
			default:
				t.Fatalf("%s: unexpected %+v", tc.class, d)
			}
		}
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%s ladder\n got %v\nwant %v", tc.class, got, tc.want)
		}
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// The live hero of 2026-11-11 (Paladin clvl ~12, Den and Blood Raven done, no living merc) is sent to Cain, not to a
// "level up to 16" grind for Andariel.
func TestPaladinClvl12GoesToCain(t *testing.T) {
	s := shipped(t)
	d := next(t, s, prog("paladin", 12, false, task.QDen, task.QBurial), campaign.NewHeroState(), campaign.Opts{})
	if d.Action != campaign.ActGo || d.Chapter != "find_cain" || d.Quest != task.QCain {
		t.Fatalf("%+v", d)
	}
	st := campaign.NewHeroState()
	_ = st.SetIntent(s, "find_cain", campaign.Skip, t0)
	d = next(t, s, prog("paladin", 12, false, task.QDen, task.QBurial), st, campaign.Opts{})
	if d.Action != campaign.ActFarm || d.Chapter != "tools_of_the_trade" || d.Level != 13 || d.Farm.Area != "Tower Cellar Level 1" {
		t.Fatalf("after skip: %+v", d)
	}
	if !strings.Contains(d.Why, "Farming Tower Cellar Level 1 for level 13 to unlock Charsi's hammer") {
		t.Errorf("why %q", d.Why)
	}
}

func TestFarmWhyNamesTheChapter(t *testing.T) {
	s := shipped(t)
	d := next(t, s, prog("sorceress", 7, false, task.QDen), campaign.NewHeroState(), campaign.Opts{})
	if d.Action != campaign.ActFarm || d.Chapter != "blood_raven" || d.Level != 10 {
		t.Fatalf("%+v", d)
	}
	if d.Why != "Farming Cold Plains for level 10 to unlock Blood Raven" {
		t.Errorf("why %q", d.Why)
	}
	st := campaign.NewHeroState()
	_ = st.SetGoAt(campaign.GoFloor)
	if d := next(t, s, prog("sorceress", 7, false, task.QDen), st, campaign.Opts{}); d.Level != 9 {
		t.Errorf("floor policy: %+v", d)
	}
}

func TestHoldBlocksDependents(t *testing.T) {
	s := shipped(t)
	st := campaign.NewHeroState()
	_ = st.SetIntent(s, "blood_raven", campaign.Hold, t0)
	d := next(t, s, prog("paladin", 20, true, task.QDen), st, campaign.Opts{})
	if d.Action != campaign.ActDone {
		t.Fatalf("held Blood Raven should leave nothing: %+v", d)
	}
	_ = st.SetIntent(s, "blood_raven", campaign.Skip, t0)
	if d := next(t, s, prog("paladin", 20, true, task.QDen), st, campaign.Opts{}); d.Chapter != "find_cain" || d.Action != campaign.ActGo {
		t.Fatalf("skip satisfies the prerequisite: %+v", d)
	}
}

func TestAskAtTheFloor(t *testing.T) {
	s := shipped(t)
	st := campaign.NewHeroState()
	_ = st.SetGoAt(campaign.GoFloor)
	p := prog("paladin", 8, false, task.QDen) // Blood Raven floor 8, recommended 9, intent ask
	d := next(t, s, p, st, campaign.Opts{AskWait: 10 * time.Minute})
	if d.Action != campaign.ActAsk || d.Chapter != "blood_raven" || d.Farm == nil {
		t.Fatalf("%+v", d)
	}
	s.Observe(p, st, d, t0)
	if d := next(t, s, p, st, campaign.Opts{Now: t0.Add(5 * time.Minute), AskWait: 10 * time.Minute}); d.Action != campaign.ActAsk {
		t.Fatalf("inside the veto window: %+v", d)
	}
	if d := next(t, s, p, st, campaign.Opts{Now: t0.Add(10 * time.Minute), AskWait: 10 * time.Minute}); d.Action != campaign.ActGo || d.Warn == "" {
		t.Fatalf("default yes: %+v", d)
	}
	st.SetAnswer("blood_raven", false, 8)
	if d := next(t, s, p, st, campaign.Opts{Now: t0.Add(20 * time.Minute), AskWait: 10 * time.Minute}); d.Action != campaign.ActFarm || d.Level != 9 {
		t.Fatalf("a no farms: %+v", d)
	}
	p.Level = 9
	if d := next(t, s, p, st, campaign.Opts{}); d.Action != campaign.ActGo {
		t.Fatalf("comfortable goes without asking: %+v", d)
	}
}

func TestOfferAfterFarmBudget(t *testing.T) {
	s := shipped(t)
	st := campaign.NewHeroState()
	p := prog("paladin", 16, false, task.QDen, task.QBurial, task.QCain, task.QTower, task.QTools) // Andariel: Ready (floor 16, rec 17)
	o := campaign.Opts{FarmBudget: 30 * time.Minute}
	d := next(t, s, p, st, o)
	if d.Action != campaign.ActFarm || d.Chapter != "andariel" {
		t.Fatalf("%+v", d)
	}
	s.Observe(p, st, d, t0)
	o.Now = t0.Add(31 * time.Minute)
	if d := next(t, s, p, st, o); d.Action != campaign.ActAsk || d.Chapter != "andariel" {
		t.Fatalf("offer after the budget: %+v", d)
	}
	st.SetAnswer("andariel", true, 16)
	if d := next(t, s, p, st, o); d.Action != campaign.ActGo || d.Warn == "" {
		t.Fatalf("yes goes at the floor: %+v", d)
	}
}

func TestGoNowAndModes(t *testing.T) {
	s := shipped(t)
	st := campaign.NewHeroState()
	if err := st.SetGoNow(s, "andariel"); err != nil {
		t.Fatal(err)
	}
	p := prog("sorceress", 6, false, task.QDen)
	d := next(t, s, p, st, campaign.Opts{})
	if d.Action != campaign.ActGo || d.Chapter != "andariel" || !strings.Contains(d.Warn, "too_early") {
		t.Fatalf("go now is never refused: %+v", d)
	}
	p.Quests[task.QAndariel] = true
	if done := s.Observe(p, st, d, t0); st.GoNow != "" || !contains(done, "andariel") || !contains(done, "den_of_evil") {
		t.Fatalf("note: %v %q", done, st.GoNow)
	}
	if done := s.Observe(p, st, d, t0); len(done) != 0 {
		t.Errorf("done twice: %v", done)
	}
	_ = st.SetMode(campaign.Free)
	if d := next(t, s, p, st, campaign.Opts{}); d.Action != campaign.ActFree {
		t.Errorf("free: %+v", d)
	}
	_ = st.SetMode(campaign.Orders)
	if d := next(t, s, p, st, campaign.Opts{}); d.Action != campaign.ActOrders {
		t.Errorf("orders: %+v", d)
	}
	if st.SetMode("auto") == nil || st.SetIntent(s, "nope", campaign.Skip, t0) == nil || st.SetIntent(s, "countess", "maybe", t0) == nil {
		t.Error("bad input accepted")
	}
}

func TestStorePersists(t *testing.T) {
	s := shipped(t)
	path := filepath.Join(t.TempDir(), "state", "campaign.json")
	st, err := campaign.LoadStore(path)
	if err != nil {
		t.Fatal(err)
	}
	h := st.Hero("Palli")
	_ = h.SetIntent(s, "countess", campaign.Skip, t0)
	_ = h.SetGoAt(campaign.GoFloor)
	if err := st.Save(path); err != nil {
		t.Fatal(err)
	}
	back, err := campaign.LoadStore(path)
	if err != nil {
		t.Fatal(err)
	}
	b := back.Hero("Palli")
	if b.Intents["countess"] != campaign.Skip || b.GoAt != campaign.GoFloor || b.Mode != campaign.Guided || b.Skipped["countess"].IsZero() {
		t.Errorf("%+v", b)
	}
	if other := back.Hero("Sorci"); other.Mode != campaign.Guided || len(other.Intents) != 0 {
		t.Errorf("new hero %+v", other)
	}
}
