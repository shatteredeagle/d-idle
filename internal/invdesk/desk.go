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
	"strings"
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
	// MaxQueue bounds the pending (queued or doing) player actions; a request beyond it is refused with code "full".
	MaxQueue = 8
	// Request kinds. An empty Kind (state files and clients of API schema 2) means equip.
	KindEquip = "equip"
	KindTrash = "trash"
	// Request states: queued -> doing -> done | failed; cancelled only from queued.
	StateQueued, StateDoing, StateDone, StateFailed, StateCancelled = "queued", "doing", "done", "failed", "cancelled"
	// TrashTTL is how long a trashed item stays on the looter's ignore list (a dropped item lies in the area until the
	// game ends; two hours covers a long area visit, and the list is capped at maxTrashed).
	TrashTTL   = 2 * time.Hour
	maxTrashed = 32
	trashNear  = 12 // subtiles: the fallback match of an item with the same fingerprint near the drop point
)

// SlotBody maps the slot keys of status.EquipSlots to the game's body locations.
var SlotBody = map[string]int{"head": d2.BodyHead, "neck": d2.BodyNeck, "torso": d2.BodyTorso, "rhand": d2.BodyRHand,
	"lhand": d2.BodyLHand, "rring": d2.BodyRRing, "lring": d2.BodyLRing, "belt": d2.BodyBelt, "feet": d2.BodyFeet, "gloves": d2.BodyGloves}

// Equipper checks and performs an equip (leveler.Leveler).
type Equipper interface {
	CheckPlayerEquip(s *d2.Snapshot, id uint32, body int) (string, error)
	PlayerEquip(ctx context.Context, id uint32, body int) error
}

// Trasher checks and performs a trash: pick the backpack item up and put it on the ground (leveler.Leveler).
type Trasher interface {
	CheckPlayerTrash(s *d2.Snapshot, id uint32) (string, error)
	PlayerTrash(ctx context.Context, id uint32) error
}

// Config wires the desk to the bot.
type Config struct {
	Path      string                                      // state file; empty: nothing is persisted
	Inventory func() status.InventoryData                 // the existing watch/status item snapshot
	Snap      func() *d2.Snapshot                         // the runner's newest snapshot
	Why       func(u d2.Unit) string                      // optional: the item goals engine's reason line (loot verdict)
	WhyFor    func(s *d2.Snapshot) func(u d2.Unit) string // optional, wins over Why: binds the reason source to ONE snapshot, built once per View
	SellBlock func(u d2.Unit) string                      // optional: why an item can never be sold ("" = it may be)
	// TrashBlock is the same for trash (quest items, keys, tomes, the start kit); nil: nothing is blocked here.
	TrashBlock func(u d2.Unit) string
	Advice     func(it status.Item) string // optional hook for upgrade suggestions (card upgradeadvisor); nil: none
	Prompt     func() string               // optional: the gear-policy prompt for the page (onboarding, no weapon); must not call back into the desk
	Now        func() time.Time
}

