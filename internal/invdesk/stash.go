package invdesk

import (
	"context"
	"errors"
	"fmt"

	"d2lod-bot/internal/d2"
)

// Stash actions (API schema 6, docs/inventory-actions.md): move a backpack item into the stash, take a stash item out
// into the backpack, deposit or withdraw gold, and identify an item where it lies in the stash. The stash opens only in
// town, so all four are town errands: in the field the bot walks home at the next safe moment (errandTripStep) and every
// queued stash job then shares ONE stash visit. Moving an item never touches equipment or the keep and sell marks.
const (
	KindStash = "stash" // backpack -> stash
	KindTake  = "take"  // stash -> backpack
	KindGold  = "gold"  // Op deposit or withdraw, Count gold; names no item (ID 0)
	// PlaceStash on an identify request: the item lies in the stash and is read there (no take-out, no put-back).
	PlaceStash = "stash"
	OpDeposit  = "deposit"
	OpWithdraw = "withdraw"
)

// WithdrawPerLevel is the most gold the backpack holds per character level (S: the game's rule; the withdraw control caps
// at it and the stash dialog refuses more).
const WithdrawPerLevel = 10000

// StashPlan is one stash visit: the town opens the stash once, runs everything and closes it.
type StashPlan struct {
	Stow     []uint32 // backpack -> stash
	Take     []uint32 // stash -> backpack
	Identify []uint32 // read in place in the stash with a scroll or the tome
	Gold     int      // 0 = no gold move
	Deposit  bool     // gold direction
}

// StashResult settles a visit. Errs holds one entry per id of the plan (nil = done); a visit-level failure fills every
// entry. Gold is what the gold move did in a sentence, GoldErr its failure.
type StashResult struct {
	Errs    map[uint32]error
	Gold    string
	GoldErr error
}

// StashServices is the optional stash side of the town (*town.Town). Services that do not implement it leave the stash
// actions off (code "off"), and the identify of a stash item stays refused with code "place".
type StashServices interface {
	// CheckStashMove says why item id cannot move now (nil: it can): toStash true for a backpack item, false for a stash
	// item. It refuses when the target has no free spot for the item's size.
	CheckStashMove(s *d2.Snapshot, id uint32, toStash bool) (name string, err error)
	// CheckStashIdentify says why the stash item id cannot be read in place (no tool, identified already).
	CheckStashIdentify(s *d2.Snapshot, id uint32) (name string, err error)
	// StashVisit runs the plan in town (see StashPlan). A fight or a cancelled context returns that error for the ids not
	// done, and the desk puts those requests back to queued.
	StashVisit(ctx context.Context, p StashPlan) StashResult
}

func (d *Desk) stashSvLocked() StashServices {
	s, _ := d.sv.(StashServices)
	return s
}

func (d *Desk) stashSv() StashServices {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stashSvLocked()
}

// errand is true for the requests that need the hero in town: the shopping and Cain errands and every stash job.
func (r Request) errand() bool {
	switch r.kind() {
	case KindBuyID, KindCain, KindStash, KindTake, KindGold:
		return true
	}
	return r.kind() == KindIdentify && r.Place == PlaceStash
}

func (r Request) isStashJob() bool {
	switch r.kind() {
	case KindStash, KindTake, KindGold:
		return true
	}
	return r.kind() == KindIdentify && r.Place == PlaceStash
}

// stashBusyLocked: an item takes part in one stash job at a time.
func (d *Desk) stashBusyLocked(id uint32) bool {
	for _, x := range d.st.Queue {
		if x.ID == id && x.isStashJob() && x.kind() != KindGold {
			return true
		}
	}
	return false
}

