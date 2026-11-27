package supply

import (
	"slices"
	"testing"
)

func TestPriorityDataIsTable42(t *testing.T) {
	want := []Step{{Healing, "min"}, {Portal, "one"}, {Mana, "min"}, {GoldCarry, "min"}, {Portal, "min"}, {Healing, "target"}, {Mana, "target"}, {Portal, "target"}}
	if !slices.Equal(DefaultPriority, want) {
		t.Fatalf("priority.json modelled order %v, want %v", DefaultPriority, want)
	}
	// D3 and the carry position: potions before the first scroll, gold carry between mana and the second portal charge
	pos := func(l Line, to string) int { return slices.Index(DefaultPriority, Step{l, to}) }
	if !(pos(Healing, "min") < pos(Portal, "one") && pos(GoldCarry, "min") > pos(Mana, "min") && pos(GoldCarry, "min") < pos(Portal, "min")) {
		t.Fatalf("order breaks D3 or the gold_carry position: %v", DefaultPriority)
	}
	for _, p := range PriorityProfiles() {
		if len(PriorityOf(p)) == 0 {
			t.Errorf("profile %s has an empty order", p)
		}
	}
}

func TestPriorityLoaderRejectsBadRows(t *testing.T) {
	for name, rows := range map[string][]stepRow{
		"unknown line": {{1, "wand", "min"}},
		"bad step":     {{1, Healing, "max"}},
		"out of order": {{2, Healing, "min"}, {1, Mana, "min"}},
	} {
		if _, err := toSteps(rows, "t"); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// With short gold the lines are bought strictly in table order on two builds, the dropped line is named and the verdict
// hands over (T6).
func TestSolveFollowsDataOrderTwoBuilds(t *testing.T) {
	for name, in := range builds {
		n, err := NeedFor("quest", "blood_raven", in) // healing 4, portal 2, mana 2 for a caster, gold_carry 250
		if err != nil {
			t.Fatal(err)
		}
		pr := Prices{Healing: 30, Mana: 30, Portal: 100}
		// 150 gold: four potions (120) then one scroll is unaffordable; a caster's mana comes after the first scroll slot
		pl := Solve(n, Count{Purse: 150, StashKnown: true}, pr, Policy{}, 0)
		if h := pl.Lines[lineIdx(pl, Healing)]; h.Buy != 4 {
			t.Fatalf("%s: healing bought %d, want 4 first (%s)", name, h.Buy, pl.Trace())
		}
		if p := pl.Lines[lineIdx(pl, Portal)]; p.Buy != 0 || p.Status != "short" {
			t.Errorf("%s: portal %+v, want unbought and short", name, p)
		}
		if pl.Verdict != ShortGold || pl.Open != "portal" && pl.Open != "mana" {
			t.Errorf("%s: verdict %s open %q", name, pl.Verdict, pl.Open)
		}
		// an explicit policy order beats the file, and the caster/melee difference is only the need, never the order code
		rev := []Step{{Portal, "min"}, {Healing, "min"}}
		pl2 := Solve(n, Count{Purse: 200, StashKnown: true}, pr, Policy{Priority: rev}, 0)
		if p := pl2.Lines[lineIdx(pl2, Portal)]; p.Buy != 2 {
			t.Errorf("%s: policy order ignored: portal bought %d, want 2 (%s)", name, p.Buy, pl2.Trace())
		}
	}
}

func TestHandoverOrderAndNeverEmpty(t *testing.T) {
	for name, in := range builds {
		n, _ := NeedFor("quest", "countess", in)
		pl := Solve(n, Count{Purse: 20, StashKnown: true}, Prices{Healing: 30, Mana: 30, Portal: 100}, Policy{}, 0)
		if pl.Verdict != ShortGold {
			t.Fatalf("%s: verdict %s", name, pl.Verdict)
		}
		h := pl.Handover(Gold{Sellable: true, StashGold: 5, VaultPrompt: true})
		want := []Source{SrcSell, SrcStash, SrcVaultPrompt, SrcRecovery}
		if !slices.Equal(h.Sources, want) {
			t.Errorf("%s: sources %v, want %v", name, h.Sources, want)
		}
		if got := pl.Handover(Gold{}); !slices.Equal(got.Sources, []Source{SrcRecovery}) || got.Reason == "" {
			t.Errorf("%s: bare hand-over %+v, want recovery with a reason", name, got)
		}
	}
	n, _ := NeedFor("x", "field", Inputs{})
	if h := Solve(n, Count{Purse: 9999, StashKnown: true}, Prices{Healing: 30, Portal: 100}, Policy{}, 0).Handover(Gold{Sellable: true}); len(h.Sources) != 0 {
		t.Errorf("ready plan handed over: %+v", h)
	}
	blocked := Solve(n, Count{Purse: 9999, StashKnown: true}, Prices{Portal: 100}, Policy{}, 0)
	if h := blocked.Handover(Gold{}); len(h.Sources) != 0 {
		t.Errorf("a blocked plan is not a gold shortage: %+v", h)
	}
}