// Desk holds the player's decisions. All methods are safe for concurrent use.
type Desk struct {
	c  Config
	mu sync.Mutex
	eq Equipper
	tr Trasher
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

// Request is a queued or finished player action on one item (API schema 3 added kind and print; a request without a
// kind is an equip).
type Request struct {
	ID   uint32 `json:"id"`
	Kind string `json:"kind,omitempty"` // equip (default) or trash
	// Target is who acts or is dressed: "" or "hero" today. "merc" is reserved for the mercenary stage (docs/research/merc-gear.md);
	// the watch route refuses any other value with code "target" until that stage exists.
	Target string `json:"target,omitempty"`
	Slot   string `json:"slot,omitempty"` // equip only
	Name   string `json:"name"`
	State  string `json:"state"` // queued, doing, done, failed, cancelled
	Reason string `json:"reason,omitempty"`
	At     int64  `json:"at"`
	Print  string `json:"print,omitempty"` // fingerprint of the item when it was queued (trash: feeds the ignore list)
}

func (r Request) kind() string {
	if r.Kind == "" {
		return KindEquip
	}
	return r.Kind
}

// trashRec is an item the player trashed: the looter skips it on the ground so the bot does not pick it up again.
type trashRec struct {
	ID    uint32 `json:"id"`
	Print string `json:"print"`
	At    int64  `json:"at"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}

type state struct {
	Marks   []markRec  `json:"marks"`
	Pins    []Pin      `json:"pins"`
	Queue   []Request  `json:"queue,omitempty"` // persisted since schema 3; a "doing" entry found at load failed with the restart
	Recent  []Request  `json:"recent,omitempty"`
	Trashed []trashRec `json:"trashed,omitempty"`
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
		d.recover()
	}
	return d, nil
}

// recover settles what a restart interrupted: a request that was "doing" may or may not have moved the item, so it is
// reported failed and never retried; the queue is cut to MaxQueue.
func (d *Desk) recover() {
	var q []Request
	for _, r := range d.st.Queue {
		if r.State == StateDoing {
			r.State, r.Reason = StateFailed, "interrupted by a restart; check the item and ask again"
			d.st.Recent = append([]Request{r}, d.st.Recent...)
			continue
		}
		if len(q) < MaxQueue {
			q = append(q, r)
		}
	}
	d.st.Queue = q
	if len(d.st.Recent) > maxRecent {
		d.st.Recent = d.st.Recent[:maxRecent]
	}
}

// SetTrasher installs the drop path; until then trash requests are refused.
func (d *Desk) SetTrasher(t Trasher) { d.mu.Lock(); d.tr = t; d.mu.Unlock() }

// SetTrashBlock installs the never-trash test (quest items and the like) after the town exists.
func (d *Desk) SetTrashBlock(f func(u d2.Unit) string) {
	d.mu.Lock()
	d.c.TrashBlock = f
	d.mu.Unlock()
}

// SetEquipper installs the click path; until then equip requests are refused.
func (d *Desk) SetEquipper(e Equipper) { d.mu.Lock(); d.eq = e; d.mu.Unlock() }

// SetWhy installs the reason-line source after the looter exists.
func (d *Desk) SetWhy(f func(u d2.Unit) string) { d.mu.Lock(); d.c.Why = f; d.mu.Unlock() }

// SetWhyFor installs a reason source that is bound to one snapshot per View. Use it when the verdict needs a game
// read: Why alone would read the game once per item on every page poll (TASK-268: about 30 reads, 1 s on d2d).
func (d *Desk) SetWhyFor(f func(s *d2.Snapshot) func(u d2.Unit) string) {
	d.mu.Lock()
	d.c.WhyFor = f
	d.mu.Unlock()
}

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
	// Callbacks (Snap, SellBlock) run unlocked: they can reach Pinned/MarkOf through the town and the leveler, and d.mu
	// is not reentrant (live deadlock 2026-11-10: the whole bot froze).
	u, _, ok := d.unit(id)
	if !ok {
		return refuse("item", "that item is not in the inventory any more")
	}
	if mark == MarkSell {
		if u.Item.Where != d2.ContInventory {
			return refuse("place", "only backpack items are sold: move it to the backpack first")
		}
		d.mu.Lock()
		sb := d.c.SellBlock
		d.mu.Unlock()
		if sb != nil {
			if why := sb(u); why != "" {
				return refuse("never", "this item is never sold (%s)", why)
			}
		}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
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

// RequestEquip queues wearing backpack item id. An empty slot lets the desk choose: the only slot that takes the item,
// else the empty one (a ring goes to a free ring finger); with two worn candidates the caller must name one (code
// "choose"). Refused with a reason when the item is not in the backpack, the slot is unknown or the requirements are
// not met. The hero need not be in town: the bot performs it at an open opportunity (docs/inventory-actions.md).
func (d *Desk) RequestEquip(id uint32, slot string) (Request, error) {
	d.mu.Lock()
	eq := d.eq
	d.mu.Unlock()
	if eq == nil {
		return Request{}, refuse("off", "equipping is not available on this bot")
	}
	u, s, ok := d.unit(id)
	if !ok {
		return Request{}, refuse("item", "that item is not in the inventory any more")
	}
	if u.Item.Where != d2.ContInventory {
		return Request{}, refuse("place", "only backpack items can be equipped: move it to the backpack first")
	}
	if slot == "" {
		var err error
		if slot, err = autoSlot(eq, s, id); err != nil {
			return Request{}, err
		}
	}
	body, ok := SlotBody[slot]
	if !ok {
		return Request{}, refuse("slot", "unknown slot %q", slot)
	}
	name, err := eq.CheckPlayerEquip(s, id, body) // unlocked, see SetMark
	if err != nil {
		return Request{}, refuse("requirements", "%v", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.queuedLocked(id, KindTrash) {
		return Request{}, refuse("trash", "that item is queued to be trashed: cancel that first")
	}
	r := Request{ID: id, Kind: KindEquip, Slot: slot, Name: name, State: StateQueued, At: d.c.Now().Unix(), Print: fingerprint(u)}
	q := d.st.Queue[:0:0]
	for _, x := range d.st.Queue {
		if x.State == StateDoing || x.kind() != KindEquip || x.Slot != slot && x.ID != id {
			q = append(q, x)
		}
	}
	if len(q) >= MaxQueue {
		return Request{}, refuse("full", "%d actions are already waiting: cancel one first", MaxQueue)
	}
	d.st.Queue = append(q, r)
	return r, d.saveLocked()
}

// autoSlot picks the slot for an equip request without a slot (see RequestEquip).
func autoSlot(eq Equipper, s *d2.Snapshot, id uint32) (string, error) {
	var fits []string
	var why error
	for _, k := range status.EquipSlots {
		_, err := eq.CheckPlayerEquip(s, id, SlotBody[k])
		switch {
		case err == nil:
			fits = append(fits, k)
		case why == nil || !isNoSlot(err):
			why = err
		}
	}
	switch len(fits) {
	case 0:
		if why == nil {
			why = errors.New("it cannot be worn")
		}
		return "", refuse("requirements", "%v", why)
	case 1:
		return fits[0], nil
	}
	worn := map[int]bool{}
	for _, u := range s.Inventory.Items {
		if u.Item != nil && u.Item.Where == d2.ContEquipment {
			worn[u.Item.Body] = true
		}
	}
	for _, k := range fits {
		if !worn[SlotBody[k]] {
			return k, nil
		}
	}
	return "", refuse("choose", "it fits %v and all are worn: name the slot", fits)
}

// isNoSlot matches the "does not fit that slot" refusal (leveler.ErrNoSlot) without importing the leveler.
func isNoSlot(err error) bool {
	return err != nil && strings.Contains(err.Error(), "does not fit that slot")
}

func (d *Desk) queuedLocked(id uint32, kind string) bool {
	for _, x := range d.st.Queue {
		if x.ID == id && x.kind() == kind {
			return true
		}
	}
	return false
}

// TrashWhy says why the item cannot be trashed ("" = it can), with the refusal code. Only backpack items are dropped
// (the pack is the one panel the click path opens for a take); equipped items must be taken off first, pinned or not;
// an item marked keep and quest items, keys, tomes and the start kit are never trashed.
func (d *Desk) trashWhy(u d2.Unit) (code, why string) {
	d.mu.Lock()
	tr, tb := d.tr, d.c.TrashBlock
	d.mu.Unlock()
	switch {
	case tr == nil:
		return "off", "trashing is not available on this bot"
	case d.MarkOf(u) == MarkKeep:
		return "keep", "it is marked keep: clear the keep mark first"
	case u.Item.Where == d2.ContEquipment:
		return "equipped", "it is worn: take it off first"
	case u.Item.Where != d2.ContInventory:
		return "place", "only backpack items are trashed: move it to the backpack first"
	}
	if tb != nil {
		if w := tb(u); w != "" {
			return "never", fmt.Sprintf("this item is never trashed (%s)", w)
		}
	}
	return "", ""
}

// RequestTrash queues dropping backpack item id on the ground. confirm must be true (the page asks twice); without it
// nothing is queued (code "confirm").
func (d *Desk) RequestTrash(id uint32, confirm bool) (Request, error) {
	u, s, ok := d.unit(id)
	if !ok {
		return Request{}, refuse("item", "that item is not in the inventory any more")
	}
	if code, why := d.trashWhy(u); code != "" {
		return Request{}, refuse(code, "%s", why)
	}
	if !confirm {
		return Request{}, refuse("confirm", "trashing needs a confirmation")
	}
	d.mu.Lock()
	tr := d.tr
	d.mu.Unlock()
	name, err := tr.CheckPlayerTrash(s, id) // unlocked, see SetMark
	if err != nil {
		return Request{}, refuse("requirements", "%v", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	r := Request{ID: id, Kind: KindTrash, Name: name, State: StateQueued, At: d.c.Now().Unix(), Print: fingerprint(u)}
	q := d.st.Queue[:0:0]
	for _, x := range d.st.Queue {
		if x.State == StateDoing || x.ID != id { // a queued equip of the same item is replaced by the trash
			q = append(q, x)
		} else if x.kind() == KindTrash {
			return Request{}, refuse("queued", "that item is already queued to be trashed")
		}
	}
	if len(q) >= MaxQueue {
		return Request{}, refuse("full", "%d actions are already waiting: cancel one first", MaxQueue)
	}
	d.st.Queue = append(q, r)
	return r, d.saveLocked()
}

// Cancel withdraws a queued request. A request the bot is already performing cannot be cancelled (code "busy").
func (d *Desk) Cancel(id uint32, kind string) error {
	if kind == "" {
		kind = KindEquip
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, x := range d.st.Queue {
		if x.ID != id || x.kind() != kind {
			continue
		}
		if x.State == StateDoing {
			return refuse("busy", "the bot is doing that right now")
		}
		d.st.Queue = append(d.st.Queue[:i:i], d.st.Queue[i+1:]...)
		x.State = StateCancelled
		d.recentLocked(x)
		return d.saveLocked()
	}
	return refuse("item", "that request is not queued any more")
}

func (d *Desk) recentLocked(r Request) {
	d.st.Recent = append([]Request{r}, d.st.Recent...)
	if len(d.st.Recent) > maxRecent {
		d.st.Recent = d.st.Recent[:maxRecent]
	}
}

// Ignored is the looter's skip test (loot.Looter.SetIgnore): a ground item the player trashed within TrashTTL. It
// matches by unit id plus fingerprint (class/quality/ilvl); because a dropped item may get a new id (U live), an item
// with the same fingerprint within trashNear subtiles of the drop point matches too.
func (d *Desk) Ignored(u d2.Unit) bool {
	if d == nil || u.Item == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.st.Trashed) == 0 {
		return false
	}
	now, fp := d.c.Now().Unix(), fingerprint(u)
	for _, t := range d.st.Trashed {
		if now-t.At > int64(TrashTTL/time.Second) || t.Print != fp {
			continue
		}
		if t.ID == u.ID || max(abs(t.X-u.X), abs(t.Y-u.Y)) <= trashNear {
			return true
		}
	}
	return false
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (d *Desk) noteTrashedLocked(r Request, x, y int) {
	now := d.c.Now().Unix()
	kept := d.st.Trashed[:0:0]
	for _, t := range d.st.Trashed {
		if now-t.At <= int64(TrashTTL/time.Second) {
			kept = append(kept, t)
		}
	}
	kept = append(kept, trashRec{ID: r.ID, Print: r.Print, At: now, X: x, Y: y})
	if len(kept) > maxTrashed {
		kept = kept[len(kept)-maxTrashed:]
	}
	d.st.Trashed = kept
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

// Pending is the number of queued player actions (bot.PlayerGear).
func (d *Desk) Pending() int {
	if d == nil {
		return 0
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for _, r := range d.st.Queue {
		if r.State == StateQueued {
			n++
		}
	}
	return n
}

// RunPending performs the queued actions, one at a time (bot.PlayerGear). The caller runs it at an open opportunity
// (docs/inventory-actions.md: in town, or in the field with nothing hostile near); each request shows "doing" while it
// runs. A successful equip pins the slot; a successful trash goes on the looter's ignore list. A fight or a cancelled
// context puts the request back to queued. It returns one line per request for the log. Callbacks run unlocked.
func (d *Desk) RunPending(ctx context.Context) []string {
	var out []string
	for ctx.Err() == nil {
		d.mu.Lock()
		i := -1
		for j, r := range d.st.Queue {
			if r.State == StateQueued {
				i = j
				break
			}
		}
		if i < 0 {
			d.mu.Unlock()
			break
		}
		r, eq, tr := d.st.Queue[i], d.eq, d.tr
		if r.kind() == KindTrash && tr == nil || r.kind() == KindEquip && eq == nil {
			d.mu.Unlock()
			break
		}
		d.st.Queue[i].State = StateDoing
		_ = d.saveLocked()
		d.mu.Unlock()

		var err error
		var hx, hy int
		if r.kind() == KindTrash {
			err = d.perform(r, func() error { return tr.PlayerTrash(ctx, r.ID) })
			if s := d.snap(); err == nil && s != nil && s.Player != nil {
				hx, hy = s.Player.X, s.Player.Y
			}
		} else {
			err = eq.PlayerEquip(ctx, r.ID, SlotBody[r.Slot])
		}

		d.mu.Lock()
		j := d.findLocked(r.ID, r.kind())
		if j >= 0 && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || isBusy(err)) {
			d.st.Queue[j].State = StateQueued // not the request's fault: try again at the next open point
			_ = d.saveLocked()
			d.mu.Unlock()
			out = append(out, fmt.Sprintf("%s %s postponed: %v", r.kind(), r.Name, err))
			break
		}
		if j >= 0 {
			d.st.Queue = append(d.st.Queue[:j:j], d.st.Queue[j+1:]...)
		}
		if err != nil {
			r.State, r.Reason = StateFailed, err.Error()
			out = append(out, fmt.Sprintf("%s %s failed: %v", r.kind(), r.Name, err))
		} else if r.kind() == KindTrash {
			r.State = StateDone
			d.noteTrashedLocked(r, hx, hy)
			out = append(out, fmt.Sprintf("trash %s done, on the ground and ignored by the looter", r.Name))
		} else {
			r.State = StateDone
			d.pinLocked(Pin{Slot: r.Slot, ID: r.ID, Name: r.Name, Since: d.c.Now().Unix()})
			out = append(out, fmt.Sprintf("equip %s -> %s done, slot pinned", r.Name, r.Slot))
		}
		d.recentLocked(r)
		_ = d.saveLocked()
		d.mu.Unlock()
	}
	return out
}

func (d *Desk) findLocked(id uint32, kind string) int {
	for i, r := range d.st.Queue {
		if r.ID == id && r.kind() == kind && r.State == StateDoing {
			return i
		}
	}
	return -1
}

func (d *Desk) snap() *d2.Snapshot {
	if d.c.Snap == nil {
		return nil
	}
	return d.c.Snap()
}

// perform runs a trash after the checks that can change while a request waits: a keep mark added since the request.
func (d *Desk) perform(r Request, do func() error) error {
	if u, _, ok := d.unit(r.ID); ok && d.MarkOf(u) == MarkKeep {
		return errors.New("it is marked keep now")
	}
	return do()
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
	// Schema 3: why an action is not offered (empty = it is), for the action menu on the page.
	NoWear  string `json:"noWear,omitempty"`  // backpack items with no slot that takes them
	NoSell  string `json:"noSell,omitempty"`  // not a backpack item, or never sold
	NoTrash string `json:"noTrash,omitempty"` // see trashWhy
}

// ViewSchema is the version of the /api/inventory reply: 2 added the optional item fields tip, base and sell (the item
// tooltip, docs/inventory-tooltip.md); 3 added the item fields noWear, noSell and noTrash and the request fields kind
// and print, and the request states doing and cancelled (the action menu, docs/inventory-actions.md). Older readers
// keep working: nothing was renamed or removed, the new fields are optional.
const ViewSchema = 3

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
	Prompt    string               `json:"prompt,omitempty"` // docs/gear-policy.md: tells the player equipment is theirs to manage
}

// View reads the snapshot and decorates it. Without a snapshot the lists are empty and Note says so.
func (d *Desk) View() View {
	// State is copied under the lock and the callbacks run after it is released: the equip check reaches Pinned
	// through the leveler and d.mu is not reentrant (live deadlock 2026-11-10: the whole bot froze).
	d.mu.Lock()
	c, eq := d.c, d.eq
	marks := append([]markRec(nil), d.st.Marks...)
	v := View{Schema: ViewSchema, Equipment: map[string]*ItemView{}, Pins: append([]Pin{}, d.st.Pins...), Queue: append([]Request{}, d.st.Queue...), Recent: append([]Request{}, d.st.Recent...)}
	d.mu.Unlock()
	sort.Slice(v.Pins, func(i, j int) bool { return v.Pins[i].Slot < v.Pins[j].Slot })
	if c.Inventory == nil {
		v.Note = "inventory is not available"
		return v
	}
	data := c.Inventory()
	var s *d2.Snapshot
	if c.Snap != nil {
		s = c.Snap()
	}
	if s != nil && s.Player != nil {
		v.InTown, v.Class = s.Player.InTown(), s.Player.Class.String()
	} else {
		v.Note = "waiting for the first look at the hero"
	}
	v.Gold, v.StashGold = data.Gold, data.StashGold
	why := c.Why
	if c.WhyFor != nil {
		why = c.WhyFor(s)
	}
	if d.c.Prompt != nil {
		v.Prompt = d.c.Prompt()
	}
	decorate := func(it status.Item, place string) ItemView {
		iv := ItemView{Item: it}
		u, _, ok := d.unit(it.ID)
		if !ok {
			return iv
		}
		for _, m := range marks {
			if m.ID == it.ID && m.Print == fingerprint(u) {
				iv.Mark = m.Mark
			}
		}
		if why != nil {
			iv.Why = why(u)
		}
		if c.Advice != nil {
			iv.Advice = c.Advice(it)
		}
		if place == "pack" {
			iv.CanSell = c.SellBlock == nil || c.SellBlock(u) == ""
			if !iv.CanSell {
				iv.NoSell = "this item is never sold"
			}
			if eq != nil && s != nil {
				var why error
				for _, k := range status.EquipSlots {
					_, err := eq.CheckPlayerEquip(s, it.ID, SlotBody[k])
					switch {
					case err == nil:
						iv.CanWear = append(iv.CanWear, k)
					case why == nil || !isNoSlot(err):
						why = err
					}
				}
				if len(iv.CanWear) == 0 && why != nil {
					iv.NoWear = why.Error()
				}
			} else if eq == nil {
				iv.NoWear = "equipping is not available on this bot"
			}
		} else if place != "worn" {
			iv.NoWear = "move it to the backpack first"
		}
		if place != "pack" {
			iv.NoSell = "only backpack items are sold"
		}
		if _, why := d.trashWhy(u); why != "" {
			iv.NoTrash = why
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
