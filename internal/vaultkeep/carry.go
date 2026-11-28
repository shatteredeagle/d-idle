package vaultkeep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/charms"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Carry and Stow (user decision 2026-11-27, docs/charms.md): a charm works only from the backpack, and carrying one is the
// player's choice, never the bot's. Carry creates the charm in the backpack at the first free cell (the engine finds it),
// puts the keep mark on it (no chore sells, drops or stashes a marked charm) and retires its vault record; Stow takes it
// out of the pack again and records it as a vault record. Same player-ordered, one-call shape as the direct Equip: no town
// trip. The bot's own play never calls either (a measured run refuses both).

const (
	ReasonCarryNotCharm = "only charms are carried from the vault"
	ReasonCarryReveal   = "Reveal it first (Gold): an unidentified charm gives nothing"
	ReasonCarryRoom     = "no free cell in the backpack for it"
	ReasonCarryOff      = "carrying is off: the vault is off, this is a measured run, or the bot has no hand"
	carryKey            = keyPrefix + "carry-"
)

// Carried is one charm the player carries out of the vault. ID is the id its vault record had.
type Carried struct {
	ID    string    `json:"id"`
	Unit  int       `json:"unit"`
	Name  string    `json:"name"`
	Code  string    `json:"code"`
	Cells int       `json:"cells"`
	At    time.Time `json:"at"`
	// The journal (card fix1): State is carry-intent or stow-intent while a move is under way ("" = carried, settled); Hex
	// is the charm's bytes (norm), by which the unit is found again after a game restart (unit ids are not stable); Tag
	// marks the vault record a Stow wrote before it took the charm out of the pack.
	State string `json:"state,omitempty"`
	Hex   string `json:"hex,omitempty"`
	Tag   string `json:"tag,omitempty"`
	// Seen is how many hero items had these bytes when the intent was written (card fix1b): the repair compares the count,
	// so an identical charm elsewhere on the hero is not taken for the one that moved.
	Seen int `json:"seen,omitempty"`
}

const (
	carryIntent = "carry-intent"
	stowIntent  = "stow-intent"
)

type carryBook struct {
	mu     sync.Mutex
	loaded bool
	by     map[string]*Carried
}

func (k *Keeper) carryLoad() {
	if k.cb.loaded {
		return
	}
	k.cb.loaded, k.cb.by = true, map[string]*Carried{}
	if k.CarryPath == "" {
		return
	}
	if b, err := os.ReadFile(k.CarryPath); err == nil {
		var f struct {
			Carried []Carried `json:"carried"`
		}
		if json.Unmarshal(b, &f) == nil {
			for i := range f.Carried {
				c := f.Carried[i]
				k.cb.by[c.ID] = &c
			}
		}
	}
}

func (k *Keeper) carrySave() error {
	if k.CarryPath == "" {
		return nil
	}
	var f struct {
		Carried []Carried `json:"carried"`
	}
	for _, c := range k.cb.by {
		f.Carried = append(f.Carried, *c)
	}
	sort.Slice(f.Carried, func(i, j int) bool { return f.Carried[i].At.Before(f.Carried[j].At) })
	b, err := json.MarshalIndent(f, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(k.CarryPath), 0o755); err != nil {
		return err
	}
	tmp := k.CarryPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, k.CarryPath)
}

// carryPut writes one journal entry (nil: remove it) and reports whether it reached the file.
func (k *Keeper) carryPut(id string, c *Carried) error {
	k.cb.mu.Lock()
	defer k.cb.mu.Unlock()
	k.carryLoad()
	if c == nil {
		delete(k.cb.by, id)
	} else {
		cp := *c
		k.cb.by[id] = &cp
	}
	return k.carrySave()
}

// carriedUnits matches every carried entry to a hero unit by item BYTES in one pass (card fix1b, review N1). Unit ids do
// not survive a new game, so a stored id only breaks a tie between units of equal bytes and never excludes a unit. Each
// unit serves one entry; open carry-intents choose last. An entry without bytes (written before the journal) is matched
// by unit id and item code only. Not restricted to the backpack: the item list carries no location (U).
func (k *Keeper) carriedUnits(have []d2dctl.VaultRecord) map[string]int {
	list := k.CarriedList()
	sort.SliceStable(list, func(i, j int) bool { return list[i].State != carryIntent && list[j].State == carryIntent })
	out, used := map[string]int{}, map[int]bool{}
	for _, sameUnit := range []bool{true, false} {
		for _, c := range list {
			if out[c.ID] != 0 {
				continue
			}
			for _, h := range have {
				fits := h.UnitID == c.Unit && h.Code == c.Code
				if c.Hex != "" {
					fits = strings.EqualFold(h.Norm, c.Hex) && (!sameUnit || h.UnitID == c.Unit)
				}
				if fits && !used[h.UnitID] {
					out[c.ID], used[h.UnitID] = h.UnitID, true
					break
				}
			}
		}
	}
	return out
}

