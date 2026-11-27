package vaultkeep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"d2lod-bot/internal/bulkid"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// The instant vault sale (docs/vault.md "Sell is instant"; user decision 2026-11-22: "sell should just delete the items from the
// vault and give you the good value"). A sale is an ORDER the player gives on the vault page. The value of each record is what
// the engine's own sell function pays for its bytes (the fork's read-only `item price`), the sum is ADDED to the stash gold at
// once by the fork's `debug creditgold` (ledgered first, once per order id, never the purse, refused in a measured run and past
// the stash limit), and only then the records leave the vault (archived, then sold). No town visit, no item created in the hero.
//
//	queued -> credited -> done | (failed: nothing was credited, the records stay)
//
// The order is saved before the credit so a restart or a lost reply is settled again with the SAME order id (the fork answers a
// repeated id with credited 0). A credit the fork refused (room, measured run, flag off) ends the order failed and every record
// stays in the vault; a credit that went through and a record that cannot leave any more (pinned meanwhile) is reported in the
// note, never silently dropped.

// Sale states.
const (
	SaleQueued   = "queued"   // saved, the credit is not confirmed
	SaleCredited = "credited" // the stash took the gold; the records are leaving
	SaleDone     = "done"
	SaleFailed   = "failed"
)

// Reasons of the sale.
const (
	ReasonSaleOff      = "the instant sale is off (loot.vaultSellInstant is 0)"
	ReasonSaleNoCredit = "this bot cannot credit the stash gold"
	ReasonSaleNoFlag   = "this d2d build cannot credit gold or read prices (start it with --vault-pay and --vault-verbs, devp20 or later)"
	ReasonSaleMeasured = "a sale is never done in a measured run"
	saleMax            = 150
	saleShown          = 10 * time.Minute
)

// SaleLine is one record in a sale and the gold the engine pays for it.
type SaleLine struct {
	ID   string `json:"id"`
	Gold int    `json:"gold"`
	Name string `json:"name,omitempty"`
}

// SaleOrder is the one sale of a hero, stored next to the vault log so a restart keeps it.
type SaleOrder struct {
	ID          string       `json:"id"`
	Lines       []SaleLine   `json:"lines"`
	Total       int          `json:"total"`
	State       string       `json:"state"`
	Done        int          `json:"done"` // records that left the vault
	StashBefore int          `json:"stashBefore,omitempty"`
	StashAfter  int          `json:"stashAfter,omitempty"`
	Note        string       `json:"note,omitempty"`
	Ended       int64        `json:"ended,omitempty"`
	All         bool         `json:"all,omitempty"`     // a "Sell all" order (sellall.go): the bot listed the records itself
	Skipped     []SaleReason `json:"skipped,omitempty"` // All: the vault records left out and why
}

// Active reports whether the sale still moves.
func (o *SaleOrder) Active() bool {
	return o != nil && (o.State == SaleQueued || o.State == SaleCredited)
}

// SaleView is the sale as the page shows it.
type SaleView struct {
	ID          string       `json:"id"`
	State       string       `json:"state"`
	Items       int          `json:"items"`
	Done        int          `json:"done"`
	Gold        int          `json:"gold"`
	StashBefore int          `json:"stashBefore"`
	StashAfter  int          `json:"stashAfter"`
	Note        string       `json:"note,omitempty"`
	Line        string       `json:"line"`
	All         bool         `json:"all,omitempty"`
	Skipped     []SaleReason `json:"skipped,omitempty"`
}

// SaleSkip is one asked record that is not in the sale and why.
type SaleSkip struct {
	ID   string `json:"id"`
	Code string `json:"code"` // missing, state, pinned, busy, price, room
	Why  string `json:"why"`
}

// SaleReason groups the skips of one code, in the order of the codes first seen.
type SaleReason struct {
	Code string `json:"code"`
	N    int    `json:"n"`
	Why  string `json:"why"`
}

// SaleQuote is what a sale of the asked records would do, worked out without changing anything.
type SaleQuote struct {
	Asked       int          `json:"asked"`
	Lines       []SaleLine   `json:"lines"`
	Total       int          `json:"total"`
	Stash       int          `json:"stash"`
	StashAfter  int          `json:"stashAfter"`
	Room        int          `json:"room"` // gold the stash can still take
	Skips       []SaleSkip   `json:"skips,omitempty"`
	Reasons     []SaleReason `json:"reasons,omitempty"`
	ConfirmAt   int          `json:"confirmAt"`
	NeedConfirm bool         `json:"needConfirm"`
	Why         string       `json:"why,omitempty"` // when the sale cannot run at all
}

