package watch

import "testing"

// MUST FAIL when two item-moving POSTs for one record can run at once (card fix1): the second is refused, another id is not.
func TestVaultInFlight(t *testing.T) {
	a := NewVaultAPI()
	leave, ok := a.enter([]string{"v-1", "v-2"})
	if !ok {
		t.Fatal("first request refused")
	}
	if _, ok := a.enter([]string{"v-2"}); ok {
		t.Fatal("a second request for the same id was let in")
	}
	if _, ok := a.enter([]string{"*"}); ok {
		t.Fatal("a whole-vault sale was let in next to a running order")
	}
	l3, ok := a.enter([]string{"v-3"})
	if !ok {
		t.Fatal("another id was refused")
	}
	leave()
	l3()
	all, ok := a.enter([]string{"*"})
	if !ok {
		t.Fatal("ids were not released")
	}
	if _, ok := a.enter([]string{"v-9"}); ok {
		t.Fatal("an order was let in during a whole-vault sale")
	}
	all()
}
