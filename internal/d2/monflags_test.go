package d2

import "testing"

func TestMonsterKindAndEnchants(t *testing.T) {
	var nilm *MonsterInfo
	if k := nilm.Kind(); k != MonKindUnknown {
		t.Fatal("nil must be unknown")
	}
	if _, ok := nilm.EnchantIDs(); ok {
		t.Fatal("nil enchants must be unknown")
	}
	if k := (&MonsterInfo{}).Kind(); k != MonKindUnknown {
		t.Fatalf("zero flags: %v", k)
	}
	u := &MonsterInfo{Flags: monFlagBoss, Enchants: [9]uint8{6, 26, 0}}
	if u.Kind() != MonKindUnique {
		t.Fatal("unique")
	}
	if ids, ok := u.EnchantIDs(); !ok || len(ids) != 2 || ids[0] != 6 || ids[1] != 26 {
		t.Fatalf("ids %v %v", ids, ok)
	}
	if (&MonsterInfo{Flags: monFlagChamp}).Kind() != MonKindChampion || (&MonsterInfo{Flags: monFlagMinion}).Kind() != MonKindMinion {
		t.Fatal("kinds")
	}
	// enchants with unknown flags are not trusted
	if _, ok := (&MonsterInfo{Enchants: [9]uint8{6}}).EnchantIDs(); ok {
		t.Fatal("unknown flags must hide enchants")
	}
	// garbage id
	if _, ok := (&MonsterInfo{Flags: monFlagBoss, Enchants: [9]uint8{200}}).EnchantIDs(); ok {
		t.Fatal("garbage id accepted")
	}
	if ids, ok := (&MonsterInfo{Flags: monFlagNormal}).EnchantIDs(); !ok || ids != nil {
		t.Fatal("normal has none")
	}
}

// Real flag bytes from the original-game census (docs/orig-spawns.md): the champion byte 0x0c carries the boss bit too.
func TestMonKindRealFlagBytes(t *testing.T) {
	for _, c := range []struct {
		name   string
		flags  uint8
		kind   MonKind
		unique bool
		champ  bool
	}{
		{"ordinary 0x00", 0x00, MonKindUnknown, false, false},
		{"bare normal bit", 0x02, MonKindNormal, false, false},
		{"champion 0x0c", 0x0c, MonKindChampion, false, true},
		{"champion bit alone", 0x04, MonKindChampion, false, true},
		{"rare 0x08", 0x08, MonKindUnique, true, false},
		{"superunique 0x0a", 0x0a, MonKindUnique, true, false},
		{"pack follower 0x10", 0x10, MonKindMinion, false, false},
		{"odd read 0x4c", 0x4c, MonKindChampion, false, true},
	} {
		m := &MonsterInfo{Flags: c.flags}
		if m.Kind() != c.kind || m.IsUnique() != c.unique || m.IsChampion() != c.champ {
			t.Errorf("%s: kind %v unique %v champion %v", c.name, m.Kind(), m.IsUnique(), m.IsChampion())
		}
	}
}
