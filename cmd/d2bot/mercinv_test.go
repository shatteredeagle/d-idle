package main

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/status"
)

func TestMercInv(t *testing.T) {
	mk := func(u d2.Unit, merc bool) status.Item { return status.Item{ID: u.ID, Name: "x"} }
	if got := mercInv(&d2.Snapshot{}, mk); got != nil {
		t.Errorf("nothing read, no unit: %+v", got)
	}
	if got := mercInv(&d2.Snapshot{Merc: &d2.MercSheet{}}, mk); got == nil || got.Hired {
		t.Errorf("read, nobody employed: %+v", got)
	}
	unit := d2.Unit{Type: d2.TypeMonster, Class: 338, Mode: d2.MonModeStand, Monster: &d2.MonsterInfo{LifePct: 50}}
	got := mercInv(&d2.Snapshot{Monsters: []d2.Unit{unit}}, mk)
	if got == nil || !got.Hired || got.GearKnown || got.Class != 338 || got.LifePct != 50 {
		t.Errorf("unit without a read: %+v", got)
	}
	gear := []d2.Unit{{ID: 7, Item: &d2.Item{Body: d2.BodyHead}}, {ID: 8, Item: &d2.Item{Body: d2.BodyRHand}}, {ID: 9, Item: &d2.Item{Body: d2.BodyBelt}}}
	got = mercInv(&d2.Snapshot{Monsters: []d2.Unit{unit}, Merc: &d2.MercSheet{Hired: true, Alive: true, Level: 17, Str: 60, Dex: 50, GearKnown: true, Gear: gear}}, mk)
	if got == nil || !got.GearKnown || got.Level != 17 || len(got.Equipment) != 2 || got.Equipment["mhead"] == nil || got.Equipment["mrhand"] == nil {
		t.Errorf("read: %+v", got)
	}
}