// refreshCarried writes the units found by bytes back into the settled entries (a new game renumbered them) and renews
// their keep mark.
func (k *Keeper) refreshCarried(units map[string]int) {
	for _, c := range k.CarriedList() {
		if u := units[c.ID]; c.State == "" && u != 0 && u != c.Unit {
			c.Unit = u
			k.Hand.Protect(uint32(u))
			_ = k.carryPut(c.ID, &c)
		}
	}
}

// countNorm is how many hero items have these bytes.
func countNorm(have []d2dctl.VaultRecord, hexNorm string) int {
	n := 0
	for _, h := range have {
		if hexNorm != "" && strings.EqualFold(h.Norm, hexNorm) {
			n++
		}
	}
	return n
}

func taggedRec(st *vault.Store, tag, hexNorm string) (string, bool) {
	for _, o := range st.Dump() {
		if tag != "" && o.State == vault.Recorded && o.Found.Area == tag && strings.EqualFold(o.Raw, hexNorm) {
			return o.ID, true
		}
	}
	return "", false
}

// repairCarry completes or rolls back the Carry and Stow moves a stop left half done, by looking at the hero. The caller
// holds actMu. A carry-intent whose charm is in the pack is completed (mark, record retired), else dropped (the record
// stays). A stow-intent whose charm is still in the pack is rolled back (its new record retired), else completed.
func (k *Keeper) repairCarry(ctx context.Context) {
	st := k.store()
	var open []Carried
	for _, c := range k.CarriedList() {
		if c.State != "" {
			open = append(open, c)
		}
	}
	if st == nil || len(open) == 0 || k.Hand == nil {
		return
	}
	have, err := k.API.ItemFull(ctx, -1)
	if err != nil {
		return // cannot look: the intents stay for the next pass
	}
	units := k.carriedUnits(have)
	k.refreshCarried(units)
	for _, c := range open {
		// A twin with equal bytes elsewhere on the hero must not pass for the charm: the count of such items before the
		// move (Seen) decides, the unit only says which one to mark.
		u, n := units[c.ID], countNorm(have, c.Hex)
		switch {
		case c.State == carryIntent && u != 0 && (c.Hex == "" || n > c.Seen):
			k.Hand.Protect(uint32(u))
			if r, ok := st.Get(c.ID); ok && r.State == vault.Recorded {
				if err := st.Discard(c.ID, "carried to the backpack"); err != nil {
					continue
				}
			}
			c.Unit, c.State = u, ""
			_ = k.carryPut(c.ID, &c)
		case c.State == carryIntent:
			_ = k.carryPut(c.ID, nil)
		case u != 0 && n >= c.Seen: // stow-intent, charm still carried
			if id, ok := taggedRec(st, c.Tag, c.Hex); ok {
				if err := st.Discard(id, "stow rolled back: the charm is still in the backpack"); err != nil {
					continue
				}
			}
			c.Unit, c.State, c.Tag = u, "", ""
			_ = k.carryPut(c.ID, &c)
		default:
			_ = k.carryPut(c.ID, nil)
		}
	}
}

// createAnswered: the game refused the create, so nothing was made.
func createAnswered(err error) bool {
	var vr *d2dctl.VaultRefused
	return errors.As(err, &vr) || errors.Is(err, d2dctl.ErrStashFull) || errors.Is(err, d2dctl.ErrVaultState) ||
		errors.Is(err, d2dctl.ErrVaultOff) || errors.Is(err, d2dctl.ErrVaultUnsupported) || errors.Is(err, d2dctl.ErrBusy) ||
		errors.Is(err, d2dctl.ErrBadHex) || errors.Is(err, d2dctl.ErrVaultVersion)
}

