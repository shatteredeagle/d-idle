package vaultkeep

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Equip v2 (docs/vault.md, "Equip v2"; ADR 0029 amended 2026-11-25). The player's Equip, or Wear set, calls the fork's direct
// verbs at once: the item goes straight into its body slot (no town trip, no pack step, no clicks), the engine hands back
// what came off, and those items become vault records in the same call. Player-ordered only: the bot's own play never calls
// it, a measured run refuses it here and in the fork, every call is ledgered with a reason by d2dctl.DirectEquip.
//
// Safety in the field: the verb is one atomic command on the control socket, which the bot's own commands share (calls are
// serialised, a fight goes on between two commands), so it needs no safe moment and no town. Nothing is read back from the
// snapshot (it may lag by a game step); the engine's answer is the result (U live).

const (
	ReasonEqReveal  = "Reveal it first (Gold): the game will not wear an unidentified item"
	ReasonDirectOff = "direct equip is switched off or this bot cannot do it"
)

const directCapsTTL = capsTTL

func (k *Keeper) directWanted() bool {
	return k.Direct != nil && k.DirectOn != nil && k.DirectOn() && !k.measured()
}

// directReady is true when the knob is on, no measured run, the build has the verbs (probed `caps`, cached) and `item decode`
// is there to turn what came off into records. Anything else leaves the order flow.
// DirectReady says the one-tap Equip wears at once through the direct verbs (the page then skips the order flow).
func (k *Keeper) DirectReady(ctx context.Context) bool { return k.directReady(ctx) }

func (k *Keeper) directReady(ctx context.Context) bool {
	if k.Hand == nil || !k.directWanted() || !k.enabledDirect() {
		return false
	}
	k.mu.Lock()
	ok, at, known := k.dcaps, k.dcapsAt, k.dcapsOK
	k.mu.Unlock()
	if known && k.now().Sub(at) < directCapsTTL {
		return ok
	}
	have, err := k.Direct.DirectCaps(ctx)
	if err == nil && have {
		_, dok := k.API.(d2dctl.Decoder)
		have = dok
		if c, known, cerr := k.vaultCaps(ctx, true); cerr == nil && known && !c.Decode {
			have = false
		}
	}
	if err != nil {
		return false // unknown: ask again next time, do not cache
	}
	k.mu.Lock()
	k.dcaps, k.dcapsAt, k.dcapsOK = have, k.now(), true
	k.mu.Unlock()
	return have
}

// enabledDirect: the vault itself is on (loot.vault) and creating has not tripped.
func (k *Keeper) enabledDirect() bool { return k.enabled() }

// directGate is EquipGate in direct mode: only a dead hero stops it (not town, not a safe moment).
func (k *Keeper) directGate() (bool, string, string) {
	if k.Ready != nil && k.Ready() == ReasonDead {
		return false, "dead", ReasonDead
	}
	return true, "", ""
}

func mapDirectCode(code string) string {
	switch code {
	case "class", "level", "str", "dex", "requirements", "hands":
		return "requirements"
	case "identified":
		return "reveal"
	}
	return code
}

func directWhy(err error) (code, why string) {
	var ref *d2dctl.DirectRefused
	switch {
	case errors.As(err, &ref):
		w := ref.Why
		if ref.Code == "identified" {
			w = ReasonEqReveal
		}
		return mapDirectCode(ref.Code), w
	case errors.Is(err, d2dctl.ErrDirectMeasured):
		return "off", ReasonEqOff
	}
	return "equip", err.Error()
}

func orderTag(id string) string {
	h := sha1.Sum([]byte(id))
	return hex.EncodeToString(h[:5])
}

// chosen is a piece that passed the checks: where it goes and what is worn there now.
type chosen struct {
	idx  int
	id   string
	rec  vault.Rec
	slot string
	pv   Preview
	hex  string
}

