package vaultkeep

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Sell from the vault (slice V6, docs/vault.md section Sell): the gold flow for items the player did not keep. The player
// queues records; at the next town visit the bot creates each one into the pack, sells it at the real vendor through the
// town's own sell chore, and checks the result in the game: the copy must be gone and the purse must have grown. Only then
// the record is Sold with the gold read. Pinned records are never queued or sold.
//
//	sellq -> sb (logged, purse noted) -> item create -> sc (logged) -> sell chore -> read back (gone, gold up) -> sold

// Seller is the game side of the sale (cmd/d2bot adapts the town; tests use a fake).
type Seller interface {
	// Gold is the purse.
	Gold(ctx context.Context) (int, error)
	// Sell marks the pack items sell and runs the sell chore for them. ErrPostpone (wrapped) means "not now, nothing was
	// sold" (a fight, a busy panel); any other error is a final refusal for this item.
	Sell(ctx context.Context, units []int) error
}

// ErrPostpone is returned (wrapped) by Seller.Sell when the sale cannot run right now and nothing changed.
var ErrPostpone = errors.New("the sale is postponed")

// Reasons of the sale.
const (
	ReasonSellRefused = "no vendor of this act would buy it"
	ReasonSellNoGold  = "the item left the hero but the gold did not rise"
	sellPrefix        = "sell: "
	sellKey           = "vs-"
	defaultSellVisit  = 10
	promptMuteFor     = 30 * time.Minute
	promptMaxItems    = 8
)

// SellReason strips the log prefix from a stored sale reason ("" when it is not one).
func SellReason(stored string) string {
	if !strings.HasPrefix(stored, sellPrefix) {
		return ""
	}
	return strings.TrimPrefix(stored, sellPrefix)
}

func sellStored(why string) string { return sellPrefix + why }

func (k *Keeper) sellPerVisit() int {
	if k.SellPerVisit != nil {
		if n := k.SellPerVisit(); n > 0 {
			return n
		}
	}
	return defaultSellVisit
}

// GateSell says whether a Sell press can be queued now.
func (k *Keeper) GateSell(ctx context.Context) (ok bool, why string) {
	if k.Vendor == nil {
		return false, "this bot cannot sell vault items"
	}
	return k.Gate(ctx)
}

// EnqueueSell queues records for the sell errand. A pinned record is refused ("pinned"); a queued one answers Already.
func (k *Keeper) EnqueueSell(ctx context.Context, ids []string, wantNow bool) []Result {
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
			case rec.State == vault.SellQ && rec.Ordered:
				r.OK, r.Already = true, true
			case rec.State == vault.SellQ: // a mark: the player's press is the confirm
				if err := st.OrderSell(id); err != nil {
					r.Code, r.Why = codeOf(err), err.Error()
				} else {
					r.OK = true
				}
			case rec.Pin:
				r.Code, r.Why = "pinned", "pinned items are never sold: unpin it first"
			case rec.State != vault.Recorded:
				r.Code, r.Why = "state", "only an item that is in the vault can be sold (it is "+string(rec.State)+")"
			default:
				if err := st.OrderSell(id); err != nil {
					r.Code, r.Why = codeOf(err), err.Error()
					if errors.Is(err, vault.ErrPinned) {
						r.Code = "pinned"
					}
				} else {
					r.OK = true
				}
			}
		}
		out = append(out, r)
	}
	if wantNow {
		k.wantTrip(out)
	}
	return out
}

