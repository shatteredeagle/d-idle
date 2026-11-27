package watch

import (
	"context"
	"testing"
)

type mercEquip struct {
	directEquip
	on bool
}

func (m *mercEquip) MercDirect(context.Context) bool { return m.on }
func (*mercEquip) EquipMerc(_ context.Context, id string) VaultKeepResult {
	if id == "illegal" {
		return VaultKeepResult{ID: id, Code: "merc", Why: "a rogue cannot use that"}
	}
	return VaultKeepResult{ID: id, OK: true, Direct: true, Slot: "mhead", Removed: []string{"Old Cap"}, Where: "vault"}
}

func TestVaultEquipMercRoute(t *testing.T) {
	s := testVault(t, 10, 100)
	a := vaultAPI(s, true)
	a.desk().Equip = &mercEquip{directEquip{fakeEquip{on: true}}, true}
	_, v := getVault(t, a, "/api/vault?limit=3")
	if v.Equip == nil || !v.Equip.Merc {
		t.Fatalf("flag %+v", v.Equip)
	}
	code, rep, _ := postVault(t, a, VaultEquipRoute, map[string]any{"ids": []string{v.Items[0].ID}, "target": "merc"})
	if code != 200 || len(rep.Equipped) != 1 || rep.Equipped[0].Slot != "mhead" {
		t.Fatalf("%d %+v", code, rep)
	}
	code, rep, _ = postVault(t, a, VaultEquipRoute, map[string]any{"ids": []string{"illegal"}, "target": "merc"})
	if code != 409 || len(rep.Failed) != 1 || rep.Failed[0].Code != "merc" {
		t.Fatalf("%d %+v", code, rep)
	}
	// Not on offer: the hero's equipper has no merc half, or the build lacks the verb.
	a.desk().Equip = &directEquip{fakeEquip{on: true}}
	if code, _, pr := postVault(t, a, VaultEquipRoute, map[string]any{"ids": []string{v.Items[0].ID}, "target": "merc"}); code != 409 || pr.Code != "off" {
		t.Fatalf("%d %+v", code, pr)
	}
	a.desk().Equip = &mercEquip{directEquip{fakeEquip{on: true}}, false}
	if code, _, pr := postVault(t, a, VaultEquipRoute, map[string]any{"ids": []string{v.Items[0].ID}, "target": "merc"}); code != 409 || pr.Code != "off" {
		t.Fatalf("%d %+v", code, pr)
	}
}
