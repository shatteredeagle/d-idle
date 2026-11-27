// Package vaultkeep is the Keep queue of the Ethereal Vault (docs/vault.md, section Keep; slice V5 of
// docs/design/loot-vault-2026-11-16.md): it turns the records the player marked Keep into real items in the hero's stash
// at the next safe moment in town, exactly as dropped, and records the outcome in the vault log.
//
// Order per item (every step is a log line of the store, so a crash anywhere resumes, never creates twice):
//
//	Keep (keepq) -> BeginKeep -> item create (key=vk-<id>) -> read-back (item full == recorded bytes) -> save -> kept
//
// The create is logged BEFORE it is sent. A record that is still in flight after a restart, a failed save or a transport
// error is resolved first, by looking at the hero (Keeper.present), never by creating again.
package vaultkeep

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/bulkid"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Reasons shown on cards and rows; each is a specific cause, never a generic failure.
const (
	ReasonStashFull = "the stash is full: free space or upgrade"
	ReasonPackFull  = "the backpack is full"
	ReasonDead      = "the hero is dead"
	ReasonNoGame    = "not in a game"
	ReasonVerbs     = "this d2d build has no vault verbs (start it with --vault-verbs)"
	ReasonBusy      = "the game is busy"
	ReasonBadData   = "the game does not accept this item's data"
	ReasonVersion   = "the item was recorded by another game version"
	ReasonReadback  = "the created item differs from the recorded one"
	ReasonDup       = "the game already holds an item created with this key but it is not in the hero"
	ReasonOff       = "Keep is switched off (loot.vaultKeep 0, loot.vault 0 or a measured run)"
	ReasonBackup    = "the save backup failed"
	ReasonSave      = "the hero could not be saved"
	reasonPrefix    = "keep: "
	defaultPerVisit = 20
	defaultBudget   = 8 * time.Second
	keyPrefix       = "vk-"
)

// Reason strips the log prefix from a stored reason.
func Reason(stored string) string { return strings.TrimPrefix(stored, reasonPrefix) }

func stored(why string) string { return reasonPrefix + why }

