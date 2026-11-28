package vaultkeep

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// The crash journal of Equip on the mercenary (card vault2; review 2026-11-28 finding F2). One intent per vault record,
// written before the first game step and removed when the move is settled.

const (
	mercCreate = "create" // the copy is about to be created in the backpack (it may or may not exist)
	mercGive   = "give"   // the copy exists (Unit); the give is about to be sent (it may or may not have run)
)

type mercIntent struct {
	ID    string    `json:"id"`    // vault record id
	Order string    `json:"order"` // ledger order id; the Found.Area of every record this order files
	Name  string    `json:"name"`
	Hex   string    `json:"hex"` // the piece's bytes (norm)
	Step  string    `json:"step"`
	Unit  int       `json:"unit,omitempty"`
	Pack  []string  `json:"pack"` // norm hex of every hero item before the create: what is new afterwards came off the mercenary
	At    time.Time `json:"at"`
}

type mercBook struct {
	mu     sync.Mutex
	loaded bool
	by     map[string]*mercIntent
}

// mercPath is the intent file, next to the equip orders ("" with no EquipPath: memory only).
func (k *Keeper) mercPath() string {
	if k.EquipPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(k.EquipPath), "merc-intents.json")
}

func (k *Keeper) mercLoad() {
	if k.mb.loaded {
		return
	}
	k.mb.loaded, k.mb.by = true, map[string]*mercIntent{}
	if p := k.mercPath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			var f struct {
				Intents []mercIntent `json:"intents"`
			}
			if json.Unmarshal(b, &f) == nil {
				for i := range f.Intents {
					k.mb.by[f.Intents[i].ID] = &f.Intents[i]
				}
			}
		}
	}
}

// mercPut writes one intent (nil: remove it), fsynced, and reports whether it reached the file.
func (k *Keeper) mercPut(id string, in *mercIntent) error {
	k.mb.mu.Lock()
	defer k.mb.mu.Unlock()
	k.mercLoad()
	if in == nil {
		delete(k.mb.by, id)
	} else {
		cp := *in
		k.mb.by[id] = &cp
	}
	p := k.mercPath()
	if p == "" {
		return nil
	}
	var f struct {
		Intents []mercIntent `json:"intents"`
	}
	for _, x := range k.mb.by {
		f.Intents = append(f.Intents, *x)
	}
	sort.Slice(f.Intents, func(i, j int) bool { return f.Intents[i].ID < f.Intents[j].ID })
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return writeDurable(p, b)
}

