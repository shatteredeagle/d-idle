// Package invdesk is the bot side of the watch screen's inventory page (docs/inventory-ui.md): the player's sell and keep
// marks, equip requests and slot pins. It reads the bot's newest snapshot and never writes game memory; an equip goes
// through the Equipper (the leveler's normal click path) at a calm point chosen by the bot.
package invdesk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/status"
)

const (
	MarkSell = "sell"
	MarkKeep = "keep"
	// StateFile is the file name of the persisted state inside the bot's state directory.
	StateFile = "inventory-ui.json"
	maxRecent = 8
)

// SlotBody maps the slot keys of status.EquipSlots to the game's body locations.
var SlotBody = map[string]int{"head": d2.BodyHead, "neck": d2.BodyNeck, "torso": d2.BodyTorso, "rhand": d2.BodyRHand,
	"lhand": d2.BodyLHand, "rring": d2.BodyRRing, "lring": d2.BodyLRing, "belt": d2.BodyBelt, "feet": d2.BodyFeet, "gloves": d2.BodyGloves}

// Equipper checks and performs an equip (leveler.Leveler).
type Equipper interface {
	CheckPlayerEquip(s *d2.Snapshot, id uint32, body int) (string, error)
	PlayerEquip(ctx context.Context, id uint32, body int) error
}

// Config wires the desk to the bot.
type Config struct {
	Path      string                      // state file; empty: nothing is persisted
	Inventory func() status.InventoryData // the existing watch/status item snapshot
	Snap      func() *d2.Snapshot         // the runner's newest snapshot
	Why       func(u d2.Unit) string      // optional: the item goals engine's reason line (loot verdict)
	SellBlock func(u d2.Unit) string      // optional: why an item can never be sold ("" = it may be)
	Advice    func(it status.Item) string // optional hook for upgrade suggestions (card upgradeadvisor); nil: none
	Now       func() time.Time
}

// Desk holds the player's decisions. All methods are safe for concurrent use.
type Desk struct {
	c  Config
	mu sync.Mutex
	eq Equipper
	st state
}

type markRec struct {
	ID    uint32 `json:"id"`
	Print string `json:"print"` // class/quality/ilvl: a reused id of another item does not inherit the mark
	Mark  string `json:"mark"`
}

// Pin is a slot the player filled; the bot's own gear logic leaves it alone until the player unpins it. A pin has no
// expiry and no "clearly worse" override (user decision 2026-11-09).
type Pin struct {
	Slot  string `json:"slot"`
	ID    uint32 `json:"id"`
	Name  string `json:"name"`
	Since int64  `json:"since"`
}

// Request is a queued or finished equip request.
type Request struct {
	ID     uint32 `json:"id"`
	Slot   string `json:"slot"`
	Name   string `json:"name"`
	State  string `json:"state"` // queued, done, failed
	Reason string `json:"reason,omitempty"`
	At     int64  `json:"at"`
}

type state struct {
	Marks  []markRec `json:"marks"`
	Pins   []Pin     `json:"pins"`
	Queue  []Request `json:"-"`
	Recent []Request `json:"-"`
}

// New makes the desk and loads the state file (a missing file is an empty state; a damaged one is an error).
func New(c Config) (*Desk, error) {
	if c.Now == nil {
		c.Now = time.Now
	}
	d := &Desk{c: c}
	if c.Path == "" {
		return d, nil
	}
	b, err := os.ReadFile(c.Path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(b, &d.st); err != nil {
			return nil, fmt.Errorf("invdesk: %s: %w", c.Path, err)
		}
	}
	return d, nil
}

// SetEquipper installs the click path; until then equip requests are refused.
func (d *Desk) SetEquipper(e Equipper) { d.mu.Lock(); d.eq = e; d.mu.Unlock() }

// SetWhy installs the reason-line source after the looter exists.
func (d *Desk) SetWhy(f func(u d2.Unit) string) { d.mu.Lock(); d.c.Why = f; d.mu.Unlock() }

