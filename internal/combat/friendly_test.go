package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

func TestIsFriendly(t *testing.T) {
	const me = 100
	cat := MapCatalog{Monsters: map[int]*data.Monster{
		1:   {Key: "fallen1", AI: "Fallen", Killable: true},
		152: {Key: "rogue1", AI: "Idle", NPC: true},
		900: {Key: "custom", AI: "NecroPet", Killable: true},  // AI table without a class entry
		901: {Key: "custom2", AI: "Hireable", Killable: true}, // likewise
	}}
	withState := func(class int, id uint32, states ...uint8) d2.Unit {
		return d2.Unit{Class: class, ID: id, Monster: &d2.MonsterInfo{States: states}}
	}
	cases := []struct {
		name string
		u    d2.Unit
		all  []d2.Unit // extra units in the snapshot
		want bool
	}{
		{"fallen", d2.Unit{Class: 1, ID: 1}, nil, false},
		{"unknown class", d2.Unit{Class: 5000, ID: 1}, nil, false},
		{"owner is player (valkyrie reads type 0)", d2.Unit{Class: 5000, ID: 2, OwnerID: me}, nil, true},
		{"owner id equals player id but owner is a monster (ids are per type)", d2.Unit{Class: 5000, ID: 2, OwnerID: me, OwnerType: 1}, nil, false},
		{"owner is another unit", d2.Unit{Class: 5000, ID: 2, OwnerID: 7}, nil, false},
		{"summon of a minion", d2.Unit{Class: 5000, ID: 3, OwnerID: 2, OwnerType: 1}, []d2.Unit{{Class: 5000, ID: 2, OwnerID: me}}, true},
		{"town npc via monstats", d2.Unit{Class: 152, ID: 4}, nil, true},
		{"hired rogue 271", d2.Unit{Class: 271, ID: 5}, nil, true},
		{"act2 hire 338", d2.Unit{Class: 338, ID: 5}, nil, true},
		{"act2 guard 358", d2.Unit{Class: 358, ID: 5}, nil, true},
		{"act3 hire 359", d2.Unit{Class: 359, ID: 5}, nil, true},
		{"act5 hire 560", d2.Unit{Class: 560, ID: 5}, nil, true},
		{"act5 hire 561", d2.Unit{Class: 561, ID: 5}, nil, true},
		{"clay golem", d2.Unit{Class: 289, ID: 6}, nil, true},
		{"valkyrie", d2.Unit{Class: 357, ID: 6}, nil, true},
		{"skeleton", d2.Unit{Class: 363, ID: 6}, nil, true},
		{"skeleton mage", d2.Unit{Class: 364, ID: 6}, nil, true},
		{"spirit wolf", d2.Unit{Class: 420, ID: 6}, nil, true},
		{"grizzly", d2.Unit{Class: 428, ID: 6}, nil, true},
		{"raven", d2.Unit{Class: 419, ID: 6}, nil, true},
		{"vine", d2.Unit{Class: 425, ID: 6}, nil, true},
		{"oak sage", d2.Unit{Class: 424, ID: 6}, nil, true},
		{"hydra", d2.Unit{Class: 351, ID: 6}, nil, true},
		{"decoy", d2.Unit{Class: 356, ID: 6}, nil, true},
		{"shadow warrior", d2.Unit{Class: 417, ID: 6}, nil, true},
		{"wake of fire sentry", d2.Unit{Class: 410, ID: 6}, nil, true},
		{"blade creeper", d2.Unit{Class: 413, ID: 6}, nil, true},
		{"AI NecroPet, class unlisted", d2.Unit{Class: 900, ID: 6}, nil, true},
		{"AI Hireable, class unlisted", d2.Unit{Class: 901, ID: 6}, nil, true},
		{"revived monster, state 105", withState(1, 8, 9, 105), nil, true},
		{"hostile with a curse state only", withState(1, 8, 9), nil, false},
		{"turret is hostile", d2.Unit{Class: 348, ID: 9}, nil, false},
	}
	for _, c := range cases {
		snap := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: me}}
		snap.Monsters = append(append(snap.Monsters, c.all...), c.u)
		if got := IsFriendly(snap, c.u, cat); got != c.want {
			t.Errorf("%s: IsFriendly = %v, want %v", c.name, got, c.want)
		}
		if IsFriendly(snap, c.u, nil) != IsFriendly(snap, c.u, nil) {
			t.Fatal("unstable")
		}
	}
	// the player's own id and a nil snapshot
	// V live: Combi is player id 1 and the town monster class 152 is monster id 1: same id, different unit type
	if IsFriendly(&d2.Snapshot{Player: &d2.Player{ID: me}}, d2.Unit{Class: 1, ID: me}, cat) {
		t.Error("a hostile monster whose id equals the player id is not the player")
	}
	if IsFriendly(nil, d2.Unit{Class: 1, ID: 1}, cat) {
		t.Error("nil snapshot: a fallen is not friendly")
	}
	if !IsFriendly(nil, d2.Unit{Class: 271, ID: 1}, nil) {
		t.Error("nil snapshot and nil catalog: class table still applies")
	}
}

// Hostiles and hostileMissiles both go through the same set: a pack of friends and one real monster.
func TestHostilesDropsAllFriends(t *testing.T) {
	cat := testCatalog()
	snap := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: 100, X: 50, Y: 50, Area: 2}}
	snap.Monsters = []d2.Unit{
		{Class: clsFallen, ID: 1, Mode: 1, X: 52, Y: 50},
		{Class: 289, ID: 2, Mode: 1, X: 51, Y: 50},
		{Class: 271, ID: 3, Mode: 1, X: 51, Y: 51},
		{Class: 363, ID: 4, Mode: 1, X: 50, Y: 51},
		{Class: clsFallen, ID: 5, Mode: 1, X: 53, Y: 50, Monster: &d2.MonsterInfo{States: []uint8{105}}},
	}
	snap.Missiles = []d2.Unit{{ID: 50, OwnerID: 2, X: 10, Y: 10}, {ID: 51, OwnerID: 1, X: 20, Y: 20}, {ID: 52, OwnerID: 100, X: 30, Y: 30}}
	hs := Hostiles(snap, HostileCtx{Cat: cat})
	if len(hs) != 1 || hs[0].Unit.ID != 1 {
		t.Fatalf("hostiles = %v, want only unit 1", ids(hs))
	}
	if ms := hostileMissiles(snap); len(ms) != 1 || ms[0].X != 20 {
		t.Errorf("hostile missiles = %v, want only the one from unit 1", ms)
	}
}
