package hand

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestPanelOpenUsesTheUnifiedTable(t *testing.T) {
	var u d2.UIState
	if PanelOpen(u) {
		t.Fatal("empty UI reads as a panel")
	}
	for _, hud := range []int{6, 7, 0x11, d2.UIMiniPanel, 0x22} {
		u = d2.UIState{}
		u[hud] = 1
		if PanelOpen(u) {
			t.Errorf("HUD flag %#x counts as a panel", hud)
		}
	}
	for _, p := range []int{d2.UIInventory, d2.UIStash, d2.UIShop, d2.UIEscMenu} {
		u = d2.UIState{}
		u[p] = 1
		if !PanelOpen(u) {
			t.Errorf("panel %#x not seen", p)
		}
	}
}