// CarriedList is the charms the player carries, oldest first.
func (k *Keeper) CarriedList() []Carried {
	k.cb.mu.Lock()
	defer k.cb.mu.Unlock()
	k.carryLoad()
	out := make([]Carried, 0, len(k.cb.by))
	for _, c := range k.cb.by {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At) || out[i].At.Equal(out[j].At) && out[i].Unit < out[j].Unit
	})
	return out
}

// CarryReady says whether Carry and Stow can be offered, and why not.
func (k *Keeper) CarryReady() (bool, string) {
	if k.Hand == nil || !k.enabled() || k.measured() {
		return false, ReasonCarryOff
	}
	if ok, _, why := k.directGate(); !ok {
		return false, why
	}
	return true, ""
}

// Carry moves the identified charm of vault record id into the backpack.
func (k *Keeper) Carry(ctx context.Context, id string) Result {
	res := Result{ID: id, Direct: true}
	if ok, why := k.CarryReady(); !ok {
		res.Code, res.Why = "off", why
		return res
	}
	st := k.store()
	if st == nil {
		res.Code, res.Why = "missing", "there is no vault for this hero yet"
		return res
	}
	k.actMu.Lock()
	defer k.actMu.Unlock()
	k.repairCarry(ctx)
	rec, ok := st.Get(id)
	switch {
	case !ok:
		res.Code, res.Why = "missing", "no such item"
		return res
	case rec.State != vault.Recorded || rec.Stage != "" || rec.Hold != "":
		res.Code, res.Why = "state", "the item is not in the vault any more or is busy ("+string(rec.State)+")"
		return res
	}
	base, isCharm := charms.Lookup(rec.Code)
	switch {
	case !isCharm || !charms.LegalQualityNum(rec.Quality):
		res.Code, res.Why = "kind", ReasonCarryNotCharm
		return res
	case !rec.Identified:
		res.Code, res.Why = "reveal", ReasonCarryReveal
		return res
	}
	for _, c := range k.CarriedList() {
		if c.ID == id {
			res.Code, res.Why = CodeInFlight, ReasonInFlight
			return res
		}
	}
	if by := k.heldBy(id); by != "" {
		res.Code, res.Why = CodeInFlight, ReasonHeld(by)
		return res
	}
	// The intent is on disk before the game is touched: a stop after the create is settled at the next start by looking.
	c := Carried{ID: id, Name: rec.Name, Code: base.Code, Cells: base.Cells(), At: k.now(), State: carryIntent, Hex: createHex(rec)}
	have, err := k.API.ItemFull(ctx, -1)
	if err != nil {
		res.Code, res.Why = carryErr(err)
		return res
	}
	c.Seen = countNorm(have, c.Hex)
	if err := k.carryPut(id, &c); err != nil {
		res.Code, res.Why = "journal", "the carry could not be written down first ("+err.Error()+"): nothing was moved"
		return res
	}
	cr, err := k.API.ItemCreate(ctx, c.Hex, d2dctl.CreateOpts{Pack: true, Key: carryKey + id})
	if err != nil {
		res.Code, res.Why = carryErr(err)
		if createAnswered(err) {
			_ = k.carryPut(id, nil)
		} else { // unanswered or "created already": the charm may be in the pack; the intent stays and the next pass looks
			k.repairCarry(ctx)
		}
		return res
	}
	k.Hand.Protect(uint32(cr.ID))
	c.Unit = cr.ID
	_ = k.carryPut(id, &c)
	if err := st.Discard(id, "carried to the backpack"); err != nil {
		// the intent stays: the next pass retires the record, and no Sell or Equip of it is offered as carried meanwhile
		k.note("carry-"+id, "A carried charm is in your backpack and its vault record could not be retired yet: "+err.Error())
		res.Code, res.Why = "record", "the charm is in your backpack; its vault record is retired at the next step"
		return res
	}
	c.State = ""
	if err := k.carryPut(id, &c); err != nil {
		k.note("carry-file", "The carried list could not be saved: "+err.Error())
	}
	k.save(ctx, "Carry")
	res.OK, res.Where = true, "backpack"
	if k.Notice != nil {
		k.Notice("carry: " + rec.Name + " is in the backpack")
	}
	return res
}