// precheck judges one record for the direct path (identified, wearable, a slot that fits). A refusal carries the code and the
// reason the toast shows.
func (k *Keeper) precheck(st *vault.Store, id, slotWord string, taken map[string]string) (c chosen, code, why string) {
	rec, ok := st.Get(id)
	switch {
	case !ok:
		return c, "missing", "no such item"
	case rec.State != vault.Recorded:
		return c, "state", "the item is not in the vault any more or is busy (" + string(rec.State) + ")"
	case !rec.Identified:
		return c, "reveal", ReasonEqReveal
	}
	pv := k.Hand.Preview(rec)
	pv.Wearable = pv.Wearable || pv.Reveal // Preview counts an unrevealed record as wearable; ours is revealed here
	group, w, cd := orderGroup(pv, slotWord)
	if w != "" {
		return c, cd, w
	}
	slot := ""
	var bestScore float64
	for _, sv := range pv.Slots {
		if !contains(group, sv.Slot) {
			continue
		}
		if o, used := taken[sv.Slot]; used {
			if slot == "" {
				code, why = "slot", "another piece of the set already takes that slot ("+o+")"
			}
			continue
		}
		if sv.Worn == nil {
			slot = sv.Slot
			break
		}
		if slot == "" || sv.Worn.Score < bestScore {
			slot, bestScore = sv.Slot, sv.Worn.Score
		}
	}
	if slot == "" {
		return c, code, why
	}
	return chosen{id: id, rec: rec, slot: slot, pv: pv, hex: createHex(rec)}, "", ""
}

func contains(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}

// wornName is what the preview says is worn in slot ("" when empty).
func (c chosen) wornName(slot string) (string, bool) {
	for _, sv := range c.pv.Slots {
		if sv.Slot == slot && sv.Worn != nil {
			return sv.Worn.Name, true
		}
	}
	return "", false
}

// vaultRoom is how many records the vault can still take once the pieces' own records have left.
func vaultRoom(st *vault.Store, leaving int) int {
	s := st.Status()
	return s.Capacity - s.Count + leaving
}

