package vaultkeep

import (
	"context"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/vault"
)

// "Sell all" (docs/vault.md, section Sell all): ONE instant sale of every vault record that is not managed. The page sends no
// ids; the bot lists the records itself under the sale lock when the order runs, so a record that became part of an order
// between the two taps is left out. The sale itself is the instant sale of sale.go (same credit, same ledger line, same
// refusals), only the list is made here, from the same plan function (SaleQuote / saleSkip) a selection uses: nothing is
// protected differently, nothing is sold that a selection would have refused.

const (
	// ScopeAllSellable is the scope word of the order.
	ScopeAllSellable = "all_sellable"
	saleAllShown     = 10 * time.Second
	orderIDMax       = 64
	// CodeOrder is the skip code of a record that belongs to an active order (equip, keep, reveal, an earlier sale).
	CodeOrder = "order"
)

var sellAllWords = map[string]string{
	"pinned":  "pinned",
	CodeOrder: "in an order",
	"state":   "not in the vault any more",
	"missing": "not in the vault",
	"price":   "price unreadable",
	"room":    "no room in the stash",
}

// SellAllPlan is the whole-vault plan: what a Sell all would sell now (from the plan function), what is left out and why.
type SellAllPlan struct {
	Quote SaleQuote // lines, total, stash, room; Reasons are the plan's own plus "order"
	Live  int       // vault records considered (live ones)
	Order int       // records left out because an order holds them
}

// managed lists the ids an active order holds: equip orders, a running gold reveal, the sale in flight.
func (k *Keeper) managed() map[string]bool {
	m := map[string]bool{}
	k.mu.Lock()
	for _, o := range k.bookLocked().orders {
		if o.active() {
			m[o.ID] = true
		}
	}
	k.mu.Unlock()
	if g := k.order(); g.Active() {
		for _, id := range g.IDs {
			m[id] = true
		}
	}
	if s := k.saleOrder(); s.Active() {
		for _, l := range s.Lines {
			m[l.ID] = true
		}
	}
	return m
}

// heldBy names the open order or intent that holds record id ("" when none): an equip order (queued or direct), a carry
// or stow intent, a mercenary intent. One rule for Sell, Carry, Equip and Equip on mercenary (card fix1b, review N3): a
// record under an open order may already be worn or carried, so nothing else may take it until the order is settled.
func (k *Keeper) heldBy(id string) string {
	k.mu.Lock()
	for _, o := range k.bookLocked().orders {
		if o.ID == id && o.active() {
			k.mu.Unlock()
			return "an Equip order"
		}
	}
	k.mu.Unlock()
	for _, c := range k.CarriedList() {
		if c.ID == id && c.State != "" {
			return "a Carry or Stow"
		}
	}
	if k.mercOpen(id) {
		return "an Equip on mercenary order"
	}
	return ""
}

// ReasonHeld is the refusal for a record another open order holds.
func ReasonHeld(by string) string {
	return by + " for this item is still being settled: wait a moment"
}

// heldByOrder says why a live record may not be sold now (true: an order holds it). Pinned records are left to the plan
// function, which names them.
func heldByOrder(r vault.Rec, managed map[string]bool) bool {
	return managed[r.ID] || r.State == vault.KeepQ || (r.State == vault.SellQ && r.Ordered) || r.RevQ || r.InFlight || r.Stage != "" || r.Hold != ""
}

// SellAllPlan works the plan out without changing anything.
func (k *Keeper) SellAllPlan(ctx context.Context) SellAllPlan {
	var p SellAllPlan
	st := k.store()
	if st == nil {
		p.Quote.Why = "there is no vault for this hero yet"
		return p
	}
	man := k.managed()
	var ids []string
	for _, r := range st.Dump() {
		if !r.State.Live() {
			continue
		}
		p.Live++
		if !r.Pin && heldByOrder(r, man) {
			p.Order++
			continue
		}
		ids = append(ids, r.ID)
	}
	p.Quote = k.SaleQuote(ctx, ids)
	p.Quote.Asked = p.Live
	if p.Order > 0 {
		p.Quote.Reasons = append(p.Quote.Reasons, SaleReason{Code: CodeOrder, N: p.Order})
	}
	for i := range p.Quote.Reasons {
		r := &p.Quote.Reasons[i]
		if w := sellAllWords[r.Code]; w != "" && r.Code != "room" {
			r.Why = w
		}
	}
	return p
}

