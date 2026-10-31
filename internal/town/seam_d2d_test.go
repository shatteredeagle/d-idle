package town

import "testing"

func TestFixedTabsUsePageAsSlot(t *testing.T) {
	tn := &Town{pages: []int{1, 3}}
	if p, _ := tn.tabPoint(3); p.X != tabX0+tabPitch { // two pages stocked: misc is the second tab on Game.exe (U)
		t.Fatalf("pages model: %v", p)
	}
	tn.fixedTabs = true
	if p, ok := tn.tabPoint(3); !ok || p.X != tabX0+3*tabPitch || p.Y != tabY {
		t.Fatalf("fixed: %v %v", p, ok)
	}
	if _, ok := tn.tabPoint(4); ok {
		t.Fatal("page 4 does not exist")
	}
}
