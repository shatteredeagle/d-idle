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
// Crash journal (card vault2, mercjournal.go): an intent record with the hero's items as they were is on disk before the
// create; a stop or a lost reply at any later step is settled at the next Run by looking (copy still in the pack: taken
// back, the record stays; copy gone: given, the record leaves and every new wearable piece in the pack is filed under the
// order id). U live: whether `item full` lists the mercenary's own items (the fake does not).

// MercHand is the optional part of a Hand that judges a record for the mercenary without needing the hero to wear it.
type MercHand interface {
	MercPreview(rec vault.Rec) MercSide
}

// mercSlotKey maps the fork's body slot numbers to the page's merc slot keys.
var mercSlotKey = map[int]string{1: "mhead", 3: "mtorso", 4: "mrhand", 5: "mlhand"}

const mercKey = keyPrefix + "eqm-"

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
	k.actMu.Lock()
	defer k.actMu.Unlock()
	rec, ok := st.Get(id)
	switch {
	case !ok:
		res.Code, res.Why = "missing", "no such item"
		return res, true
	case rec.State != vault.Recorded || rec.Hold != "":
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

	k.repairMerc(ctx) // an order a stop left half done is settled before a new give can displace its piece
	if k.mercOpen(id) {
		res.Code, res.Why = CodeInFlight, ReasonInFlight
		return res, true
	}
	if r, ok := st.Get(id); !ok || r.State != vault.Recorded { // the repair finished an earlier give of this record
		res.Code, res.Why = "state", "the item is not in the vault any more: an earlier order gave it to the mercenary"
		return res, true
	}
	// The intent is on disk before the game is touched, with the hero's items as they are now: after a stop or a lost
	// reply the bot tells by looking whether the copy was made, whether it was given, and which item came off.
	pre, err := k.API.ItemFull(ctx, -1)
	if err != nil {
		res.Code, res.Why = "busy", "the backpack could not be read first ("+err.Error()+"): nothing was moved"
		return res, true
	}
	order := "eqm-" + orderTag(id) + fmt.Sprintf("-%d", k.now().UnixMilli())
	in := mercIntent{ID: id, Order: order, Name: rec.Name, Hex: createHex(rec), Step: mercCreate, At: k.now()}
	for _, h := range pre {
		in.Pack = append(in.Pack, strings.ToLower(h.Norm))
	}
	if err := k.mercPut(id, &in); err != nil {
		res.Code, res.Why = "journal", "the order could not be written down first ("+err.Error()+"): nothing was moved"
		return res, true
	}
	// The key makes a repeated create of the same record answer "dup" instead of a second copy (per d2d process: U live).
	cr, err := k.API.ItemCreate(ctx, in.Hex, d2dctl.CreateOpts{Pack: true, Key: mercKey + id})
	if errors.Is(err, d2dctl.ErrVaultDup) {
		_ = k.mercPut(id, nil)
		res.Code, res.Why = "dup", "an earlier order already took this item out of the vault: check your backpack and the mercenary"
		k.note("eqm-dup-"+id, "Equip on mercenary: "+rec.Name+" was created by an earlier order that did not finish: check your backpack and the mercenary.")
		return res, true
	}
	if err != nil {
		res.Code, res.Why = "room", "the item could not be put in your backpack ("+err.Error()+"): make room first"
		if createAnswered(err) {
			_ = k.mercPut(id, nil)
		} else { // unanswered: the copy may be in the pack; settled by looking, now or at the next pass
			k.repairMerc(ctx)
			res.Code, res.Why = "unknown", "the game did not answer: check your backpack"
		}
		return res, true
	}
	in.Step, in.Unit = mercGive, cr.ID
	if err := k.mercPut(id, &in); err != nil {
		k.note("eqm-journal", "Equip on mercenary: the order file could not be updated ("+err.Error()+").")
	}
	ir, err := dm.MercEquipDirect(ctx, uint32(cr.ID), order)
	if _, refused := d2dctl.IsItemRefusal(err); err != nil && !refused && !answered(err) && !errors.Is(err, d2dctl.ErrDirectOff) {
		// No answer: the give may have run. The intent says what was there before; look.
		out := k.repairMerc(ctx)[id]
		switch {
		case out.open:
			k.note("eqm-open-"+order, "Equip on mercenary: the game did not answer for "+rec.Name+"; the bot looks again at the next step.")
			res.Code, res.Why = CodeInFlight, ReasonUnknown
		case !out.given:
			res.Code, res.Why = "equip", "the game did not answer and nothing changed: try again"
		default:
			res.OK, res.Removed, res.Where = true, out.names, out.where
		}
		return res, true
	}
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
		_ = k.mercPut(id, nil)
		return res, true
	}

	k.mercRetire(st, id, rec.Name)
	slot := mercSlotKey[ir.Slot]
	if slot == "" {
		slot = fmt.Sprintf("merc slot %d", ir.Slot)
	}
	var names []string
	where := "vault"
	open := false
	if ir.Displaced >= 0 {
		// The record is written BEFORE the piece leaves the pack (its bytes are readable while it lies there), as Stow does.
		have, ferr := k.API.ItemFull(ctx, ir.Displaced)
		switch {
		case ferr != nil: // cannot look now: the intent stays and the next pass files the piece by the pack difference
			where, open = "backpack", true
			names = append(names, "an item")
			k.Hand.Protect(uint32(ir.Displaced))
			k.note("eqm-off-"+id, "The piece the mercenary took off is in your backpack; the bot files it at the next step.")
		case len(have) != 1:
			where = "backpack"
			names = append(names, "an item")
			k.note("eqm-off-"+id, "The piece the mercenary took off could not be found in your backpack: check it.")
		default:
			var n string
			n, where, open = k.mercFile(ctx, st, have[0], order)
			names = append(names, n)
		}
	}
	if !open { // an unanswered removal keeps the intent: the next pass looks
		_ = k.mercPut(id, nil)
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
