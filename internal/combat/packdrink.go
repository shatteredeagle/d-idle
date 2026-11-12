package combat

import (
	"context"
	"errors"
	"strconv"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
)

// Pack drink (docs/beltfix.md): the belt holds no healing or rejuvenation potion, life is under combat.drinkRejuvPct and the
// pack holds one. In D2 a right click on a potion in the open inventory drinks it (S, docs/research/belt-trace.md Q3 table
// "right on a pack potion = use"; the portal scroll uses the same click here, V live). The panel has to be opened and
// closed around it, so this is the fallback for an empty belt, never the normal drink: about 100 ms hover, up to 900 ms
// for the effect, plus the panel opening (waitUI). Not verified live on d2d (U).

// packDrinkBackoff keeps a failed pack drink from re-firing every tick.
const packDrinkBackoff = 3 * time.Second

// packPotion is the pack potion to drink in an emergency: a rejuvenation first when kind is rejuvenation, else the
// strongest healing potion (an emergency takes the strongest, like the belt rule).
func (f *Fighter) packPotion(snap *d2.Snapshot, kind PotionKind) *d2.Unit {
	if snap == nil || snap.Inventory == nil {
		return nil
	}
	var best *d2.Unit
	bestTier := -1
	for i := range snap.Inventory.Items {
		u := &snap.Inventory.Items[i]
		if u.Item == nil || u.Item.Where != d2.ContInventory {
			continue
		}
		c := f.cat.Item(u.Class)
		if c.Potion == kind && c.Tier > bestTier {
			best, bestTier = u, c.Tier
		}
	}
	return best
}

// packDrinkWanted says whether Safety should ask for a healing drink although the belt has none: in a fight, below
// the rejuvenation line, with a pack potion, not backed off. The chicken line stays above it (decideBase runs first).
func (f *Fighter) packDrinkWanted(snap *d2.Snapshot, in SafetyInput) bool {
	if in.InTown || !in.InFight || !in.HealReady || in.Belt.Count(PotionHealing)+in.Belt.Count(PotionRejuv) > 0 {
		return false
	}
	if pct(in.Life, in.Max) >= in.Tune.DrinkRejuvPct {
		return false
	}
	f.diagMu.Lock()
	backed := f.now().Before(f.packDrinkBackoff)
	f.diagMu.Unlock()
	return !backed && (f.packPotion(snap, PotionHealing) != nil || f.packPotion(snap, PotionRejuv) != nil)
}

// drinkFromPack opens the inventory, right-clicks the strongest pack potion of the kind and checks it is gone.
func (f *Fighter) drinkFromPack(ctx context.Context, snap *d2.Snapshot, kind PotionKind) (err error) {
	it := f.packPotion(snap, kind)
	if kind == PotionHealing && it == nil {
		kind = PotionRejuv
		it = f.packPotion(snap, kind)
	}
	if it == nil {
		return ErrNoPotion
	}
	defer func() {
		if err != nil {
			f.diagMu.Lock()
			f.packDrinkBackoff = f.now().Add(packDrinkBackoff)
			f.diagMu.Unlock()
		}
	}()
	grid, err := f.inventoryGrid(snap)
	if err != nil {
		return err
	}
	if !snap.UI.Open(d2.UIInventory) {
		if err := f.e.In.Key("i"); err != nil {
			return err
		}
		if !f.waitUI(ctx, d2.UIInventory) {
			return errors.New("combat: inventory panel did not open for the pack drink")
		}
	}
	defer f.e.In.CloseAll()
	w, h := f.itemSize(it.Class)
	c := grid.ItemCenter(it.Item.GX, it.Item.GY, w, h)
	if err := f.e.In.MoveMouse(c.X, c.Y); err != nil {
		return err
	}
	if err := f.sleep(ctx, 100*time.Millisecond); err != nil {
		return err
	}
	if err := f.e.In.Click(input.Right, c.X, c.Y); err != nil {
		return err
	}
	for i := 0; i < 6; i++ {
		if err := f.sleep(ctx, 150*time.Millisecond); err != nil {
			return err
		}
		s, e := f.snapFn()
		if e != nil || s == nil || s.Inventory == nil {
			continue
		}
		if !packHas(s, it.ID) {
			f.lastDrink[kind] = f.now()
			if kind == PotionHealing && snap.Player != nil {
				f.heal = f.heal.drink(f.now(), f.cat.Item(it.Class).Tier, snap.Player.Class)
			}
			f.info("potion: pack drink", "kind", kind.String(), "tier", f.cat.Item(it.Class).Tier, "life", snap.Player.Life, "max", snap.Player.MaxLife)
			f.emit(goals.Event{Type: goals.EvPotion, Potion: kind.String(),
				Message: "Drank a " + kind.String() + " potion from the pack at " + strconv.Itoa(pct(snap.Player.Life, snap.Player.MaxLife)) + "% life"})
			return nil
		}
	}
	return errors.New("combat: the pack potion was not used after the click")
}

func packHas(s *d2.Snapshot, id uint32) bool {
	for i := range s.Inventory.Items {
		if u := &s.Inventory.Items[i]; u.ID == id && u.Item != nil && u.Item.Where == d2.ContInventory {
			return true
		}
	}
	return false
}