// Keeper is the Keep queue and its executor for one hero. Zero values of the optional fields are usable; Store and API
// are required. All callbacks run unlocked and may block for one verb round trip.
type Keeper struct {
	Store    func() *vault.Store         // the hero's vault (nil: none yet)
	API      d2dctl.VaultAPI             // the fork verbs (a VaultFake offline)
	Enabled  func() bool                 // loot.vault and loot.vaultKeep on and no measured run; nil counts as on
	Ready    func() string               // "" when the hero is alive, in town, no blocking panel, cursor free; else the reason
	Backup   func(context.Context) error // before the first create of a session (the fork writes its .bak); required
	Save     func(context.Context) error // persists the created items (the fork's save verb with read-back); required
	Notice   func(string)                // a Needs-you info line, once per state change
	Now      func() time.Time
	PerVisit func() int    // loot.vaultKeepPerVisit
	Budget   time.Duration // most wall time of one Run before it yields (default 8 s)

	// Reveal and Sell (slice V6); nil Tools or Vendor leaves that errand off.
	Tools          Identifier
	Vendor         Seller
	SellPerVisit   func() int                                 // loot.vaultSellPerVisit
	Namer          func(d2dctl.VaultRecord, vault.Rec) string // display name of an identified record (default: the old name)
	ShortBy        func() int                                 // gold the town budget is short by (repair, potions, merc); 0 none
	Prompt         func() bool                                // loot.vaultSellPrompt
	AutoSell       func() bool                                // loot.vaultAutoTrim: marked (unordered) sales run without the player's confirm; nil or false: only ordered sales run
	Ether          *bulkid.EtherPayer                         // identify.etherPerReveal payer; nil or price 0: Ether is not used
	IdentifyPolicy func() int                                 // identify.policy after the measured-run override (nil: manual)

	// Reveal paid in gold (gold.go): the payer, the knobs and where the order is kept. nil Payer: no gold orders.
	Payer       GoldPayer
	GoldOn      func() bool     // loot.vaultRevealGold
	GoldConfirm func() int      // loot.vaultRevealGoldConfirm
	Methods     func() []string // loot.vaultRevealMethods: the payments the page offers (free, ether, gold); nil: all
	OrderFile   func() string
	Measured    func() bool // a measured run (D2BOT_RUN): a gold reveal is refused and never reaches the fork

	// The instant sale (sale.go): the credit verb, the price verb, the stash read and the knobs. nil Credit or Pricer: the old queue.
	Credit      d2dctl.GoldCreditor
	Pricer      d2dctl.Pricer
	StashGold   func(context.Context) (int, error) // the stash gold, fresh
	SellInstant func() bool                        // loot.vaultSellInstant
	SellConfirm func() int                         // loot.vaultSellConfirm
	SaleFile    func() string                      // where the sale order is kept

	smu     sync.Mutex // one sale at a time
	smem    *SaleOrder
	sloaded string
	prices  sync.Map // record hash and identified flag -> gold

	nop    sync.Map   // record ids whose bytes decoded to no properties (BackfillProps tries them once per process)
	pmu    sync.Mutex // serialises one gold order's unlock, payment and settle (taken before omu)
	omu    sync.Mutex // guards mem and loaded (never taken while mu is held)
	mem    *GoldOrder
	loaded string

	// One-tap Equip (equip.go); a nil Hand leaves it off.
	Hand      Hand
	EquipPath string                     // the orders file (empty: memory only)
	EquipOn   func() bool                // loot.vaultEquip; nil counts as on
	Chain     func(code string) []string // base type chain of an item code, for the record of a displaced item

	// Direct equip (equipdirect.go; ADR 0029 amended 2026-11-25): the player's Equip writes the item into its body slot and
	// the displaced items come back as vault records. nil Direct or DirectOn false: the order flow stays.
	Direct   d2dctl.DirectEquipper
	DirectOn func() bool // loot.vaultDirectEquip
	Pinner   func(slot string, unit uint32, name string)
	dcapsAt  time.Time
	dcapsOK  bool
	dcaps    bool

	CarryPath string // the carried charms file (carry.go); empty: memory only
	cb        carryBook

	eq         equipBook
	mu         sync.Mutex
	promptMute time.Time
	backedUp   bool
	tripped    string // a read-back mismatch switches creating off for the process
	info       Info
	tripWant   bool
	noticed    map[string]bool
	caps       d2dctl.VaultCaps // cached for capsTTL: the page polls, the socket is shared
	capsAt     time.Time
	capsOK     bool
}

const capsTTL = 60 * time.Second

// vaultCaps answers from the cache when it is fresh. With fetch false it never touches the socket (the page's poll).
func (k *Keeper) vaultCaps(ctx context.Context, fetch bool) (d2dctl.VaultCaps, bool, error) {
	k.mu.Lock()
	c, at, ok := k.caps, k.capsAt, k.capsOK
	k.mu.Unlock()
	if ok && k.now().Sub(at) < capsTTL || !fetch {
		return c, ok, nil
	}
	c, err := k.API.VaultCaps(ctx)
	if err != nil {
		return c, false, err
	}
	k.mu.Lock()
	k.caps, k.capsAt, k.capsOK = c, k.now(), true
	k.mu.Unlock()
	return c, true, nil
}

// Info is the visible state of the queue for the progress line and the Needs-you row.
type Info struct {
	Queued   int    `json:"queued"`            // waiting, not started
	Creating int    `json:"creating"`          // started and not confirmed
	Waiting  string `json:"waiting,omitempty"` // why the queue does not move right now
	Full     bool   `json:"full,omitempty"`    // the stash (or pack) has no room: a Needs-you row
	Tripped  string `json:"tripped,omitempty"` // creating is off until restart, and why
	Done     int    `json:"done"`              // created in this process

	RevQueued   int `json:"revQueued"`  // Reveal waiting
	RevWorking  int `json:"revWorking"` // Reveal with a step done in the game
	Revealed    int `json:"revealed"`   // revealed in this process
	SellQueued  int `json:"sellQueued"` // Sell waiting (ordered by the player, or marked with loot.vaultAutoTrim on)
	SellMarked  int `json:"sellMarked"` // marked for sale and waiting for the player's confirm (never sold by itself)
	SellWorking int `json:"sellWorking"`
	Sold        int `json:"sold"` // sold in this process
	GoldEarned  int `json:"goldEarned"`
}

