package invdesk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
)

// Identify commands (API schema 5, docs/inventory-actions.md). Identifying is a player command, not an equipment
// decision (docs/gear-policy.md): the bot never identifies for the sake of a gear choice on this path. Three requests:
//   - KindIdentify: read a Scroll or Tome of Identify on one item (backpack or worn), anywhere the bot is free;
//   - KindBuyID: a town errand that buys identify scrolls (or fills the tome) within the gold above the reserve;
//   - KindCain: a town errand that has Deckard Cain identify everything, free, once he is rescued.
// The two errands name no item (ID 0) and wait until the hero is in town.

// NoToolReason is shown on an Identify entry when the pack holds no scroll and no tome.
const NoToolReason = "no Scroll or Tome of Identify in the backpack: use Buy identify scrolls or Identify all (Cain)"

// IdentifySafeWait is how long a queued identify waits for a safe moment (the repair trip's bound, repair.Trip.SafeWaitSecs):
// after it the request fails with a reason and the player asks again.
const IdentifySafeWait = 60 * time.Second

// WaitsForTown is the reason line of a queued errand while the hero is in the field.
const WaitsForTown = "waits until the hero is in town"

// CainTeaser is the text of the locked Identify all (Cain) command.
const CainTeaser = "opens when Cain is rescued"

// Services is the town side of the identify commands (*town.Town). Check methods only read the snapshot; the others
// click. The perform methods may return a "fight is on" error (isBusy) to postpone the request.
type Services interface {
	// CheckPlayerIdentify says why backpack or worn item id cannot be identified now (nil: it can); name is its name.
	CheckPlayerIdentify(s *d2.Snapshot, id uint32) (name string, err error)
	// PlayerIdentifyBatch identifies the items in one inventory-open (the panel is not reopened per item) and returns one
	// error per id (nil = identified). Without a scroll or tome every entry fails at once with the hint to buy scrolls.
	PlayerIdentifyBatch(ctx context.Context, ids []uint32) []error
	// IdentifyTools reports the Tome of Identify (scrolls in it, -1 without a tome), its capacity and the loose scrolls.
	IdentifyTools(s *d2.Snapshot) (tome, tomeCap, scrolls int)
	// CheckBuyIdentify validates n (0 = fill the tome, or the default without one) and names the errand.
	CheckBuyIdentify(s *d2.Snapshot, n int) (name string, err error)
	// BuyIdentify runs the errand in town and returns a sentence such as "bought 12, short of gold for the rest".
	BuyIdentify(ctx context.Context, n int) (string, error)
	CheckCainIdentify(s *d2.Snapshot) error
	CainIdentify(ctx context.Context) (string, error)
}

// SetServices installs the town side; until then the identify commands are refused with code "off".
func (d *Desk) SetServices(s Services) { d.mu.Lock(); d.sv = s; d.mu.Unlock() }

// SetCainOpen installs the "Cain is rescued" check (Config.CainOpen).
func (d *Desk) SetCainOpen(f func(s *d2.Snapshot) bool) { d.mu.Lock(); d.c.CainOpen = f; d.mu.Unlock() }

func (d *Desk) heroInTown() bool {
	s := d.snap()
	return s != nil && s.Player != nil && s.Player.InTown()
}

// runnableLocked says whether the bot could start r now: the needed backend is installed and, for a town errand, the
// hero is in town. inTown is read before the lock is taken (the snapshot callback must run unlocked).
func (d *Desk) runnableLocked(r Request, inTown bool) bool {
	switch r.kind() {
	case KindTrash:
		return d.tr != nil
	case KindIdentify:
		if r.Place == PlaceStash {
			return d.stashSvLocked() != nil && inTown
		}
		return d.sv != nil
	case KindBuyID, KindCain:
		return d.sv != nil && inTown
	case KindStash, KindTake, KindGold:
		return d.stashSvLocked() != nil && inTown
	}
	return d.eq != nil
}

func (d *Desk) services() (Services, func(*d2.Snapshot) bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sv, d.c.CainOpen
}

// identifyWhy says why the unidentified item u cannot be identified now ("" = it can), with the refusal code. The
// pack holds the tool check (tools), so the view asks once per page build instead of once per item.
func identifyWhy(u d2.Unit, hasTool bool) (code, why string) {
	switch {
	case u.Item.Where != d2.ContInventory && u.Item.Where != d2.ContEquipment:
		return "place", "only backpack and worn items are identified: move it to the backpack first"
	case !hasTool:
		return "requirements", NoToolReason
	}
	return "", ""
}

