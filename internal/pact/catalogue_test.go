package pact

import (
	"fmt"
	"strings"
	"testing"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/resource"
)

// The seven class names of the game (the class table of the plan data).
var sevenClasses = map[string]bool{"amazon": true, "assassin": true, "barbarian": true, "druid": true, "necromancer": true, "paladin": true, "sorceress": true}

// shippedRefs checks the catalogue against the real policy table, the real knobs and the seven class rows.
func shippedRefs(t *testing.T) Refs {
	t.Helper()
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	tbl, err := decide.ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	knobs := map[string]env.Knob{}
	for _, k := range append(env.Knobs(), resource.Knobs()...) {
		knobs[k.Key] = k
	}
	return Refs{
		Class: func(n string) bool { return sevenClasses[n] },
		Option: func(row, opt string) error {
			r, err := tbl.Row(row)
			if err != nil {
				return err
			}
			_, err = r.Option(opt)
			return err
		},
		Knob: func(k string) (float64, float64, bool) { kn, ok := knobs[k]; return kn.Min, kn.Max, ok },
	}
}

func shipped(t *testing.T) *Catalogue {
	t.Helper()
	c, err := Shipped(shippedRefs(t))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestShippedHasTwelvePacts(t *testing.T) {
	c := shipped(t)
	if len(c.Pacts) != 12 {
		t.Errorf("%d pacts, want 12", len(c.Pacts))
	}
	for _, id := range []string{"blood_ravens_tithe", "andariels_shadow", "rakanishus_bargain", "pact_of_nightmares"} {
		if _, err := c.Get(id); err != nil {
			t.Error(err)
		}
	}
}

// mutate the shipped JSON text and expect the cause in the error.
func TestLoaderErrorsNameTheCause(t *testing.T) {
	good := string(mustJSON(t))
	for _, c := range []struct{ name, from, to, want string }{
		{"unknown field", `"learn_text"`, `"learn_texts"`, "learn_texts"},
		{"unknown family", `"lever_family": "overlay"`, `"lever_family": "magic"`, `"magic"`},
		{"unknown kind", `"kind": "fixed"`, `"kind": "rare"`, `"rare"`},
		{"unknown trigger", `"unlock_trigger": "baal_normal"`, `"unlock_trigger": "never"`, `"never"`},
		{"unknown away", `"away_safety": "present_only"`, `"away_safety": "maybe"`, `"maybe"`},
		{"unknown class", `"amazon",`, `"monk",`, `"monk"`},
		{"unknown row", `"row": "repair"`, `"row": "nosuchrow"`, "nosuchrow"},
		{"unknown option", `"option": "low"`, `"option": "weekly"`, "weekly"},
		{"unknown knob", `"town.tripMinHPPotions"`, `"town.nosuchknob"`, "town.nosuchknob"},
		{"knob out of range", `"town.tpReserve": 1`, `"town.tpReserve": 99`, "outside"},
		{"duplicate id", `"id": "cains_silence"`, `"id": "griswolds_debt"`, `duplicate id "griswolds_debt"`},
		{"missing incompat target", `"id": "strangers_toll",`, `"id": "ghost_pact",`, `"ghost_pact"`},
		{"rank without heat", `"heat": 8`, `"heat": 0`, "no heat"},
	} {
		if !strings.Contains(good, c.from) {
			t.Fatalf("%s: %q not in the data", c.name, c.from)
		}
		bad := strings.Replace(good, c.from, c.to, 1)
		_, err := Parse([]byte(bad), shippedRefs(t))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}

func mustJSON(t *testing.T) []byte {
	t.Helper()
	b, err := readShipped()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func pick(s string, r int) Pick { return Pick{s, r} }

func TestHeatCapAndSlots(t *testing.T) {
	c := shipped(t)
	if c.HeatCap(0) != 12 || c.HeatCap(3) != 15 || c.HeatCap(99) != 20 {
		t.Errorf("caps %d %d %d", c.HeatCap(0), c.HeatCap(3), c.HeatCap(99))
	}
	for _, class := range []string{"paladin", "sorceress"} {
		// 3 + 2 + 1 heat fits the cap of 12 but not two slots
		set := []Pick{pick("blood_ravens_tithe", 1), pick("andariels_shadow", 1)}
		if err := c.Check(set, class, 2, 0); err != nil {
			t.Errorf("%s: %v", class, err)
		}
		if err := c.Check(append(set, pick("griswolds_debt", 1)), class, 2, 0); err == nil || !strings.Contains(err.Error(), "slots") {
			t.Errorf("%s: three picks in two slots: %v", class, err)
		}
		// rank 3 Tithe (9) + rank 3 Shadow (6) = 15 over 12; masteries lift the cap to 15
		over := []Pick{pick("blood_ravens_tithe", 3), pick("andariels_shadow", 3)}
		if err := c.Check(over, class, 2, 0); err == nil || !strings.Contains(err.Error(), "over the cap 12") {
			t.Errorf("%s: over cap: %v", class, err)
		}
		if err := c.Check(over, class, 2, 3); err != nil {
			t.Errorf("%s: cap 15: %v", class, err)
		}
	}
}

func TestIncompatibility(t *testing.T) {
	c := shipped(t)
	for _, class := range []string{"paladin", "sorceress"} {
		// declared pair, either side
		if err := c.Check([]Pick{pick("blood_ravens_tithe", 1), pick("strangers_toll", 1)}, class, 2, 0); err == nil || !strings.Contains(err.Error(), "No merc and no potions") {
			t.Errorf("%s: declared pair: %v", class, err)
		}
		if err := c.Check([]Pick{pick("strangers_toll", 1), pick("blood_ravens_tithe", 1)}, class, 2, 0); err == nil {
			t.Errorf("%s: declared pair, other order: %v", class, err)
		}
		// rank-limited: Tithe 1 with Warriv 1 is fine (no_safety below rank 3); both at 3 is not
		if err := c.Check([]Pick{pick("blood_ravens_tithe", 1), pick("rakanishus_bargain", 1)}, class, 2, 0); err != nil {
			t.Errorf("%s: low ranks: %v", class, err)
		}
		if err := c.Check([]Pick{pick("blood_ravens_tithe", 3), pick("rakanishus_bargain", 3)}, class, 2, 4); err == nil || !strings.Contains(err.Error(), "safety net") {
			t.Errorf("%s: no_safety tag rule: %v", class, err)
		}
		// the same lever twice
		if err := c.Check([]Pick{pick("rakanishus_bargain", 1), pick("warrivs_long_road", 1)}, class, 2, 0); err == nil || !strings.Contains(err.Error(), "same lever") {
			t.Errorf("%s: same lever: %v", class, err)
		}
		// a poor choice below the cap is allowed
		if err := c.Check([]Pick{pick("griswolds_debt", 5)}, class, 2, 0); err != nil {
			t.Errorf("%s: poor but legal: %v", class, err)
		}
	}
}

func TestClassScopeAndUnknowns(t *testing.T) {
	c := shipped(t)
	if err := c.Check([]Pick{pick("pact_of_rotting_bone", 1)}, "paladin", 2, 0); err != nil {
		t.Errorf("paladin: %v", err)
	}
	for _, class := range []string{"necromancer", "druid"} {
		if err := c.Check([]Pick{pick("pact_of_rotting_bone", 1)}, class, 2, 0); err == nil || !strings.Contains(err.Error(), class) {
			t.Errorf("%s: %v", class, err)
		}
	}
	if err := c.Check([]Pick{pick("ghost", 1)}, "paladin", 2, 0); err == nil || !strings.Contains(err.Error(), `"ghost"`) {
		t.Errorf("unknown pact: %v", err)
	}
	if err := c.Check([]Pick{pick("andariels_shadow", 2)}, "paladin", 2, 0); err == nil || !strings.Contains(err.Error(), "no rank 2") {
		t.Errorf("undefined rank: %v", err)
	}
	if err := c.Check([]Pick{pick("andariels_shadow", 1), pick("andariels_shadow", 3)}, "paladin", 2, 0); err == nil || !strings.Contains(fmt.Sprint(err), "twice") {
		t.Errorf("duplicate pick: %v", err)
	}
}

func readShipped() ([]byte, error) { return pactdataJSON(), nil }