func (k *Keeper) now() time.Time {
	if k.Now != nil {
		return k.Now()
	}
	return time.Now()
}

func (k *Keeper) enabled() bool {
	k.mu.Lock()
	t := k.tripped
	k.mu.Unlock()
	return t == "" && (k.Enabled == nil || k.Enabled())
}

func (k *Keeper) perVisit() int {
	if k.PerVisit != nil {
		if n := k.PerVisit(); n > 0 {
			return n
		}
	}
	return defaultPerVisit
}

func (k *Keeper) store() *vault.Store {
	if k.Store == nil {
		return nil
	}
	return k.Store()
}

func (k *Keeper) note(key, text string) {
	k.mu.Lock()
	if k.noticed == nil {
		k.noticed = map[string]bool{}
	}
	first := !k.noticed[key]
	k.noticed[key] = true
	k.mu.Unlock()
	if first && k.Notice != nil && text != "" {
		k.Notice(text)
	}
}

func (k *Keeper) clearNote(key string) {
	k.mu.Lock()
	delete(k.noticed, key)
	k.mu.Unlock()
}

func (k *Keeper) setWaiting(why string) {
	k.mu.Lock()
	k.info.Waiting = why
	k.mu.Unlock()
}

// Gate says whether a Keep press can be queued now. It refuses only what cannot become true by waiting for the next town
// visit: Keep off, a dead hero, a build without the verbs. Not being in town is not a refusal (the item queues).
func (k *Keeper) Gate(ctx context.Context) (ok bool, why string) { return k.gate(ctx, true) }

// Peek is Gate without any round trip: the cached capabilities (unknown counts as fine). The page's poll uses it.
func (k *Keeper) Peek() (ok bool, why string) { return k.gate(context.Background(), false) }

func (k *Keeper) gate(ctx context.Context, fetch bool) (ok bool, why string) {
	k.mu.Lock()
	t := k.tripped
	k.mu.Unlock()
	if t != "" {
		return false, "Keep stopped for this run: " + t
	}
	if !k.enabled() {
		return false, ReasonOff
	}
	if k.API != nil {
		if caps, known, err := k.vaultCaps(ctx, fetch); err == nil && known && !caps.ItemCreate {
			return false, ReasonVerbs
		}
	}
	if k.Ready != nil {
		if w := k.Ready(); w == ReasonDead {
			return false, ReasonDead
		}
	}
	return true, ""
}

// Result is the outcome of queueing one id.
type Result struct {
	ID      string
	OK      bool
	Already bool   // it was queued or kept already: a double press changes nothing
	Code    string // missing, state, target
	Why     string

	// Direct equip (equipdirect.go): what the one-tap path did, so the page can say it per piece.
	Direct  bool     // the swap was done by the direct verbs, at once
	Slot    string   // the body slot it went to
	Removed []string // names of the items that came off (they are vault records now)
	Where   string   // vault, stash or backpack (where the removed items are)
}

// Enqueue queues records for creation (target stash or inv; "" is the stash). It is idempotent: a record that is queued or
// kept answers OK with Already. wantNow also asks the bot for an errand trip home (Keep now); it never creates anything
// by itself. The create happens in Run, at a safe moment.
func (k *Keeper) Enqueue(ctx context.Context, ids []string, target string, wantNow bool) []Result {
	st := k.store()
	out := make([]Result, 0, len(ids))
	for _, id := range ids {
		r := Result{ID: id}
		switch {
		case st == nil:
			r.Code, r.Why = "missing", "there is no vault for this hero yet"
		default:
			rec, found := st.Get(id)
			switch {
			case !found:
				r.Code, r.Why = "missing", "no such item"
			case rec.State == vault.KeepQ || rec.State == vault.Kept:
				r.OK, r.Already = true, true
			default:
				if err := st.KeepTo(id, target); err != nil {
					r.Code, r.Why = codeOf(err), err.Error()
				} else {
					r.OK = true
				}
			}
		}
		out = append(out, r)
	}
	if wantNow {
		for _, r := range out {
			if r.OK {
				k.mu.Lock()
				k.tripWant = true
				k.mu.Unlock()
				break
			}
		}
	}
	return out
}

