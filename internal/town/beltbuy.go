package town

import (
	"context"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/town/belt"
)

// beltBuyReserve is the gold that must remain after a belt purchase: the reserve plus a minimal kit (a town portal
// scroll and the first healing potions), so a belt never eats the money for potions (docs/sme/potion-policy.md).
func (t *Town) beltBuyReserve() int {
	tn := t.tune()
	return tn.MinGoldReserve + t.e.Tune.Resource.KitTPGold + t.e.Tune.Resource.HPFirst*t.e.Tune.Resource.KitPotionGold
}

// needBelt says whether a belt purchase is worth trying: the knob is on, the character is high enough, the worn belt
// has fewer potion slots than a belt gives (12), and the gold covers the reserve plus a little. A visit that bought
// nothing is not repeated until the gold has grown by FutileGoldStep.
func (t *Town) needBelt(s *d2.Snapshot) bool {
	tn := t.tune()
	if tn.BeltBuyLevel <= 0 || s.Player == nil || s.Player.Level < tn.BeltBuyLevel || s.Inventory == nil {
		return false
	}
	if belt.Slots(t.e.Data, s.Inventory) >= 3*belt.Columns {
		return false
	}
	gold := s.Player.Gold
	return gold >= t.beltBuyReserve()+tn.FutileGoldStep/2 && (t.beltBuyGold == 0 || gold >= t.beltBuyGold+tn.FutileGoldStep)
}

// buyBelt buys the cheapest belt in the repair vendor's stock that has more potion slots than the worn one and equips
// it (equipUnit swaps over a worn belt and the old one goes to the pack for the sell chore, V rebuy.go; the swap over a
// belt specifically is U). Belts come before no armor: docs/sme/early-gear-policy.md, "a belt beats any armor at clvl 12".
func (t *Town) buyBelt(ctx context.Context, a *Act, rep *ChoreReport) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	cur := belt.Slots(t.e.Data, s.Inventory)
	g := gearSlot{"belt", d2.BodyBelt, "belt", "belt", 2, func(db *data.DB, b *data.ItemBase) bool {
		bl := db.BeltByIndex(b.BeltIndex)
		return bl != nil && bl.Slots > cur
	}}
	gold := s.Player.Gold
	var out RebuyReport
	t.rebuyGear(ctx, a, []gearSlot{g}, t.beltBuyReserve(), &out)
	if len(out.Equipped) == 0 {
		t.beltBuyGold = gold
	} else {
		t.beltBuyGold = 0
		rep.Bought["belt"]++
	}
	t.e.Log.Info("belt buy", "had", cur, "equipped", out.Equipped, "errors", out.Errors, "gold", gold)
	if len(out.Equipped) == 0 && len(out.Errors) > 0 {
		rep.add("beltbuy", true, out.Errors[0], nil)
	}
	return nil
}