func (k *Keeper) directOne(ctx context.Context, id, slotWord string) (Result, bool) {
	if !k.directReady(ctx) {
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
	c, code, why := k.precheck(st, id, slotWord, nil)
	if why != "" {
		res.Code, res.Why = code, why
		return res, true
	}
	if _, occupied := c.wornName(c.slot); occupied && vaultRoom(st, 1) < 2 { // the worn item, and a shield a two-hander may push off
		if vaultRoom(st, 1) < 1 {
			res.Code, res.Why = "room", ReasonNoRoom
			return res, true
		}
	}
	order := "eq-" + orderTag(id) + fmt.Sprintf("-%d", k.now().UnixMilli())
	k.putEq(&EquipOrder{ID: id, Name: c.rec.Name, Slot: slotWord, Group: []string{c.slot}, State: EqEquipping, Direct: true})
	dr, err := k.Direct.EquipDirect(ctx, c.hex, c.slot, order)
	if err != nil {
		if errors.Is(err, d2dctl.ErrDirectOff) { // an older image: the order flow
			k.mu.Lock()
			delete(k.bookLocked().orders, id)
			k.dcapsOK = false
			k.mu.Unlock()
			return Result{}, false
		}
		res.Code, res.Why = directWhy(err)
		k.failOrder(id, res.Code, res.Why)
		return res, true
	}
	k.setOrder(id, func(o *EquipOrder) { o.Applied, o.Removed, o.Worn, o.Unit = true, dr.Removed, c.slot, dr.Unit })
	k.finishDirect(ctx, st, c, dr, &res)
	return res, true
}

// finishDirect is the store step after the engine swapped: the equipped record leaves the vault, what came off becomes
// records ("previously worn", identified state kept), the character is saved. Idempotent (repairDirect re-runs it).
func (k *Keeper) finishDirect(ctx context.Context, st *vault.Store, c chosen, dr d2dctl.DirectResult, res *Result) {
	if r, ok := st.Get(c.id); ok && r.State == vault.Recorded {
		_ = st.Discard(c.id, "equipped directly (vault equip v2)")
	}
	where := "vault"
	var names []string
	for i, h := range dr.Removed {
		name, _ := c.wornName(c.slot)
		if i > 0 || name == "" {
			name = ""
		}
		n, w := k.recordRemoved(ctx, st, h, name)
		names = append(names, n)
		switch {
		case w == "backpack" || where == "backpack":
			where = "backpack"
		case w == "stash":
			where = "stash"
		}
	}
	if k.Pinner != nil {
		k.Pinner(c.slot, uint32(dr.Unit), c.rec.Name)
	}
	if k.Save != nil {
		if err := k.Save(ctx); err != nil {
			k.note("save", "Equip: "+ReasonSave+" ("+err.Error()+")")
		}
	}
	k.setOrder(c.id, func(o *EquipOrder) { o.State, o.Displaced, o.Where, o.Removed = EqDone, names, where, nil })
	k.mu.Lock()
	k.info.Done++
	k.mu.Unlock()
	res.OK, res.Slot, res.Removed, res.Where = true, c.slot, names, where
	line := fmt.Sprintf("equip: %s worn in %s", c.rec.Name, c.slot)
	if len(names) > 0 {
		line += fmt.Sprintf(", %s went to the %s", strings.Join(names, ", "), where)
	}
	if k.Notice != nil {
		k.Notice(line) // the bot log line the Proving Ground scenarios read
	}
}

// recordRemoved turns the bytes the engine returned into a vault record. When the vault cannot take it the item is created
// into the stash, else the backpack, so nothing is lost; the order says where it is.
func (k *Keeper) recordRemoved(ctx context.Context, st *vault.Store, hexRaw, name string) (string, string) {
	dec, _ := k.API.(d2dctl.Decoder)
	var r d2dctl.VaultRecord
	var err error
	if dec != nil {
		r, err = dec.ItemDecode(ctx, hexRaw)
	} else {
		err = errors.New("no decoder")
	}
	if err != nil {
		w := k.createBack(ctx, hexRaw)
		k.note("equip-"+hexRaw[:min(8, len(hexRaw))], "An item you replaced could not be read for the vault ("+err.Error()+"); it is in your "+w+".")
		return "an item", w
	}
	return k.recordItem(ctx, st, r, name)
}

// recordItem files a decoded item as a "previously worn" vault record (the vault full: the stash, else the backpack).
func (k *Keeper) recordItem(ctx context.Context, st *vault.Store, r d2dctl.VaultRecord, name string) (string, string) {
	if name == "" && k.Namer != nil {
		name = k.Namer(r, vault.Rec{})
	}
	if name == "" {
		name = r.Code
	}
	it := vault.Item{Raw: r.Norm, Code: r.Code, Name: name, Quality: r.Quality, Ilvl: r.ILvl, Identified: true, Props: r.Props,
		Sockets: r.Sockets, Eth: r.Ethereal, Found: vault.Found{Time: k.now().UTC(), Zone: "previously worn"}}
	it.Pic, it.Tint, it.FileIndex = r.Pic, r.Tint, r.FileIndex()
	if k.Chain != nil {
		it.Chain = k.Chain(r.Code)
	}
	if _, err := st.Add(it); err != nil && !errors.Is(err, vault.ErrDuplicate) {
		w := k.createBack(ctx, r.Norm)
		k.note("equip-"+name, "The "+name+" you replaced is in your "+w+": the vault could not take it ("+err.Error()+").")
		return name, w
	}
	return name, "vault"
}

// createBack puts an item that has no vault record back on the hero: the stash, else the backpack under the keep mark.
func (k *Keeper) createBack(ctx context.Context, hexNorm string) string {
	if _, err := k.API.ItemCreate(ctx, hexNorm, d2dctl.CreateOpts{}); err == nil {
		return "stash"
	}
	if r, err := k.API.ItemCreate(ctx, hexNorm, d2dctl.CreateOpts{Pack: true}); err == nil {
		k.Hand.Protect(uint32(r.ID))
		return "backpack"
	}
	return "backpack (not placed: " + hexNorm + ")"
}

func (k *Keeper) putEq(o *EquipOrder) {
	k.mu.Lock()
	defer k.mu.Unlock()
	o.At, o.Updated = k.now(), k.now()
	k.bookLocked().orders[o.ID] = o
	k.saveBookLocked()
}

// directSet is Wear set: every piece is judged, the pieces that fit go to the engine in ONE wearset call (in slotRank order,
// rings left then right), each piece gets its own result.
func (k *Keeper) directSet(ctx context.Context, ids []string) ([]Result, bool) {
	if !k.directReady(ctx) {
		return nil, false
	}
	out := make([]Result, len(ids))
	for i, id := range ids {
		out[i] = Result{ID: id, Direct: true}
	}
	if ok, code, why := k.directGate(); !ok {
		for i := range out {
			out[i].Code, out[i].Why = code, why
		}
		return out, true
	}
	st := k.store()
	if st == nil {
		for i := range out {
			out[i].Code, out[i].Why = "missing", "there is no vault for this hero yet"
		}
		return out, true
	}
	type cand struct {
		pos, rank int
		id        string
	}
	var cs []cand
	for i, id := range ids {
		rk := 1
		if rec, ok := st.Get(id); ok {
			rk = slotRank(k.Hand.Preview(rec))
		}
		cs = append(cs, cand{i, rk, id})
	}
	sort.SliceStable(cs, func(a, b int) bool { return cs[a].rank < cs[b].rank })
	taken := map[string]string{}
	var picks []chosen
	rings := 0
	for _, c := range cs {
		word := SlotAuto
		if c.rank == 4 {
			if rings < 2 {
				word = []string{SlotLeft, SlotRight}[rings]
			}
			rings++
		}
		ch, code, why := k.precheck(st, c.id, word, taken)
		if why != "" {
			out[c.pos].Code, out[c.pos].Why = code, why
			continue
		}
		taken[ch.slot] = ch.rec.Name
		ch.idx = c.pos
		picks = append(picks, ch)
	}
	if len(picks) == 0 {
		return out, true
	}
	worn := 0
	for _, ch := range picks {
		if _, occ := ch.wornName(ch.slot); occ {
			worn++
		}
	}
	if vaultRoom(st, len(picks)) < worn {
		for _, ch := range picks {
			out[ch.idx].Code, out[ch.idx].Why = "room", ReasonNoRoom
		}
		return out, true
	}
	order := "eq-set-" + orderTag(strings.Join(ids, ",")) + fmt.Sprintf("-%d", k.now().UnixMilli())
	var pieces []d2dctl.DirectPiece
	for _, ch := range picks {
		pieces = append(pieces, d2dctl.DirectPiece{Hex: ch.hex, Slot: ch.slot})
		k.putEq(&EquipOrder{ID: ch.id, Name: ch.rec.Name, Slot: SlotAuto, Group: []string{ch.slot}, State: EqEquipping, Direct: true})
	}
	rs, err := k.Direct.WearSet(ctx, pieces, order)
	if err != nil {
		if errors.Is(err, d2dctl.ErrDirectOff) {
			k.mu.Lock()
			for _, ch := range picks {
				delete(k.bookLocked().orders, ch.id)
			}
			k.dcapsOK = false
			k.mu.Unlock()
			return nil, false
		}
		code, why := directWhy(err)
		for _, ch := range picks {
			out[ch.idx].Code, out[ch.idx].Why = code, why
			k.failOrder(ch.id, code, why)
		}
		return out, true
	}
	for i, ch := range picks {
		r := rs[i]
		if !r.OK {
			code, why := directWhy(&d2dctl.DirectRefused{Code: r.Code, Why: r.Why})
			out[ch.idx].Code, out[ch.idx].Why = code, why
			k.failOrder(ch.id, code, why)
			continue
		}
		k.setOrder(ch.id, func(o *EquipOrder) { o.Applied, o.Removed, o.Worn, o.Unit = true, r.Removed, ch.slot, r.Unit })
		k.finishDirect(ctx, st, ch, r, &out[ch.idx])
	}
	return out, true
}

// repairDirect finishes the store step of an order the engine applied but the process left before the records were written
// (a restart), and ends an order whose engine answer never arrived. Called at the top of every Run.
func (k *Keeper) repairDirect(ctx context.Context) {
	st := k.store()
	if st == nil {
		return
	}
	k.mu.Lock()
	var todo []EquipOrder
	for _, o := range k.bookLocked().orders {
		if o.Direct && o.active() {
			todo = append(todo, *o)
		}
	}
	k.mu.Unlock()
	for _, o := range todo {
		if !o.Applied {
			k.failOrder(o.ID, "interrupted", "the bot restarted before the game answered: check the Inventory page")
			continue
		}
		var res Result
		k.finishDirect(ctx, st, chosen{id: o.ID, rec: vault.Rec{Item: vault.Item{Name: o.Name}}, slot: o.Worn}, d2dctl.DirectResult{Removed: o.Removed, Unit: o.Unit}, &res)
	}
}
