package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/invdesk"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/merc"
	"d2lod-bot/internal/vault"
	"d2lod-bot/internal/vaultkeep"
	"d2lod-bot/internal/watch"
)

// One-tap Equip of a vault item: the hero side (vaultkeep.Hand) over the inventory desk, the leveler's gear rule and the
// snapshot, and the page face (watch.VaultEquipper). Docs: docs/vault.md, section "Equip".

// equipOn is the knob loot.vaultEquip (never in a measured run).
func equipOn(e *env.Env) func() bool {
	return func() bool {
		return e != nil && e.Tune != nil && os.Getenv("D2BOT_RUN") == "" && e.Tune.Loot.VaultEquip != 0
	}
}

type equipHand struct {
	ref  *runnerRef
	e    *env.Env
	snap func() *d2.Snapshot
}

var slotKeyOf = func() map[int]string {
	m := map[int]string{}
	for k, b := range invdesk.SlotBody {
		m[b] = k
	}
	return m
}()

var slotLabel = map[string]string{"head": "helm", "neck": "amulet", "torso": "armour", "rhand": "right hand", "lhand": "left hand",
	"rring": "right ring", "lring": "left ring", "belt": "belt", "feet": "boots", "gloves": "gloves"}

// ringSide names the two ring slots with the words of the Inventory page ("left ring" is the game's lring).
func ringSide(slot string) string {
	switch slot {
	case "lring":
		return vaultkeep.SlotLeft
	case "rring":
		return vaultkeep.SlotRight
	}
	return ""
}

func (h equipHand) fresh() *d2.Snapshot {
	if h.e != nil {
		if s, err := h.e.Snapshot(); err == nil && s != nil {
			return s
		}
	}
	return h.snap()
}

// freshWith is fresh, waiting on the bot clock until the pack lists unit. The Keep's create answers before the game shows
// the item (under d2d's lockstep it appears a few steps later), and the verb replied with the id at once: F2 second cause
// (pgvault live: the id was 20, the dump listed 1..19). Gives up after about a second of game time and returns what it has.
func (h equipHand) freshWith(unit uint32) *d2.Snapshot {
	var s *d2.Snapshot
	for try := 0; try < 12; try++ {
		s = h.fresh()
		if s != nil && s.Inventory != nil {
			for _, u := range s.Inventory.Items {
				if u.ID == unit {
					return s
				}
			}
		}
		if h.e == nil || h.e.Clock == nil {
			break
		}
		h.e.Clock.Sleep(100 * time.Millisecond)
	}
	return s
}

// wornUnits maps slot key to the item unit worn there.
func wornUnits(s *d2.Snapshot) map[string]d2.Unit {
	out := map[string]d2.Unit{}
	if s == nil || s.Inventory == nil {
		return out
	}
	for _, u := range s.Inventory.Items {
		if u.Item != nil && u.Item.Where == d2.ContEquipment {
			if k, ok := slotKeyOf[u.Item.Body]; ok {
				out[k] = u
			}
		}
	}
	return out
}

func (h equipHand) baseIndex(code string) int {
	db := h.e.Data
	if db == nil {
		return -1
	}
	b := db.ItemByCode(code)
	if b == nil {
		return -1
	}
	for i, x := range db.Items {
		if x == b {
			return i
		}
	}
	return -1
}

func (h equipHand) score(s *d2.Snapshot, u d2.Unit) float64 {
	var st d2.Stats
	if u.Item != nil && u.Item.Identified() && h.e.Read != nil {
		st, _ = h.e.Read.UnitStats(u.Addr)
	}
	return leveler.Candidate(h.e.Data, s.Player, u, st).Score
}

