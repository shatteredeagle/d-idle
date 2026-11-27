package advisor

import (
	"strings"
	"testing"
)

func spiritRunes() map[string]int { return map[string]int{"r07": 1, "r10": 1, "r09": 1, "r11": 1} }

func TestTableRunesKnown(t *testing.T) {
	for _, r := range Recipes() {
		if len(r.Runes) < 2 || len(r.Types) == 0 || r.Why == "" {
			t.Errorf("%s: incomplete row", r.Name)
		}
	}
}

func TestMakeLineNamesBaseAndPlace(t *testing.T) {
	st := Stock{Runes: spiritRunes(), Bases: []Base{
		{Name: "Broad Sword", Code: "bsd", Chain: []string{"swor", "weap"}, Quality: "normal", Sockets: 4, Where: "stash"}}}
	l, err := Lines(st, Options{Class: "sorceress", Ladder: true})
	if err != nil || len(l) == 0 {
		t.Fatalf("%v %v", l, err)
	}
	if !strings.Contains(l[0], "Spirit") || !strings.Contains(l[0], "Broad Sword") || !strings.Contains(l[0], "stash") {
		t.Errorf("line: %s", l[0])
	}
}

func TestLadderGate(t *testing.T) {
	st := Stock{Runes: spiritRunes()}
	l, _ := Lines(st, Options{Class: "sorceress"})
	for _, x := range l {
		if strings.Contains(x, "Spirit") {
			t.Errorf("ladder runeword advised with the knob off: %s", x)
		}
	}
	l, _ = Lines(st, Options{Class: "sorceress", Ladder: true})
	if len(l) == 0 || !strings.Contains(l[0], "plain 4-socket") {
		t.Errorf("want a base hint: %v", l)
	}
}

func TestBaseMustBeWhiteEmptyAndExactSockets(t *testing.T) {
	for name, b := range map[string]Base{
		"magic":   {Code: "bsd", Chain: []string{"swor"}, Quality: "magic", Sockets: 4},
		"3 sock":  {Code: "bsd", Chain: []string{"swor"}, Quality: "normal", Sockets: 3},
		"filled":  {Code: "bsd", Chain: []string{"swor"}, Quality: "normal", Sockets: 4, Filled: 1},
		"wrong t": {Code: "buc", Chain: []string{"tors"}, Quality: "normal", Sockets: 4},
	} {
		l, _ := Lines(Stock{Runes: spiritRunes(), Bases: []Base{b}}, Options{Class: "paladin", Ladder: true})
		if len(l) == 0 || !strings.Contains(l[0], "you need a plain") {
			t.Errorf("%s: %v", name, l)
		}
	}
}

// Two classes: the Necromancer sees Bone, the Sorceress does not; Treachery shows for neither.
func TestClassRelevance(t *testing.T) {
	st := Stock{Runes: map[string]int{"r12": 1, "r22": 2}} // Sol Um Um
	nec, _ := Lines(st, Options{Class: "necromancer", Max: 9})
	sor, _ := Lines(st, Options{Class: "sorceress", Max: 9})
	has := func(l []string, w string) bool {
		for _, x := range l {
			if strings.Contains(x, w) {
				return true
			}
		}
		return false
	}
	if !has(nec, "for Bone") || has(sor, "for Bone") {
		t.Errorf("nec %v sor %v", nec, sor)
	}
	if has(nec, "Treachery") {
		t.Errorf("assassin recipe shown to a necromancer")
	}
}

