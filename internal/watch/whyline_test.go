package watch

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
)

func fakeWhy(t *testing.T) (*Whyline, *float64) {
	t.Helper()
	now := new(float64)
	w, err := NewWhyline(func() float64 { return *now })
	if err != nil {
		t.Fatal(err)
	}
	return w, now
}

func p(kv ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

// Every code renders in both phrasings with its parameters filled and no brace left; the examples of the brief read
// as the card says.
func TestWhyTemplates(t *testing.T) {
	w, _ := fakeWhy(t)
	params := map[string]map[string]string{
		"quest_step": p("step", "Clear the Den of Evil", "level", "6"), "walk": p("area", "Blood Moor", "level", "6"),
		"fight": p("area", "Cold Plains", "life", "80"), "town_trip": p("why", "2 potions left (I go at 3)"),
		"town_chores": p("why", "selling"), "heal_trip": p("life", "40", "line", "50"), "wait_heal": p("why", "no potion"),
		"guard_exit": p("why", "too many"), "farm_runs": p("area", "Mausoleum"), "farm_gold": p("level", "9"), "saving_for": p("area", "Blood Moor", "item", "healing potions", "need", "40"),
		"farm_user": p("area", "Stony Field", "have", "40", "want", "190"),
		"hold":      p("question", "the shrine"), "activity": p("activity", "Fighting"),
	}
	for _, code := range WhyCodes() {
		for v := 0; v < 2; v++ {
			text, ok := w.render(code, params[code], v)
			if !ok || strings.ContainsAny(text, "{}") || text == "" {
				t.Errorf("%s variant %d: %q ok=%v", code, v, text, ok)
			}
		}
		if n := len(w.tab.Codes[code].Say); n != 2 {
			t.Errorf("%s has %d phrasings, want 2", code, n)
		}
	}
	if got, _ := w.render("fight", p("area", "Cold Plains", "life", "80"), 0); got != "Fighting in Cold Plains, my life is 80%." {
		t.Errorf("fight: %q", got)
	}
	if got, _ := w.render("town_trip", p("why", "2 potions left (I go at 3)"), 0); got != "Going back to town: 2 potions left (I go at 3)." {
		t.Errorf("town_trip: %q", got)
	}
}

func TestWhyUnknownCode(t *testing.T) {
	w, _ := fakeWhy(t)
	w.Observe(goals.Why{Code: "nope"}, "x", "g", "")
	if v := w.View(); v.Text != "Doing what the plan says." {
		t.Errorf("production line: %q", v.Text)
	}
	w2, now2 := fakeWhy(t)
	w2.Debug = true
	w2.Observe(goals.Why{Code: "nope"}, "x", "g", "")
	if v := w2.View(); v.Text != "unknown reason code nope" {
		t.Errorf("debug line: %q", v.Text)
	}
	*now2 = 10
	w2.Observe(goals.Why{Code: "fight", Params: p("area", "A")}, "x", "g", "") // missing life
	if v := w2.View(); !strings.Contains(v.Text, "lacks parameter life") {
		t.Errorf("missing parameter: %q", v.Text)
	}
}

// The line is read once a second, kept for 3 s, and replaced at once when the goal changes.
func TestWhyRateLimitAndHold(t *testing.T) {
	w, now := fakeWhy(t)
	fight := func(life string) goals.Why { return goals.Why{Code: "fight", Params: p("area", "A", "life", life)} }
	w.Observe(fight("90"), "", "g1", "")
	*now = 0.5
	w.Observe(fight("50"), "", "g1", "") // inside the read interval
	if v := w.View(); !strings.Contains(v.Text, "90%") {
		t.Errorf("read twice inside a second: %q", v.Text)
	}
	*now = 1.5
	w.Observe(fight("50"), "", "g1", "") // readable, but the line is held
	if v := w.View(); !strings.Contains(v.Text, "90%") {
		t.Errorf("line changed inside 3 s: %q", v.Text)
	}
	*now = 2.0
	w.Observe(goals.Why{Code: "to_town"}, "", "g2", "") // a new goal ends the hold
	if v := w.View(); v.Code != "to_town" {
		t.Errorf("goal change ignored: %q", v.Code)
	}
	*now = 5.5
	w.Observe(fight("50"), "", "g2", "")
	if v := w.View(); v.Code != "fight" || !strings.Contains(v.Text, "50%") {
		t.Errorf("after the hold: %+v", v)
	}
	// same code, new number: the line updates but the history and the count do not grow
	*now = 9
	w.Observe(fight("40"), "", "g2", "")
	if n := w.Counts()["fight"]; n != 2 {
		t.Errorf("fight started %d times, want 2", n)
	}
	if h := w.History(); len(h) != 3 || h[0].Code != "fight" {
		t.Errorf("history: %+v", h)
	}
}

func TestWhyHistoryKeepsFiveNewestFirst(t *testing.T) {
	w, now := fakeWhy(t)
	codes := []string{"walk", "to_town", "back_to_field", "area_done", "claim_trip", "farm_runs", "farm_gold"}
	for i, c := range codes {
		*now = float64(i) * 10
		w.Observe(goals.Why{Code: c, Params: p("area", "A", "level", "3")}, "", "g"+c, "")
	}
	h := w.History()
	if len(h) != 5 || h[0].Code != "farm_gold" || h[4].Code != "back_to_field" {
		t.Errorf("history: %+v", h)
	}
	if w.Counts()["walk"] != 1 {
		t.Errorf("counts: %v", w.Counts())
	}
}

// With no reason from the bot an open question holds the hero ("Hold: waiting for your yes"); with none of either the
// activity text is the line.
func TestWhyHoldAndActivityFallback(t *testing.T) {
	w, _ := fakeWhy(t)
	w.Observe(goals.Why{}, "Fighting", "g", "")
	if v := w.View(); v.Code != "activity" || v.Text != "Now: Fighting." {
		t.Errorf("fallback: %+v", v)
	}
	w2, _ := fakeWhy(t)
	w2.Observe(goals.Why{}, "Idle", "g", "the shrine")
	if v := w2.View(); v.Code != "hold" || !strings.HasPrefix(v.Text, "Hold: waiting for your yes about the shrine") {
		t.Errorf("hold: %+v", v)
	}
	w3, _ := fakeWhy(t)
	w3.Observe(goals.Why{}, "", "g", "")
	if v := w3.View(); v.Text != "" {
		t.Errorf("nothing to say: %+v", v)
	}
}

// Parameters from the game or the player are plain short text.
func TestWhyParamsArePlain(t *testing.T) {
	w, _ := fakeWhy(t)
	w.Observe(goals.Why{Code: "walk", Params: p("area", "<b>x</b>\n"+strings.Repeat("y", 300), "level", "1")}, "", "g", "")
	v := w.View()
	if strings.Contains(v.Text, "\n") || len(v.Text) > 260 {
		t.Errorf("not plain/short: %q", v.Text)
	}
}

// Two classes: the table has no class, the same codes serve both heroes through the API; golden JSON of the field.
func TestWhylineAPIGolden(t *testing.T) {
	for _, build := range []string{"sorceress", "paladin"} {
		r := newRig(t, build)
		w := r.sim.Hub.Whyline()
		r.sim.Step(2 * time.Second)
		r.sim.Hub.SetHero(HeroInfo{Class: build, Level: 6, Area: "Blood Moor"})
		w.Observe(goals.Why{Code: "heal_trip", Params: p("life", "40", "line", "50")}, "Going to town to heal", "g1", "")
		r.sim.Step(5 * time.Second)
		w.Observe(goals.Why{Code: "fight", Params: p("area", "Blood Moor", "life", "80")}, "Fighting in Blood Moor", "g2", "")
		golden(t, "whyline-"+build, r.get(WhylineRoute, nil).Body.String())
		if m := r.postRaw(r.view, WhylineRoute, "{}", nil); m.Code != 405 {
			t.Errorf("POST on the viewer: %d", m.Code)
		}
	}
}

// The reason is the Why card of the command centre (side band, /api/hero), no longer a line in the top band.
func TestWhyCardInTheCommandCentre(t *testing.T) {
	ms, err := Manifests()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		if m.Band == "status" && m.ID != "hero-status" {
			t.Errorf("%s in the top band", m.ID)
		}
		if m.ID == "why" && (m.Band != "side" || m.Endpoint != HeroRoute) {
			t.Errorf("why: band %s endpoint %s", m.Band, m.Endpoint)
		}
	}
	js, _ := os.ReadFile("web/panels/why/panel.js")
	for _, want := range []string{"'Would change it'", "textContent"} {
		if !strings.Contains(string(js), want) {
			t.Errorf("why panel lacks %s", want)
		}
	}
}

