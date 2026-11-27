package d2dctl

import (
	"context"
	"errors"
	"time"

	"d2lod-bot/internal/cheat"
)

// The vault's Equip on the mercenary (card mercvault): the player's own request gives a backpack item to the mercenary
// through the fork's `item merc-equip` verb (docs/research/merc-gear.md section 9). The verb is the same one the Inventory
// page uses; this wrapper adds the ledger line that every player-ordered direct write carries (ADR 0014) and the measured-run
// refusal, exactly like DirectEquip. Never called by the bot's own play.

// DirectMercer is the optional merc half of a DirectEquipper.
type DirectMercer interface {
	// MercDirectCaps says whether the build lists item.merc-equip (probed `caps`).
	MercDirectCaps(ctx context.Context) (bool, error)
	// MercEquipDirect gives backpack item id to the mercenary; the result carries the body slot and the unit id of the
	// item that came off (now in the backpack, -1 none). A rule refusal is an *ItemRefused.
	MercEquipDirect(ctx context.Context, id uint32, order string) (ItemResult, error)
}

var _ DirectMercer = (*DirectEquip)(nil)

func (d *DirectEquip) MercDirectCaps(ctx context.Context) (bool, error) {
	caps, err := d.C.Caps(ctx)
	if err != nil {
		return false, err
	}
	return HasCap(caps, CapItemMercEquip), nil
}

func (d *DirectEquip) MercEquipDirect(ctx context.Context, id uint32, order string) (ItemResult, error) {
	if err := d.ready(order); err != nil {
		return ItemResult{}, err
	}
	e := cheat.Entry{Time: time.Now().UTC(), Actor: d.Actor, Instance: d.Instance, Character: d.Character, Kind: KindD2DDebug,
		Params: map[string]any{"verb": "item merc-equip", "order": order, "unit": id}, Reason: "player vault equip on the mercenary, order " + order}
	if err := d.Ledger.Append(e); err != nil {
		return ItemResult{}, err
	}
	r, err := d.C.ItemMercEquip(ctx, id)
	e.Time, e.Success = time.Now().UTC(), err == nil
	e.After = map[string]any{"slot": r.Slot, "displaced": r.Displaced}
	if err != nil {
		e.Error = err.Error()
	}
	if lerr := d.Ledger.Append(e); lerr != nil && err == nil {
		err = lerr
	}
	return r, err
}

// IsItemRefusal reports whether err is a rule refusal of an item verb and returns it.
func IsItemRefusal(err error) (*ItemRefused, bool) {
	var ir *ItemRefused
	if errors.As(err, &ir) {
		return ir, true
	}
	return nil, false
}