func codeOf(err error) string {
	switch {
	case errors.Is(err, vault.ErrNotFound):
		return "missing"
	case errors.Is(err, vault.ErrState):
		return "state"
	case errors.Is(err, vault.ErrInvalid):
		return "target"
	}
	return "internal"
}

// Pending is the number of records waiting for a create (queued or in flight).
func (k *Keeper) Pending() int {
	st := k.store()
	if st == nil {
		return 0
	}
	n := len(st.KeepQueued()) + k.sellRunnable(st.SellQueued()) + k.equipPending()
	if k.GoldPending() {
		n++
	}
	for _, r := range st.RevealQueued() {
		if r.State == vault.Recorded { // a keep-combined reveal is counted with its keep
			n++
		}
	}
	return n
}

// TripWanted reports whether the player pressed Keep now and the hero is not yet home; Run clears it.
func (k *Keeper) TripWanted() bool {
	pend := k.Pending() // before mu: the gold order has its own lock
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.tripWant && pend > 0
}

// Info reads the visible state.
func (k *Keeper) Info() Info {
	k.mu.Lock()
	in := k.info
	in.Tripped = k.tripped
	k.mu.Unlock()
	in.Queued, in.Creating = 0, 0
	if st := k.store(); st != nil {
		for _, r := range st.KeepQueued() {
			if r.InFlight {
				in.Creating++
			} else {
				in.Queued++
			}
		}
		for _, r := range st.RevealQueued() {
			if r.Stage != "" {
				in.RevWorking++
			} else {
				in.RevQueued++
			}
		}
		for _, r := range st.SellQueued() {
			if r.Stage != "" {
				in.SellWorking++
			} else if !r.Ordered && !k.autoSell() {
				in.SellMarked++
			} else {
				in.SellQueued++
			}
		}
	}
	return in
}

type created struct {
	rec   vault.Rec
	place string
}

// Run creates the queued Keep items, then runs the Reveal and Sell errands, each bounded by its per-visit cap and Budget.
// It returns one log line per notable event. Call it only at a safe moment in town; it checks Ready itself.
func (k *Keeper) Run(ctx context.Context) []string {
	lines := k.runKeep(ctx)
	lines = append(lines, k.resumeGold(ctx)...) // an order a restart or a lost reply left unsettled
	lines = append(lines, k.resumeSale(ctx)...)
	lines = append(lines, k.flushQueued(ctx)...)
	eq := k.runEquip(ctx)
	lines = append(lines, eq...)
	// A set bigger than the free pack cells goes in chunks: the pieces made so far are worn (or sent back to the vault),
	// which frees their cells, then the next ones are taken out, until nothing moves any more.
	for round := 0; round < maxSetRounds && len(eq) > 0 && ctx.Err() == nil && k.packWaiting(); round++ {
		lines = append(lines, k.runKeep(ctx)...)
		eq = k.runEquip(ctx)
		lines = append(lines, eq...)
	}
	lines = append(lines, k.runReveal(ctx)...)
	return append(lines, k.runSell(ctx)...)
}

// runGate is the shared start of every errand: enabled, hero ready, fork verbs present. It sets Waiting when it says no.
func (k *Keeper) runGate(ctx context.Context) bool {
	if !k.enabled() {
		k.setWaiting(ReasonOff)
		return false
	}
	if k.Ready != nil {
		if why := k.Ready(); why != "" {
			k.setWaiting(why)
			return false
		}
	}
	caps, _, err := k.vaultCaps(ctx, true)
	switch {
	case err != nil:
		k.setWaiting(ReasonBusy)
		return false
	case !caps.ItemCreate || !caps.ItemFull || !caps.ItemVault:
		k.setWaiting(ReasonVerbs)
		k.note("verbs", "Keep, Reveal and Sell are waiting: "+ReasonVerbs)
		return false
	}
	k.clearNote("verbs")
	k.setWaiting("")
	k.mu.Lock()
	k.tripWant = false
	k.mu.Unlock()
	return true
}

