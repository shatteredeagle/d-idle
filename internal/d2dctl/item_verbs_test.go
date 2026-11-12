package d2dctl

import (
	"context"
	"errors"
	"testing"
)

func TestCapsOnForkAndStock(t *testing.T) {
	_, c := startStub(t, map[string]string{"caps": "caps=state.caps,item.equip,item.unequip\nok\n"}, nil)
	caps, err := c.Caps(context.Background())
	if err != nil || !HasCap(caps, CapItemEquip) || !HasCap(caps, CapItemUnequip) || HasCap(caps, "item.drop") {
		t.Fatalf("caps %v %v", caps, err)
	}
	// Stock d2d (and an older fork) does not know the verb: no capability, no error.
	_, c = startStub(t, map[string]string{"caps": "err unknown verb caps\n"}, nil)
	if caps, err := c.Caps(context.Background()); err != nil || len(caps) != 0 {
		t.Fatalf("stock: %v %v", caps, err)
	}
}

func TestItemEquipReplies(t *testing.T) {
	ctx := context.Background()
	s, c := startStub(t, map[string]string{
		"item equip 5 4":  "equip id=5 slot=4 displaced=9 at=3,2\nok\n",
		"item equip 6 0":  "noop slot=1 at=-1,-1\nok\n",
		"item equip 7 4":  "err twohand a second item would have to come off\n",
		"item equip 8 4":  "err lab verbs off: item needs d2d --lab-verbs\n",
		"item unequip 10": "unequip id=10 slot=4 displaced=-1 at=0,1\nok\n",
	}, nil)
	r, err := c.ItemEquip(ctx, 5, 4)
	if err != nil || r.Noop || r.Slot != 4 || r.Displaced != 9 || r.AtCol != 3 || r.AtRow != 2 {
		t.Fatalf("equip: %+v %v", r, err)
	}
	if r, err = c.ItemEquip(ctx, 6, 0); err != nil || !r.Noop || r.Slot != 1 {
		t.Fatalf("noop: %+v %v", r, err)
	}
	_, err = c.ItemEquip(ctx, 7, 4)
	var ref *ItemRefused
	if !errors.As(err, &ref) || ref.Code != ItemErrTwoHand || !ref.RuleRefusal() {
		t.Fatalf("refusal: %v", err)
	}
	_, err = c.ItemEquip(ctx, 8, 4) // lab verbs off is not a rule refusal: the caller falls back
	if err == nil || errors.As(err, &ref) {
		t.Fatalf("lab verbs off: %v", err)
	}
	if r, err = c.ItemUnequip(ctx, 10); err != nil || r.AtCol != 0 || r.AtRow != 1 {
		t.Fatalf("unequip: %+v %v", r, err)
	}
	if got := s.seen[len(s.seen)-1]; got != "item unequip 10" {
		t.Fatalf("sent %q", got)
	}
	// The adapter invdesk uses.
	noop, err := ItemVerbs{C: c}.ItemEquip(ctx, 6, 0)
	if err != nil || !noop {
		t.Fatalf("adapter: %v %v", noop, err)
	}
}
