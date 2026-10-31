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