// runSell sells the queued records one by one, bounded by SellPerVisit and Budget. Resumes what a restart left.
func (k *Keeper) runSell(ctx context.Context) []string {
	st := k.store()
	if st == nil || k.Vendor == nil {
		return nil
	}
	if ok, _ := k.SaleOn(ctx); ok && !k.autoSell() && !slices.ContainsFunc(st.SellQueued(), func(r vault.Rec) bool { return r.Stage != "" }) {
		return nil // the instant sale took over (flushQueued ran before); only a sale a restart left in flight still finishes at the vendor
	}
	var list []vault.Rec
	for _, r := range st.SellQueued() {
		if r.Stage != "" || r.Ordered || k.autoSell() { // a mere mark never sells without loot.vaultAutoTrim or the player's confirm
			list = append(list, r)
		}
	}
	if len(list) == 0 {
		return nil
	}
	if !k.runGate(ctx) {
		return nil
	}
	var lines []string
	deadline := k.now().Add(orDur(k.Budget, defaultBudget))
	sold := 0
	for _, r := range list {
		if ctx.Err() != nil {
			break
		}
		if r.Stage != "" {
			have, err := k.API.ItemFull(ctx, -1)
			if err != nil {
				return append(lines, "sell: cannot read the hero, in-flight sales stay as they are: "+err.Error())
			}
			if line, halt := k.resumeSell(ctx, st, r, have); line != "" {
				lines = append(lines, line)
				if halt {
					break
				}
			}
			continue
		}
		if sold >= k.sellPerVisit() || !k.now().Before(deadline) {
			break
		}
		sold++
		line, halt := k.sellOne(ctx, st, r)
		if line != "" {
			lines = append(lines, line)
		}
		if halt {
			break
		}
	}
	return lines
}

// sellOne creates the record's item in the pack and sells it.
func (k *Keeper) sellOne(ctx context.Context, st *vault.Store, r vault.Rec) (string, bool) {
	if r.Pin {
		_ = st.FailSell(r.ID, sellStored("pinned items are never sold"))
		return "sell: " + r.Name + " is pinned, not sold", false
	}
	if err := d2dctl.ValidHex(r.Raw); err != nil {
		_ = st.FailSell(r.ID, sellStored(ReasonBadData))
		return "sell: " + r.Name + " not sold: " + ReasonBadData, false
	}
	var lines []string
	if !k.backupOnce(ctx, &lines) {
		return strings.Join(lines, "; "), true
	}
	gold, err := k.Vendor.Gold(ctx)
	if err != nil {
		k.setWaiting(ReasonBusy)
		return "sell: purse unreadable, nothing created: " + err.Error(), true
	}
	if err := st.SetStage(r.ID, vault.StageSellBegin, gold); err != nil {
		return "sell: " + r.ID + " begin: " + err.Error(), true
	}
	res, err := k.API.ItemCreate(ctx, r.Raw, d2dctl.CreateOpts{Pack: true, Key: fmt.Sprintf("%s%s-%d", sellKey, r.ID, r.Attempts+1)})
	if err != nil {
		wait := func(why string) (string, bool) {
			_ = st.SetStage(r.ID, "", 0)
			k.setWaiting(why)
			return "sell: " + r.Name + " waits: " + why, true
		}
		switch {
		case errors.Is(err, d2dctl.ErrStashFull):
			k.note("sellfull", "Sell: "+ReasonPackFull+". The sale stays queued.")
			return wait(ReasonPackFull)
		case errors.Is(err, d2dctl.ErrVaultState):
			return wait(deadOrNoGame(err))
		case errors.Is(err, d2dctl.ErrVaultOff), errors.Is(err, d2dctl.ErrVaultUnsupported):
			return wait(ReasonVerbs)
		case errors.Is(err, d2dctl.ErrBusy):
			return wait(ReasonBusy)
		case errors.Is(err, d2dctl.ErrBadHex):
			_ = st.FailSell(r.ID, sellStored(ReasonBadData))
			return "sell: " + r.Name + " not sold: " + ReasonBadData, false
		case errors.Is(err, d2dctl.ErrVaultVersion):
			_ = st.FailSell(r.ID, sellStored(ReasonVersion))
			return "sell: " + r.Name + " not sold: " + ReasonVersion, false
		case errors.Is(err, d2dctl.ErrVaultDup):
			return "sell: " + r.Name + " already created in this game, resolved at the next look", true
		}
		var vr *d2dctl.VaultRefused
		if errors.As(err, &vr) {
			_ = st.FailSell(r.ID, sellStored(vr.Why))
			return "sell: " + r.Name + " not sold: " + vr.Why, false
		}
		return "sell: " + r.Name + " create unanswered, left in flight: " + err.Error(), true
	}
	k.clearNote("sellfull")
	back, err := k.API.ItemFull(ctx, res.ID)
	if err != nil || len(back) == 0 {
		return fmt.Sprintf("sell: %s created (unit %d) but the read-back failed, left in flight", r.Name, res.ID), true
	}
	if !strings.EqualFold(back[0].Norm, r.Raw) {
		why := ReasonReadback
		if _, uerr := k.API.ItemVault(ctx, res.ID); uerr != nil {
			why += "; it could not be removed again, check the backpack"
		} else {
			why += "; the new copy was removed again"
		}
		k.trip("read-back mismatch")
		_ = st.FailSell(r.ID, sellStored(why))
		return "sell: " + r.Name + " not sold: " + why, true
	}
	if err := st.SetStage(r.ID, vault.StageSellMade, gold); err != nil {
		return "sell: " + r.ID + " stage: " + err.Error(), true
	}
	return k.sellUnit(ctx, st, r, res.ID, gold)
}

