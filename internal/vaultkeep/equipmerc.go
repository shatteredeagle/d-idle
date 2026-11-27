package vaultkeep

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Equip on the mercenary (card mercvault), the merc twin of the instant Equip (equipdirect.go). The record is created in
// the backpack, the fork's `item merc-equip` gives it to the mercenary at once, and the piece that came off (now in the
// backpack) is vaulted into a "previously worn" record. Player-ordered only, ledgered by DirectEquip.MercEquipDirect,
// refused in a measured run. Which pieces his kind may wear is the merc gear rule (Hand.MercPreview, leveler.MercSlot);
// his level, Str and Dex are checked by the fork with the server's own rule and come back as a refusal.
//
// Not covered: a restart between the create and the give leaves the created copy in the backpack under no mark (the
// record stays in the vault, so the item is duplicated until the player drops one); U live.

// MercHand is the optional part of a Hand that judges a record for the mercenary without needing the hero to wear it.
type MercHand interface {
	MercPreview(rec vault.Rec) MercSide
}

// mercSlotKey maps the fork's body slot numbers to the page's merc slot keys.
var mercSlotKey = map[int]string{1: "mhead", 3: "mtorso", 4: "mrhand", 5: "mlhand"}

const ReasonMercOff = "equipping the mercenary from the Vault needs the fork build with item merc-equip (and the instant equip switched on)"

func (k *Keeper) mercDirect() (d2dctl.DirectMercer, bool) {
	dm, ok := k.Direct.(d2dctl.DirectMercer)
	return dm, ok && k.Direct != nil
}

// MercReady says Equip on merc may be offered: the instant path is ready and the build lists item.merc-equip.
func (k *Keeper) MercReady(ctx context.Context) bool {
	dm, ok := k.mercDirect()
	if !ok || !k.directReady(ctx) {
		return false
	}
	_, mh := k.Hand.(MercHand)
	have, err := dm.MercDirectCaps(ctx)
	return mh && err == nil && have
}

func mercCode(err error) (code, why string) {
	if ir, ok := d2dctl.IsItemRefusal(err); ok {
		switch ir.Code {
		case d2dctl.ItemErrIdentified:
			return "reveal", ReasonEqReveal
		case d2dctl.ItemErrMercType:
			return "merc", "your mercenary cannot wear that"
		case d2dctl.ItemErrRequirements, d2dctl.ItemErrClass:
			return "requirements", "your mercenary's level, strength or dexterity is too low for that: " + ir.Why
		case d2dctl.ItemErrNoSpace:
			return "room", "no room in your backpack for what the mercenary would take off"
		case d2dctl.ItemErrDead:
			return "dead", "the mercenary is dead: revive him first"
		case "merc":
			return "merc", "no mercenary is hired"
		}
		return ir.Code, ir.Why
	}
	return directWhy(err)
}

// EquipMerc gives the vault record id to the mercenary at once. ok is false only when the path is not on offer at all
// (the caller then answers ReasonMercOff).
func (k *Keeper) EquipMerc(ctx context.Context, id string) (Result, bool) {
	dm, has := k.mercDirect()
	if !has || !k.MercReady(ctx) {
		return Result{}, false
	}
	res := Result{ID: id, Direct: true}
	if ok, code, why := k.directGate(); !ok {
		res.Code, res.Why = code, why
		return res, true
	}
	st := k.store()
	if st == nil {
		res.Code, res.Why = "missing", "there is no vault for this hero yet"
		return res, true
	}
	rec, ok := st.Get(id)
	switch {
	case !ok:
		res.Code, res.Why = "missing", "no such item"
		return res, true
	case rec.State != vault.Recorded:
		res.Code, res.Why = "state", "the item is not in the vault any more or is busy ("+string(rec.State)+")"
		return res, true
	case !rec.Identified:
		res.Code, res.Why = "reveal", ReasonEqReveal
		return res, true
	}
	side := k.Hand.(MercHand).MercPreview(rec)
	if !side.Owned || !side.Can {
		res.Code, res.Why = "merc", side.Why
		if res.Why == "" {
			res.Why = "your mercenary cannot wear that"
		}
		return res, true
	}

	cr, err := k.API.ItemCreate(ctx, createHex(rec), d2dctl.CreateOpts{Pack: true})
	if err != nil {
		res.Code, res.Why = "room", "the item could not be put in your backpack ("+err.Error()+"): make room first"
		return res, true
	}
	order := "eqm-" + orderTag(id) + fmt.Sprintf("-%d", k.now().UnixMilli())
	ir, err := dm.MercEquipDirect(ctx, uint32(cr.ID), order)
	if err != nil {
		res.Code, res.Why = mercCode(err)
		if errors.Is(err, d2dctl.ErrDirectOff) {
			k.mu.Lock()
			k.dcapsOK = false
			k.mu.Unlock()
		}
		// Nothing moved on the mercenary: take the created copy away again, the record stays in the vault.
		if _, verr := k.API.ItemVault(ctx, cr.ID); verr != nil {
			k.Hand.Protect(uint32(cr.ID))
			k.note("eqm-"+id, "The mercenary refused the "+rec.Name+" and its copy is still in your backpack ("+verr.Error()+"): drop it.")
		}
		return res, true
	}

	if r, ok := st.Get(id); ok && r.State == vault.Recorded {
		_ = st.Discard(id, "equipped on the mercenary (vault equip)")
	}
	slot := mercSlotKey[ir.Slot]
	if slot == "" {
		slot = fmt.Sprintf("merc slot %d", ir.Slot)
	}
	var names []string
	where := "vault"
	if ir.Displaced >= 0 {
		rr, verr := k.API.ItemVault(ctx, ir.Displaced)
		if verr != nil {
			where = "backpack"
			names = append(names, "an item")
			k.Hand.Protect(uint32(ir.Displaced))
			k.note("eqm-off-"+id, "The piece the mercenary took off is in your backpack: the vault could not take it ("+verr.Error()+").")
		} else {
			n, w := k.recordItem(ctx, st, rr, "")
			names, where = append(names, n), w
		}
	}
	if k.Save != nil {
		if err := k.Save(ctx); err != nil {
			k.note("save", "Equip: "+ReasonSave+" ("+err.Error()+")")
		}
	}
	k.mu.Lock()
	k.info.Done++
	k.mu.Unlock()
	res.OK, res.Slot, res.Removed, res.Where = true, slot, names, where
	line := fmt.Sprintf("equip: %s worn by the mercenary in %s", rec.Name, slot)
	if len(names) > 0 {
		line += fmt.Sprintf(", %s went to the %s", strings.Join(names, ", "), where)
	}
	if k.Notice != nil {
		k.Notice(line)
	}
	return res, true
}
