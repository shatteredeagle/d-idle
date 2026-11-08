package watch

import (
	"strings"
	"testing"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
)

// stanceKnobs returns the last knob call as plain values.
func (r *polRig) lastKnobs() map[string]float64 {
	out := map[string]float64{}
	for _, c := range r.calls {
		for k, v := range c {
			if v == nil {
				delete(out, k)
				continue
			}
			out[k] = *v
		}
	}
	return out
}

// The stance row: the same option gives a melee hero and a caster different numbers; Balanced changes nothing.
func TestStanceKnobsDifferByClass(t *testing.T) {
	pal := newPolRig(t, "paladin", true)
	sor := newPolRig(t, "sorceress", true)
	pal.set("stance", "aggressive")
	sor.set("stance", "aggressive")
	p, s := pal.lastKnobs(), sor.lastKnobs()
	if p["bot.clearRadius"] != 40 || s["bot.clearRadius"] != 36 {
		t.Errorf("aggressive notice radius: paladin %v sorceress %v", p["bot.clearRadius"], s["bot.clearRadius"])
	}
	if p["combat.kiteLifePct"] == s["combat.kiteLifePct"] {
		t.Errorf("a melee hero and a caster must not share the kite line: %v", p["combat.kiteLifePct"])
	}
	// back to Balanced resets every knob the stance set
	pal.set("stance", "default")
	for k, v := range pal.calls[len(pal.calls)-1] {
		if v != nil {
			t.Errorf("balanced left %s = %v", k, *v)
		}
	}
	// cautious is tighter than balanced on both classes
	for _, r := range []*polRig{pal, sor} {
		r.set("stance", "cautious")
		k := r.lastKnobs()
		if k["bot.clearRadius"] >= 30 || k["combat.chickenPct"] <= 25 || k["combat.drinkLifePct"] <= 55 {
			t.Errorf("%s cautious: %v", r.class, k)
		}
	}
}

// Every knob value of the table is a real knob and inside its range, for every class.
func TestStanceKnobValuesAreInRange(t *testing.T) {
	cat, _ := decide.Shipped()
	tbl, err := decide.ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	row, err := tbl.Row("stance")
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]env.Knob{}
	for _, k := range env.Knobs() {
		byKey[k.Key] = k
	}
	for _, o := range row.Options {
		for class := range cat.Classes {
			kv, err := o.KnobsFor(class)
			if err != nil {
				t.Fatalf("%s/%s: %v", o.ID, class, err)
			}
			if o.ID == "default" && len(kv) != 0 {
				t.Errorf("Balanced must change nothing: %v", kv)
			}
			for k, v := range kv {
				kn, ok := byKey[k]
				if !ok || v < kn.Min || v > kn.Max {
					t.Errorf("%s/%s: %s = %v (knob %v)", o.ID, class, k, v, ok)
				}
			}
		}
		if o.ID != "default" && o.Says == "" {
			t.Errorf("%s has no plain words", o.ID)
		}
	}
	if _, err := row.Options[0].KnobsFor("necromancer"); err == nil || !strings.Contains(err.Error(), `"necromancer"`) {
		t.Errorf("unknown class: %v", err)
	}
}

// The Paladin starts Aggressive (a melee hero has to walk up to its target); the Sorceress starts Balanced, and a
// named run (SetStarts with nothing) starts nobody on a stance.
func TestStanceClassStart(t *testing.T) {
	for class, want := range map[string]string{"paladin": "aggressive", "sorceress": "default"} {
		r := newPolRig(t, class, true)
		r.sheet.SetStarts(nil) // the embedded table
		if err := r.sheet.EnsureStart(); err != nil {
			t.Fatal(err)
		}
		if got := r.sheet.CurrentOption("stance"); got != want {
			t.Errorf("%s starts on %q, want %q", class, got, want)
		}
		if len(r.sheet.Changes()) != 0 {
			t.Errorf("%s: a start is not a change", class)
		}
	}
	r := newPolRig(t, "paladin", true) // starts off, as under D2BOT_RUN
	if err := r.sheet.EnsureStart(); err != nil || r.sheet.CurrentOption("stance") != "default" || len(r.calls) != 0 {
		t.Errorf("named run: %v %q %d", err, r.sheet.CurrentOption("stance"), len(r.calls))
	}
	// no class yet: nothing happens, no error
	r.class = ""
	if err := r.sheet.EnsureStart(); err != nil {
		t.Errorf("no class: %v", err)
	}
}

// The card says it in plain words, and a stored class-specific option comes back after a restart once the class is known.
func TestStanceCardAndReapplyAfterClassKnown(t *testing.T) {
	r := newPolRig(t, "paladin", true)
	r.now = 100
	r.set("stance", "cautious")
	sv := r.sheet.Section().Build(nil)
	if sv == nil || !strings.Contains(sv.Lines[0], "Cautious: only fights what comes close") {
		t.Fatalf("card: %+v", sv)
	}
	st, err := decide.Open(r.dir, "pol", r.cat, &decide.ManualClock{})
	if err != nil {
		t.Fatal(err)
	}
	var calls []map[string]*float64
	cls := ""
	ps, _ := NewPolicySheet(st, r.cat, r.tbl, func() string { return cls }, func() float64 { return 0 }, NewAwayLog(nil),
		func(o map[string]*float64) error { calls = append(calls, o); return nil })
	if err := ps.Reapply(); err != nil {
		t.Fatal(err)
	}
	cls = "paladin"
	if err := ps.EnsureStart(); err != nil {
		t.Fatal(err)
	}
	if len(calls) == 0 || calls[len(calls)-1]["bot.clearRadius"] == nil || *calls[len(calls)-1]["bot.clearRadius"] != 20 {
		t.Errorf("class knobs after the class showed up: %v", calls)
	}
}

// POST /api/policy {row: stance} on two classes: the reply and the stored change are golden files.
func TestStancePostGolden(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newPolRig(t, class, true)
		ans := r.sheet.PolicyAnswerHandler(AnswerOptions{Token: "tok"})
		w := polPost(ans, map[string]string{"Authorization": "Bearer tok"}, `{"row":"stance","option":"aggressive"}`)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", class, w.Code, w.Body)
		}
		polGolden(t, "post-stance-"+class+".json", w.Body.Bytes())
		if r.sheet.CurrentOption("stance") != "aggressive" || len(r.calls) != 1 {
			t.Errorf("%s: option %q, knob calls %d", class, r.sheet.CurrentOption("stance"), len(r.calls))
		}
		if w := polPost(ans, map[string]string{"Authorization": "Bearer tok"}, `{"row":"stance","option":"reckless"}`); w.Code != 400 || !strings.Contains(w.Body.String(), "reckless") {
			t.Errorf("%s: unknown option: %d %s", class, w.Code, w.Body)
		}
	}
}