// SellAllCount is the number the button shows: records a sale would take, without the engine prices (the page polls it) and
// without the stash room (the quote of the first tap says that). It uses the same exclusions as SellAllPlan.
func (k *Keeper) SellAllCount() (n int, left []SaleReason) {
	st := k.store()
	if st == nil {
		return 0, nil
	}
	man := k.managed()
	var pinned, order int
	st.Scan(func(r *vault.Rec) bool {
		switch {
		case !r.State.Live():
		case r.Pin:
			pinned++
		case heldByOrder(*r, man):
			order++
		default:
			n++
		}
		return true
	})
	if pinned > 0 {
		left = append(left, SaleReason{Code: "pinned", N: pinned, Why: sellAllWords["pinned"]})
	}
	if order > 0 {
		left = append(left, SaleReason{Code: CodeOrder, N: order, Why: sellAllWords[CodeOrder]})
	}
	return n, left
}

// SellAllIDs is the list the old town-visit queue takes when the instant sale is not available (the fork lacks the verbs).
func (k *Keeper) SellAllIDs() []string {
	st := k.store()
	if st == nil {
		return nil
	}
	man := k.managed()
	var ids []string
	for _, r := range st.Dump() {
		if r.State.Live() && !r.Pin && r.State != vault.SellQ && !heldByOrder(r, man) {
			ids = append(ids, r.ID)
		}
	}
	return ids
}

// SellAllOutcome is the answer of SellAll: Code and Why when nothing was sold, the sale view otherwise.
type SellAllOutcome struct {
	Code string
	Why  string
	View *SaleView
	Plan SellAllPlan
}

// cleanOrderID makes the client's order id fit the fork's rule (1..64 of letters, digits and . _ : -).
func cleanOrderID(id string) string {
	var b strings.Builder
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == ':', c == '-':
			b.WriteRune(c)
		}
	}
	s := b.String()
	if len(s) > orderIDMax-3 {
		s = s[:orderIDMax-3]
	}
	return s
}

// SellAll sells everything that is not managed, as ONE order with the client's id: the same id again (a double tap, a resend
// after a lost reply) settles that order and never makes a second one, so nothing is sold or credited twice.
func (k *Keeper) SellAll(ctx context.Context, clientID string) SellAllOutcome {
	cid := cleanOrderID(clientID)
	if cid == "" {
		return SellAllOutcome{Code: "ids", Why: "send an order id"}
	}
	oid := "sa-" + cid
	if ok, why := k.SaleOn(ctx); !ok {
		return SellAllOutcome{Code: saleOffCode(why), Why: why}
	}
	k.actMu.Lock()
	defer k.actMu.Unlock()
	k.smu.Lock()
	defer k.smu.Unlock()
	if o := k.saleOrder(); o != nil && o.ID == oid {
		if o.Active() {
			k.settleSale(ctx, *o)
		}
		return SellAllOutcome{View: k.SaleView()}
	}
	if o := k.saleOrder(); o.Active() {
		return SellAllOutcome{Code: "active", Why: "a sale is still being settled: wait a moment"}
	}
	var p SellAllPlan
	_, held, err := k.quoteReserved(oid, true, func() SaleQuote { p = k.SellAllPlan(ctx); return p.Quote })
	if err != nil {
		return SellAllOutcome{Code: "busy", Why: err.Error(), Plan: p}
	}
	if p.Quote.Why != "" {
		return SellAllOutcome{Code: "busy", Why: p.Quote.Why, Plan: p}
	}
	if len(p.Quote.Lines) == 0 {
		why := "nothing in the vault can be sold now"
		if l := skippedLine(p.Quote.Reasons); l != "" {
			why += ":" + strings.TrimPrefix(l, ";")
		}
		return SellAllOutcome{Code: "state", Why: why, Plan: p}
	}
	if !held {
		return SellAllOutcome{Code: "busy", Why: "the vault could not reserve the items: nothing was sold", Plan: p}
	}
	o := SaleOrder{ID: oid, Lines: p.Quote.Lines, Total: p.Quote.Total, State: SaleQueued, All: true, Skipped: p.Quote.Reasons}
	if err := k.startSale(o); err != nil {
		return SellAllOutcome{Code: "journal", Why: err.Error(), Plan: p}
	}
	k.settleSale(ctx, o)
	return SellAllOutcome{View: k.SaleView(), Plan: p}
}

func saleOffCode(why string) string {
	switch why {
	case ReasonSaleMeasured:
		return "measured"
	case ReasonSaleNoCredit:
		return "noshop"
	}
	return "off"
}

// skippedLine words the left-out groups: "; 9 skipped: 4 pinned, 5 in an order" ("" when none).
func skippedLine(rs []SaleReason) string {
	var parts []string
	n := 0
	for _, r := range rs {
		w := sellAllWords[r.Code]
		if w == "" {
			w = r.Why
		}
		if r.Code == "room" {
			w = sellAllWords["room"]
		}
		parts = append(parts, fmt.Sprintf("%d %s", r.N, w))
		n += r.N
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("; %d skipped: %s", n, strings.Join(parts, ", "))
}