// Preview judges the vault record for the hero with the gear rule (leveler.Candidate): the same slots, requirements and
// refusals the Inventory page's Wear uses, on a stand-in unit built from the record (class row, quality, level, flags). The
// stat-driven score of an affixed candidate is not readable from the record, so "upgrade" is answered for plain and
// superior items only (U: shown as unknown otherwise).
func (h equipHand) Preview(rec vault.Rec) vaultkeep.Preview {
	s := h.snap()
	db := h.e.Data
	if s == nil || s.Player == nil || db == nil {
		return vaultkeep.Preview{Why: "the hero is not readable right now"}
	}
	idx := h.baseIndex(rec.Code)
	if idx < 0 {
		return vaultkeep.Preview{Why: "this item is not in the game tables"}
	}
	b := db.Items[idx]
	// A player may wear an unidentified item (the affixes apply unseen): the candidate check counts it as identified, so an
	// unrevealed record is as wearable as a revealed one (user decision 2026-11-21: Equip does not need a reveal).
	flags := uint32(d2.FlagIdentified)
	if rec.Eth {
		flags |= d2.FlagEthereal
	}
	u := d2.Unit{Type: d2.TypeItem, Class: idx, Item: &d2.Item{Quality: rec.Quality, ILvl: rec.Ilvl, Flags: flags, Where: d2.ContInventory}}
	c := leveler.Candidate(db, s.Player, u, nil)
	if len(c.Slots) == 0 {
		why := "this item is not worn"
		if db.IsA(b.Type, "char") {
			why = "charms are not equipped: belt and backpack items are out of scope for Equip"
		}
		return vaultkeep.Preview{Why: why}
	}
	p := vaultkeep.Preview{Wearable: true, Merc: h.mercSide(s, b)}
	if c.Req.Lvl > 0 {
		p.Req = append(p.Req, fmt.Sprintf("level %d", c.Req.Lvl))
	}
	if c.Req.Str > 0 {
		p.Req = append(p.Req, fmt.Sprintf("strength %d", c.Req.Str))
	}
	if c.Req.Dex > 0 {
		p.Req = append(p.Req, fmt.Sprintf("dexterity %d", c.Req.Dex))
	}
	worn := wornUnits(s)
	var view map[string]*invdesk.ItemView
	if h.ref.inv != nil {
		view = h.ref.inv.View().Equipment
	}
	plain := rec.Quality <= d2.QualitySuperior
	for _, body := range c.Slots {
		key, ok := slotKeyOf[body]
		if !ok {
			continue
		}
		sv := vaultkeep.SlotView{Slot: key, Label: slotLabel[key], Side: ringSide(key)}
		sv.Can = c.Usable
		if !c.Usable {
			sv.Why = c.Why
		}
		if wu, has := worn[key]; has {
			w := &vaultkeep.Worn{Unit: wu.ID, Name: "item"}
			if tw := h.ref.tw.Load(); tw != nil {
				w.Name = tw.ItemName(wu)
			}
			if iv := view[key]; iv != nil {
				if iv.Name != "" {
					w.Name = iv.Name
				}
				w.Lines = iv.Props
				if len(w.Lines) > 4 {
					w.Lines = w.Lines[:4]
				}
			}
			w.Score = h.score(s, wu)
			sv.Worn = w
			if plain && sv.Can {
				up := c.Score > w.Score
				sv.Upgrade, sv.Gain = &up, c.Score-w.Score
			}
		}
		if sv.Can && c.TwoHanded && key == "rhand" {
			if _, shield := worn["lhand"]; shield {
				sv.Note = "takes the shield off (a two-hander uses both hands)"
			}
		}
		if sv.Can && (key == "lhand") {
			if o, two := worn["rhand"]; two && h.twoHanded(s, o) {
				sv.Can, sv.Why = false, "a two-handed weapon is worn: take it off first"
			}
		}
		p.Slots = append(p.Slots, sv)
	}
	sort.SliceStable(p.Slots, func(i, j int) bool { return p.Slots[i].Slot < p.Slots[j].Slot })
	return p
}

// mercSide answers the compare toggle's merc question with the merc gear rule (leveler.MercSlot). The kind comes from the
// merc unit in the snapshot, as the Inventory page's merc doll does (U: it is unknown while he is out of view).
func (h equipHand) mercSide(s *d2.Snapshot, b *data.ItemBase) *vaultkeep.MercSide {
	if s == nil || s.Merc == nil || !s.Merc.Hired {
		return &vaultkeep.MercSide{Why: "no mercenary is hired"}
	}
	fm := d2.FindMerc(s)
	m := &vaultkeep.MercSide{Owned: true, Kind: merc.KindName(fm.Class)}
	if m.Kind == "" {
		m.Why = "your mercenary is out of view, so his kind is not known yet"
		return m
	}
	key, ok := leveler.MercSlot(h.e.Data, fm.Class, b)
	switch {
	case ok:
		m.Can, m.Slot = true, key
	case key == "":
		m.Why = "a mercenary cannot wear this"
	default:
		m.Why = "a " + m.Kind + " cannot use that"
	}
	return m
}