// sellUnit runs the sale of the created copy and checks it: the copy must be gone and the purse must have grown.
func (k *Keeper) sellUnit(ctx context.Context, st *vault.Store, r vault.Rec, unit, goldBefore int) (string, bool) {
	serr := k.Vendor.Sell(ctx, []int{unit})
	gone, gerr := k.gone(ctx, unit)
	after, perr := k.Vendor.Gold(ctx)
	if gerr != nil || perr != nil {
		return "sell: " + r.Name + " sold or not, the hero could not be read back; left in flight", true
	}
	if !gone {
		note := ""
		if _, err := k.API.ItemVault(ctx, unit); err != nil {
			_ = st.SetStage(r.ID, vault.StageSellMade, goldBefore)
			return "sell: " + r.Name + " is still in the backpack and could not be removed: " + err.Error(), true
		}
		switch {
		case errors.Is(serr, ErrPostpone):
			_ = st.SetStage(r.ID, "", 0) // stays queued
			k.setWaiting("the sale waits for a quiet moment")
			return "sell: " + r.Name + " waits for a quiet moment", true
		case serr != nil:
			note = serr.Error()
		default:
			note = ReasonSellRefused
		}
		_ = st.FailSell(r.ID, sellStored(note))
		return "sell: " + r.Name + " not sold: " + note, false
	}
	rise := after - goldBefore
	if rise <= 0 {
		k.trip("sale read-back: the gold did not rise")
		_ = st.SoldNote(r.ID, 0, sellStored(ReasonSellNoGold))
		return "sell: " + r.Name + " left the hero but the gold did not rise: sales stopped for this run, check the purse", true
	}
	if err := st.Sold(r.ID, rise); err != nil {
		return "sell: " + r.ID + " sold: " + err.Error(), true
	}
	k.mu.Lock()
	k.info.Sold++
	k.info.GoldEarned += rise
	k.mu.Unlock()
	return fmt.Sprintf("sell: %s sold for %d gold", r.Name, rise), false
}

// gone reports whether the carried unit no longer exists.
func (k *Keeper) gone(ctx context.Context, unit int) (bool, error) {
	recs, err := k.API.ItemFull(ctx, unit)
	if err != nil {
		var vr *d2dctl.VaultRefused
		if errors.As(err, &vr) && vr.Code == d2dctl.VaultErrItem {
			return true, nil
		}
		return false, err
	}
	return len(recs) == 0, nil
}