func (k *Keeper) runKeep(ctx context.Context) []string {
	st := k.store()
	if st == nil || len(st.KeepQueued()) == 0 {
		k.setWaiting("")
		return nil
	}
	if !k.runGate(ctx) {
		return nil
	}
	var lines []string
	var confirm []created
	deadline := k.now().Add(orDur(k.Budget, defaultBudget))
	stop := false

	// 1. what a restart or a failed save left in flight: look at the hero first
	var pack []d2dctl.VaultRecord
	havePack := false
	look := func() ([]d2dctl.VaultRecord, error) {
		if !havePack {
			p, err := k.API.ItemFull(ctx, -1)
			if err != nil {
				return nil, err
			}
			pack, havePack = p, true
		}
		return pack, nil
	}
	for _, r := range st.KeepQueued() {
		if !r.InFlight {
			continue
		}
		have, err := look()
		if err != nil {
			k.setWaiting(ReasonBusy)
			return append(lines, "keep: cannot read the hero, in-flight items stay as they are: "+err.Error())
		}
		if r.RevQ && k.Tools != nil {
			if c, line := k.resumeKeepReveal(ctx, st, r, have); c != nil {
				confirm = append(confirm, *c)
				if line != "" {
					lines = append(lines, line)
				}
			} else if line != "" {
				lines = append(lines, line)
			}
			havePack = false
		} else if k.presentIn(st, r, have) {
			confirm = append(confirm, created{r, placeOf(r)})
		} else if err := st.RetryKeep(r.ID, ""); err != nil {
			lines = append(lines, "keep: "+r.ID+" requeue: "+err.Error())
		} else {
			lines = append(lines, "keep: "+r.ID+" was not created before the restart, queued again")
		}
	}

	// 2. create
	k.mu.Lock()
	k.info.Full = false // set again below when the stash refuses
	k.mu.Unlock()
	made := 0
	queue := st.KeepQueued()
	// Backpack creates (the Equip orders) go first: a full stash halts the stash creates, and must not hold a set back.
	sort.SliceStable(queue, func(a, b int) bool { return queue[a].Where == vault.WhereInv && queue[b].Where != vault.WhereInv })
	for _, r := range queue {
		if stop || ctx.Err() != nil {
			break
		}
		if r.InFlight {
			continue // confirmed above, or unresolved: not ours to create again
		}
		if made >= k.perVisit() || !k.now().Before(deadline) {
			break
		}
		// the kill switch and the safe moment are judged again before every create, not only at the start
		if !k.enabled() {
			k.setWaiting(ReasonOff)
			break
		}
		if k.Ready != nil {
			if why := k.Ready(); why != "" {
				k.setWaiting(why)
				break
			}
		}
		if !k.backupOnce(ctx, &lines) {
			stop = true
			break
		}
		made++
		c, line, halt := k.createOne(ctx, st, r)
		if line != "" {
			lines = append(lines, line)
		}
		if c != nil {
			confirm = append(confirm, *c)
		}
		stop = halt
	}

	// 3. persist, then close the books
	if len(confirm) > 0 {
		if err := k.Save(ctx); err != nil {
			k.setWaiting(ReasonSave)
			k.note("save", "Keep: "+ReasonSave+" ("+err.Error()+"); the items stay in the queue's in-flight state and are confirmed after the next save")
			return append(lines, "keep: save failed, nothing confirmed: "+err.Error())
		}
		k.clearNote("save")
		for _, c := range confirm {
			if err := st.CompleteKeepAt(c.rec.ID, c.place); err != nil {
				lines = append(lines, "keep: "+c.rec.ID+" complete: "+err.Error())
				continue
			}
			k.mu.Lock()
			k.info.Done++
			k.mu.Unlock()
			lines = append(lines, fmt.Sprintf("keep: %s kept, %s", c.rec.Name, c.place))
		}
	}
	if !k.Info().Full {
		k.clearNote("full") // the next full stash is a new episode
	}
	return lines
}

