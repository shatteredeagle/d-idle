package sentinel

import (
	"reflect"
	"testing"

	"d2lod-bot/internal/d2"
)

func TestOpenPanelsIgnoresHUDAndEscMenu(t *testing.T) {
	var ui d2.UIState
	ui[6], ui[0x11], ui[d2.UIMiniPanel], ui[d2.UIEscMenu] = 1, 1, 1, 1
	if got := openPanels(ui); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	ui[d2.UIStash], ui[d2.UIInventory] = 1, 1
	if got := openPanels(ui); !reflect.DeepEqual(got, []int{d2.UIInventory, d2.UIStash}) {
		t.Fatalf("got %v", got)
	}
}