// SetSellBlock installs the never-sell test after the town exists.
func (d *Desk) SetSellBlock(f func(u d2.Unit) string) { d.mu.Lock(); d.c.SellBlock = f; d.mu.Unlock() }

func fingerprint(u d2.Unit) string {
	if u.Item == nil {
		return ""
	}
	return fmt.Sprintf("%d/%d/%d", u.Class, u.Item.Quality, u.Item.ILvl)
}

// MarkOf is town.ItemMarks: the player's mark for the unit, "" when none or when the id now names another item.
func (d *Desk) MarkOf(u d2.Unit) string {
	if d == nil {
		return ""
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, m := range d.st.Marks {
		if m.ID == u.ID && m.Print == fingerprint(u) {
			return m.Mark
		}
	}
	return ""
}

// Pinned says whether the player pinned the body slot (leveler.Options.Pinned, town.SetPinned). Nil desk: no.
func (d *Desk) Pinned(body int) bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, p := range d.st.Pins {
		if SlotBody[p.Slot] == body {
			return true
		}
	}
	return false
}

func (d *Desk) unit(id uint32) (d2.Unit, *d2.Snapshot, bool) {
	var s *d2.Snapshot
	if d.c.Snap != nil {
		s = d.c.Snap()
	}
	if s == nil || s.Inventory == nil {
		return d2.Unit{}, s, false
	}
	for _, u := range s.Inventory.Items {
		if u.ID == id && u.Item != nil {
			return u, s, true
		}
	}
	return d2.Unit{}, s, false
}

// Refusal is an error with a reason line for the player.
type Refusal struct{ Code, Reason string }

func (r *Refusal) Error() string { return r.Reason }

func refuse(code, f string, a ...any) error { return &Refusal{code, fmt.Sprintf(f, a...)} }