// MercPreview is the merc rule alone for a vault record (vaultkeep.MercHand): Equip on merc does not need the hero to wear it.
func (h equipHand) MercPreview(rec vault.Rec) vaultkeep.MercSide {
	s := h.snap()
	idx := h.baseIndex(rec.Code)
	if idx < 0 || h.e.Data == nil {
		return vaultkeep.MercSide{Why: "this item is not in the game tables"}
	}
	if m := h.mercSide(s, h.e.Data.Items[idx]); m != nil {
		return *m
	}
	return vaultkeep.MercSide{Why: "no mercenary is hired"}
}

func (h equipHand) twoHanded(s *d2.Snapshot, u d2.Unit) bool {
	if h.e.Data == nil || u.Class < 0 || u.Class >= len(h.e.Data.Items) {
		return false
	}
	b := h.e.Data.Items[u.Class]
	return b.TwoHanded && b.Kind == data.KindWeapon
}

// Pick resolves the order's slot word for the created pack unit.
func (h equipHand) Pick(unit uint32, slot string) (string, error) {
	d := h.ref.inv
	if d == nil {
		return "", errors.New("equipping is not available on this bot")
	}
	fs := h.freshWith(unit)
	defer h.ref.holdSnap(fs)()
	fits, err := d.FitSlots(unit)
	if err != nil {
		return "", err
	}
	want := map[string]string{vaultkeep.SlotLeft: "lring", vaultkeep.SlotRight: "rring"}[slot]
	if want != "" {
		for _, f := range fits {
			if f == want {
				return want, nil
			}
		}
		return "", fmt.Errorf("it does not fit the %s ring slot", slot)
	}
	if len(fits) == 1 {
		return fits[0], nil
	}
	s := h.fresh()
	worn := wornUnits(s)
	best, bestScore := "", 0.0
	for _, f := range fits {
		wu, has := worn[f]
		if !has {
			return f, nil // an empty slot first
		}
		if sc := h.score(s, wu); best == "" || sc < bestScore {
			best, bestScore = f, sc
		}
	}
	return best, nil
}

// Wear equips through the desk (verified, pinned) and reports what left the body.
func (h equipHand) Wear(ctx context.Context, unit uint32, slot string) ([]vaultkeep.Worn, error) {
	d := h.ref.inv
	if d == nil {
		return nil, errors.New("equipping is not available on this bot")
	}
	fresh := h.freshWith(unit)
	defer h.ref.holdSnap(fresh)()
	before := wornUnits(fresh)
	names := map[uint32]string{}
	for _, u := range before {
		names[u.ID] = "item"
		if tw := h.ref.tw.Load(); tw != nil {
			names[u.ID] = tw.ItemName(u)
		}
	}
	if err := d.EquipNow(ctx, unit, slot); err != nil {
		return nil, err
	}
	after := wornUnits(h.fresh())
	still := map[uint32]bool{}
	for _, u := range after {
		still[u.ID] = true
	}
	var off []vaultkeep.Worn
	for _, u := range before {
		if !still[u.ID] {
			off = append(off, vaultkeep.Worn{Unit: u.ID, Name: names[u.ID]})
		}
	}
	return off, nil
}

func (h equipHand) WornUnit(slot string) uint32 {
	if u, ok := wornUnits(h.fresh())[slot]; ok {
		return u.ID
	}
	return 0
}