// writeDurable replaces path with b: temp file, fsync, rename.
func writeDurable(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// mercIntents lists the open merc orders, by record id.
func (k *Keeper) mercIntents() []mercIntent {
	k.mb.mu.Lock()
	defer k.mb.mu.Unlock()
	k.mercLoad()
	out := make([]mercIntent, 0, len(k.mb.by))
	for _, x := range k.mb.by {
		out = append(out, *x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (k *Keeper) mercOpen(id string) bool {
	for _, in := range k.mercIntents() {
		if in.ID == id {
			return true
		}
	}
	return false
}

func (k *Keeper) mercRetire(st *vault.Store, id, name string) {
	if r, ok := st.Get(id); ok && r.State == vault.Recorded {
		if err := st.Discard(id, "equipped on the mercenary (vault equip)"); err != nil {
			k.note("eqm-rec-"+id, "Equip on mercenary: the vault record of "+name+" could not be retired ("+err.Error()+").")
		}
	}
}

// mercFile files a piece that came off the mercenary and lies in the backpack: the vault record first (tagged with the
// order id), then the removal. open: the removal was not answered, so the piece may be in the pack and in the vault; the
// intent stays and the next pass looks (a repeated pass finds the tagged record and only removes).
func (k *Keeper) mercFile(ctx context.Context, st *vault.Store, vr d2dctl.VaultRecord, order string) (name, where string, open bool) {
	name, _, filed := k.recordItem(ctx, st, vr, "", order, true)
	if !filed {
		k.Hand.Protect(uint32(vr.UnitID))
		k.note("eqm-off-"+order, "The "+name+" the mercenary took off is in your backpack: the vault could not take it (it is full or holds an identical one).")
		return name, "backpack", false
	}
	if _, err := k.API.ItemVault(ctx, vr.UnitID); err != nil {
		if !createAnswered(err) {
			return name, "vault", true
		}
		if rid, ok := taggedRec(st, order, vr.Norm); ok {
			_ = st.Discard(rid, "the piece stayed in the backpack: the game refused to take it out")
		}
		k.Hand.Protect(uint32(vr.UnitID))
		k.note("eqm-off-"+order, "The "+name+" the mercenary took off is in your backpack: the vault could not take it ("+err.Error()+").")
		return name, "backpack", false
	}
	return name, "vault", false
}

type mercOutcome struct {
	given bool     // the mercenary took the piece
	open  bool     // not settled: the intent stays
	names []string // what came off
	where string
}

// repairMerc settles every open merc intent by looking at the hero and returns what it found, by record id. The caller
// holds actMu. It moves items (item vault), so a measured run leaves the intents for later.
func (k *Keeper) repairMerc(ctx context.Context) map[string]mercOutcome {
	open := k.mercIntents()
	st := k.store()
	if len(open) == 0 || st == nil || k.Hand == nil || k.measured() {
		return nil
	}
	out := map[string]mercOutcome{}
	for _, in := range open {
		have, err := k.API.ItemFull(ctx, -1)
		if err != nil {
			out[in.ID] = mercOutcome{open: true} // cannot look: the intent stays for the next pass
			continue
		}
		out[in.ID] = k.settleMerc(ctx, st, in, have)
	}
	return out
}

func (k *Keeper) settleMerc(ctx context.Context, st *vault.Store, in mercIntent, have []d2dctl.VaultRecord) mercOutcome {
	extra := map[string]int{} // items the hero holds now, minus those held before the create
	for _, h := range have {
		extra[strings.ToLower(h.Norm)]++
	}
	for _, n := range in.Pack {
		extra[n]--
	}
	hex := strings.ToLower(in.Hex)
	if extra[hex] > 0 { // the copy is still in the pack: not given. It goes back; the record stays in the vault.
		unit := 0
		for _, h := range have {
			if strings.EqualFold(h.Norm, hex) && (unit == 0 || h.UnitID == in.Unit) {
				unit = h.UnitID
			}
		}
		if _, err := k.API.ItemVault(ctx, unit); err != nil {
			if !createAnswered(err) {
				return mercOutcome{open: true}
			}
			k.Hand.Protect(uint32(unit))
			k.note("eqm-"+in.ID, "A copy of "+in.Name+" made for the mercenary is still in your backpack ("+err.Error()+"): drop it, the item is in the vault.")
		}
		_ = k.mercPut(in.ID, nil)
		return mercOutcome{}
	}
	if in.Step == mercCreate { // never made
		_ = k.mercPut(in.ID, nil)
		return mercOutcome{}
	}
	// The copy left the pack: the mercenary wears it. Whatever wearable piece is new on the hero came off him.
	o := mercOutcome{given: true, where: "vault"}
	k.mercRetire(st, in.ID, in.Name)
	mh, _ := k.Hand.(MercHand)
	for _, h := range have {
		n := strings.ToLower(h.Norm)
		if extra[n] <= 0 || mh == nil || !mh.MercPreview(vault.Rec{Item: vault.Item{Code: h.Code, Identified: true}}).Can {
			continue
		}
		extra[n]--
		name, where, open := k.mercFile(ctx, st, h, in.Order)
		o.names = append(o.names, name)
		if where != "vault" {
			o.where = where
		}
		o.open = o.open || open
	}
	if !o.open {
		_ = k.mercPut(in.ID, nil)
		k.save(ctx, "Equip on mercenary")
	}
	return o
}