// SetMark sets, changes or (mark "") clears the player's mark on an item. "sell" needs the item in the backpack (the
// town sells from there only) and is refused for items that can never be sold.
func (d *Desk) SetMark(id uint32, mark string) error {
	if mark != "" && mark != MarkSell && mark != MarkKeep {
		return refuse("mark", "mark is %q, %q or empty", MarkSell, MarkKeep)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	u, _, ok := d.unit(id)
	if !ok {
		return refuse("item", "that item is not in the inventory any more")
	}
	if mark == MarkSell {
		if u.Item.Where != d2.ContInventory {
			return refuse("place", "only backpack items are sold: move it to the backpack first")
		}
		if d.c.SellBlock != nil {
			if why := d.c.SellBlock(u); why != "" {
				return refuse("never", "this item is never sold (%s)", why)
			}
		}
	}
	kept := d.st.Marks[:0]
	for _, m := range d.st.Marks {
		if m.ID != id {
			kept = append(kept, m)
		}
	}
	d.st.Marks = kept
	if mark != "" {
		d.st.Marks = append(d.st.Marks, markRec{ID: id, Print: fingerprint(u), Mark: mark})
	}
	return d.saveLocked()
}

// RequestEquip queues wearing backpack item id in the slot. Refused with a reason when the hero is not in town, the item
// is not in the backpack, the slot is unknown or the item's class, level or stat requirements are not met.
func (d *Desk) RequestEquip(id uint32, slot string) (Request, error) {
	body, ok := SlotBody[slot]
	if !ok {
		return Request{}, refuse("slot", "unknown slot %q", slot)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.eq == nil {
		return Request{}, refuse("off", "equipping is not available on this bot")
	}
	u, s, ok := d.unit(id)
	if !ok {
		return Request{}, refuse("item", "that item is not in the inventory any more")
	}
	if s.Player == nil || !s.Player.InTown() {
		return Request{}, refuse("town", "the hero is not in town: equip works in town only")
	}
	if u.Item.Where != d2.ContInventory {
		return Request{}, refuse("place", "only backpack items can be equipped: move it to the backpack first")
	}
	name, err := d.eq.CheckPlayerEquip(s, id, body)
	if err != nil {
		return Request{}, refuse("requirements", "%v", err)
	}
	r := Request{ID: id, Slot: slot, Name: name, State: "queued", At: d.c.Now().Unix()}
	q := d.st.Queue[:0]
	for _, x := range d.st.Queue {
		if x.Slot != slot && x.ID != id {
			q = append(q, x)
		}
	}
	d.st.Queue = append(q, r)
	return r, nil
}

// Unpin releases the slot to the bot's own gear logic. This is the only way a pin ends.
func (d *Desk) Unpin(slot string) error {
	if _, ok := SlotBody[slot]; !ok {
		return refuse("slot", "unknown slot %q", slot)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	kept := d.st.Pins[:0]
	found := false
	for _, p := range d.st.Pins {
		if p.Slot == slot {
			found = true
			continue
		}
		kept = append(kept, p)
	}
	if !found {
		return refuse("pin", "that slot is not pinned")
	}
	d.st.Pins = kept
	return d.saveLocked()
}

// Pending is the number of queued equip requests (bot.PlayerGear).
func (d *Desk) Pending() int {
	if d == nil {
		return 0
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.st.Queue)
}

// RunPending performs the queued requests, one at a time, through the Equipper (bot.PlayerGear). The caller runs it at a
// safe point (in town); a success pins the slot. It returns one line per request for the log.
func (d *Desk) RunPending(ctx context.Context) []string {
	d.mu.Lock()
	q := append([]Request(nil), d.st.Queue...)
	d.st.Queue = nil
	eq := d.eq
	d.mu.Unlock()
	var out []string
	for i, r := range q {
		if ctx.Err() != nil || eq == nil {
			d.mu.Lock()
			d.st.Queue = append(d.st.Queue, q[i:]...)
			d.mu.Unlock()
			break
		}
		err := eq.PlayerEquip(ctx, r.ID, SlotBody[r.Slot])
		d.mu.Lock()
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || isBusy(err) {
			d.st.Queue = append(d.st.Queue, q[i:]...) // not the request's fault: try again at the next calm point
			d.mu.Unlock()
			out = append(out, fmt.Sprintf("equip %s -> %s postponed: %v", r.Name, r.Slot, err))
			break
		}
		if err != nil {
			r.State, r.Reason = "failed", err.Error()
			out = append(out, fmt.Sprintf("equip %s -> %s failed: %v", r.Name, r.Slot, err))
		} else {
			r.State = "done"
			d.pinLocked(Pin{Slot: r.Slot, ID: r.ID, Name: r.Name, Since: d.c.Now().Unix()})
			out = append(out, fmt.Sprintf("equip %s -> %s done, slot pinned", r.Name, r.Slot))
		}
		d.st.Recent = append([]Request{r}, d.st.Recent...)
		if len(d.st.Recent) > maxRecent {
			d.st.Recent = d.st.Recent[:maxRecent]
		}
		_ = d.saveLocked()
		d.mu.Unlock()
	}
	return out
}

// isBusy matches leveler.ErrBusy without importing the leveler.
func isBusy(err error) bool {
	return err != nil && err.Error() == "leveler: a fight is on"
}

func (d *Desk) pinLocked(p Pin) {
	kept := d.st.Pins[:0]
	for _, x := range d.st.Pins {
		if x.Slot != p.Slot {
			kept = append(kept, x)
		}
	}
	d.st.Pins = append(kept, p)
}

func (d *Desk) saveLocked() error {
	if d.c.Path == "" {
		return nil
	}
	b, err := json.MarshalIndent(d.st, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(d.c.Path), 0o755); err != nil {
		return err
	}
	tmp := d.c.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, d.c.Path)
}

// ItemView is a status.Item with the player's mark and the reason lines.
type ItemView struct {
	status.Item
	Mark    string   `json:"mark,omitempty"`
	Why     string   `json:"why,omitempty"`
	CanSell bool     `json:"canSell,omitempty"`
	CanWear []string `json:"canWear,omitempty"` // backpack items: slots a request would be accepted for right now
	Advice  string   `json:"advice,omitempty"`  // upgrade suggestion hook (card upgradeadvisor); empty today
}

// ViewSchema is the version of the /api/inventory reply: 2 added the optional item fields tip, base and sell (the item
// tooltip, docs/inventory-tooltip.md). Version 1 readers keep working: nothing was renamed or removed.
const ViewSchema = 2

// View is the inventory page's data.
type View struct {
	Schema    int                  `json:"schema"`
	InTown    bool                 `json:"inTown"`
	Class     string               `json:"class,omitempty"`
	Gold      int                  `json:"gold"`
	StashGold int                  `json:"stashGold"`
	Equipment map[string]*ItemView `json:"equipment"`
	Backpack  []ItemView           `json:"backpack"`
	Stash     []ItemView           `json:"stash"`
	Belt      []ItemView           `json:"belt"`
	Cube      []ItemView           `json:"cube,omitempty"`
	Pins      []Pin                `json:"pins"`
	Queue     []Request            `json:"queue"`
	Recent    []Request            `json:"recent"`
	Note      string               `json:"note,omitempty"`
}

// View reads the snapshot and decorates it. Without a snapshot the lists are empty and Note says so.
func (d *Desk) View() View {
	d.mu.Lock()
	defer d.mu.Unlock()
	v := View{Schema: ViewSchema, Equipment: map[string]*ItemView{}, Pins: append([]Pin{}, d.st.Pins...), Queue: append([]Request{}, d.st.Queue...), Recent: append([]Request{}, d.st.Recent...)}
	sort.Slice(v.Pins, func(i, j int) bool { return v.Pins[i].Slot < v.Pins[j].Slot })
	if d.c.Inventory == nil {
		v.Note = "inventory is not available"
		return v
	}
	data := d.c.Inventory()
	var s *d2.Snapshot
	if d.c.Snap != nil {
		s = d.c.Snap()
	}
	if s != nil && s.Player != nil {
		v.InTown, v.Class = s.Player.InTown(), s.Player.Class.String()
	} else {
		v.Note = "waiting for the first look at the hero"
	}
	v.Gold, v.StashGold = data.Gold, data.StashGold
	decorate := func(it status.Item, place string) ItemView {
		iv := ItemView{Item: it}
		u, _, ok := d.unit(it.ID)
		if !ok {
			return iv
		}
		for _, m := range d.st.Marks {
			if m.ID == it.ID && m.Print == fingerprint(u) {
				iv.Mark = m.Mark
			}
		}
		if d.c.Why != nil {
			iv.Why = d.c.Why(u)
		}
		if d.c.Advice != nil {
			iv.Advice = d.c.Advice(it)
		}
		if place == "pack" {
			iv.CanSell = d.c.SellBlock == nil || d.c.SellBlock(u) == ""
			if d.eq != nil && s != nil {
				for _, k := range status.EquipSlots {
					if _, err := d.eq.CheckPlayerEquip(s, it.ID, SlotBody[k]); err == nil {
						iv.CanWear = append(iv.CanWear, k)
					}
				}
			}
		}
		return iv
	}
	for k, it := range data.Equipment {
		if it != nil {
			iv := decorate(*it, "worn")
			v.Equipment[k] = &iv
		}
	}
	for _, it := range data.Inventory {
		v.Backpack = append(v.Backpack, decorate(it, "pack"))
	}
	for _, it := range data.Stash {
		v.Stash = append(v.Stash, decorate(it, "stash"))
	}
	for _, it := range data.Belt {
		v.Belt = append(v.Belt, decorate(it, "belt"))
	}
	for _, it := range data.Cube {
		v.Cube = append(v.Cube, decorate(it, "cube"))
	}
	return v
}

// Kind is the short machine code of the refusal (item, place, never, town, requirements, slot, pin, mark, off).
func (r *Refusal) Kind() string { return r.Code }
