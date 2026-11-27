package watch

import (
	"context"
	"testing"

	"d2lod-bot/internal/vault"
)

type fakeCarrier struct {
	fakeEquip
	carried map[string]VaultCarried
	cells   map[string]int
}

func (f *fakeCarrier) CarryReady() (bool, string) { return f.on, "off for the test" }
func (f *fakeCarrier) Carry(_ context.Context, id string) VaultKeepResult {
	if id == "ring" {
		return VaultKeepResult{ID: id, Code: "kind", Why: "only charms are carried from the vault"}
	}
	if f.carried == nil {
		f.carried = map[string]VaultCarried{}
	}
	f.carried[id] = VaultCarried{ID: id, Name: "charm", Code: "cm3", Cells: f.cells[id]}
	return VaultKeepResult{ID: id, OK: true, Where: "backpack"}
}
func (f *fakeCarrier) Stow(_ context.Context, id string) VaultKeepResult {
	if _, ok := f.carried[id]; !ok {
		return VaultKeepResult{ID: id, Code: "missing", Why: "that charm is not carried"}
	}
	delete(f.carried, id)
	return VaultKeepResult{ID: id, OK: true, Where: "vault"}
}
func (f *fakeCarrier) Carried() []VaultCarried {
	var out []VaultCarried
	for _, c := range f.carried {
		out = append(out, c)
	}
	return out
}

func TestVaultCarryRoutes(t *testing.T) {
	s := testVault(t, 5, 100)
	a := vaultAPI(s, true)
	fc := &fakeCarrier{fakeEquip: fakeEquip{on: true}, cells: map[string]int{"c1": 3}}
	a.desk().Equip = fc
	_, v := getVault(t, a, "/api/vault?limit=2")
	if v.Carry == nil || !v.Carry.Enabled || len(v.Carry.Carried) != 0 {
		t.Fatalf("view %+v", v.Carry)
	}
	code, rep, _ := postVault(t, a, VaultCarryRoute, map[string]any{"ids": []string{"c1"}})
	if code != 200 || !rep.OK || rep.Where != "backpack" {
		t.Fatalf("carry %d %+v", code, rep)
	}
	_, v = getVault(t, a, "/api/vault?limit=2")
	if len(v.Carry.Carried) != 1 || v.Carry.Cells != 3 {
		t.Fatalf("after carry %+v", v.Carry)
	}
	if code, rep, _ = postVault(t, a, VaultCarryRoute, map[string]any{"ids": []string{"ring"}}); code != 409 || len(rep.Failed) != 1 || rep.Failed[0].Code != "kind" {
		t.Fatalf("refusal %d %+v", code, rep)
	}
	if code, _, _ = postVault(t, a, VaultStowRoute, map[string]any{"ids": []string{"c1", "c2"}}); code != 400 {
		t.Fatalf("two ids: %d", code)
	}
	if code, rep, _ = postVault(t, a, VaultStowRoute, map[string]any{"ids": []string{"c1"}}); code != 200 || rep.Where != "vault" {
		t.Fatalf("stow %d %+v", code, rep)
	}
	if code, _, _ = postVault(t, a, VaultStowRoute, map[string]any{"ids": []string{"c1"}}); code != 409 {
		t.Fatalf("a second stow: %d", code)
	}
}

func TestVaultCarryAbsentWithoutACarrier(t *testing.T) {
	a, _ := equipDesk(t, true)
	_, v := getVault(t, a, "/api/vault?limit=1")
	if v.Carry != nil {
		t.Fatalf("a bot without Carry must not offer it: %+v", v.Carry)
	}
	if code, _, _ := postVault(t, a, VaultCarryRoute, map[string]any{"ids": []string{"x"}}); code != 409 {
		t.Fatalf("carry without a carrier: %d", code)
	}
}

func TestVaultRowFlagsCarriableCharms(t *testing.T) {
	for _, tc := range []struct {
		code string
		q    int
		want bool
	}{{"cm1", 4, true}, {"cm3", 7, true}, {"cm2", 6, false}, {"rin", 4, false}} {
		r := vault.Rec{Item: vault.Item{Code: tc.code, Quality: tc.q}}
		if got := vaultRow(&r, nil).Charm; got != tc.want {
			t.Errorf("%s q%d: %v", tc.code, tc.q, got)
		}
	}
}
