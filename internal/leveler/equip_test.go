package leveler

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestSwapLanded(t *testing.T) {
	inv := func(where d2.Container, body int) *d2.Inventory {
		return &d2.Inventory{Items: []d2.Unit{{ID: 7, Item: &d2.Item{Where: where, Body: body}}}}
	}
	cases := []struct {
		name string
		inv  *d2.Inventory
		slot int
		want bool
	}{
		{"still in the pack", inv(d2.ContInventory, 0), d2.BodyRRing, false},
		{"worn at the slot", inv(d2.ContEquipment, d2.BodyRRing), d2.BodyRRing, true},
		{"ring in the other ring slot", inv(d2.ContEquipment, d2.BodyLRing), d2.BodyRRing, true},
		{"worn elsewhere", inv(d2.ContEquipment, d2.BodyBelt), d2.BodyRRing, false},
		{"gone", &d2.Inventory{}, d2.BodyRRing, false},
	}
	for _, c := range cases {
		if got := swapLanded(c.inv, 7, c.slot); got != c.want {
			t.Errorf("%s: %v", c.name, got)
		}
	}
}