// resumeSell settles a sale a restart interrupted by looking at the hero.
func (k *Keeper) resumeSell(ctx context.Context, st *vault.Store, r vault.Rec, have []d2dctl.VaultRecord) (string, bool) {
	if u := k.presentUnit(st, r.Raw, have); u != 0 {
		gold, err := k.Vendor.Gold(ctx)
		if err != nil {
			return "sell: purse unreadable, in-flight sale stays as it is", true
		}
		if r.Stage == vault.StageSellBegin {
			_ = st.SetStage(r.ID, vault.StageSellMade, r.GoldPre)
		}
		_ = gold
		return k.sellUnit(ctx, st, r, u, r.GoldPre)
	}
	if r.Stage == vault.StageSellBegin { // the copy never appeared: the sale starts again
		_ = st.SetStage(r.ID, "", 0)
		return "sell: " + r.Name + " was not created before the restart, queued again", false
	}
	// created, now gone: it was sold. The purse tells how much, if it grew.
	gold, err := k.Vendor.Gold(ctx)
	if err != nil {
		return "sell: purse unreadable, in-flight sale stays as it is", true
	}
	rise := max(gold-r.GoldPre, 0)
	_ = st.SoldNote(r.ID, rise, sellStored("settled after a restart: the item was gone, the gold is the purse change"))
	k.mu.Lock()
	k.info.Sold++
	k.info.GoldEarned += rise
	k.mu.Unlock()
	return fmt.Sprintf("sell: %s sold (settled after a restart), about %d gold", r.Name, rise), false
}

// SellOffer is the opt-in "short on gold: sell these?" prompt (loot.vaultSellPrompt, default off): never sells by itself.
type SellOffer struct {
	Short int      `json:"short"`
	IDs   []string `json:"ids"`
	Gold  int      `json:"gold"` // the vault's lower-bound price of the listed items
	Names []string `json:"names"`
}

// maxPromptQuality is the highest quality the prompt offers: low (1) to magic (4); sets, rares, uniques and crafted items
// are never offered (the rare+ rule).
const maxPromptQuality = 4

// Offer builds the prompt, or nil when it is off, muted, nothing is short or nothing is offered. The list is the records
// of the lowest value class that are not pinned and not rare or better, highest price first, until the shortage is covered
// (at most promptMaxItems).
func (k *Keeper) Offer() *SellOffer {
	st := k.store()
	if st == nil || k.Prompt == nil || !k.Prompt() || k.ShortBy == nil || k.Vendor == nil || !k.enabled() {
		return nil
	}
	k.mu.Lock()
	muted := k.now().Before(k.promptMute)
	k.mu.Unlock()
	if muted {
		return nil
	}
	short := k.ShortBy()
	if short <= 0 {
		return nil
	}
	var c []vault.Rec
	st.Scan(func(r *vault.Rec) bool {
		if r.State == vault.Recorded && !r.Pin && r.Quality <= maxPromptQuality && !r.InFlight && r.Stage == "" {
			c = append(c, *r)
		}
		return true
	})
	if len(c) == 0 {
		return nil
	}
	low := c[0].Quality
	for _, r := range c {
		low = min(low, r.Quality)
	}
	var pool []vault.Rec
	for _, r := range c {
		if r.Quality == low {
			pool = append(pool, r)
		}
	}
	sort.SliceStable(pool, func(i, j int) bool { return pool[i].Value > pool[j].Value })
	o := &SellOffer{Short: short}
	for _, r := range pool {
		if o.Gold >= short || len(o.IDs) >= promptMaxItems {
			break
		}
		o.IDs, o.Gold, o.Names = append(o.IDs, r.ID), o.Gold+r.Value, append(o.Names, r.Name)
	}
	return o
}

// DismissOffer mutes the prompt for half an hour (the "Not now" button).
func (k *Keeper) DismissOffer() {
	k.mu.Lock()
	k.promptMute = k.now().Add(promptMuteFor)
	k.mu.Unlock()
}
