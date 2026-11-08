package pact

import (
	"d2lod-bot/internal/unlock"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeSheet struct{ cur map[string]string }

func (f *fakeSheet) Current(r string) string {
	if v := f.cur[r]; v != "" {
		return v
	}
	return "default"
}
func (f *fakeSheet) Default(string) string { return "default" }
func (f *fakeSheet) Set(r, o string) error { f.cur[r] = o; return nil }

type rig struct {
	t       *testing.T
	c       *Controller
	knobs   map[string]float64 // what the tuner holds after the edits
	edits   int
	sheet   *fakeSheet
	paused  []string
	present bool
	now     time.Time
	eng     *unlock.Engine
	consent []string
}

func newRig(t *testing.T, class, path string) *rig {
	t.Helper()
	r := &rig{t: t, knobs: map[string]float64{}, sheet: &fakeSheet{cur: map[string]string{}}, now: time.Unix(1_700_000_000, 0)}
	lad, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	if r.eng, err = unlock.Open("", "t", lad, nil); err != nil {
		t.Fatal(err)
	}
	c, err := New(Config{
		Gate: r.eng, ConsentLog: func(l string) { r.consent = append(r.consent, l) },
		Cat: shipped(t), Class: class, Path: path, Sheet: r.sheet,
		Knobs: func(over map[string]*float64) error {
			r.edits++
			for k, v := range over {
				if v == nil {
					delete(r.knobs, k)
				} else {
					r.knobs[k] = *v
				}
			}
			return nil
		},
		Pause:   func(why string) error { r.paused = append(r.paused, why); return nil },
		Present: func() bool { return r.present },
		Now:     func() time.Time { return r.now },
	})
	if err != nil {
		t.Fatal(err)
	}
	r.c = c
	return r
}

var field = Where{Level: 6, Age: time.Hour}

func TestOffInGatesAndMeasuredRuns(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	for _, c := range []struct {
		env map[string]string
		off bool
	}{
		{nil, false},
		{map[string]string{"D2BOT_PACTS": "off"}, true},
		{map[string]string{"D2BOT_PACTS": "OFF"}, true},
		{map[string]string{"D2BOT_PACTS": "on"}, false},
		{map[string]string{"D2BOT_RUN": "g2c"}, true}, // the runstart protocol marks a measured run
	} {
		if got := Off(env(c.env)); got != c.off {
			t.Errorf("%v: Off = %v, want %v", c.env, got, c.off)
		}
	}
}

func TestTithePresentConsentAndSafeZones(t *testing.T) {
	for _, class := range []string{"paladin", "sorceress"} {
		r := newRig(t, class, "")
		if err := r.c.Select([]Pick{{"blood_ravens_tithe", 1}}); err != nil {
			t.Fatal(err)
		}
		step := func(w Where) {
			t.Helper()
			if err := r.c.Evaluate(w); err != nil {
				t.Fatal(err)
			}
		}
		held := func() bool { return len(r.knobs) > 0 }
		// safe zones: town, the Den, the first minutes, below clvl 5
		step(Where{Level: 6, Age: time.Hour, InTown: true})
		step(Where{Level: 6, Age: time.Hour, InDen: true})
		step(Where{Level: 6, Age: 3 * time.Minute})
		step(Where{Level: 3, Age: time.Hour})
		r.present = true
		step(Where{Level: 3, Age: time.Hour})
		if held() {
			t.Fatalf("%s: a Pact acted in a safe zone: %v", class, r.knobs)
		}
		// a human is connected, out of the zones: all four belt columns empty
		step(field)
		for _, k := range []string{"resource.beltCol1", "resource.beltCol2", "resource.beltCol3", "resource.beltCol4"} {
			if r.knobs[k] != 4 {
				t.Errorf("%s: %s = %v", class, k, r.knobs[k])
			}
		}
		n := r.edits
		step(field)
		if r.edits != n {
			t.Errorf("%s: an unchanged state edited the tuner again", class)
		}
		// rank 1 lifts at clvl 8
		step(Where{Level: 8, Age: time.Hour})
		if held() {
			t.Errorf("%s: still held at clvl 8: %v", class, r.knobs)
		}
		step(field)
		// the human leaves and no consent: back to the profile values
		r.present = false
		step(field)
		if held() {
			t.Errorf("%s: acted with nobody watching and no consent", class)
		}
		// consent: the name typed out, dated, per Pact
		if err := r.c.Grant("blood_ravens_tithe", "yes", 0); err == nil || !strings.Contains(err.Error(), "typed out") {
			t.Errorf("%s: a short word is not consent: %v", class, err)
		}
		if err := r.c.Grant("andariels_shadow", "Andariel's Shadow", 0); err == nil {
			t.Errorf("%s: consent for a Pact that is not chosen", class)
		}
		if err := r.c.Grant("blood_ravens_tithe", "  blood raven's  TITHE ", 0); err != nil {
			t.Fatal(err)
		}
		step(field)
		if !held() {
			t.Errorf("%s: consent did not switch it on", class)
		}
		// consent ends: after the days, with a difficulty change, with a rank change
		step(Where{Level: 6, Age: time.Hour, Difficulty: 1})
		if held() {
			t.Errorf("%s: consent survived a difficulty change", class)
		}
		r.now = r.now.Add(8 * 24 * time.Hour)
		step(field)
		if held() {
			t.Errorf("%s: consent survived 8 days", class)
		}
		if last := r.consent[len(r.consent)-1]; !strings.Contains(last, "ran out") {
			t.Errorf("%s: expiry not said on the return line: %q", class, last)
		}
		r.now = r.now.Add(-8 * 24 * time.Hour)
		if err := r.c.Grant("blood_ravens_tithe", "Blood Raven's Tithe", 0); err != nil {
			t.Fatal(err)
		}
		step(field)
		if !held() {
			t.Errorf("%s: a new consent should switch it on again", class)
		}
		if err := r.c.Select([]Pick{{"blood_ravens_tithe", 3}}); err != nil {
			t.Fatal(err)
		}
		step(field)
		if held() {
			t.Errorf("%s: a rank change must end the consent", class)
		}
	}
}

func TestRankLadderForPotions(t *testing.T) {
	r := newRig(t, "paladin", "")
	r.present = true
	r.c.st.Masteries = 3 // rank 5 costs 15 heat; the cap starts at 12
	for _, c := range []struct {
		rank, level int
		held        bool
	}{{1, 7, true}, {1, 8, false}, {3, 13, true}, {3, 14, false}, {5, 40, true}, {5, 90, true}} {
		if err := r.c.Select([]Pick{{"blood_ravens_tithe", c.rank}}); err != nil {
			t.Fatal(err)
		}
		if err := r.c.Evaluate(Where{Level: c.level, Age: time.Hour}); err != nil {
			t.Fatal(err)
		}
		if (len(r.knobs) > 0) != c.held {
			t.Errorf("rank %d clvl %d: held %v, want %v", c.rank, c.level, len(r.knobs) > 0, c.held)
		}
	}
}

func TestShadowDeathLimitPausesAndResumes(t *testing.T) {
	for _, class := range []string{"paladin", "sorceress"} {
		r := newRig(t, class, "")
		r.present = true
		if err := r.c.Select([]Pick{{"andariels_shadow", 3}}); err != nil { // limit 2
			t.Fatal(err)
		}
		// a death in a safe zone does not count
		r.c.Evaluate(Where{Level: 3, Age: time.Hour})
		r.c.OnDeath()
		r.c.OnDeath()
		r.c.Evaluate(field)
		if err := r.c.OnDeath(); err != nil || len(r.paused) != 0 {
			t.Fatalf("%s: first counted death: %v %v", class, err, r.paused)
		}
		if err := r.c.OnDeath(); err != nil || len(r.paused) != 1 || !strings.Contains(r.paused[0], "limit is 2") {
			t.Fatalf("%s: limit not reached: %v %v", class, err, r.paused)
		}
		r.c.OnDeath()
		if len(r.paused) != 1 {
			t.Errorf("%s: paused twice", class)
		}
		if !strings.HasPrefix(r.c.Paused(), "Andariel's Shadow") {
			t.Errorf("%s: Paused() = %q", class, r.c.Paused())
		}
		if l := r.c.Lines(); len(l) != 1 || l[0].Active || !strings.Contains(l[0].Status, "paused") {
			t.Errorf("%s: lines while paused: %+v", class, l)
		}
		r.c.Resume()
		r.c.Evaluate(field)
		r.c.OnDeath()
		if len(r.paused) != 1 || r.c.Paused() != "" {
			t.Errorf("%s: the count must start again after Resume", class)
		}
	}
	// nobody watching and no consent: deaths are not counted
	r := newRig(t, "paladin", "")
	r.c.Select([]Pick{{"andariels_shadow", 5}})
	r.c.Evaluate(field)
	r.c.OnDeath()
	if len(r.paused) != 0 {
		t.Error("counted a death while the Pact was not live")
	}
}

func TestRowsAreRestored(t *testing.T) {
	r := newRig(t, "sorceress", "")
	r.present = true
	r.sheet.cur["repair"] = "always" // the player's own choice
	if err := r.c.Select([]Pick{{"griswolds_debt", 1}}); err != nil {
		t.Fatal(err)
	}
	r.c.Evaluate(field)
	if r.sheet.cur["repair"] != "low" {
		t.Fatalf("repair = %q", r.sheet.cur["repair"])
	}
	r.c.Evaluate(Where{Level: 6, Age: time.Hour, InTown: true})
	if r.sheet.cur["repair"] != "always" {
		t.Errorf("the player's option was not put back: %q", r.sheet.cur["repair"])
	}
}

func TestSelectRulesStateAndLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pacts-state.json")
	r := newRig(t, "paladin", path)
	if err := r.c.Select([]Pick{{"blood_ravens_tithe", 3}, {"andariels_shadow", 3}}); err == nil || !strings.Contains(err.Error(), "over the cap") {
		t.Errorf("over the cap: %v", err)
	}
	if err := r.c.Select([]Pick{{"blood_ravens_tithe", 1}, {"andariels_shadow", 1}}); err != nil {
		t.Fatal(err)
	}
	if err := r.c.Grant("andariels_shadow", "Andariel's Shadow", 0); err != nil {
		t.Fatal(err)
	}
	r2 := newRig(t, "paladin", path) // a restart keeps the choice and the consent
	if len(r2.c.Picks()) != 2 {
		t.Errorf("picks after restart: %v", r2.c.Picks())
	}
	r2.c.Evaluate(field)
	lines := r2.c.Lines()
	if len(lines) != 2 || lines[0].Active || !lines[1].Active || !strings.Contains(lines[1].Text, "stops in town") {
		t.Errorf("lines: %+v", lines)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("state file: %v %v", fi, err)
	}
	// a corrupt file and a file the rules refuse are errors naming the file
	os.WriteFile(path, []byte("{"), 0o600)
	if _, err := New(Config{Cat: shipped(t), Class: "paladin", Path: path}); err == nil || !strings.Contains(err.Error(), "pacts-state.json") {
		t.Errorf("corrupt: %v", err)
	}
	os.WriteFile(path, []byte(`{"picks":[{"ID":"pact_of_rotting_bone","Rank":1}]}`), 0o600)
	if _, err := New(Config{Cat: shipped(t), Class: "necromancer", Path: path}); err == nil || !strings.Contains(err.Error(), "necromancer") {
		t.Errorf("class scope on load: %v", err)
	}
}