func TestCubePathAndCap(t *testing.T) {
	// Lore = Ort Sol. Ort in hand, Sol short: 3 Amn... use Amn x3 -> needs gem for Sol (r12 needs a chipped amethyst).
	st := Stock{Runes: map[string]int{"r09": 1, "r11": 3}, Gems: map[string]int{"gcv": 1}}
	l, _ := Lines(st, Options{Class: "sorceress", Max: 9})
	found := false
	for _, x := range l {
		if strings.Contains(x, "for Lore") && strings.Contains(x, "3 Amn and a Chipped Amethyst make Sol") {
			found = true
		}
	}
	if !found {
		t.Errorf("no cube line: %v", l)
	}
	// Without the gem it is one rune short, not a cube line.
	st.Gems = nil
	l, _ = Lines(st, Options{Class: "sorceress", Max: 9})
	for _, x := range l {
		if strings.Contains(x, "Lore") && !strings.HasPrefix(x, "One rune short") {
			t.Errorf("cube without the gem: %s", x)
		}
	}
	// Spare runes the recipe needs are never consumed: Stealth Tal Eth, only one Tal, 3 Tir spare cannot become Tal.
	if steps, ok := cubePlan(Stock{Runes: map[string]int{"r03": 3, "r05": 1}}, map[string]int{"r07": 1, "r05": 1}, false); ok {
		t.Errorf("unexpected plan %v (needs 3 Nef and so on)", steps)
	}
}

func TestChainCascadeAndVanillaCap(t *testing.T) {
	// Nine Eld... Tir(r03) x9 -> 3 Nef? r04 needs 3 r03; r05 needs 3 r04: Eth from 9 Tir.
	st := Stock{Runes: map[string]int{"r03": 9, "r07": 1}}
	steps, ok := cubePlan(st, map[string]int{"r07": 1, "r05": 1}, false)
	if !ok || len(steps) != 4 {
		t.Fatalf("steps %v ok %v", steps, ok)
	}
	// Vanilla cap 13 stops Hel (r15) production.
	if _, ok := cubePlan(Stock{Runes: map[string]int{"r14": 3}, Gems: map[string]int{"gfw": 3}}, map[string]int{"r15": 1}, false); ok {
		t.Error("rune above the vanilla cap")
	}
}

func TestNoClassIsAnError(t *testing.T) {
	if _, err := Advise(Stock{}, Options{}); err == nil {
		t.Error("want an error naming the missing class")
	}
}

func TestMaxAndOrder(t *testing.T) {
	st := Stock{Runes: map[string]int{"r07": 2, "r05": 2, "r09": 1, "r12": 1}, Bases: []Base{{Code: "buc", Chain: []string{"tors"}, Quality: "normal", Sockets: 2, Where: "pack"}}}
	l, _ := Advise(st, Options{Class: "necromancer", Max: 2})
	if len(l) != 2 || l[0].Kind != Make {
		t.Errorf("%+v", l)
	}
}

func TestStockAdd(t *testing.T) {
	var s Stock
	s.Add("r07", "", nil, "", 0, 0, "pack")
	s.Add("gcv", "", nil, "", 0, 0, "pack")
	s.Add("r40", "", nil, "", 0, 0, "pack") // not a rune
	s.Add("bsd", "Broad Sword", []string{"swor"}, "normal", 4, 0, "stash")
	s.Add("cap", "Cap", []string{"helm"}, "normal", 0, 0, "pack") // no sockets: not a base
	if s.Runes["r07"] != 1 || s.Gems["gcv"] != 1 || len(s.Bases) != 1 || len(s.Runes) != 1 {
		t.Errorf("%+v", s)
	}
}

// SME review 2026-11-27, confidence medium: an offline Necromancer (ladder knob off) is never advised Spirit, a ladder
// runeword; with the knob on it is. Bone (not ladder) stays advised either way.
func TestSpiritSkippedOfflineForNecromancer(t *testing.T) {
	st := Stock{Runes: spiritRunes()}
	off, _ := Lines(st, Options{Class: "necromancer", Max: 9})
	for _, x := range off {
		if strings.Contains(x, "Spirit") {
			t.Errorf("Spirit advised offline: %s", x)
		}
	}
	on, _ := Lines(st, Options{Class: "necromancer", Ladder: true, Max: 9})
	found := false
	for _, x := range on {
		found = found || strings.Contains(x, "Spirit")
	}
	if !found {
		t.Errorf("Spirit not advised with the ladder knob on: %v", on)
	}
	bone, _ := Lines(Stock{Runes: map[string]int{"r12": 1, "r22": 2}}, Options{Class: "necromancer", Max: 9})
	if len(bone) == 0 {
		t.Error("Bone must stay advised offline")
	}
}
