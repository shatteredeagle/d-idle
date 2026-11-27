package vaultkeep

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
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
}

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

func (k *Keeper) carrySave() {
	if k.CarryPath == "" {
		return
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
		return
	}
	_ = os.MkdirAll(filepath.Dir(k.CarryPath), 0o755)
	tmp := k.CarryPath + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, k.CarryPath)
	}
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
	sort.Slice(out, func(i, j int) bool { return out[i].At.Before(out[j].At) || out[i].At.Equal(out[j].At) && out[i].Unit < out[j].Unit })
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
	rec, ok := st.Get(id)
	switch {
	case !ok:
		res.Code, res.Why = "missing", "no such item"
		return res
	case rec.State != vault.Recorded:
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
	cr, err := k.API.ItemCreate(ctx, createHex(rec), d2dctl.CreateOpts{Pack: true, Key: carryKey + id})
	if err != nil {
		res.Code, res.Why = carryErr(err)
		return res
	}
	k.Hand.Protect(uint32(cr.ID))
	k.cb.mu.Lock()
	k.carryLoad()
	k.cb.by[id] = &Carried{ID: id, Unit: cr.ID, Name: rec.Name, Code: base.Code, Cells: base.Cells(), At: k.now()}
	k.carrySave()
	k.cb.mu.Unlock()
	if err := st.Discard(id, "carried to the backpack"); err != nil {
		k.note("carry-"+id, "A carried charm is in your backpack and its vault record could not be retired: "+err.Error())
	}
	k.save(ctx, "Carry")
	res.OK, res.Where = true, "backpack"
	if k.Notice != nil {
		k.Notice("carry: " + rec.Name + " is in the backpack")
	}
	return res
}

// Stow takes carried charm id (the old record id) out of the backpack and records it in the vault again.
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
	k.cb.mu.Lock()
	k.carryLoad()
	c := k.cb.by[id]
	k.cb.mu.Unlock()
	if c == nil {
		res.Code, res.Why = "missing", "that charm is not carried"
		return res
	}
	if vaultRoom(st, 0) < 1 {
		res.Code, res.Why = "room", ReasonNoRoom
		return res
	}
	have, err := k.API.ItemFull(ctx, c.Unit)
	if err != nil {
		res.Code, res.Why = carryErr(err)
		return res
	}
	if len(have) == 0 { // sold, dropped or moved by the player: nothing to stow, forget it
		k.forgetCarried(id)
		res.Code, res.Why = "gone", "the charm is not in the game any more"
		return res
	}
	vr, err := k.API.ItemVault(ctx, c.Unit)
	if err != nil {
		res.Code, res.Why = carryErr(err)
		return res
	}
	_, where := k.recordRemoved(ctx, st, vr.Norm, c.Name)
	k.forgetCarried(id)
	k.save(ctx, "Stow")
	res.OK, res.Where = true, where
	if k.Notice != nil {
		k.Notice("stow: " + c.Name + " went to the " + where)
	}
	return res
}

func (k *Keeper) forgetCarried(id string) {
	k.cb.mu.Lock()
	delete(k.cb.by, id)
	k.carrySave()
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