func (d *Desk) requestMove(id uint32, toStash bool) (Request, error) {
	sv := d.stashSv()
	if sv == nil {
		return Request{}, refuse("off", "moving items to and from the stash is not available on this bot")
	}
	u, s, ok := d.unit(id)
	if !ok {
		return Request{}, refuse("item", "that item is not in the inventory any more")
	}
	switch {
	case toStash && u.Item.Where != d2.ContInventory:
		return Request{}, refuse("place", "only backpack items move to the stash")
	case !toStash && u.Item.Where != d2.ContStash:
		return Request{}, refuse("place", "only stash items are taken out")
	}
	name, err := sv.CheckStashMove(s, id, toStash)
	if err != nil {
		return Request{}, refuse("room", "%v", err)
	}
	kind := KindTake
	if toStash {
		kind = KindStash
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.queuedLocked(id, KindTrash) {
		return Request{}, refuse("queued", "it is queued to be trashed: cancel that first")
	}
	if d.stashBusyLocked(id) {
		return Request{}, refuse("queued", "a stash action for that item is already waiting")
	}
	return d.enqueueLocked(Request{ID: id, Kind: kind, Name: name, Print: fingerprint(u)})
}

// RequestStash queues moving backpack item id into the stash.
func (d *Desk) RequestStash(id uint32) (Request, error) { return d.requestMove(id, true) }

// RequestTake queues taking stash item id out into the backpack. Refused at once when the backpack has no free spot for
// its size (the reason says to make room).
func (d *Desk) RequestTake(id uint32) (Request, error) { return d.requestMove(id, false) }

// requestStashIdentify is RequestIdentify for an item in the stash: it is read where it lies, in town.
func (d *Desk) requestStashIdentify(id uint32, u d2.Unit, s *d2.Snapshot) (Request, error) {
	sv := d.stashSv()
	name, err := sv.CheckStashIdentify(s, id)
	if err != nil {
		return Request{}, refuse("requirements", "%v", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stashBusyLocked(id) || d.queuedLocked(id, KindIdentify) {
		return Request{}, refuse("queued", "that item is already queued to be identified")
	}
	return d.enqueueLocked(Request{ID: id, Kind: KindIdentify, Place: PlaceStash, Name: name, Print: fingerprint(u)})
}

// packGoldRoom is how much more gold the backpack holds at the hero's level.
func packGoldRoom(s *d2.Snapshot, packGold int) int {
	return max(0, s.Player.Level*WithdrawPerLevel-packGold)
}

// RequestGold queues a gold move between the backpack and the stash (deposit true: into the stash).
func (d *Desk) RequestGold(deposit bool, amount int) (Request, error) {
	if d.stashSv() == nil {
		return Request{}, refuse("off", "the stash gold controls are not available on this bot")
	}
	s := d.snap()
	if s == nil || s.Player == nil || s.Inventory == nil {
		return Request{}, refuse("waiting", "waiting for the first look at the hero")
	}
	if amount <= 0 {
		return Request{}, refuse("amount", "enter an amount of gold above zero")
	}
	op, limit := OpDeposit, s.Player.Gold
	why := "the backpack holds only %d gold"
	if !deposit {
		op, limit = OpWithdraw, min(s.Player.StashGold, packGoldRoom(s, s.Player.Gold))
		why = "at most %d gold can come out (the stash holds it, the backpack takes level x 10000)"
	}
	if amount > limit {
		return Request{}, refuse("amount", why, limit)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.queuedLocked(0, KindGold) {
		return Request{}, refuse("queued", "a gold move is already waiting: cancel it first")
	}
	return d.enqueueLocked(Request{Kind: KindGold, Op: op, Count: amount, Name: fmt.Sprintf("%s %d gold", map[string]string{OpDeposit: "Deposit", OpWithdraw: "Withdraw"}[op], amount)})
}

// StashOpsView is the stash gold controls' data (schema 6): the largest amounts the boxes take and why a control is off.
type StashOpsView struct {
	DepositMax  int    `json:"depositMax"`
	WithdrawMax int    `json:"withdrawMax"`
	NoDeposit   string `json:"noDeposit,omitempty"`
	NoWithdraw  string `json:"noWithdraw,omitempty"`
}

func stashOpsView(s *d2.Snapshot, gold, stashGold int) *StashOpsView {
	v := &StashOpsView{DepositMax: gold, WithdrawMax: stashGold}
	if s != nil && s.Player != nil {
		v.WithdrawMax = min(stashGold, packGoldRoom(s, gold))
		if v.WithdrawMax == 0 && stashGold > 0 {
			v.NoWithdraw = "the backpack holds level x 10000 gold already"
		}
	}
	if v.DepositMax == 0 {
		v.NoDeposit = "the backpack has no gold"
	}
	if v.WithdrawMax == 0 && v.NoWithdraw == "" {
		v.NoWithdraw = "the stash has no gold"
	}
	return v
}

// stashItemView fills the stash fields of a decorated item (called with the lock free: the checks read the snapshot).
func stashItemView(iv *ItemView, sv StashServices, s *d2.Snapshot, place string, hasTool bool) {
	switch place {
	case "pack":
		iv.StashOp = "stash"
	case "stash":
		iv.StashOp = "take"
	default:
		return
	}
	if _, err := sv.CheckStashMove(s, iv.ID, place == "pack"); err != nil {
		iv.NoStash = err.Error()
	}
	if place == "stash" && !iv.Identified {
		iv.NoIdentify = ""
		if _, err := sv.CheckStashIdentify(s, iv.ID); err != nil {
			iv.NoIdentify = err.Error()
		}
	}
}

// runStashBatch performs every queued stash job in one visit and settles each request. A fight, a cancelled context or a
// visit that never started puts the requests back to queued; it then reports stop.
func (d *Desk) runStashBatch(ctx context.Context, out *[]string) (stop bool) {
	d.mu.Lock()
	sv := d.stashSvLocked()
	var batch []Request
	var plan StashPlan
	for i := range d.st.Queue {
		r := d.st.Queue[i]
		if r.State != StateQueued || !r.isStashJob() {
			continue
		}
		d.st.Queue[i].State = StateDoing
		batch = append(batch, r)
		switch {
		case r.kind() == KindStash:
			plan.Stow = append(plan.Stow, r.ID)
		case r.kind() == KindTake:
			plan.Take = append(plan.Take, r.ID)
		case r.kind() == KindIdentify:
			plan.Identify = append(plan.Identify, r.ID)
		default:
			plan.Gold, plan.Deposit = r.Count, r.Op == OpDeposit
		}
	}
	_ = d.saveLocked()
	d.mu.Unlock()

	res := StashResult{}
	if sv != nil {
		res = sv.StashVisit(ctx, plan)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, r := range batch {
		var err error
		result := ""
		switch {
		case sv == nil:
			err = errors.New("the stash is not available")
		case r.kind() == KindGold:
			err, result = res.GoldErr, res.Gold
		default:
			var found bool
			if err, found = res.Errs[r.ID]; !found {
				err = context.Canceled
			}
		}
		j := d.findLocked(r.ID, r.kind())
		if j < 0 {
			continue
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || isBusy(err) {
			d.st.Queue[j].State = StateQueued
			*out = append(*out, fmt.Sprintf("%s %s postponed: %v", r.kind(), r.Name, err))
			stop = true
			continue
		}
		d.st.Queue = append(d.st.Queue[:j:j], d.st.Queue[j+1:]...)
		if err != nil {
			r.State, r.Reason = StateFailed, err.Error()
			*out = append(*out, fmt.Sprintf("%s %s failed: %v", r.kind(), r.Name, err))
		} else {
			r.State, r.Result = StateDone, result
			*out = append(*out, fmt.Sprintf("%s %s done", r.kind(), r.Name))
		}
		d.recentLocked(r)
	}
	_ = d.saveLocked()
	return stop
}

// ErrandKind is the kind of the first waiting town errand mapped to the Why line's `what` (stash jobs say "stash").
func (d *Desk) ErrandKind() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, r := range d.st.Queue {
		if r.State == StateQueued && r.errand() && d.sv != nil {
			if r.isStashJob() {
				return "stash"
			}
			return r.kind()
		}
	}
	return ""
}
