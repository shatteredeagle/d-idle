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
	key  string // the order's key in the book ("" = id)
	tag  string // the order id: the mark of the records this order files
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
	case rec.State != vault.Recorded || rec.Hold != "":
		return c, "state", "the item is not in the vault any more or is busy (" + string(rec.State) + ")"
	case !rec.Identified:
		return c, "reveal", ReasonEqReveal
	}
	if by := k.heldBy(id); by != "" {
		return c, CodeInFlight, ReasonHeld(by)
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
	k.actMu.Lock()
	defer k.actMu.Unlock()
	k.journalOnce()
	k.repairDirectLocked(ctx) // an order a restart left open is settled before a new swap can displace its item
	if k.directOpen(id) {
		res.Code, res.Why = CodeInFlight, ReasonInFlight
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
	c.tag = order
	bu, bh := k.wornBefore(ctx, c.slot)
	ou, oh := k.wornBefore(ctx, otherHand[c.slot])
	k.putEq(&EquipOrder{ID: id, Name: c.rec.Name, Slot: slotWord, Group: []string{c.slot}, State: EqEquipping, Direct: true,
		Order: order, New: c.hex, BeforeUnit: bu, Before: bh, BeforeOffUnit: ou, BeforeOff: oh})
	dr, err := k.Direct.EquipDirect(ctx, c.hex, c.slot, order)
	if dr.OK { // the engine swapped: whatever failed after it, the displaced item is filed
		k.markApplied(id, c.slot, dr)
		if err != nil {
			k.note("equip-late-"+order, "Equip: the "+c.rec.Name+" is worn; a step after the swap failed ("+err.Error()+").")
		}
		k.finishDirect(ctx, st, c, dr, &res)
		return res, true
	}
	if errors.Is(err, d2dctl.ErrDirectOff) { // an older image: the order flow
		k.mu.Lock()
		delete(k.bookLocked().orders, id)
		k.dcapsOK = false
		k.mu.Unlock()
		return Result{}, false
	}
	if answered(err) {
		res.Code, res.Why = directWhy(err)
		k.failOrder(id, res.Code, res.Why)
		return res, true
	}
	// No usable answer: the swap may have happened. Never "nothing happened": look at the hero, else the order stays open.
	k.resolveDirect(ctx, st, id, &res)
	if res.Code == "" && !res.OK {
		res.Code, res.Why = directWhy(err)
	}
	return res, true
}

// CodeInFlight is the refusal of an order for a record that another order is still moving.
const (
	CodeInFlight   = "order_in_flight"
	ReasonInFlight = "an order for this item is still being settled: wait a moment"
	ReasonUnknown  = "the game did not answer; the order stays open until the bot can see what happened (check the Inventory page)"
)

// answered says the engine itself refused the call, so nothing moved.
func answered(err error) bool {
	var ref *d2dctl.DirectRefused
	return errors.As(err, &ref) || errors.Is(err, d2dctl.ErrDirectMeasured) || errors.Is(err, d2dctl.ErrBusy)
}

func (k *Keeper) journalOnce() {
	k.jonce.Do(func() {
		if j, ok := k.Direct.(d2dctl.ReplyJournaler); ok {
			j.JournalReplies(k.journalReply)
		}
	})
}

// journalReply writes the engine's answer into the order file before the direct client does anything else with it. A reply
// that did not parse is kept as text: it is the only copy of what came off.
func (k *Keeper) journalReply(order, raw string, res []d2dctl.DirectResult) {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, o := range k.bookLocked().orders {
		if !o.Direct || o.Order != order || !o.active() {
			continue
		}
		switch {
		case res == nil:
			o.Reply = raw
		case o.Piece < len(res) && res[o.Piece].OK:
			r := res[o.Piece]
			o.Applied, o.Removed, o.Unit = true, r.Removed, r.Unit
			if len(o.Group) > 0 {
				o.Worn = o.Group[0]
			}
		}
		o.Updated = k.now()
	}
	k.saveBookLocked()
}

func (k *Keeper) markApplied(key, slot string, dr d2dctl.DirectResult) {
	k.setOrder(key, func(o *EquipOrder) { o.Applied, o.Removed, o.Worn, o.Unit = true, dr.Removed, slot, dr.Unit })
}

// directOpen reports an unsettled direct order for the record (its own or one moved aside by its order id).
func (k *Keeper) directOpen(id string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, o := range k.bookLocked().orders {
		if o.ID == id && o.Direct && o.active() {
			return true
		}
	}
	return false
}

// otherHand pairs the two hand slots of the body (the same on every class).
var otherHand = map[string]string{"rhand": "lhand", "lhand": "rhand"}

// wornBefore reads what the slot holds before the swap (unit and item bytes), so an unanswered call can be judged by looking
// and the displaced item filed from this copy. 0 and "": the slot is empty. A unit with "": occupied but not readable (such
// an order is never judged, it stays open). U live.
func (k *Keeper) wornBefore(ctx context.Context, slot string) (int, string) {
	if slot == "" {
		return 0, ""
	}
	u := int(k.Hand.WornUnit(slot))
	if u == 0 {
		return 0, ""
	}
	have, err := k.API.ItemFull(ctx, u)
	if err != nil || len(have) != 1 {
		return u, ""
	}
	return u, have[0].Norm
}

// resolveDirect settles an order whose engine answer never arrived, by looking at item BYTES, never at unit ids (a new game
// or a game restart renumbers every unit; card fix1b, review N2). The slot holds the bytes read before the call: nothing
// moved. The slot holds the new item's bytes: the swap ran and the copy read before it is filed. Anything else (the look
// failed, another item, an old order without the new bytes) leaves the order open for the next pass. Equal bytes old and
// new count as "nothing moved": both verdicts leave one such item worn and one in the vault.
// U live: that the fork reports a worn item with the norm bytes it was created from.
func (k *Keeper) resolveDirect(ctx context.Context, st *vault.Store, key string, res *Result) {
	k.mu.Lock()
	op := k.bookLocked().orders[key]
	if op == nil {
		k.mu.Unlock()
		return
	}
	o := *op
	k.mu.Unlock()
	slot := o.Worn
	if slot == "" && len(o.Group) > 0 {
		slot = o.Group[0]
	}
	open := func() {
		k.note("equip-open-"+o.Order, "Equip: "+ReasonUnknown)
		res.Code, res.Why = CodeInFlight, ReasonUnknown
	}
	have, err := k.API.ItemFull(ctx, -1)
	if err != nil {
		open()
		return
	}
	// worn: the bytes in a body slot now ("" and true: empty; false: occupied and not readable).
	worn := func(s string) (string, bool) {
		u := int(k.Hand.WornUnit(s))
		if u == 0 {
			return "", true
		}
		for _, h := range have {
			if h.UnitID == u {
				return h.Norm, h.Norm != ""
			}
		}
		return "", false
	}
	now, ok := worn(slot)
	switch {
	case !ok, o.BeforeUnit != 0 && o.Before == "":
		open()
		return
	case o.Before != "" && strings.EqualFold(now, o.Before):
		k.failOrder(key, "equip", "the game did not answer and nothing changed: try again")
		res.Code, res.Why = "equip", "the game did not answer and nothing changed: try again"
		return
	case o.Before == "" && now == "":
		k.failOrder(key, "interrupted", "the game did not answer and the item is not worn: check the Inventory page")
		res.Code, res.Why = "interrupted", "the game did not answer: check the Inventory page"
		return
	case o.New == "" || !strings.EqualFold(now, o.New):
		open()
		return
	}
	var removed []string
	if o.Before != "" {
		removed = append(removed, o.Before)
	}
	// The other hand (a two-hander pushes the shield off as well): its bytes still in that slot means it stayed; gone from
	// the slot and from the whole hero means it left with the swap and the copy read before the call is the only one.
	// Off the body but still on the hero (or a twin of it) cannot be judged.
	if o.BeforeOffUnit != 0 {
		off, ok := worn(otherHand[slot])
		onHero := false
		for _, h := range have {
			onHero = onHero || (o.BeforeOff != "" && strings.EqualFold(h.Norm, o.BeforeOff))
		}
		switch {
		case ok && off != "" && (o.BeforeOff == "" || strings.EqualFold(off, o.BeforeOff)):
		case ok && o.BeforeOff != "" && !onHero:
			removed = append(removed, o.BeforeOff)
		default:
			open()
			return
		}
	}
	dr := d2dctl.DirectResult{OK: true, Slot: slot, Removed: removed}
	k.markApplied(key, slot, dr)
	k.finishDirect(ctx, st, chosen{key: key, id: o.ID, tag: o.Order, rec: vault.Rec{Item: vault.Item{Name: o.Name}}, slot: slot}, dr, res)
}

// finishDirect is the store step after the engine swapped: the equipped record leaves the vault, what came off becomes
// records ("previously worn", identified state kept), the character is saved. Idempotent (repairDirect re-runs it).
func (k *Keeper) finishDirect(ctx context.Context, st *vault.Store, c chosen, dr d2dctl.DirectResult, res *Result) {
	if c.key == "" {
		c.key = c.id
	}
	if r, ok := st.Get(c.id); ok && r.State == vault.Recorded && c.key == c.id {
		if err := st.Discard(c.id, "equipped directly (vault equip v2)"); err != nil {
			k.note("equip-rec-"+c.id, "Equip: the "+c.rec.Name+" is worn but its vault record could not be retired ("+err.Error()+").")
		}
	}
	quiet := k.measured() // a repair during a measured run files records only: no create, no save (F12)
	where := "vault"
	var names, left []string
	for i, h := range dr.Removed {
		name, _ := c.wornName(c.slot)
		if i > 0 || name == "" {
			name = ""
		}
		n, w, ok := k.recordRemoved(ctx, st, h, name, c.tag, quiet)
		if !ok {
			left = append(left, h)
			continue
		}
		names = append(names, n)
		switch {
		case w == "backpack" || where == "backpack":
			where = "backpack"
		case w == "stash":
			where = "stash"
		}
	}
	if len(left) > 0 { // never dropped: the bytes stay in the order file and the next pass files them
		why := "what you took off could not be filed yet; its data is kept in the order and the bot tries again"
		k.setOrder(c.key, func(o *EquipOrder) { o.Removed, o.Code, o.Reason = left, "unfiled", why })
		k.note("equip-unfiled-"+c.tag, "Equip: "+c.rec.Name+" is worn, but "+why+".")
		res.Code, res.Why, res.Slot = "unfiled", why, c.slot
		return
	}
	if k.Pinner != nil {
		k.Pinner(c.slot, uint32(dr.Unit), c.rec.Name)
	}
	if k.Save != nil && !quiet {
		if err := k.Save(ctx); err != nil {
			k.note("save", "Equip: "+ReasonSave+" ("+err.Error()+")")
		}
	}
	k.setOrder(c.key, func(o *EquipOrder) {
		o.State, o.Displaced, o.Where, o.Removed, o.Code, o.Reason = EqDone, names, where, nil, "", ""
	})
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
// into the stash, else the backpack, so nothing is lost; the order says where it is. ok false: the item has no home yet
// and the caller keeps its bytes. tag marks the record as this order's, so a second pass knows it filed it. quiet: no
// game write (a measured run).
func (k *Keeper) recordRemoved(ctx context.Context, st *vault.Store, hexRaw, name, tag string, quiet bool) (string, string, bool) {
	dec, _ := k.API.(d2dctl.Decoder)
	var r d2dctl.VaultRecord
	var err error
	if dec != nil {
		r, err = dec.ItemDecode(ctx, hexRaw)
	} else {
		err = errors.New("no decoder")
	}
	if err != nil {
		w, ok := k.createBack(ctx, hexRaw, tag, quiet)
		if ok {
			k.note("equip-"+hexRaw[:min(8, len(hexRaw))], "An item you replaced could not be read for the vault ("+err.Error()+"); it is in your "+w+".")
		}
		return "an item", w, ok
	}
	return k.recordItem(ctx, st, r, name, tag, quiet)
}

// recordItem files a decoded item as a "previously worn" vault record (the vault full: the stash, else the backpack).
// An identical live record is NOT taken as "filed" (d2d items have no unique id, two real items can have equal bytes):
// only a record this same order wrote counts; a real twin goes back to the hero and the player is told.
func (k *Keeper) recordItem(ctx context.Context, st *vault.Store, r d2dctl.VaultRecord, name, tag string, quiet bool) (string, string, bool) {
	if name == "" && k.Namer != nil {
		name = k.Namer(r, vault.Rec{})
	}
	if name == "" {
		name = r.Code
	}
	it := vault.Item{Raw: r.Norm, Code: r.Code, Name: name, Quality: r.Quality, Ilvl: r.ILvl, Identified: true, Props: r.Props,
		Sockets: r.Sockets, Eth: r.Ethereal, Found: vault.Found{Time: k.now().UTC(), Zone: "previously worn", Area: tag}}
	it.Pic, it.Tint, it.FileIndex = r.Pic, r.Tint, r.FileIndex()
	if k.Chain != nil {
		it.Chain = k.Chain(r.Code)
	}
	_, err := st.Add(it)
	if err == nil {
		return name, "vault", true
	}
	if errors.Is(err, vault.ErrDuplicate) && tag != "" {
		for _, o := range st.Dump() {
			if o.State == vault.Recorded && o.Found.Area == tag && strings.EqualFold(o.Raw, r.Norm) {
				return name, "vault", true // this order filed it on an earlier pass
			}
		}
	}
	w, ok := k.createBack(ctx, r.Norm, tag, quiet)
	if ok {
		text := "The " + name + " you replaced is in your " + w + ": the vault could not take it (" + err.Error() + ")."
		if errors.Is(err, vault.ErrDuplicate) {
			text = "The " + name + " you took off is in your " + w + ": an identical item is already in the vault, so it was not filed as a second record."
		}
		k.note("equip-"+name+tag, text)
	}
	return name, w, ok
}

// createBack puts an item that has no vault record back on the hero: the stash, else the backpack under the keep mark.
// The create carries a key (order and bytes), so a repeated pass does not make a second copy. ok false: not placed.
func (k *Keeper) createBack(ctx context.Context, hexNorm, tag string, quiet bool) (string, bool) {
	if quiet {
		return "", false
	}
	key := keyPrefix + "rb-" + orderTag(tag+hexNorm)
	_, err := k.API.ItemCreate(ctx, hexNorm, d2dctl.CreateOpts{Key: key})
	if err == nil || errors.Is(err, d2dctl.ErrVaultDup) {
		return "stash", true
	}
	r, err := k.API.ItemCreate(ctx, hexNorm, d2dctl.CreateOpts{Pack: true, Key: key + "p"})
	if err == nil {
		k.Hand.Protect(uint32(r.ID))
		return "backpack", true
	}
	if errors.Is(err, d2dctl.ErrVaultDup) {
		return "backpack", true
	}
	return "", false
}

func (k *Keeper) putEq(o *EquipOrder) {
	k.mu.Lock()
	defer k.mu.Unlock()
	o.At, o.Updated = k.now(), k.now()
	k.bookLocked().put(o)
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
	k.actMu.Lock()
	defer k.actMu.Unlock()
	k.journalOnce()
	k.repairDirectLocked(ctx)
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
		if k.directOpen(c.id) {
			code, why = CodeInFlight, ReasonInFlight
		}
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
	for i := range picks {
		ch := &picks[i]
		ch.tag = order
		pieces = append(pieces, d2dctl.DirectPiece{Hex: ch.hex, Slot: ch.slot})
		bu, bh := k.wornBefore(ctx, ch.slot)
		ou, oh := k.wornBefore(ctx, otherHand[ch.slot])
		k.putEq(&EquipOrder{ID: ch.id, Name: ch.rec.Name, Slot: SlotAuto, Group: []string{ch.slot}, State: EqEquipping, Direct: true,
			Order: order, Piece: i, New: ch.hex, BeforeUnit: bu, Before: bh, BeforeOffUnit: ou, BeforeOff: oh})
	}
	rs, err := k.Direct.WearSet(ctx, pieces, order)
	if errors.Is(err, d2dctl.ErrDirectOff) {
		k.mu.Lock()
		for _, ch := range picks {
			delete(k.bookLocked().orders, ch.id)
		}
		k.dcapsOK = false
		k.mu.Unlock()
		return nil, false
	}
	if err != nil && len(rs) > 0 {
		k.note("equip-late-"+order, "Wear set: the pieces are worn; a step after the swap failed ("+err.Error()+").")
	}
	for i, ch := range picks {
		switch {
		case i < len(rs) && rs[i].OK: // worn: filed whatever failed after the engine answered
			k.markApplied(ch.id, ch.slot, rs[i])
			k.finishDirect(ctx, st, ch, rs[i], &out[ch.idx])
		case i < len(rs):
			code, why := directWhy(&d2dctl.DirectRefused{Code: rs[i].Code, Why: rs[i].Why})
			out[ch.idx].Code, out[ch.idx].Why = code, why
			k.failOrder(ch.id, code, why)
		case answered(err):
			code, why := directWhy(err)
			out[ch.idx].Code, out[ch.idx].Why = code, why
			k.failOrder(ch.id, code, why)
		default: // no answer for this piece: judged by looking, else it stays open
			k.resolveDirect(ctx, st, ch.id, &out[ch.idx])
			if out[ch.idx].Code == "" && !out[ch.idx].OK {
				out[ch.idx].Code, out[ch.idx].Why = "equip", ReasonUnknown
			}
		}
	}
	return out, true
}

// repairDirect settles the direct orders a restart or a failed step left open: an applied one gets its displaced items
// filed (from the order file), an unanswered one is judged by looking at the hero. Called at the top of every Run and before
// every new direct order. In a measured run it only files records (finishDirect, quiet).
func (k *Keeper) repairDirect(ctx context.Context) {
	k.actMu.Lock()
	defer k.actMu.Unlock()
	k.repairDirectLocked(ctx)
	k.repairCarry(ctx)
	k.repairMerc(ctx)
}

func (k *Keeper) repairDirectLocked(ctx context.Context) {
	st := k.store()
	if st == nil {
		return
	}
	k.mu.Lock()
	todo := map[string]EquipOrder{}
	for key, o := range k.bookLocked().orders {
		if o.Direct && o.active() {
			todo[key] = *o
		}
	}
	k.mu.Unlock()
	for key, o := range todo {
		var res Result
		switch {
		case o.Applied:
			k.finishDirect(ctx, st, chosen{key: key, id: o.ID, tag: o.Order, rec: vault.Rec{Item: vault.Item{Name: o.Name}}, slot: o.Worn},
				d2dctl.DirectResult{Removed: o.Removed, Unit: o.Unit}, &res)
		case o.Reply != "": // an answer the bot could not read: the text is the only copy, it is never ended or overwritten
			k.note("equip-reply-"+o.Order, "Equip: the game's answer for "+o.Name+" could not be read; it is kept in the order file (order "+o.Order+"). Check the Inventory page.")
		case k.Hand == nil:
		default:
			k.resolveDirect(ctx, st, key, &res)
		}
	}
}