// SaleFacts is the knob state the page reads with every poll (no game call).
type SaleFacts struct {
	On        bool   `json:"on"`
	Why       string `json:"why,omitempty"`
	ConfirmAt int    `json:"confirmAt"`
	Limit     int    `json:"limit"`
}

func (k *Keeper) saleKnob() bool { return k.SellInstant != nil && k.SellInstant() }

func (k *Keeper) saleConfirm() int {
	if k.SellConfirm == nil {
		return 0
	}
	return k.SellConfirm()
}

// SaleOn says whether a sale can run now: the knob, the wiring, no measured run, and a fork that lists both verbs. Without it the
// old queue (sold at the next town visit) stays the way, so a bot on an older d2d still works.
func (k *Keeper) SaleOn(ctx context.Context) (bool, string) {
	switch {
	case !k.saleKnob():
		return false, ReasonSaleOff
	case k.Credit == nil || k.Pricer == nil || k.StashGold == nil:
		return false, ReasonSaleNoCredit
	case k.measured():
		return false, ReasonSaleMeasured
	case !k.enabled():
		return false, ReasonOff
	}
	caps, ok, err := k.vaultCaps(ctx, true)
	if err != nil || !ok {
		return false, ReasonBusy
	}
	if !caps.CreditGold || !caps.Price {
		return false, ReasonSaleNoFlag
	}
	return true, ""
}

// SaleFacts reports the instant sale's state from the cache only.
func (k *Keeper) SaleFacts(ctx context.Context) SaleFacts {
	f := SaleFacts{ConfirmAt: k.saleConfirm(), Limit: d2dctl.StashLimit}
	f.On, f.Why = k.saleKnob(), ""
	switch {
	case !f.On:
		f.Why = ReasonSaleOff
	case k.Credit == nil || k.Pricer == nil || k.StashGold == nil:
		f.On, f.Why = false, ReasonSaleNoCredit
	case k.measured():
		f.On, f.Why = false, ReasonSaleMeasured
	default:
		if caps, ok, _ := k.vaultCaps(ctx, false); ok && (!caps.CreditGold || !caps.Price) {
			f.On, f.Why = false, ReasonSaleNoFlag
		}
	}
	return f
}

func (k *Keeper) salePath() string {
	if k.SaleFile == nil {
		return ""
	}
	return k.SaleFile()
}

func (k *Keeper) saleOrder() *SaleOrder {
	k.omu.Lock()
	defer k.omu.Unlock()
	return k.saleLocked()
}

func (k *Keeper) saleLocked() *SaleOrder {
	path := k.salePath()
	if path == "" {
		return k.smem
	}
	if k.sloaded != path {
		k.smem, k.sloaded = nil, path
		if b, err := os.ReadFile(path); err == nil {
			var o SaleOrder
			if json.Unmarshal(b, &o) == nil && o.ID != "" {
				k.smem = &o
			}
		}
	}
	return k.smem
}

