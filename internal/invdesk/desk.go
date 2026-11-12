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
	"d2lod-bot/internal/merc"
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
	// Schema 5: identify one item, and two town errands that name no item (ID 0): buy identify scrolls, and have
	// Deckard Cain identify everything (docs/inventory-actions.md).
	KindIdentify = "identify"
	KindBuyID    = "buyid"
	KindCain     = "cain"
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
	// CainOpen says whether Deckard Cain is rescued (the Search for Cain quest done, or Cain seen in town); nil: the
	// "Identify all (Cain)" command is not offered at all.
	CainOpen func(s *d2.Snapshot) bool
	Advice   func(it status.Item) string // optional hook for upgrade suggestions (card upgradeadvisor); nil: none
	Prompt   func() string               // optional: the gear-policy prompt for the page (onboarding, no weapon); must not call back into the desk
	Now      func() time.Time
}

// Desk holds the player's decisions. All methods are safe for concurrent use.
type Desk struct {
	c  Config
	mu sync.Mutex
	eq Equipper
	tr Trasher
	sv Services
	st state
}

type markRec struct {
	ID    uint32 `json:"id"`
	Print string `json:"print"` // class/quality/ilvl: a reused id of another item does not inherit the mark
	Mark  string `json:"mark"`
	// Class and the grid cell let Reconcile find the item again when its id changed (a game restart gives new ids)
	// or its print changed (identified). Marks saved before this field existed have Class 0 and match by print only.
	Class int    `json:"class,omitempty"` // base row + 1, 0 for a legacy mark
	ILvl  int    `json:"ilvl,omitempty"`
	Where string `json:"where,omitempty"`
	GX    int    `json:"gx,omitempty"`
	GY    int    `json:"gy,omitempty"`
}

func newMarkRec(u d2.Unit, mark string) markRec {
	m := markRec{ID: u.ID, Print: fingerprint(u), Mark: mark, Class: u.Class + 1}
	if u.Item != nil {
		m.ILvl = u.Item.ILvl
		m.Where, m.GX, m.GY = u.Item.Where.String(), u.Item.GX, u.Item.GY
	}
	return m
}

