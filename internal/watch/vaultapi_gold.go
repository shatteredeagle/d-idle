package watch

import (
	"context"
	"net/http"

	"d2lod-bot/internal/bulkid"
	"d2lod-bot/internal/vault"
)

// Reveal paid in gold (docs/vault.md, "Reveal paid in gold"; ADR 0029): the order routes and the Reveal bar the page draws.
// An order settles in the same request: the stash pays, the items are revealed, the answer carries the order.

const (
	VaultRevealGoldRoute       = "/api/vault/revealgold"        // POST {ids | scope, maxGold, confirm}
	VaultRevealGoldCancelRoute = "/api/vault/revealgold/cancel" // POST {}
)

// VaultGoldOrder is the gold order as the page shows it (cmd/d2bot copies internal/vaultkeep.GoldView).
type VaultGoldOrder struct {
	ID         string `json:"id"`
	State      string `json:"state"` // queued, paid, revealed, partial, failed, cancelled
	Total      int    `json:"total"` // items the stash pays for
	Want       int    `json:"want"`  // items the player asked for
	Done       int    `json:"done"`
	Paid       int    `json:"paid"`
	StashAfter int    `json:"stashAfter"` // the stash gold after the payment
	Note       string `json:"note,omitempty"`
	Line       string `json:"line"`
}

// VaultGoldInfo is what the bar needs of the stash gold and the order, read per request.
type VaultGoldInfo struct {
	On         bool            `json:"on"`         // loot.vaultRevealGold with a payer wired, not a measured run
	ScrollGold int             `json:"scrollGold"` // the price of one reveal
	Stash      int             `json:"stash"`      // the stash gold a gold reveal pays from
	ConfirmAt  int             `json:"confirmAt"`
	Decode     bool            `json:"decode,omitempty"` // the fork decodes the affixes of an unidentified item from its bytes (item decode): any unidentified record can be gold-revealed
	Why        string          `json:"why,omitempty"`    // why gold cannot be offered now
	Unrevealed int             `json:"unrevealed"`       // records of the WHOLE vault a gold reveal can still pay for (the "Identify all" count; no filter applies)
	Order      *VaultGoldOrder `json:"order,omitempty"`
}

// VaultGoldActor is the optional part of a VaultActor that runs gold orders.
type VaultGoldActor interface {
	GoldInfo(ctx context.Context) VaultGoldInfo
	EnqueueRevealGold(ctx context.Context, ids []string, maxGold int, confirm bool) ([]VaultKeepResult, *VaultGoldOrder)
	// EnqueueRevealGoldAll is the order of scope all_unrevealed: the bot lists the records itself when it runs.
	EnqueueRevealGoldAll(ctx context.Context, maxGold int, confirm bool) ([]VaultKeepResult, *VaultGoldOrder)
	CancelGold() (VaultGoldOrder, bool)
}

// bar quotes the Reveal bar for the filtered records.
func (d *VaultDesk) bar(ctx context.Context, recs []vault.Rec, r VaultRevealView, gi VaultGoldInfo, hasGold bool) *bulkid.Bar {
	in := bulkid.BarIn{Free: r.Free, Charges: r.Charges, EtherPer: r.EtherPer, EtherBal: r.Balance}
	if hasGold {
		in.GoldOn, in.ScrollGold, in.Stash, in.ConfirmAt, in.GoldWhy, in.Decode = gi.On, gi.ScrollGold, gi.Stash, gi.ConfirmAt, gi.Why, gi.Decode
		if gi.Order != nil && (gi.Order.State == "queued" || gi.Order.State == "paid") {
			in.ActiveGoldOrderID = gi.Order.ID
		}
	}
	b := bulkid.MakeBar(bulkid.Candidates(d.MassFilter, recs, d.Quality), in).Only(r.Methods)
	return &b
}

// goldRoute serves the order and its cancel. Like an Ether reveal it needs no game step and no gate: the keeper refuses
// by itself (knob off, measured run, no payer, the stash short).
func (a *VaultAPI) goldRoute(w http.ResponseWriter, r *http.Request, d *VaultDesk, route string, b *vaultBody) {
	ga, ok := d.Act.(VaultGoldActor)
	if d.Act == nil || !ok {
		writeJSON(w, http.StatusConflict, policyReply{Code: "off", Error: "this bot cannot pay a reveal in gold"})
		return
	}
	type reply struct {
		vaultReply
		Order *VaultGoldOrder `json:"order,omitempty"`
	}
	if route == VaultRevealGoldCancelRoute {
		o, did := ga.CancelGold()
		rep := reply{vaultReply: vaultReply{OK: did, Done: []string{}}}
		if o.ID != "" {
			rep.Order = &o
		}
		code := 200
		if !did {
			code = http.StatusConflict
		}
		writeJSON(w, code, rep)
		return
	}
	var res []VaultKeepResult
	var order *VaultGoldOrder
	if b.Scope == bulkid.ScopeAllUnrevealed {
		res, order = ga.EnqueueRevealGoldAll(r.Context(), b.MaxGold, b.Confirm)
	} else {
		res, order = ga.EnqueueRevealGold(r.Context(), b.IDs, b.MaxGold, b.Confirm)
	}
	rep := reply{vaultReply: vaultReply{Done: []string{}}, Order: order}
	for _, x := range res {
		switch {
		case x.OK && x.Already:
			rep.Already = append(rep.Already, x.ID)
			rep.Done = append(rep.Done, x.ID)
		case x.OK:
			rep.Done = append(rep.Done, x.ID)
		default:
			rep.Failed = append(rep.Failed, vaultFail{ID: x.ID, Code: x.Code, Error: x.Why})
		}
	}
	rep.OK = len(rep.Failed) == 0
	code := 200
	if len(rep.Done) == 0 {
		code = http.StatusConflict
	}
	writeJSON(w, code, rep)
}