func orDur(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}

func placeOf(r vault.Rec) string {
	if r.Where == vault.WhereInv {
		return "backpack"
	}
	return "stash"
}

// presentIn reports whether the hero carries the record: more items with its bytes than kept records of the same bytes
// the vault already counts (two identical items are legitimate; one already kept must not stand in for this one).
func (k *Keeper) presentIn(st *vault.Store, r vault.Rec, have []d2dctl.VaultRecord) bool {
	n := 0
	for _, h := range have {
		if strings.EqualFold(h.Norm, createHex(r)) {
			n++
		}
	}
	kept := 0
	for _, o := range st.Dump() {
		if o.State == vault.Kept && o.ID != r.ID && strings.EqualFold(o.Raw, r.Raw) {
			kept++
		}
	}
	return n > kept
}

// backupOnce takes the save backup before the first create of the session; a failure stops the batch (fail closed).
func (k *Keeper) backupOnce(ctx context.Context, lines *[]string) bool {
	k.mu.Lock()
	done := k.backedUp
	k.mu.Unlock()
	if done {
		return true
	}
	if k.Backup != nil {
		if err := k.Backup(ctx); err != nil {
			k.setWaiting(ReasonBackup)
			k.note("backup", "Keep is waiting: "+ReasonBackup+" ("+err.Error()+")")
			*lines = append(*lines, "keep: backup failed, no item created: "+err.Error())
			return false
		}
	}
	k.clearNote("backup")
	k.mu.Lock()
	k.backedUp = true
	k.mu.Unlock()
	return true
}

// createOne runs BeginKeep, the create and the read-back for one record. It returns the item to confirm after the save, a
// log line, and whether the batch must stop.
func (k *Keeper) createOne(ctx context.Context, st *vault.Store, r vault.Rec) (*created, string, bool) {
	fail := func(why string) (*created, string, bool) {
		if err := st.FailKeep(r.ID, stored(why)); err != nil {
			return nil, "keep: " + r.ID + " fail: " + err.Error(), true
		}
		return nil, fmt.Sprintf("keep: %s not kept: %s", r.Name, why), false
	}
	wait := func(why string, halt bool) (*created, string, bool) {
		_ = st.RetryKeep(r.ID, stored(why))
		k.setWaiting(why)
		return nil, fmt.Sprintf("keep: %s waits: %s", r.Name, why), halt
	}
	if err := d2dctl.ValidHex(r.Raw); err != nil {
		return fail(ReasonBadData)
	}
	if err := st.BeginKeep(r.ID); err != nil {
		return nil, "keep: " + r.ID + " begin: " + err.Error(), true
	}
	// Identify works on backpack items, so a Keep that also asks for Reveal lands in the backpack (the stash chore files it).
	o := d2dctl.CreateOpts{Pack: r.Where == vault.WhereInv || r.RevQ && !r.Identified && k.Tools != nil, Key: keyPrefix + r.ID}
	res, err := k.API.ItemCreate(ctx, createHex(r), o)
	if err != nil {
		switch {
		case errors.Is(err, d2dctl.ErrStashFull):
			why := ReasonStashFull
			if o.Pack {
				why = ReasonPackFull
			}
			k.mu.Lock()
			k.info.Full = true
			k.mu.Unlock()
			k.note("full", "Keep: "+why+". The item stays queued.")
			return wait(why, true)
		case errors.Is(err, d2dctl.ErrVaultState):
			return wait(deadOrNoGame(err), true)
		case errors.Is(err, d2dctl.ErrVaultOff), errors.Is(err, d2dctl.ErrVaultUnsupported):
			return wait(ReasonVerbs, true)
		case errors.Is(err, d2dctl.ErrBusy):
			return wait(ReasonBusy, true) // rejected before it ran: nothing was created
		case errors.Is(err, d2dctl.ErrBadHex):
			return fail(ReasonBadData)
		case errors.Is(err, d2dctl.ErrVaultVersion):
			return fail(ReasonVersion)
		case errors.Is(err, d2dctl.ErrVaultDup):
			have, lerr := k.API.ItemFull(ctx, -1)
			if lerr == nil && k.presentIn(st, r, have) {
				return &created{r, placeOf(r)}, "", false
			}
			if lerr != nil {
				return nil, "keep: " + r.ID + " dup, hero unreadable: " + lerr.Error(), true
			}
			return fail(ReasonDup)
		}
		var vr *d2dctl.VaultRefused
		if errors.As(err, &vr) {
			return fail(vr.Why)
		}
		// transport error or timeout: whether the item was created is unknown; it stays in flight and the next Run looks
		return nil, "keep: " + r.Name + " create unanswered, left in flight: " + err.Error(), true
	}

	back, err := k.API.ItemFull(ctx, res.ID)
	if err != nil || len(back) == 0 {
		return nil, fmt.Sprintf("keep: %s created (unit %d) but the read-back failed, left in flight", r.Name, res.ID), true
	}
	if !strings.EqualFold(back[0].Norm, createHex(r)) {
		why := ReasonReadback
		if _, uerr := k.API.ItemVault(ctx, res.ID); uerr != nil {
			why += "; it could not be removed again, check the stash"
		} else {
			why += "; the new copy was removed again"
		}
		k.trip("read-back mismatch")
		_, line, _ := fail(why)
		return nil, line, true
	}
	place := fmt.Sprintf("%s %d,%d", map[bool]string{true: "backpack", false: "stash"}[o.Pack], res.Col, res.Row)
	if r.RevQ && !r.Identified && k.Tools != nil {
		nr, why, unknown := k.revealInPlace(ctx, st, r, res.ID)
		switch {
		case unknown:
			return nil, fmt.Sprintf("keep: %s created, reveal %s; left in flight", r.Name, why), true
		case why != "":
			return &created{nr, place}, fmt.Sprintf("keep: %s kept unidentified, not revealed: %s", r.Name, why), false
		}
		return &created{nr, place}, "", false
	}
	return &created{r, place}, "", false
}

