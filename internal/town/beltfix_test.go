package town_test

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/hand"
)

// putInput is the d2d input: a belt put that lands the potion in belt box `box` (GX = box, GY 0, like the fork's dump).
type putInput struct {
	input.Controller
	r     *rig
	fail  []error // errors the next puts return, one each
	boxes []int
}

func (p *putInput) BeltPut(id uint32, box int) error {
	if len(p.fail) > 0 {
		err := p.fail[0]
		p.fail = p.fail[1:]
		return err
	}
	for i, u := range p.r.W.Items {
		if u.ID == id {
			p.r.W.Items[i].Item.Where, p.r.W.Items[i].Mode = d2.ContBelt, d2.ItemModeBelt
			p.r.W.Items[i].X, p.r.W.Items[i].Y = box, 0
			p.r.W.Items[i].Item.GX, p.r.W.Items[i].Item.GY = box, 0 // the fork's dump: box in GX
			p.boxes = append(p.boxes, box)
			return nil
		}
	}
	return hand.ErrNoItem
}

func d2dRig(t *testing.T) (*rig, *putInput) {
	r := newRig(t)
	pi := &putInput{Controller: r.Env.In, r: r}
	r.Env.In = pi
	return r, pi
}

// LOG SIGNATURE (BELTFIX F1): a belt put fails once with "item not found" right after a re-entry. The belt must not be
// marked full: the same call retries after a fresh read, and the next call fills too.
func TestBeltPutItemNotFoundDoesNotStick(t *testing.T) {
	r, pi := d2dRig(t)
	addPack(r, "hp1", 4)
	pi.fail = []error{hand.ErrNoItem}
	rep, err := r.T.FillBelt(context.Background())
	if err != nil || rep.Moved < 2 || rep.Failed != 0 || rep.Moved != count(r.W, d2.ContBelt) {
		t.Fatalf("retry after a vanished id: %+v err %v belt %d", rep, err, count(r.W, d2.ContBelt))
	}

	// every put fails with item-not-found: nothing sticks, the next call (a later tick) fills
	r, pi = d2dRig(t)
	addPack(r, "hp1", 4)
	pi.fail = []error{hand.ErrNoItem, hand.ErrNoItem, hand.ErrNoItem, hand.ErrNoItem}
	_, _ = r.T.FillBelt(context.Background())
	if !r.T.NeedsBeltFill(mustSnap(t, r)) {
		t.Fatal("belt marked full after item-not-found")
	}
	rep, err = r.T.FillBelt(context.Background())
	if err != nil || rep.Moved < 2 {
		t.Fatalf("second call: %+v err %v", rep, err)
	}
}

// Another failure kind still sticks, and a new game (ForgetBelt) or a changed pack lifts the mark.
func TestBeltFullMarkResets(t *testing.T) {
	r, pi := d2dRig(t)
	addPack(r, "hp1", 4)
	boom := errors.New("boom")
	pi.fail = []error{boom, boom}
	if _, err := r.T.FillBelt(context.Background()); err == nil {
		t.Fatal("want the failure")
	}
	if r.T.NeedsBeltFill(mustSnap(t, r)) {
		t.Fatal("a real failure should stick until something changes")
	}
	r.T.ForgetBelt()
	if !r.T.NeedsBeltFill(mustSnap(t, r)) {
		t.Fatal("ForgetBelt did not reset the mark")
	}
	pi.fail = []error{boom, boom}
	_, _ = r.T.FillBelt(context.Background())
	if r.T.NeedsBeltFill(mustSnap(t, r)) {
		t.Fatal("mark not set again")
	}
	addPack(r, "hp1", 1) // the pack's potion set changed
	if !r.T.NeedsBeltFill(mustSnap(t, r)) {
		t.Fatal("a changed pack did not lift the mark")
	}
}

// BELTFIX F8: a mana potion is never put above a healing column, and a healing potion not above a mana column.
func TestBeltPutRespectsColumnKind(t *testing.T) {
	r, pi := d2dRig(t)
	r.W.Player.Level = 14
	// row 0: hp, mp, mp, mp (columns 0..3)
	r.W.Add("hp1", d2.ContBelt, 0, 0)
	for c := 1; c < 4; c++ {
		r.W.Add("mp1", d2.ContBelt, c, 0)
	}
	addPack(r, "hp1", 2)
	if _, err := r.T.FillBelt(context.Background()); err != nil {
		t.Fatal(err)
	}
	// BELTFIX layout rule: a Sorceress plans two mana columns, so one of the three mana columns may be emptied for healing;
	// what must hold is that no column mixes kinds (a healing potion never sits above a mana potion)
	assertColumnsPure(t, r)
	_ = pi
	// a belt whose only free boxes are above mana columns takes no healing potion
	r, pi = d2dRig(t)
	r.W.Add("hp1", d2.ContBelt, 0, 0)
	r.W.Add("hp1", d2.ContBelt, 4, 0)
	r.W.Add("hp1", d2.ContBelt, 8, 0)
	r.W.Add("hp1", d2.ContBelt, 12, 0)
	for c := 1; c < 4; c++ {
		r.W.Add("mp1", d2.ContBelt, c, 0)
	}
	addPack(r, "hp1", 1)
	_, _ = r.T.FillBelt(context.Background())
	assertColumnsPure(t, r)
	_ = pi
}

