package watch

import "testing"

// TASK-167: a Pact sets a row like the player does (knobs, log line) but leaves no "change it back" record.
func TestSetByPactIsQuiet(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newPolRig(t, class, true)
		if got := r.sheet.CurrentOption("potionmix"); got != "default" || r.sheet.DefaultOption("potionmix") != "default" {
			t.Fatalf("%s: current %q", class, got)
		}
		if err := r.sheet.SetByPact("potionmix", "allmana"); err != nil {
			t.Fatal(err)
		}
		if r.sheet.CurrentOption("potionmix") != "allmana" || len(r.calls) != 1 {
			t.Errorf("%s: option %q, knob calls %d", class, r.sheet.CurrentOption("potionmix"), len(r.calls))
		}
		if len(r.sheet.Changes()) != 0 {
			t.Errorf("%s: a Pact change must not be a recorded change: %v", class, r.sheet.Changes())
		}
		if err := r.sheet.SetByPact("nosuchrow", "x"); err == nil {
			t.Errorf("%s: unknown row accepted", class)
		}
		if err := r.sheet.SetByPact("potionmix", "nosuchoption"); err == nil {
			t.Errorf("%s: unknown option accepted", class)
		}
	}
}