// Stow takes carried charm id (the old record id) out of the backpack and records it in the vault again. The vault record
// is written BEFORE the charm leaves the pack (the bytes are readable while it is there), and the unit is found by the
// charm's bytes, never by the stored id alone (a game restart renumbers units).
func (k *Keeper) Stow(ctx context.Context, id string) Result {
	res := Result{ID: id, Direct: true}
	if ok, why := k.CarryReady(); !ok {
		res.Code, res.Why = "off", why
		return res
	}
	st := k.store()
	if st == nil {
		res.Code, res.Why = "missing", "there is no vault for this hero yet"
		return res
	}
	k.actMu.Lock()
	defer k.actMu.Unlock()
	k.repairCarry(ctx)
	var c *Carried
	for _, x := range k.CarriedList() {
		if x.ID == id {
			c = &x
			break
		}
	}
	if c == nil {
		res.Code, res.Why = "missing", "that charm is not carried"
		return res
	}
	if c.State != "" {
		res.Code, res.Why = CodeInFlight, ReasonInFlight
		return res
	}
	if vaultRoom(st, 0) < 1 {
		res.Code, res.Why = "room", ReasonNoRoom
		return res
	}
	have, err := k.API.ItemFull(ctx, -1)
	if err != nil {
		res.Code, res.Why = carryErr(err)
		return res
	}
	units := k.carriedUnits(have)
	k.refreshCarried(units)
	unit := units[id]
	if unit == 0 { // sold, dropped or moved by the player: nothing to stow, forget it
		k.forgetCarried(id)
		res.Code, res.Why = "gone", "the charm is not in the game any more"
		return res
	}
	var vr d2dctl.VaultRecord
	for _, h := range have {
		if h.UnitID == unit {
			vr = h
		}
	}
	c.Seen = countNorm(have, vr.Norm)
	c.Unit, c.Hex, c.State, c.Tag = unit, vr.Norm, stowIntent, fmt.Sprintf("stow-%s-%d", orderTag(id), k.now().UnixMilli())
	if err := k.carryPut(id, c); err != nil {
		res.Code, res.Why = "journal", "the stow could not be written down first ("+err.Error()+"): nothing was moved"
		return res
	}
	settle := func() { c.State, c.Tag = "", ""; _ = k.carryPut(id, c) }
	_, _, filed := k.recordItem(ctx, st, vr, c.Name, c.Tag, true)
	if !filed {
		settle()
		res.Code, res.Why = "room", "the vault could not take the charm (it is full or holds an identical one): it stays in your backpack"
		return res
	}
	if _, err := k.API.ItemVault(ctx, unit); err != nil {
		res.Code, res.Why = carryErr(err)
		if createAnswered(err) { // refused: the charm is still carried, the new record goes again
			if rid, ok := taggedRec(st, c.Tag, vr.Norm); ok {
				_ = st.Discard(rid, "stow refused: the charm is still in the backpack")
			}
			settle()
		} else {
			k.repairCarry(ctx) // unanswered: settled by looking, now or at the next pass
		}
		return res
	}
	k.forgetCarried(id)
	k.save(ctx, "Stow")
	res.OK, res.Where = true, "vault"
	if k.Notice != nil {
		k.Notice("stow: " + c.Name + " went to the vault")
	}
	return res
}

func (k *Keeper) forgetCarried(id string) {
	k.cb.mu.Lock()
	delete(k.cb.by, id)
	_ = k.carrySave()
	k.cb.mu.Unlock()
}

func (k *Keeper) save(ctx context.Context, what string) {
	if k.Save == nil {
		return
	}
	if err := k.Save(ctx); err != nil {
		k.note("save", what+": "+ReasonSave+" ("+err.Error()+")")
	}
}

func carryErr(err error) (code, why string) {
	var vr *d2dctl.VaultRefused
	switch {
	case errors.Is(err, d2dctl.ErrStashFull):
		return "room", ReasonCarryRoom
	case errors.Is(err, d2dctl.ErrVaultState):
		return "dead", deadOrNoGame(err)
	case errors.Is(err, d2dctl.ErrVaultOff), errors.Is(err, d2dctl.ErrVaultUnsupported):
		return "verbs", ReasonVerbs
	case errors.Is(err, d2dctl.ErrBusy):
		return "busy", ReasonBusy
	case errors.Is(err, d2dctl.ErrVaultDup):
		return "dup", "that charm was created already: check your backpack"
	case errors.As(err, &vr):
		return "game", vr.Why
	}
	return "game", err.Error()
}