// The status feed drives the line, and an open question shows as a hold while the runner idles.
func TestWhylineFromStatus(t *testing.T) {
	r := newRig(t, "paladin")
	b := &GoalsOrders{Hub: r.sim.Hub}
	b.OnStatus(goals.Status{State: goals.RunnerRunning, Activity: "Walking to Cold Plains", GoalID: "g",
		Why: goals.Why{Code: "walk", Params: p("area", "Cold Plains", "level", "6")}})
	var v WhylineView
	r.get(WhylineRoute, &v)
	if !strings.Contains(v.Text, "Cold Plains") {
		t.Errorf("status line: %+v", v)
	}
}

func TestWhylineRace(t *testing.T) {
	w, now := fakeWhy(t)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				w.Observe(goals.Why{Code: "fight", Params: p("area", "A", "life", "5")}, "", "g", "")
				_ = w.View()
				_ = w.Counts()
			}
		}()
	}
	wg.Wait()
	_ = now
}

func TestChronicleCarriesReasons(t *testing.T) {
	w, now := fakeWhy(t)
	*now = 4
	w.Observe(goals.Why{Code: "to_town"}, "", "g", "")
	c := &Chronicle{}
	c.SetWhy(w.History)
	if h := c.why(); len(h) != 1 || h[0].Code != "to_town" {
		t.Errorf("history: %+v", h)
	}
}
