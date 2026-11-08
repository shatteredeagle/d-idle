package watch

import (
	"fmt"
	"strconv"
	"strings"

	"d2lod-bot/internal/currency"
)

const currencySchema = "watch.currency/1"

// SetCurrency attaches the currency wallet (d.idle mode). Without one, /api/currency answers 404 and the page shows
// no strip. Safe to call once before Serve.
func (h *Hub) SetCurrency(w *currency.Wallet) { h.cur.Store(w) }

// CurrencyRow is one currency in the header strip.
type CurrencyRow struct {
	ID      string   `json:"id"`
	Scope   string   `json:"scope"`
	Name    string   `json:"name"`
	Blurb   string   `json:"blurb"`
	Icon    string   `json:"icon"`
	Label   string   `json:"label"`
	Order   int      `json:"order"`
	Where   []string `json:"where"`
	Amount  *int64   `json:"amount"` // null until known (the bot has not reported the pack yet)
	Sealed  int64    `json:"sealed,omitempty"`
	Cap     int64    `json:"cap,omitempty"`
	Storage string   `json:"storage"`
}

// SpendRow is one upgrade on the spend sheet. Enabled is false unless the upgrade is wired and affordable.
type SpendRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Effect     string `json:"effect"`
	Tier       string `json:"tier"`
	Currency   string `json:"currency"`
	Level      int    `json:"level"`
	MaxLevel   int    `json:"maxLevel"`
	Cost       *int64 `json:"cost"` // null at the top level
	Affordable bool   `json:"affordable"`
	Enabled    bool   `json:"enabled"`
	Why        string `json:"why,omitempty"`
}

// CurrencyView is GET /api/currency.
type CurrencyView struct {
	Schema     string        `json:"schema"`
	Enabled    bool          `json:"enabled"`
	Mode       string        `json:"mode"`
	Text       string        `json:"text"`
	Strip      string        `json:"strip"`
	Error      string        `json:"error,omitempty"`
	Currencies []CurrencyRow `json:"currencies"`
	Spend      []SpendRow    `json:"spend"`
}

// Currency builds the view. It only reads: the ledger is verified on every read, a failed check shows as an error line
// and never as a guessed balance. nil when no wallet is attached.
func (h *Hub) Currency() *CurrencyView {
	w := h.cur.Load()
	if w == nil {
		return nil
	}
	v := &CurrencyView{Schema: currencySchema, Mode: w.Mode, Currencies: []CurrencyRow{}, Spend: []SpendRow{}}
	if w.Mode == "" || w.Mode == currency.VanillaMode || w.Ledger == nil {
		v.Mode = currency.VanillaMode
		v.Text = "Currencies are off in vanilla mode."
		v.Strip = v.Text
		return v
	}
	if w.Mode == currency.EventsMode {
		v.Text = "Marks and the bank need the d.idle overlay; the tally runs without it (see the tally strip)."
		v.Strip = v.Text
		return v
	}
	v.Enabled = true
	levels, err := w.Ledger.Levels()
	if err != nil {
		levels = map[string]int{}
		v.Error = "The ledger failed its check, so balances are hidden: " + err.Error()
	}
	bal := map[string]currency.Balance{}
	var parts []string
	for _, c := range w.Reg.Ordered() {
		row := CurrencyRow{ID: c.ID, Scope: c.Scope, Name: c.Name, Blurb: c.Blurb, Icon: c.Display.Icon, Label: c.Display.Label,
			Order: c.Display.Order, Where: append([]string{}, c.Display.Where...), Cap: c.Cap, Storage: c.Storage}
		switch {
		case c.Storage == currency.StorageItem:
			if n, ok := w.Carried(c.ID); ok {
				a := int64(n)
				row.Amount = &a
			}
		case v.Error == "":
			if b, err := w.Ledger.Balance(c.ID); err == nil {
				bal[c.ID] = b
				row.Amount, row.Sealed = &b.Spendable, b.Sealed
			}
		}
		v.Currencies = append(v.Currencies, row)
		if row.Amount != nil {
			parts = append(parts, fmt.Sprintf("%s %d %s", c.Name, *row.Amount, c.Display.Label))
		} else {
			parts = append(parts, c.Name+" -")
		}
	}
	v.Strip = strings.Join(parts, " · ")
	v.Text = "Marks the hero finds are banked in town as tally. Tally buys upgrades."
	for _, u := range w.Reg.Upgrades {
		lv := levels[u.ID]
		r := SpendRow{ID: u.ID, Name: u.Name, Effect: u.Effect, Tier: u.Tier, Currency: u.Currency, Level: lv, MaxLevel: len(u.Costs)}
		have := int64(0)
		if b, ok := bal[u.Currency]; ok {
			have = b.Spendable
		}
		switch {
		case lv >= len(u.Costs):
			r.Why = "Top level reached."
		default:
			cost := u.Costs[lv]
			r.Cost = &cost
			r.Affordable = have >= cost
			switch {
			case !r.Affordable:
				r.Why = "Needs " + strconv.FormatInt(cost-have, 10) + " more " + u.Currency + "."
			case !u.Wired:
				r.Why = "Not available yet."
			default:
				r.Enabled = true
			}
		}
		v.Spend = append(v.Spend, r)
	}
	return v
}