func (k *Keeper) putSale(o SaleOrder) error {
	k.omu.Lock()
	defer k.omu.Unlock()
	k.saleLocked()
	k.smem = &o
	path := k.salePath()
	if path == "" {
		return nil
	}
	b, err := json.Marshal(o)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (k *Keeper) saveSale(o SaleOrder) {
	if err := k.putSale(o); err != nil {
		k.note("saleorder", "The sale order could not be saved ("+err.Error()+"): it is kept until the bot restarts.")
	}
}

// value is the gold the engine pays for the record: the price of its bytes as identified when the player has revealed it (a
// reveal is what an identify scroll would do), else as recorded. Cached per record bytes and flag: the page polls.
func (k *Keeper) value(ctx context.Context, rec vault.Rec) (int, error) {
	id := rec.Identified || rec.Unlocked
	key := fmt.Sprintf("%s|%t", rec.Hash, id)
	if v, ok := k.prices.Load(key); ok {
		return v.(int), nil
	}
	p, err := k.Pricer.ItemPrice(ctx, rec.Raw, "")
	if err != nil {
		return 0, err
	}
	g := p.Sell
	if id {
		g = p.SellID
	}
	if g < 1 {
		g = 1
	}
	k.prices.Store(key, g)
	return g, nil
}

func (k *Keeper) saleSkip(ctx context.Context, id string, rec vault.Rec, found bool) (SaleLine, *SaleSkip) {
	switch {
	case !found:
		return SaleLine{}, &SaleSkip{ID: id, Code: "missing", Why: "no such item"}
	case rec.Pin:
		return SaleLine{}, &SaleSkip{ID: id, Code: "pinned", Why: "pinned items are never sold: unpin them first"}
	case rec.State != vault.Recorded && rec.State != vault.SellQ:
		return SaleLine{}, &SaleSkip{ID: id, Code: "state", Why: "only an item that is in the vault can be sold (it is " + string(rec.State) + ")"}
	case rec.Stage != "":
		return SaleLine{}, &SaleSkip{ID: id, Code: "busy", Why: "a reveal or an old sale of it is still running"}
	}
	g, err := k.value(ctx, rec)
	if err != nil {
		return SaleLine{}, &SaleSkip{ID: id, Code: "price", Why: "the engine price could not be read: " + err.Error()}
	}
	return SaleLine{ID: id, Gold: g, Name: rec.Name}, nil
}

var saleWords = map[string]string{
	"missing": "not in the vault",
	"state":   "already sold, kept or discarded",
	"pinned":  "pinned (never sold)",
	"busy":    "busy with another step",
	"price":   "price unreadable",
	"room":    "no room in the stash",
}

// SaleQuote works a sale out without changing anything: the records that would go (in the order asked), what the engine pays
// for them, the stash before and after, and every skipped record with its reason. The stash limit decides how many fit; the
// rest are skipped with code "room".
func (k *Keeper) SaleQuote(ctx context.Context, ids []string) SaleQuote {
	q := SaleQuote{ConfirmAt: k.saleConfirm()}
	ids = dedupe(ids)
	q.Asked = len(ids)
	if ok, why := k.SaleOn(ctx); !ok {
		q.Why = why
		return q
	}
	st := k.store()
	if st == nil {
		q.Why = "there is no vault for this hero yet"
		return q
	}
	stash, err := k.StashGold(ctx)
	if err != nil {
		q.Why = "the stash gold could not be read: " + err.Error()
		return q
	}
	q.Stash, q.Room = stash, max(d2dctl.StashLimit-stash, 0)
	for _, id := range ids {
		rec, found := st.Get(id)
		line, skip := k.saleSkip(ctx, id, rec, found)
		switch {
		case skip != nil:
			q.Skips = append(q.Skips, *skip)
		case q.Total+line.Gold > q.Room:
			q.Skips = append(q.Skips, SaleSkip{ID: id, Code: "room", Why: fmt.Sprintf("the stash holds %s of %s: room for %s more", bulkid.Thousands(stash), bulkid.Thousands(d2dctl.StashLimit), bulkid.Thousands(q.Room-q.Total))})
		default:
			q.Lines = append(q.Lines, line)
			q.Total += line.Gold
		}
	}
	q.StashAfter = stash + q.Total
	q.NeedConfirm = q.ConfirmAt > 0 && q.Total >= q.ConfirmAt
	for _, s := range q.Skips {
		i := slices.IndexFunc(q.Reasons, func(r SaleReason) bool { return r.Code == s.Code })
		if i < 0 {
			q.Reasons = append(q.Reasons, SaleReason{Code: s.Code, Why: s.Why})
			i = len(q.Reasons) - 1
		}
		q.Reasons[i].N++
	}
	for i := range q.Reasons {
		if w := saleWords[q.Reasons[i].Code]; w != "" && q.Reasons[i].Code != "room" {
			q.Reasons[i].Why = w
		}
	}
	return q
}

func dedupe(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (k *Keeper) saleView(o *SaleOrder) SaleView {
	v := SaleView{ID: o.ID, State: o.State, Items: len(o.Lines), Done: o.Done, Gold: o.Total, StashBefore: o.StashBefore, StashAfter: o.StashAfter, Note: o.Note, All: o.All, Skipped: o.Skipped}
	switch o.State {
	case SaleQueued:
		v.Line = fmt.Sprintf("Selling %d items for %s gold: waiting for the stash to confirm", len(o.Lines), bulkid.Thousands(o.Total))
	case SaleCredited:
		v.Line = fmt.Sprintf("Sold for %s gold: removing %d items from the vault", bulkid.Thousands(o.Total), len(o.Lines))
	case SaleDone:
		if o.All {
			v.Line = fmt.Sprintf("Sold %d for %s gold%s", o.Done, bulkid.Thousands(o.Total), skippedLine(o.Skipped))
			break
		}
		v.Line = fmt.Sprintf("Sold %d items for %s gold. Stash gold: %s", o.Done, bulkid.Thousands(o.Total), bulkid.Thousands(o.StashAfter))
	default:
		v.Line = "Not sold: " + o.Note
	}
	return v
}

// SaleView returns the sale the page shows (the active one, or the last for ten minutes), nil when none.
func (k *Keeper) SaleView() *SaleView {
	o := k.saleOrder()
	shown := saleShown
	if o != nil && o.All {
		shown = saleAllShown
	}
	if o == nil || (!o.Active() && o.Ended > 0 && k.now().Sub(time.Unix(o.Ended, 0)) > shown) {
		return nil
	}
	v := k.saleView(o)
	return &v
}

// SellNow gives the sale for ids and settles it before it returns. Refusals have a code: off, measured, noshop, ids, active,
// confirm, missing, state, pinned, busy, price, room, pay, gold. A cost at or over loot.vaultSellConfirm needs confirm.
func (k *Keeper) SellNow(ctx context.Context, ids []string, confirm bool) ([]Result, *SaleView) {
	ids = dedupe(ids)
	res := make([]Result, len(ids))
	for i, id := range ids {
		res[i].ID = id
	}
	failAll := func(code, why string) ([]Result, *SaleView) {
		for i := range res {
			res[i].Code, res[i].Why = code, why
		}
		return res, nil
	}
	if len(ids) == 0 || len(ids) > saleMax {
		return failAll("ids", fmt.Sprintf("name 1 to %d items", saleMax))
	}
	if ok, why := k.SaleOn(ctx); !ok {
		code := "off"
		switch why {
		case ReasonSaleMeasured:
			code = "measured"
		case ReasonSaleNoCredit:
			code = "noshop"
		}
		return failAll(code, why)
	}
	k.smu.Lock()
	defer k.smu.Unlock()
	if o := k.saleOrder(); o.Active() {
		if !sameSet(ids, lineIDs(o.Lines)) {
			return failAll("active", "a sale is still being settled: wait a moment")
		}
		k.settleSale(ctx, *o)
		return k.saleResults(res, k.saleOrder()), k.SaleView()
	}
	q := k.SaleQuote(ctx, ids)
	if q.Why != "" {
		return failAll("busy", q.Why)
	}
	at := map[string]int{}
	for i, r := range res {
		at[r.ID] = i
	}
	for _, s := range q.Skips {
		res[at[s.ID]].Code, res[at[s.ID]].Why = s.Code, s.Why
	}
	if len(q.Lines) == 0 {
		return res, nil
	}
	if q.NeedConfirm && !confirm {
		for _, l := range q.Lines {
			res[at[l.ID]].Code, res[at[l.ID]].Why = "confirm", fmt.Sprintf("this adds %s gold to your stash: confirm it", bulkid.Thousands(q.Total))
		}
		return res, nil
	}
	o := SaleOrder{ID: fmt.Sprintf("s-%d", k.now().UnixNano()), Lines: q.Lines, Total: q.Total, State: SaleQueued}
	k.saveSale(o)
	k.settleSale(ctx, o)
	return k.saleResults(res, k.saleOrder()), k.SaleView()
}

func lineIDs(ls []SaleLine) []string {
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = l.ID
	}
	return out
}

// saleResults fills the per-record results of ids that are in the order from the order's end state.
func (k *Keeper) saleResults(res []Result, o *SaleOrder) []Result {
	if o == nil {
		return res
	}
	in := map[string]bool{}
	for _, l := range o.Lines {
		in[l.ID] = true
	}
	st := k.store()
	for i := range res {
		if !in[res[i].ID] {
			continue
		}
		switch o.State {
		case SaleDone, SaleCredited:
			res[i].OK = true
			res[i].Code, res[i].Why = "", ""
			if rec, ok := st.Get(res[i].ID); ok && rec.State != vault.Sold {
				res[i].OK, res[i].Code, res[i].Why = false, "state", "the gold was credited but this item could not leave the vault: "+o.Note
			}
		case SaleFailed:
			res[i].Code, res[i].Why = salePayCode(o.Note), o.Note
		default:
			res[i].Code, res[i].Why = "pay", "the stash did not confirm yet; it is settled again at the next step"
		}
	}
	return res
}

func salePayCode(note string) string {
	switch note {
	case ReasonSaleMeasured:
		return "measured"
	case ReasonSaleNoFlag:
		return "payoff"
	}
	if strings.HasPrefix(note, "the stash has no room") {
		return "room"
	}
	return "pay"
}

// settleSale runs a sale from where it stands. The caller holds smu. A refusal ends it failed with every record in the vault; an
// unanswered credit leaves it queued so the next settle repeats it with the same order id.
func (k *Keeper) settleSale(ctx context.Context, o SaleOrder) {
	st := k.store()
	if st == nil || !o.Active() {
		return
	}
	end := func(state, note string) {
		o.State, o.Note, o.Ended = state, note, k.now().Unix()
		k.saveSale(o)
	}
	if o.State == SaleQueued {
		if k.measured() {
			end(SaleFailed, ReasonSaleMeasured)
			return
		}
		cr, err := k.Credit.CreditGold(ctx, o.Total, o.ID, len(o.Lines))
		switch {
		case err == nil:
			o.State, o.StashBefore, o.StashAfter = SaleCredited, cr.StashBefore, cr.StashAfter
			k.saveSale(o)
		case errors.Is(err, d2dctl.ErrCreditRoom):
			end(SaleFailed, "the stash has no room for that much gold: "+err.Error())
			return
		case errors.Is(err, d2dctl.ErrPayMeasured):
			end(SaleFailed, ReasonSaleMeasured)
			return
		case errors.Is(err, d2dctl.ErrPayOff):
			end(SaleFailed, ReasonSaleNoFlag)
			return
		case errors.Is(err, d2dctl.ErrPayAmount):
			end(SaleFailed, "the fork refused the amount: "+err.Error())
			return
		default: // unanswered: the gold may or may not be in the stash; the same order id settles it
			k.note("salepay", "A vault sale is waiting for the stash to confirm ("+err.Error()+"); it is tried again at the next step.")
			return
		}
	}
	k.clearNote("salepay")
	left := 0
	var why string
	var todo []vault.SoldLine
	for _, l := range o.Lines {
		rec, ok := st.Get(l.ID)
		if ok && rec.State == vault.Sold && rec.Reason == o.ID {
			continue // a first pass of this order took it already
		}
		todo = append(todo, vault.SoldLine{ID: l.ID, Gold: l.Gold})
	}
	// One batch: the per-record fsyncs of a whole-vault sale took seconds, and the page waits for this.
	if _, failed, err := st.SellNowBatch(todo, o.ID); err != nil {
		left, why = max(failed, 1), err.Error()
	}
	o.Done = 0
	for _, l := range o.Lines {
		if rec, ok := st.Get(l.ID); ok && rec.State == vault.Sold && rec.Reason == o.ID {
			o.Done++
		}
	}
	k.mu.Lock()
	k.info.Sold += o.Done
	k.info.GoldEarned += o.Total
	k.mu.Unlock()
	if left > 0 {
		end(SaleDone, fmt.Sprintf("%d item(s) stayed in the vault after the gold was credited (%s)", left, why))
		return
	}
	end(SaleDone, "")
}

// resumeSale settles a sale a restart or a lost reply left unsettled.
func (k *Keeper) resumeSale(ctx context.Context) []string {
	k.smu.Lock()
	defer k.smu.Unlock()
	o := k.saleOrder()
	if !o.Active() {
		return nil
	}
	if ok, _ := k.SaleOn(ctx); !ok {
		return nil
	}
	k.settleSale(ctx, *o)
	if f := k.saleOrder(); f != nil && !f.Active() {
		return []string{"vault sale " + f.State + ": " + k.saleView(f).Line}
	}
	return nil
}

// flushQueued sells the records the player ORDERED under the old queue (sold at the next town visit) the moment the instant sale
// is on: they were confirmed with a price shown, and the player's order stands. Marks (no order) are never touched.
func (k *Keeper) flushQueued(ctx context.Context) []string {
	st := k.store()
	if st == nil {
		return nil
	}
	if ok, _ := k.SaleOn(ctx); !ok {
		return nil
	}
	var ids []string
	for _, r := range st.SellQueued() {
		if r.Ordered && r.Stage == "" && !r.Pin {
			ids = append(ids, r.ID)
		}
	}
	var lines []string
	for len(ids) > 0 {
		n := min(len(ids), saleMax)
		_, v := k.SellNow(ctx, ids[:n], true)
		if v == nil || v.State != SaleDone {
			break
		}
		lines = append(lines, "vault sale: "+v.Line)
		ids = ids[n:]
	}
	return lines
}