// Reconcile re-attaches the player's marks to the items they were put on after an item id changed (item ids belong
// to one game session: a restart renumbers the whole pack, live 2026-11-11: marked items stayed in the bag, never
// sold) or after the print changed (identify). A mark whose id is gone is moved to the unmarked item of the same
// base at the same grid cell; marks of items that are no longer carried stay until the player clears them. It also
// follows an item that moved in the pack. Returns the number of marks that changed.
func (d *Desk) Reconcile(items []d2.Unit) int {
	if d == nil {
		return 0
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	byID := map[uint32]int{}
	for i, u := range items {
		if u.Item != nil {
			byID[u.ID] = i
		}
	}
	taken := map[int]bool{}
	var orphans []int
	changed := 0
	for mi := range d.st.Marks {
		m := &d.st.Marks[mi]
		if i, ok := byID[m.ID]; ok && markFits(*m, items[i]) {
			taken[i] = true
			n := newMarkRec(items[i], m.Mark)
			if n != *m {
				*m = n
				changed++
			}
			continue
		}
		orphans = append(orphans, mi)
	}
	for _, mi := range orphans {
		m := &d.st.Marks[mi]
		for i, u := range items {
			if taken[i] || u.Item == nil || !markFits(*m, u) || u.Item.Where.String() != m.Where || u.Item.GX != m.GX || u.Item.GY != m.GY {
				continue
			}
			taken[i] = true
			*m = newMarkRec(u, m.Mark)
			changed++
			break
		}
	}
	if changed > 0 {
		_ = d.saveLocked()
	}
	return changed
}

// markFits: the item is still the one the mark was put on. The base must agree; the print may differ once it is
// identified, never for a legacy mark (Class 0) which has only the print.
func markFits(m markRec, u d2.Unit) bool {
	if m.Class == 0 {
		return m.Print == fingerprint(u)
	}
	return m.Class == u.Class+1 && (m.Print == fingerprint(u) || u.Item != nil && m.ILvl == u.Item.ILvl)
}

// Forget drops the mark of an item that left the pack (sold).
func (d *Desk) Forget(id uint32) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	kept := d.st.Marks[:0]
	for _, m := range d.st.Marks {
		if m.ID != id {
			kept = append(kept, m)
		}
	}
	if len(kept) != len(d.st.Marks) {
		d.st.Marks = kept
		_ = d.saveLocked()
	}
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
	// Schema 5. Count is the number of scrolls the player asked for (buyid; 0 = fill the tome); Result is what a done
	// request did, in a sentence ("bought 12, short of gold for the rest").
	Count  int    `json:"count,omitempty"`
	Result string `json:"result,omitempty"`
	// Schema 6 (stash actions): Place "stash" marks an identify of an item that lies in the stash (done in place, in
	// town); Op is "deposit" or "withdraw" on a gold request.
	Place string `json:"place,omitempty"`
	Op    string `json:"op,omitempty"`
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
		if m.ID == u.ID && markFits(m, u) {
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
		d.st.Marks = append(d.st.Marks, newMarkRec(u, mark))
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

// Pending is the number of queued player actions the bot could start now (bot.PlayerGear): the town errands count only
// while the hero is in town, so the bot does not report "doing what you asked" for a request that must wait.
func (d *Desk) Pending() int {
	if d == nil {
		return 0
	}
	inTown := d.heroInTown()
	d.mu.Lock()
	defer d.mu.Unlock()
	d.expireStaleLocked()
	n := 0
	for _, r := range d.st.Queue {
		if r.State == StateQueued && (inTown || !r.errand()) {
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
		inTown := d.heroInTown() // the snapshot callback runs unlocked
		d.mu.Lock()
		d.expireStaleLocked()
		i := -1
		for j, r := range d.st.Queue {
			if r.State == StateQueued && d.runnableLocked(r, inTown) && (i < 0 || priority(r) < priority(d.st.Queue[i])) {
				i = j
			}
		}
		if i < 0 {
			d.mu.Unlock()
			break
		}
		if q := d.st.Queue[i]; q.isStashJob() { // every queued stash job shares one visit
			d.mu.Unlock()
			if d.runStashBatch(ctx, &out) {
				break
			}
			continue
		}
		if d.st.Queue[i].kind() == KindIdentify { // all queued identifies in one inventory-open
			sv := d.sv
			d.mu.Unlock()
			if d.runIdentifyBatch(ctx, sv, &out) {
				break
			}
			continue
		}
		r, eq, tr, sv := d.st.Queue[i], d.eq, d.tr, d.sv
		d.st.Queue[i].State = StateDoing
		_ = d.saveLocked()
		d.mu.Unlock()

		var err error
		var hx, hy int
		result := ""
		switch r.kind() {
		case KindTrash:
			err = d.perform(r, func() error { return tr.PlayerTrash(ctx, r.ID) })
			if s := d.snap(); err == nil && s != nil && s.Player != nil {
				hx, hy = s.Player.X, s.Player.Y
			}
		case KindBuyID:
			result, err = sv.BuyIdentify(ctx, r.Count)
		case KindCain:
			result, err = sv.CainIdentify(ctx)
		default:
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
		} else {
			r.State, r.Result = StateDone, result
			switch r.kind() {
			case KindTrash:
				d.noteTrashedLocked(r, hx, hy)
				out = append(out, fmt.Sprintf("trash %s done, on the ground and ignored by the looter", r.Name))
			case KindBuyID, KindCain:
				out = append(out, fmt.Sprintf("%s done: %s", r.kind(), result))
			default:
				d.pinLocked(Pin{Slot: r.Slot, ID: r.ID, Name: r.Name, Since: d.c.Now().Unix()})
				out = append(out, fmt.Sprintf("equip %s -> %s done, slot pinned", r.Name, r.Slot))
			}
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

// isBusy matches leveler.ErrBusy and town.ErrFight without importing the leveler.
func isBusy(err error) bool {
	return err != nil && (err.Error() == "leveler: a fight is on" || err.Error() == "town: a fight is on")
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
	// Schema 5: set only on an unidentified item that cannot be identified now (the reason); empty on an identified item
	// (identifying it is refused) and on an unidentified one that can be.
	NoIdentify string `json:"noIdentify,omitempty"`
	// Schema 6: the stash move this item offers ("stash" for a backpack item, "take" for a stash item; empty when the
	// bot has no stash backend or the item lies elsewhere) and, when it is refused now, the reason.
	StashOp string `json:"stashOp,omitempty"`
	NoStash string `json:"noStash,omitempty"`
}

// ViewSchema is the version of the /api/inventory reply: 2 added the optional item fields tip, base and sell (the item
// tooltip, docs/inventory-tooltip.md); 3 added the item fields noWear, noSell and noTrash and the request fields kind
// and print, and the request states doing and cancelled (the action menu, docs/inventory-actions.md); 4 added the
// optional merc block (docs/research/merc-gear.md); 5 added the optional identify block, the item field noIdentify and the
// request kinds identify, buyid and cain with the fields count and result (the identify commands); 6 added the item fields stashOp and noStash, the stashOps block and the request kinds stash, take and gold (docs/inventory-actions.md, stash actions). Older readers keep working: nothing was renamed or removed, the new
// fields are optional.
const ViewSchema = 6

// View is the inventory page's data.
type View struct {
	Schema      int                  `json:"schema"`
	InTown      bool                 `json:"inTown"`
	Class       string               `json:"class,omitempty"`
	Gold        int                  `json:"gold"`
	SellPending int                  `json:"sellPending,omitempty"` // backpack items marked "sell" that the next town visit sells
	StashGold   int                  `json:"stashGold"`
	Equipment   map[string]*ItemView `json:"equipment"`
	Backpack    []ItemView           `json:"backpack"`
	Stash       []ItemView           `json:"stash"`
	Belt        []ItemView           `json:"belt"`
	Cube        []ItemView           `json:"cube,omitempty"`
	Pins        []Pin                `json:"pins"`
	Queue       []Request            `json:"queue"`
	Recent      []Request            `json:"recent"`
	Note        string               `json:"note,omitempty"`
	Prompt      string               `json:"prompt,omitempty"`   // docs/gear-policy.md: tells the player equipment is theirs to manage
	StashOps    *StashOpsView        `json:"stashOps,omitempty"` // schema 6: the stash gold controls; nil = no stash backend
	Identify    *IdentifyView        `json:"identify,omitempty"` // schema 5: the identify commands' state; nil = no town backend
	Merc        *MercView            `json:"merc,omitempty"`     // schema 3: the hired merc and what he wears; nil = the backend cannot read a merc
}

// Merc states of MercView.State.
const (
	MercNone    = "none"    // nobody is employed
	MercReady   = "ready"   // employed and alive, gear read
	MercDead    = "dead"    // employed, dead (gear is shown, not changeable)
	MercUnknown = "unknown" // employed, but the backend cannot list his gear
)

// MercView is the merc doll's data (read only in stage A: no merc action exists yet).
type MercView struct {
	State     string               `json:"state"`
	Owned     bool                 `json:"owned"`
	Alive     bool                 `json:"alive"`
	Class     int                  `json:"class,omitempty"`
	Kind      string               `json:"kind,omitempty"` // rogue, desert, ironwolf, barbarian
	Level     int                  `json:"level,omitempty"`
	Str       int                  `json:"str,omitempty"`
	Dex       int                  `json:"dex,omitempty"`
	LifePct   int                  `json:"lifePct,omitempty"`
	Slots     []string             `json:"slots,omitempty"` // the slots this kind can use; the page greys the others
	Equipment map[string]*ItemView `json:"equipment"`
	Note      string               `json:"note,omitempty"`
}

// View reads the snapshot and decorates it. Without a snapshot the lists are empty and Note says so.
func (d *Desk) View() View {
	// State is copied under the lock and the callbacks run after it is released: the equip check reaches Pinned
	// through the leveler and d.mu is not reentrant (live deadlock 2026-11-10: the whole bot froze).
	d.mu.Lock()
	c, eq, sv := d.c, d.eq, d.sv
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
	if s != nil && s.Inventory != nil {
		d.Reconcile(s.Inventory.Items)
	}
	d.mu.Lock()
	marks := append([]markRec(nil), d.st.Marks...)
	d.mu.Unlock()
	if s != nil && s.Player != nil {
		v.InTown, v.Class = s.Player.InTown(), s.Player.Class.String()
	} else {
		v.Note = "waiting for the first look at the hero"
	}
	v.Gold, v.StashGold = data.Gold, data.StashGold
	hasTool := false
	ssv, _ := sv.(StashServices)
	if ssv != nil && s != nil && s.Inventory != nil {
		v.StashOps = stashOpsView(s, data.Gold, data.StashGold)
	}
	if sv != nil && s != nil && s.Inventory != nil {
		v.Identify = d.identifyView(sv, c.CainOpen, s)
		hasTool = v.Identify.Tool != "none"
	}
	for i, r := range v.Queue { // a town errand cannot start in the field: say so on the queue card
		if r.errand() && r.State == StateQueued && !v.InTown && r.Reason == "" {
			v.Queue[i].Reason = WaitsForTown
		}
	}
	why := c.Why
	if c.WhyFor != nil {
		why = c.WhyFor(s)
	}
	if d.c.Prompt != nil {
		v.Prompt = d.c.Prompt()
	}
	decorate := func(it status.Item, place string) ItemView {
		iv := ItemView{Item: it}
		if place == "mercworn" {
			return iv // his own id space: no marks, pins or advice by id (stage A is read only)
		}
		u, _, ok := d.unit(it.ID)
		if !ok {
			return iv
		}
		for _, m := range marks {
			if m.ID == it.ID && markFits(m, u) {
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
		if sv != nil && !it.Identified {
			_, iv.NoIdentify = identifyWhy(u, hasTool)
		}
		if ssv != nil && s != nil {
			stashItemView(&iv, ssv, s, place, hasTool)
		}
		return iv
	}
	for k, it := range data.Equipment {
		if it != nil {
			iv := decorate(*it, "worn")
			v.Equipment[k] = &iv
		}
	}
	v.Merc = mercView(data.Merc, decorate)
	for _, it := range data.Inventory {
		iv := decorate(it, "pack")
		if iv.Mark == MarkSell {
			v.SellPending++
		}
		v.Backpack = append(v.Backpack, iv)
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

// mercView turns the status block into the page's view. Items are decorated like the worn ones (tip, base, sell).
func mercView(m *status.MercInv, decorate func(status.Item, string) ItemView) *MercView {
	if m == nil {
		return nil
	}
	v := &MercView{State: MercNone, Equipment: map[string]*ItemView{}}
	if !m.Hired {
		return v
	}
	v.Owned, v.Alive, v.Class, v.Level, v.Str, v.Dex, v.LifePct = true, m.Alive, m.Class, m.Level, m.Str, m.Dex, m.LifePct
	v.Kind, v.Slots = merc.KindName(m.Class), merc.WearableKeys(m.Class)
	for k, it := range m.Equipment {
		if it != nil {
			iv := decorate(*it, "mercworn")
			v.Equipment[k] = &iv
		}
	}
	switch {
	case !m.GearKnown:
		v.State, v.Note = MercUnknown, "Equipment unreadable on this backend."
	case !m.Alive:
		v.State, v.Note = MercDead, "Mercenary is dead: revive first; gear cannot be changed while he is dead."
	default:
		v.State = MercReady
	}
	return v
}