// Protect puts the keep mark on the pack unit so no chore sells or drops it. The snapshot may not list a created item yet:
// a few short retries.
func (h equipHand) Protect(unit uint32) {
	if h.ref.inv == nil {
		return
	}
	defer h.ref.holdSnap(h.freshWith(unit))()
	for try := 0; try < 6; try++ {
		err := h.ref.inv.SetMark(unit, invdesk.MarkKeep)
		var rf *invdesk.Refusal
		if err == nil || !errors.As(err, &rf) || rf.Code != "item" {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// StashRoom: the size of a worn item and the stash layout are not read before the wear, so the answer is no. An order whose
// Vault is over its capacity is therefore refused up front; the stash only catches an item the Vault turns away after the
// wear (Stash below, a backpack item the desk can check).
func (h equipHand) StashRoom(string) error {
	return errors.New("the stash room for a worn item is not read before the wear")
}

// Stash files the taken-off item (now in the backpack) in the stash through the desk's stash visit.
func (h equipHand) Stash(ctx context.Context, unit uint32) error {
	d := h.ref.inv
	if d == nil {
		return errors.New("the stash is not available on this bot")
	}
	defer h.ref.holdSnap(h.fresh())()
	return d.StashNow(ctx, unit)
}

var (
	_ vaultkeep.MercHand = equipHand{}
	_ vaultkeep.Hand     = equipHand{}
	_ vaultkeep.Stasher  = equipHand{}
)

// equipFace adapts the keeper of a vaultHome to watch.VaultEquipper.
type equipFace struct{ h *vaultHome }

func (f equipFace) k() *vaultkeep.Keeper {
	f.h.mu.Lock()
	defer f.h.mu.Unlock()
	return f.h.keep
}

// Direct says the one-tap Equip wears at once (the verbs are in this build and the knob is on).
func (f equipFace) Direct(ctx context.Context) bool {
	k := f.k()
	return k != nil && k.DirectReady(ctx)
}

// MercDirect says the page may offer Equip on merc (instant path ready and the build has item merc-equip).
func (f equipFace) MercDirect(ctx context.Context) bool {
	k := f.k()
	return k != nil && k.MercReady(ctx)
}

// EquipMerc is the merc twin of Enqueue: the item is given to the mercenary at once, or refused with a reason.
func (f equipFace) EquipMerc(ctx context.Context, id string) watch.VaultKeepResult {
	k := f.k()
	if k == nil {
		return watch.VaultKeepResult{ID: id, Code: "off", Why: vaultkeep.ReasonEqNone}
	}
	r, ok := k.EquipMerc(ctx, id)
	if !ok {
		return watch.VaultKeepResult{ID: id, Code: "off", Why: vaultkeep.ReasonMercOff}
	}
	return keepResult(r)
}

func (f equipFace) Gate(ctx context.Context) (bool, string, string) {
	k := f.k()
	if k == nil {
		return false, "off", vaultkeep.ReasonEqNone
	}
	return k.EquipGate(ctx)
}

// equipSheet is the reply of GET /api/vault/equip/<id>.
type equipSheet struct {
	Enabled bool   `json:"enabled"`
	Direct  bool   `json:"direct,omitempty"` // one tap wears at once (Equip v2)
	Off     string `json:"offWhy,omitempty"` // why Equip cannot be pressed now (knob off, dead hero ...)
	vaultkeep.Preview
	Order *vaultkeep.EquipOrder `json:"order,omitempty"`
	Line  string                `json:"line,omitempty"`
}

func (f equipFace) Preview(id string) (any, bool) {
	k := f.k()
	if k == nil {
		return nil, false
	}
	p, o, ok := k.EquipPreview(id)
	if !ok {
		return nil, false
	}
	v := equipSheet{Preview: p, Order: o}
	v.Enabled = true
	v.Direct = k.DirectReady(context.Background())
	if g, _, why := k.EquipGate(context.Background()); !g {
		v.Enabled, v.Off = false, why
	}
	if o != nil && o.State != vaultkeep.EqDone && o.State != vaultkeep.EqFailed && o.State != vaultkeep.EqCancelled {
		_, v.Line = k.EquipInfo()
	}
	if v.Slots == nil {
		v.Slots = []vaultkeep.SlotView{}
	}
	return v, true
}

func (f equipFace) Enqueue(ctx context.Context, id, slot string, now bool) watch.VaultKeepResult {
	k := f.k()
	if k == nil {
		return watch.VaultKeepResult{ID: id, Code: "off", Why: vaultkeep.ReasonEqNone}
	}
	r := k.EquipEnqueue(ctx, id, slot, now)
	return keepResult(r)
}

func (f equipFace) EnqueueSet(ctx context.Context, ids []string, now bool) []watch.VaultKeepResult {
	k := f.k()
	out := make([]watch.VaultKeepResult, 0, len(ids))
	if k == nil {
		for _, id := range ids {
			out = append(out, watch.VaultKeepResult{ID: id, Code: "off", Why: vaultkeep.ReasonEqNone})
		}
		return out
	}
	for _, r := range k.EquipSet(ctx, ids, now) {
		out = append(out, keepResult(r))
	}
	return out
}

func (f equipFace) Cancel(id string) watch.VaultKeepResult {
	k := f.k()
	if k == nil {
		return watch.VaultKeepResult{ID: id, Code: "off", Why: vaultkeep.ReasonEqNone}
	}
	r := k.EquipCancel(id)
	return keepResult(r)
}

func keepResult(r vaultkeep.Result) watch.VaultKeepResult {
	return watch.VaultKeepResult{ID: r.ID, OK: r.OK, Already: r.Already, Code: r.Code, Why: r.Why, Direct: r.Direct, Slot: r.Slot, Removed: r.Removed, Where: r.Where}
}
