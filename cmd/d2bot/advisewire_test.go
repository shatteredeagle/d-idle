package main

import (
	"strings"
	"testing"

	"d2lod-bot/internal/advisor"
	"d2lod-bot/internal/d2"
)

func rc(name string) int { c, _ := advisor.RuneCode(name); return int(c[1]-'0')*10 + int(c[2]-'0') } // class number = fake row

// fakeRows: class 1xx = runes (100+n for rNN), 200 = Shako-like helm, 201 = Broad Sword, 300 potion.
func fakeRows(class int) (string, string, []string, bool) {
	switch {
	case class >= 100 && class < 140:
		return "r" + string(rune('0'+(class-100)/10)) + string(rune('0'+(class-100)%10)), "Rune", []string{"rune"}, true
	case class == 200:
		return "cap", "Cap", []string{"helm", "armo"}, true
	case class == 201:
		return "bsd", "Broad Sword", []string{"swor", "weap"}, true
	case class == 300:
		return "hp1", "Healing Potion", []string{"pott"}, true
	}
	return "", "", nil, false
}

func unit(id uint32, class int, where d2.Container, q int) d2.Unit {
	return d2.Unit{ID: id, Class: class, Item: &d2.Item{Where: where, Quality: q}}
}

func rune_(id uint32, name string) d2.Unit { return unit(id, 100+rc(name), d2.ContInventory, 2) }

func source(class string, on bool, items []d2.Unit, socks map[uint32]int) *adviceSource {
	snap := &d2.Snapshot{Inventory: &d2.Inventory{Items: items}}
	return &adviceSource{on: func() bool { return on }, class: func() string { return class }, snap: func() *d2.Snapshot { return snap },
		row: fakeRows, sockets: func(id uint32) int { return socks[id] }}
}

func TestAdviceNecromancerLore(t *testing.T) {
	items := []d2.Unit{rune_(1, "Ort"), rune_(2, "Sol"), unit(3, 200, d2.ContStash, 2), unit(4, 300, d2.ContBelt, 2)}
	a := source("Necromancer", true, items, map[uint32]int{3: 2})
	l := a.Lines()
	if len(l) == 0 || !strings.Contains(l[0], "Lore") || !strings.Contains(l[0], "Cap") || !strings.Contains(l[0], "stash") {
		t.Fatalf("want a Lore make line naming the Cap in the stash: %v", l)
	}
}

func TestAdvicePaladinSteel(t *testing.T) {
	items := []d2.Unit{rune_(1, "Tir"), rune_(2, "El"), unit(3, 201, d2.ContInventory, 2)}
	a := source("paladin", true, items, map[uint32]int{3: 2})
	l := a.Lines()
	if len(l) == 0 || !strings.Contains(l[0], "Steel") || !strings.Contains(l[0], "Broad Sword") {
		t.Fatalf("want Steel on the sword: %v", l)
	}
	// The same stock advises nothing Lore for the Paladin (class relevance).
	for _, x := range l {
		if strings.Contains(x, "Lore") {
			t.Errorf("Lore advised to a Paladin: %s", x)
		}
	}
}

// Must fail if the knob gate is dropped: knob 0 shows nothing, whatever the hero owns.
func TestAdviceKnobOffShowsNothing(t *testing.T) {
	items := []d2.Unit{rune_(1, "Ort"), rune_(2, "Sol"), unit(3, 200, d2.ContStash, 2)}
	a := source("necromancer", false, items, map[uint32]int{3: 2})
	if l := a.Lines(); len(l) != 0 {
		t.Fatalf("knob 0: %v", l)
	}
	if a.runs != 0 {
		t.Errorf("advisor ran with the knob off")
	}
}

func TestAdviceCachedByInventory(t *testing.T) {
	items := []d2.Unit{rune_(1, "Ort"), rune_(2, "Sol"), unit(3, 200, d2.ContStash, 2)}
	a := source("necromancer", true, items, map[uint32]int{3: 2})
	a.Lines()
	a.Lines()
	a.Lines()
	if a.runs != 1 {
		t.Fatalf("recomputed on an unchanged stock: %d runs", a.runs)
	}
	snap := &d2.Snapshot{Inventory: &d2.Inventory{Items: items[:2]}} // the cap left
	a.snap = func() *d2.Snapshot { return snap }
	l := a.Lines()
	if a.runs != 2 {
		t.Errorf("no recompute after the stock changed: %d", a.runs)
	}
	for _, x := range l {
		if strings.Contains(x, "Cap") {
			t.Errorf("stale line: %s", x)
		}
	}
}

func TestAdviceUnknownClassIsNoAdvice(t *testing.T) {
	a := source("", true, []d2.Unit{rune_(1, "Ort")}, nil)
	if l := a.Lines(); len(l) != 0 {
		t.Fatalf("%v", l)
	}
}

func TestAdviceNeverCountsEquippedOrBelt(t *testing.T) {
	// Runes worn or in the belt column are not stock; two runes in the belt must not make Lore.
	items := []d2.Unit{unit(1, 100+rc("Ort"), d2.ContBelt, 2), unit(2, 100+rc("Sol"), d2.ContEquipment, 2)}
	if st := heroStock(items, fakeRows, nil); len(st.Runes) != 0 {
		t.Fatalf("stock %v", st.Runes)
	}
}