// BELTFIX F3 + the empty-belt visit: a visit that starts with an empty belt and potions in the pack ends with a full
// belt, and potions bought by the shop are belted before the visit ends.
func TestVisitEndsWithBeltFilled(t *testing.T) {
	r, _ := d2dRig(t)
	dressed(r.W)
	addPack(r, "hp1", 4)
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true}); err != nil {
		t.Fatal(err)
	}
	if n := count(r.W, d2.ContBelt); n == 0 {
		t.Fatal("belt still empty after the visit")
	}
	left := 0
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContInventory && r.Env.Data.Items[u.Class].Code == "hp1" {
			left++
		}
	}
	if left > 0 && count(r.W, d2.ContBelt) < 4 {
		t.Errorf("%d healing potions left in the pack with %d belt slots used", left, count(r.W, d2.ContBelt))
	}
}

// BELTFIX F3: d2d purchases land in the pack; the belt step after the shop belts them in the same visit.
func TestShopPurchaseIsBelted(t *testing.T) {
	r, _ := d2dRig(t)
	r.W.NoAutoBelt = true
	dressed(r.W)
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true})
	if err != nil {
		t.Fatal(err)
	}
	if bought(rep) == 0 {
		t.Skipf("nothing bought: %+v", rep.Steps)
	}
	if count(r.W, d2.ContBelt) == 0 {
		t.Fatalf("bought %v but the belt is empty: %+v", rep.Bought, rep.Steps)
	}
}

// LOG SIGNATURE (FIXBELT2, post-fix soak 1): a sash whose four columns hold healing potions except a full mana column
// ("no belt column for Minor Mana Potion (kind mp)", belt marked full). A mana potion in the pack has no room: that is no
// failure, nothing is marked full, nothing is clicked, and a healing potion for the open healing box still moves.
func TestNoBeltColumnIsNoRoomNotFull(t *testing.T) {
	r, pi := d2dRig(t)
	r.W.Add("lbl", d2.ContEquipment, 0, 0).Item.Body = d2.BodyBelt
	for _, b := range []int{0, 4, 1, 5, 3, 7, 2} {
		code := "hp1"
		if b%4 == 3 {
			code = "mp1"
		}
		r.W.Add(code, d2.ContBelt, b, 0)
	}
	addPack(r, "mp1", 2)
	if r.T.NeedsBeltFill(mustSnap(t, r)) {
		t.Fatal("a mana potion with no box is a candidate")
	}
	rep, err := r.T.FillBelt(context.Background())
	if err != nil || rep.Failed != 0 || rep.Moved != 0 || len(pi.boxes) != 0 {
		t.Fatalf("no-room fill: %+v err %v boxes %v", rep, err, pi.boxes)
	}
}

// FIXBELT2: healing potions must not take the empty columns the plan keeps for mana: an empty sash and a pack of both
// kinds ends with a mana column on the belt.
func TestHealingDoesNotTakeTheManaColumns(t *testing.T) {
	r, pi := d2dRig(t)
	r.W.Add("lbl", d2.ContEquipment, 0, 0).Item.Body = d2.BodyBelt
	addPack(r, "hp1", 6)
	addPack(r, "mp1", 4)
	if _, err := r.T.FillBelt(context.Background()); err != nil {
		t.Fatal(err)
	}
	mp := 0
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContBelt && r.Env.Data.Items[u.Class].Code == "mp1" {
			mp++
		}
	}
	if mp < 2 {
		t.Fatalf("belt holds %d mana potions of 4 in the pack (boxes %v)", mp, pi.boxes)
	}
}

// assertColumnsPure fails when a belt column holds both kinds.
func assertColumnsPure(t *testing.T, r *rig) {
	t.Helper()
	kind := map[int]string{}
	for _, u := range r.W.Items {
		if u.Item.Where != d2.ContBelt {
			continue
		}
		c := (u.Item.GY*4 + u.Item.GX) % 4
		k := r.W.DB.Items[u.Class].Code[:2]
		if prev, ok := kind[c]; ok && prev != k {
			t.Errorf("column %d mixes %s and %s", c, prev, k)
		}
		kind[c] = k
	}
}