// RequestIdentify queues reading a Scroll or Tome of Identify on item id. Refused for an identified item, an item
// outside the backpack and the doll, and when the pack holds no scroll or tome (the reason names the two other
// commands).
func (d *Desk) RequestIdentify(id uint32) (Request, error) {
	sv, _ := d.services()
	if sv == nil {
		return Request{}, refuse("off", "identifying is not available on this bot")
	}
	u, s, ok := d.unit(id)
	if !ok {
		return Request{}, refuse("item", "that item is not in the inventory any more")
	}
	if u.Item.Identified() {
		return Request{}, refuse("identified", "it is identified already")
	}
	if u.Item.Where == d2.ContStash && d.stashSv() != nil {
		return d.requestStashIdentify(id, u, s) // in place, in town
	}
	if code, why := identifyWhy(u, true); code != "" {
		return Request{}, refuse(code, "%s", why)
	}
	name, err := sv.CheckPlayerIdentify(s, id) // unlocked, see SetMark
	if err != nil {
		return Request{}, refuse("requirements", "%v", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.queuedLocked(id, KindTrash) {
		return Request{}, refuse("queued", "it is queued to be trashed: cancel that first")
	}
	if d.queuedLocked(id, KindIdentify) {
		return Request{}, refuse("queued", "that item is already queued to be identified")
	}
	return d.enqueueLocked(Request{ID: id, Kind: KindIdentify, Name: name, Print: fingerprint(u)})
}

// RequestBuyID queues the shopping errand: buy n identify scrolls (0: fill the Tome of Identify, else buy a tome when
// the gold above the reserve allows, else a few scrolls). It never spends below the repair and kit reserve.
func (d *Desk) RequestBuyID(n int) (Request, error) {
	sv, _ := d.services()
	if sv == nil {
		return Request{}, refuse("off", "shopping for identify scrolls is not available on this bot")
	}
	s := d.snap()
	if s == nil || s.Player == nil || s.Inventory == nil {
		return Request{}, refuse("waiting", "waiting for the first look at the hero")
	}
	name, err := sv.CheckBuyIdentify(s, n)
	if err != nil {
		return Request{}, refuse("requirements", "%v", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.queuedLocked(0, KindBuyID) {
		return Request{}, refuse("queued", "a shopping errand is already waiting: cancel it first")
	}
	return d.enqueueLocked(Request{Kind: KindBuyID, Name: name, Count: n})
}

// RequestCain queues the errand to have Deckard Cain identify every item. Refused while he is not rescued (code
// "locked", the teaser is the reason) and when nothing needs identifying.
func (d *Desk) RequestCain() (Request, error) {
	sv, open := d.services()
	if sv == nil || open == nil {
		return Request{}, refuse("off", "Deckard Cain is not available on this bot")
	}
	s := d.snap()
	if s == nil || s.Player == nil || s.Inventory == nil {
		return Request{}, refuse("waiting", "waiting for the first look at the hero")
	}
	if !open(s) {
		return Request{}, refuse("locked", "%s", CainTeaser)
	}
	if err := sv.CheckCainIdentify(s); err != nil {
		return Request{}, refuse("requirements", "%v", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.queuedLocked(0, KindCain) {
		return Request{}, refuse("queued", "that errand is already waiting: cancel it first")
	}
	return d.enqueueLocked(Request{Kind: KindCain, Name: "Identify all at Deckard Cain"})
}

// enqueueLocked stamps and appends a new request (bounded by MaxQueue) and saves.
func (d *Desk) enqueueLocked(r Request) (Request, error) {
	if len(d.st.Queue) >= MaxQueue {
		return Request{}, refuse("full", "%d actions are already waiting: cancel one first", MaxQueue)
	}
	r.State, r.At = StateQueued, d.c.Now().Unix()
	d.st.Queue = append(d.st.Queue, r)
	return r, d.saveLocked()
}

// IdentifyView is the "Identify" block of the view (schema 5): what the pack holds, the default of the shopping
// errand and the state of the Cain command. Absent when the bot has no town backend.
type IdentifyView struct {
	Tool    string `json:"tool"`    // "tome", "scrolls" or "none"
	Tome    int    `json:"tome"`    // scrolls in the Tome of Identify; -1 = no tome
	TomeCap int    `json:"tomeCap"` // its capacity
	Scrolls int    `json:"scrolls"` // loose scrolls
	Unid    int    `json:"unid"`    // unidentified items carried or worn
	// Buy: the default count of the shopping errand (0 = fill the tome, or buy one and fill it), the largest count the
	// page offers, and why the command is refused now ("" = it is available).
	BuyDefault int    `json:"buyDefault"`
	BuyMax     int    `json:"buyMax"`
	BuyWhy     string `json:"buyWhy,omitempty"`
	// Cain: "open" or "locked" ("" = the command is not offered); CainWhy is the teaser while locked, or why an open
	// command is refused now.
	Cain    string `json:"cain,omitempty"`
	CainWhy string `json:"cainWhy,omitempty"`
}

func countUnid(s *d2.Snapshot) int {
	n := 0
	for _, c := range []d2.Container{d2.ContInventory, d2.ContEquipment} {
		for _, u := range s.Inventory.Container(c) {
			if u.Item != nil && !u.Item.Identified() {
				n++
			}
		}
	}
	return n
}

func (d *Desk) identifyView(sv Services, open func(*d2.Snapshot) bool, s *d2.Snapshot) *IdentifyView {
	tome, capv, scrolls := sv.IdentifyTools(s)
	v := &IdentifyView{Tool: "none", Tome: tome, TomeCap: capv, Scrolls: scrolls, Unid: countUnid(s), BuyMax: capv}
	switch {
	case tome > 0:
		v.Tool = "tome"
	case scrolls > 0:
		v.Tool = "scrolls"
	}
	if tome >= 0 {
		v.BuyDefault = capv - tome
	}
	if tome >= 0 && tome >= capv {
		v.BuyWhy = "the Tome of Identify is full"
	}
	if open != nil {
		switch {
		case !open(s):
			v.Cain, v.CainWhy = "locked", CainTeaser
		default:
			v.Cain = "open"
			if v.Unid == 0 {
				v.CainWhy = "nothing to identify"
			}
		}
	}
	return v
}

// Priority ranks the queued kinds: an identify (a scroll read, seconds) goes before equip and trash, those before the
// town errands. Lower runs first.
func priority(r Request) int {
	switch {
	case r.errand():
		return 2
	case r.kind() == KindIdentify:
		return 0
	}
	return 1
}

// expireStale fails the queued identifies that found no safe moment within IdentifySafeWait of being asked. Called with
// the lock held by Pending, which the bot asks every tick.
func (d *Desk) expireStaleLocked() {
	now := d.c.Now()
	q := d.st.Queue[:0:0]
	for _, r := range d.st.Queue {
		if r.kind() == KindIdentify && r.Place != PlaceStash && r.State == StateQueued && now.Sub(time.Unix(r.At, 0)) > IdentifySafeWait {
			r.State, r.Reason = StateFailed, fmt.Sprintf("no safe moment within %d s (monsters close or the hero was busy): ask again", int(IdentifySafeWait/time.Second))
			d.recentLocked(r)
			continue
		}
		q = append(q, r)
	}
	if len(q) != len(d.st.Queue) {
		d.st.Queue = q
		_ = d.saveLocked()
	}
}

// NextKind is the kind of the request the bot would run first now ("" = none); the Why line names it.
func (d *Desk) NextKind() string {
	inTown := d.heroInTown()
	d.mu.Lock()
	defer d.mu.Unlock()
	best, k := 99, ""
	for _, r := range d.st.Queue {
		if r.State == StateQueued && (inTown || !r.errand()) && priority(r) < best {
			best, k = priority(r), r.kind()
			if r.isStashJob() {
				k = "stash" // the Why line's `what`
			}
		}
	}
	return k
}

// ErrandsWaiting is the number of queued town errands; the bot goes home for them at its next safe moment.
func (d *Desk) ErrandsWaiting() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for _, r := range d.st.Queue {
		if r.State == StateQueued && r.errand() && d.sv != nil {
			n++
		}
	}
	return n
}

// runIdentifyBatch performs every queued identify in one call and settles each request. It reports whether the run
// must stop (a fight or a cancelled context put requests back to queued).
func (d *Desk) runIdentifyBatch(ctx context.Context, sv Services, out *[]string) (stop bool) {
	d.mu.Lock()
	var batch []Request
	for i := range d.st.Queue {
		if r := d.st.Queue[i]; r.State == StateQueued && r.kind() == KindIdentify && r.Place != PlaceStash {
			d.st.Queue[i].State = StateDoing
			batch = append(batch, r)
		}
	}
	_ = d.saveLocked()
	d.mu.Unlock()
	ids := make([]uint32, len(batch))
	for i, r := range batch {
		ids[i] = r.ID
	}
	errs := sv.PlayerIdentifyBatch(ctx, ids)

	d.mu.Lock()
	defer d.mu.Unlock()
	for i, r := range batch {
		var err error
		if i < len(errs) {
			err = errs[i]
		} else {
			err = context.Canceled
		}
		j := d.findLocked(r.ID, r.kind())
		if j < 0 {
			continue
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || isBusy(err) {
			d.st.Queue[j].State = StateQueued
			*out = append(*out, fmt.Sprintf("identify %s postponed: %v", r.Name, err))
			stop = true
			continue
		}
		d.st.Queue = append(d.st.Queue[:j:j], d.st.Queue[j+1:]...)
		if err != nil {
			r.State, r.Reason = StateFailed, err.Error()
			*out = append(*out, fmt.Sprintf("identify %s failed: %v", r.Name, err))
		} else {
			r.State = StateDone
			*out = append(*out, fmt.Sprintf("identify %s done", r.Name))
		}
		d.recentLocked(r)
	}
	_ = d.saveLocked()
	return stop
}