// Rakanishu's Bargain rank 3 goes through the same generic path: a sheet row and a knob.
func TestBargainRank3SetsRowAndKnob(t *testing.T) {
	for _, class := range []string{"paladin", "sorceress"} {
		r := newRig(t, class, "")
		r.present = true
		if err := r.c.Select([]Pick{{"rakanishus_bargain", 3}}); err != nil {
			t.Fatal(err)
		}
		if err := r.c.Evaluate(field); err != nil {
			t.Fatal(err)
		}
		if r.sheet.cur["tp"] != "walk" || r.knobs["town.tpReserve"] != 1 {
			t.Errorf("%s: row %q knobs %v", class, r.sheet.cur["tp"], r.knobs)
		}
		r.c.Evaluate(Where{Level: 6, Age: time.Hour, InTown: true})
		if r.sheet.cur["tp"] != "default" || len(r.knobs) != 0 {
			t.Errorf("%s: not put back: row %q knobs %v", class, r.sheet.cur["tp"], r.knobs)
		}
	}
}

// kill feeds a boss kill to the ladder, as the bot's kill events do.
func (r *rig) kill(boss, diff string) {
	r.t.Helper()
	if _, err := r.eng.Observe(unlock.Event{Type: unlock.EvBossKill, Name: boss, Sub: diff}); err != nil {
		r.t.Fatal(err)
	}
}
