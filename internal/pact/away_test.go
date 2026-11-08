package pact

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPresenceWindowAndStream(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	p := NewPresence(5*time.Minute, func() time.Time { return now })
	if p.Present() {
		t.Fatal("present before any look")
	}
	p.Touch()
	now = now.Add(4 * time.Minute)
	if !p.Present() {
		t.Fatal("not present inside the window")
	}
	now = now.Add(2 * time.Minute)
	if p.Present() {
		t.Fatal("still present after the window")
	}
	open := 1
	p.Open = func() int { return open }
	if !p.Present() {
		t.Fatal("an open stream is a human")
	}
	open = 0
	if p.Present() {
		t.Fatal("stream closed and window over")
	}
}

func TestPresenceGatesPact(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newRig(t, class, "")
		if err := r.c.Select([]Pick{{"blood_ravens_tithe", 1}}); err != nil {
			t.Fatal(err)
		}
		pr := NewPresence(0, func() time.Time { return r.now })
		r.c.cfg.Present = pr.Present
		r.c.Evaluate(field)
		if len(r.knobs) > 0 {
			t.Fatalf("%s: acted with nobody there", class)
		}
		pr.Touch()
		r.c.Evaluate(field)
		if len(r.knobs) == 0 {
			t.Fatalf("%s: did not act with a viewer", class)
		}
		r.now = r.now.Add(6 * time.Minute)
		r.c.Evaluate(field)
		if len(r.knobs) > 0 {
			t.Fatalf("%s: kept acting after the viewer left", class)
		}
	}
}

func TestConsentHoursExpiryAndRankChange(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newRig(t, class, "")
		r.c.Select([]Pick{{"blood_ravens_tithe", 1}})
		if err := r.c.GrantFor("blood_ravens_tithe", "Blood Raven's Tithe", 0, 8*24); err == nil {
			t.Fatalf("%s: more than consent_days accepted", class)
		}
		if err := r.c.GrantFor("blood_ravens_tithe", "Blood Raven's Tithe", 0, 2); err != nil {
			t.Fatal(err)
		}
		r.c.Evaluate(field)
		if len(r.knobs) == 0 {
			t.Fatalf("%s: consent not honoured", class)
		}
		r.now = r.now.Add(3 * time.Hour)
		r.c.Evaluate(field)
		if len(r.knobs) > 0 {
			t.Fatalf("%s: consent outlived its 2 hours", class)
		}
		r.c.GrantFor("blood_ravens_tithe", "Blood Raven's Tithe", 0, 2)
		r.c.Select([]Pick{{"blood_ravens_tithe", 3}}) // rank change
		r.c.Evaluate(field)
		if len(r.knobs) > 0 {
			t.Fatalf("%s: rank change kept the consent", class)
		}
		// difficulty change
		r.c.GrantFor("blood_ravens_tithe", "Blood Raven's Tithe", 0, 0)
		r.c.Evaluate(Where{Level: 6, Age: time.Hour, Difficulty: 1})
		if len(r.knobs) > 0 {
			t.Fatalf("%s: a Normal consent counted in Nightmare", class)
		}
	}
}

func TestLowLifePauseAndResume(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newRig(t, class, "")
		var log []string
		r.c.cfg.Journal = func(s string) { log = append(log, s) }
		r.c.Select([]Pick{{"blood_ravens_tithe", 1}})
		r.present = true
		at := func(life int) { r.c.Evaluate(Where{Level: 6, Age: time.Hour, Life: life, MaxLife: 100}) }
		at(100)
		if len(r.knobs) == 0 {
			t.Fatalf("%s: not active at full life", class)
		}
		at(34)
		if len(r.knobs) > 0 || len(log) != 1 || !strings.Contains(log[0], "35%") {
			t.Fatalf("%s: no pause under the floor: knobs %v log %v", class, r.knobs, log)
		}
		if l := r.c.Lines(); len(l) != 1 || l[0].Active || !strings.Contains(l[0].Status, "life") {
			t.Fatalf("%s: line %+v", class, l)
		}
		at(40) // above the floor but inside the margin: still paused
		if len(r.knobs) > 0 {
			t.Fatalf("%s: resumed inside the margin", class)
		}
		at(45)
		if len(r.knobs) == 0 || len(log) != 2 {
			t.Fatalf("%s: did not resume: log %v", class, log)
		}
		// unknown life never pauses
		r.c.Evaluate(Where{Level: 6, Age: time.Hour})
		if len(r.knobs) == 0 {
			t.Fatalf("%s: unknown life paused", class)
		}
	}
}

func TestOfferNumbersAndWarnings(t *testing.T) {
	r := newRig(t, "paladin", "")
	r.c.Select([]Pick{{"andariels_shadow", 1}})
	o := r.c.Offer(1)
	if len(o.Lines) != 1 || !strings.Contains(strings.Join(o.Lines[0].CanLose, " "), "3 deaths") {
		t.Fatalf("offer %+v", o)
	}
	if !strings.Contains(strings.Join(o.Warnings, " "), "40 lower") || !strings.Contains(strings.Join(o.Warnings, " "), "5%") {
		t.Fatalf("nightmare warning missing: %v", o.Warnings)
	}
	if o.MaxHours != 168 || o.Lines[0].TypeToGive != "Andariel's Shadow" || o.Lines[0].Consent != "none" {
		t.Fatalf("offer %+v", o)
	}
	r.c.GrantFor("andariels_shadow", "Andariel's Shadow", 1, 4)
	if o = r.c.Offer(1); !strings.HasPrefix(o.Lines[0].Consent, "until ") {
		t.Fatalf("consent not shown: %+v", o.Lines[0])
	}
	if o = r.c.Offer(0); o.Lines[0].Consent != "none" {
		t.Fatal("Nightmare consent shown on Normal")
	}
}

func TestHardcoreSeparateAndStubbed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hardcore.json")
	h, err := OpenHardcore(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Confirm("Hero", "yes"); err == nil {
		t.Fatal("a click-word confirmed hardcore")
	}
	if err := h.Confirm("Hero", "Blood Raven's Tithe"); err == nil {
		t.Fatal("a Pact name confirmed hardcore")
	}
	if err := h.AllowAway(true); err == nil {
		t.Fatal("away allowed before the confirmation")
	}
	if err := h.Confirm("Hero", "hero"); err != nil {
		t.Fatal(err)
	}
	if s := h.State(); !s.Confirmed || s.AwayAllowed || s.Active || !strings.Contains(s.Note, "not active") {
		t.Fatalf("%+v", s)
	}
	if err := h.Activate(); err == nil {
		t.Fatal("hardcore activated")
	}
	h2, _ := OpenHardcore(path, nil) // survives a restart, still inactive
	if s := h2.State(); !s.Confirmed || s.Active {
		t.Fatalf("%+v", s)
	}
	// the Pact state file never mentions it
	r := newRig(t, "paladin", filepath.Join(t.TempDir(), "p.json"))
	r.c.Select([]Pick{{"blood_ravens_tithe", 1}})
	h2.Revoke()
	if s := h2.State(); s.Confirmed {
		t.Fatal("revoke kept it")
	}
}