func deadOrNoGame(err error) string {
	var vr *d2dctl.VaultRefused
	if errors.As(err, &vr) && vr.Code == d2dctl.VaultErrNoGame {
		return ReasonNoGame
	}
	return ReasonDead
}

// NeedsText is the Needs-you sentence, "" when nothing needs the player.
func (k *Keeper) NeedsText() string {
	in := k.Info()
	switch {
	case in.Tripped != "":
		return "Keep is stopped: " + in.Tripped + ". Restart the bot after checking the stash."
	case in.Full && in.Queued+in.Creating > 0:
		return fmt.Sprintf("Stash full: free space or upgrade. %d kept item(s) wait.", in.Queued+in.Creating)
	}
	return ""
}

// ProgressLine is the sentence under the toolbar of the vault page, "" when the queue is empty.
func (k *Keeper) ProgressLine() string {
	in := k.Info()
	n := in.Queued + in.Creating
	if n == 0 {
		return ""
	}
	s := fmt.Sprintf("Keeping %d item%s", n, plural(n))
	switch {
	case in.Full:
		return s + ": the stash is full, free space or upgrade"
	case in.Waiting != "":
		return s + ": waiting, " + in.Waiting
	}
	return s + ": arrives at your next town visit"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// resumeKeepReveal settles an in-flight Keep that also asked for Reveal: it finds the created copy (unidentified, or
// already identified in the game) and finishes the reveal in place. A record whose copy is nowhere is queued again.
func (k *Keeper) resumeKeepReveal(ctx context.Context, st *vault.Store, r vault.Rec, have []d2dctl.VaultRecord) (*created, string) {
	if u := k.presentUnit(st, r.Raw, have); u != 0 {
		nr, why, unknown := k.revealInPlace(ctx, st, r, u)
		switch {
		case unknown:
			return nil, "keep: " + r.Name + " reveal " + why + "; left in flight"
		case why != "":
			return &created{nr, "backpack"}, fmt.Sprintf("keep: %s kept unidentified, not revealed: %s", r.Name, why)
		}
		return &created{nr, "backpack"}, ""
	}
	if h := k.adoptable(st, r, have); h != nil {
		if why := k.recordReveal(st, r, *h); why != "" {
			_ = st.RevealFail(r.ID, revStored(why))
			return &created{r, "backpack"}, "keep: " + r.Name + " kept, not revealed: " + why
		}
		nr, _ := st.Get(r.ID)
		return &created{nr, "backpack"}, ""
	}
	if err := st.RetryKeep(r.ID, ""); err != nil {
		return nil, "keep: " + r.ID + " requeue: " + err.Error()
	}
	return nil, "keep: " + r.ID + " was not created before the restart, queued again"
}

func (k *Keeper) waitLine(verb string, n int) string {
	in := k.Info()
	s := fmt.Sprintf("%s %d item%s", verb, n, plural(n))
	switch {
	case in.Tripped != "":
		return s + ": stopped, " + in.Tripped
	case in.Waiting != "":
		return s + ": waiting, " + in.Waiting
	}
	return s + ": at your next town visit"
}

// RevealLine is the progress sentence of Reveal requests, "" when none is queued.
func (k *Keeper) RevealLine() string {
	in := k.Info()
	if n := in.RevQueued + in.RevWorking; n > 0 {
		return k.waitLine("Revealing", n)
	}
	return ""
}

// SellLine is the progress sentence of the sell errand, "" when none is queued.
func (k *Keeper) SellLine() string {
	in := k.Info()
	if n := in.SellQueued + in.SellWorking; n > 0 {
		return k.waitLine("Selling", n)
	}
	return ""
}

func (k *Keeper) autoSell() bool { return k.AutoSell != nil && k.AutoSell() }

// sellRunnable counts the queued sales the errand may run now: in flight, ordered by the player, or any with AutoSell.
func (k *Keeper) sellRunnable(list []vault.Rec) int {
	n := 0
	for _, r := range list {
		if r.Stage != "" || r.Ordered || k.autoSell() {
			n++
		}
	}
	return n
}

// ConfirmMarked is the player's "Sell these N now": every marked record becomes an order.
func (k *Keeper) ConfirmMarked() (int, error) {
	st := k.store()
	if st == nil {
		return 0, nil
	}
	return st.ConfirmSell()
}

// OfferedMethods are the reveal payments the Vault page draws (hidden for now: user decision 2026-11-21, default only gold). The
// keeper still runs every route; this is what the page is told.
func (k *Keeper) OfferedMethods() []string {
	if k.Methods == nil {
		return []string{"free", "ether", "gold"}
	}
	return k.Methods()
}

// maxSetRounds bounds the take-out and wear rounds of one visit (a full body is ten pieces).
const maxSetRounds = 12

// packWaiting reports that a backpack create (an Equip order's) is queued and was held back because the pack had no room.
func (k *Keeper) packWaiting() bool {
	st := k.store()
	if st == nil {
		return false
	}
	for _, r := range st.KeepQueued() {
		if r.Where == vault.WhereInv && !r.InFlight && Reason(r.Reason) == ReasonPackFull {
			return true
		}
	}
	return false
}

// createHex is the item bytes the game is given for a record. A record that was revealed in the vault (gold, Ether:
// Identified and Unlocked) keeps the bytes the game dropped, which carry the affixes but the identified flag clear; the
// game refuses to wear such a unit ("identify it first", V live lab d2d devp28). Setting bit 4 of the first flag byte (the
// identified flag of the item header, S: the save format; U live until the lab run) creates the unit identified, so a
// revealed record comes out of the vault revealed, with no scroll and no Cain.
func createHex(r vault.Rec) string {
	if !r.Identified || !r.Unlocked {
		return r.Raw
	}
	b, err := hex.DecodeString(r.Raw)
	if err != nil || len(b) < 4 {
		return r.Raw
	}
	b[2] |= 0x10
	return hex.EncodeToString(b)
}
