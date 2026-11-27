package supply

import (
	"strings"
	"testing"
)

// two builds: a mana-need-2 caster-like build and a mana-need-0 melee-like build. Names are labels only.
var builds = map[string]Inputs{
	"sorc": {ManaNeed: 2, Capacity: map[Line]int{Healing: 8, Mana: 4}},
	"pal":  {ManaNeed: 0, Capacity: map[Line]int{Healing: 8}},
}

func TestNeedForUnknownProfileNamesIt(t *testing.T) {
	_, err := NeedFor("x", "nope", Inputs{})
	if err == nil || !strings.Contains(err.Error(), `"nope"`) {
		t.Fatalf("want an error naming the profile, got %v", err)
	}
}

func TestNeedForManaFollowsBuildNotClass(t *testing.T) {
	for name, want := range map[string]int{"sorc": 2, "pal": 0} {
		n, err := NeedFor("quest", "monastery_road", builds[name])
		if err != nil {
			t.Fatal(err)
		}
		if got := n.Lines[Mana].Min; got != want {
			t.Errorf("%s: mana min %d, want %d", name, got, want)
		}
		if n.Lines[Healing].Min != 4 {
			t.Errorf("%s: healing min %d, want today's 4", name, n.Lines[Healing].Min)
		}
	}
}

// must-fail on today's behaviour: the guard counted mana in the belt only and wanted 4 healing in a 4-slot belt that holds 3.
func TestSolveGuardBlockedRestock(t *testing.T) {
	prices := Prices{Healing: 30, Mana: 30, Portal: 100}
	for name, in := range builds {
		in.Capacity = map[Line]int{Healing: 3, Mana: 1} // 4-slot belt, three healing columns
		n, _ := NeedFor("quest:malus", "monastery_road", in)
		c := CountOf(append(pots(3, 0, HP), pots(1, 0, MP)...), Floors{})
		c.PortalCharges, c.Purse, c.StashKnown = 2, 1220, true
		pl := Solve(n, c, prices, Policy{}, 0)
		if pl.Verdict == ShortGold || pl.Verdict == Blocked {
			t.Fatalf("%s: verdict %s, gold 1220 closes the line: %s", name, pl.Verdict, pl.Trace())
		}
		if got := pl.Lines[lineIdx(pl, Healing)].Buy; got != 0 {
			// min clamped to the capacity 3: nothing to buy to reach the min, the 4th is not asked for
			t.Fatalf("%s: buys %d healing, want 0 (min clamped to capacity 3)", name, got)
		}
		if pl.Lines[lineIdx(pl, Healing)].Status != "clamped" {
			t.Errorf("%s: status %q, want clamped", name, pl.Lines[lineIdx(pl, Healing)].Status)
		}
	}
}

func lineIdx(p Plan, l Line) int {
	for i, ln := range p.Lines {
		if ln.Line == l {
			return i
		}
	}
	return -1
}

func TestSolveShortGoldPriorities(t *testing.T) {
	for name, in := range builds {
		n, _ := NeedFor("quest", "countess", in)
		c := Count{Purse: 130, StashKnown: true}
		pl := Solve(n, c, Prices{Healing: 30, Mana: 30, Portal: 100}, Policy{}, 0)
		h := pl.Lines[lineIdx(pl, Healing)]
		if h.Buy != 4 {
			t.Fatalf("%s: potions first: bought %d healing, want 4 (%s)", name, h.Buy, pl.Trace())
		}
		if p := pl.Lines[lineIdx(pl, Portal)]; p.Buy != 0 || p.Status != "dropped" && p.Status != "short" {
			t.Fatalf("%s: portal %+v, want none bought and named", name, p)
		}
		if pl.Verdict != ShortGold || pl.Open == "" {
			t.Fatalf("%s: verdict %s open %q", name, pl.Verdict, pl.Open)
		}
	}
}

func TestSolveBlockedWhenNotBuyable(t *testing.T) {
	n, _ := NeedFor("x", "field", Inputs{})
	pl := Solve(n, Count{Purse: 9999, StashKnown: true}, Prices{Portal: 100}, Policy{}, 0)
	if pl.Verdict != Blocked || pl.Open != "healing" {
		t.Fatalf("verdict %s open %q, want blocked on healing", pl.Verdict, pl.Open)
	}
}

func TestSolveDeterministic(t *testing.T) {
	n, _ := NeedFor("x", "blood_raven", builds["sorc"])
	c := Count{Purse: 500, Stash: 2000, StashKnown: true}
	a := Solve(n, c, Prices{Healing: 30, Mana: 30, Portal: 100}, Policy{}, 40).Trace()
	for i := 0; i < 20; i++ {
		if b := Solve(n, c, Prices{Healing: 30, Mana: 30, Portal: 100}, Policy{}, 40).Trace(); a != b {
			t.Fatalf("plan differs:\n%s\n%s", a, b)
		}
	}
}

// The Andariel profile is the boss kit (V live 2026-11-24: 3 healing potions at 120/279 life, no belt healing at the chicken): 8
// healing, clamped to what the build carries, and the boss purse as gold_carry. Two builds: the 4-slot belt clamps the line, a
// bigger belt keeps it. Must fail with the old profile (4 healing, no gold line).
func TestAndarielProfileIsTheBossKit(t *testing.T) {
	for name, tc := range map[string]struct{ capHP, wantMax int }{"pal-sash": {6, 6}, "sorc-belt": {10, 10}} {
		in := builds["sorc"]
		in.Capacity = map[Line]int{Healing: tc.capHP, Mana: 4}
		n, err := NeedFor("quest", "andariel", in)
		if err != nil {
			t.Fatal(err)
		}
		if q := n.Lines[Healing]; q.Min != 8 && q.Min != tc.capHP || q.Max != tc.wantMax {
			t.Errorf("%s: healing %+v, want min 8 (or the capacity %d when that is lower) and max %d", name, q, tc.capHP, tc.wantMax)
		}
		if got := n.Lines[GoldCarry].Min; got != 3000 {
			t.Errorf("%s: boss purse %d, want 3000", name, got)
		}
	}
}

func TestViewTwoBuilds(t *testing.T) {
	prices := Prices{Healing: 30, Mana: 30, Portal: 100}
	for name, in := range builds {
		n, err := NeedFor("quest", "monastery_road", in)
		if err != nil {
			t.Fatal(err)
		}
		rich := Count{Purse: 5000, StashKnown: true, Stash: 700}
		v := ViewOf(Solve(n, rich, prices, Policy{}, 0), rich, []string{"akara", "stash"}, 1)
		if v.Schema != ViewSchema || v.Budget.Purse != 5000 || v.Budget.Stash != 700 || v.Done != 1 || len(v.Lines) != len(n.Lines) {
			t.Errorf("%s: view %+v", name, v)
		}
		if !strings.HasPrefix(v.Waiting, "Waiting: ") || !strings.Contains(v.Waiting, "Buying") {
			t.Errorf("%s: waiting %q", name, v.Waiting)
		}
		poor := Count{Purse: 10, StashKnown: true}
		pv := ViewOf(Solve(n, poor, prices, Policy{}, 0), poor, nil, 9)
		if pv.Chip != "Short of gold" || !strings.HasPrefix(pv.Waiting, "Short of gold: needs ") || pv.Done != 0 {
			t.Errorf("%s: poor view %q %q", name, pv.Chip, pv.Waiting)
		}
		if !strings.Contains(pv.Minute, "verdict=short-gold") || !strings.Contains(pv.Minute, "dropped=") {
			t.Errorf("%s: minute %q", name, pv.Minute)
		}
	}
}
