package watch

import (
	"context"
	"testing"
)

// Card vault2: the HTTP 409 order_in_flight itself, with the first request stopped inside the keeper.

type blockCarrier struct {
	*fakeCarrier
	in, out chan struct{}
}

func (b blockCarrier) Carry(ctx context.Context, id string) VaultKeepResult {
	b.in <- struct{}{}
	<-b.out
	return b.fakeCarrier.Carry(ctx, id)
}

// MUST FAIL when a second POST for a record reaches the keeper while the first is still inside it.
func TestVault2InFlight409(t *testing.T) {
	s := testVault(t, 5, 100)
	a := vaultAPI(s, true)
	bc := blockCarrier{&fakeCarrier{fakeEquip: fakeEquip{on: true}}, make(chan struct{}), make(chan struct{})}
	a.desk().Equip = bc
	_, v := getVault(t, a, "/api/vault?limit=2")
	id, other := v.Items[0].ID, v.Items[1].ID
	body := map[string]any{"ids": []string{id}}
	first := make(chan int)
	go func() {
		code, _, _ := postVault(t, a, VaultCarryRoute, body)
		first <- code
	}()
	<-bc.in // the first request is inside the keeper
	for _, route := range []string{VaultCarryRoute, VaultStowRoute, VaultEquipRoute, VaultSellRoute} {
		code, _, pr := postVault(t, a, route, body)
		if code != 409 || pr.Code != VaultCodeInFlight {
			t.Errorf("%s for a record in flight: %d %+v", route, code, pr)
		}
	}
	if code, _, pr := postVault(t, a, VaultSellRoute, map[string]any{"scope": "all_sellable", "orderId": "tap1"}); code != 409 || pr.Code != VaultCodeInFlight {
		t.Errorf("a whole-vault sale next to a running order: %d %+v", code, pr)
	}
	if code, _, pr := postVault(t, a, VaultStowRoute, map[string]any{"ids": []string{other}}); pr.Code == VaultCodeInFlight {
		t.Errorf("another record was refused as in flight: %d %+v", code, pr)
	}
	close(bc.out)
	if code := <-first; code != 200 {
		t.Fatalf("the first request: %d", code)
	}
	go func() { <-bc.in }()
	if code, _, pr := postVault(t, a, VaultCarryRoute, body); pr.Code == VaultCodeInFlight {
		t.Fatalf("the record stayed in flight after the first request ended: %d %+v", code, pr)
	}
}

// A record a sale has reserved in the store refuses Pin, Keep and Discard over HTTP: nothing changes.
func TestVault2ReservedRecordRefusesPinKeepDiscard(t *testing.T) {
	s := testVault(t, 5, 100)
	a := vaultAPI(s, true)
	_, v := getVault(t, a, "/api/vault?limit=1")
	id := v.Items[0].ID
	if _, err := s.Reserve([]string{id}, "s-1", s.Version()); err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{VaultPinRoute, VaultKeepRoute, VaultDiscardRoute} {
		code, rep, _ := postVault(t, a, route, map[string]any{"ids": []string{id}, "confirm": true})
		if code != 409 || rep.OK || len(rep.Done) != 0 {
			t.Errorf("%s on a reserved record: %d %+v", route, code, rep)
		}
	}
	if r, _ := s.Get(id); r.Hold != "s-1" || r.Pin || !r.State.Live() {
		t.Fatalf("the reserved record changed: %+v", r)
	}
}
