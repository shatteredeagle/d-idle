package vaultkeep

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"d2lod-bot/internal/vault"
)

// The read-only vault check (card vault2; `d2bot ops vault check`): what a repair would have to look at, worked out from
// the files of one hero's vault directory alone. It never opens the store for writing, never talks to the game and
// changes nothing, so it can run next to a live bot.

// Finding kinds.
const (
	FindOrphanIntent = "orphan-intent" // a move that was written down and not settled (carry, stow, merc, direct equip, sale)
	FindDuplicate    = "duplicate"     // the same item bytes on the hero and as a live vault record
	FindUnreserved   = "unreserved"    // a sale took (or may have taken) gold for a record it does not hold
	FindOrphanHold   = "orphan-hold"   // a record reserved by an order that no sale file names as running
	FindLog          = "log"           // the vault log itself: skipped lines, a torn tail
)

// Finding is one line of the check.
type Finding struct {
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"` // record or order id
	Text string `json:"text"`
}

// CheckFiles are the journal files next to the vault log.
const (
	checkSale  = "sale.json"
	checkEquip = "equip.json"
	checkCarry = "carried.json"
	checkMerc  = "merc-intents.json"
)

func readJSON(path string, v any) bool {
	b, err := os.ReadFile(path)
	return err == nil && json.Unmarshal(b, v) == nil
}

// Check reads the vault in dir and its journals. heroItems is the norm hex of the items the hero holds (pack, stash,
// body), when the caller has them; nil compares only the items the journals know (the carried charms).
func Check(dir string, heroItems []string) ([]Finding, error) {
	recs, anomalies, torn, err := vault.Inspect(dir)
	if err != nil {
		return nil, err
	}
	var out []Finding
	add := func(kind, id, format string, a ...any) {
		out = append(out, Finding{Kind: kind, ID: id, Text: fmt.Sprintf(format, a...)})
	}
	if anomalies > 0 || torn {
		add(FindLog, "", "the log has %d skipped line(s), torn last line: %t", anomalies, torn)
	}
	by := map[string]vault.Rec{}
	live := map[string]string{} // bytes -> record id
	for _, r := range recs {
		by[r.ID] = r
		if r.State.Live() {
			live[strings.ToLower(r.Raw)] = r.ID
		}
	}

	var sale SaleOrder
	haveSale := readJSON(filepath.Join(dir, checkSale), &sale) && sale.ID != ""
	if haveSale {
		if sale.Active() {
			add(FindOrphanIntent, sale.ID, "sale order is %s (%d item(s), %d gold): the next town step settles it", sale.State, len(sale.Lines), sale.Total)
		}
		paid := sale.State == SaleCredited || sale.State == SaleDone
		for _, l := range sale.Lines {
			r, ok := by[l.ID]
			switch {
			case !ok || !r.State.Live():
			case sale.Active() && r.Hold != sale.ID:
				add(FindUnreserved, l.ID, "sale %s (%s) lists it for %d gold but does not hold it (state %s, hold %q)", sale.ID, sale.State, l.Gold, r.State, r.Hold)
			case !sale.Active() && paid:
				add(FindUnreserved, l.ID, "sale %s credited %d gold for it and it is still in the vault (%s)", sale.ID, l.Gold, r.State)
			}
		}
	}
	for _, r := range recs {
		if r.Hold != "" && !(haveSale && sale.Active() && sale.ID == r.Hold) {
			add(FindOrphanHold, r.ID, "reserved by order %s, which is not a running sale: the next town step releases it", r.Hold)
		}
	}

	var cf struct {
		Carried []Carried `json:"carried"`
	}
	readJSON(filepath.Join(dir, checkCarry), &cf)
	hero := append([]string(nil), heroItems...)
	for _, c := range cf.Carried {
		if c.State != "" {
			add(FindOrphanIntent, c.ID, "%s of %s was not settled", c.State, c.Name)
			continue
		}
		if r, ok := by[c.ID]; ok && r.State.Live() {
			add(FindDuplicate, c.ID, "%s is carried and its vault record is still %s", c.Name, r.State)
		}
		if heroItems == nil && c.Hex != "" {
			hero = append(hero, c.Hex)
		}
	}
	var mf struct {
		Intents []mercIntent `json:"intents"`
	}
	readJSON(filepath.Join(dir, checkMerc), &mf)
	for _, in := range mf.Intents {
		add(FindOrphanIntent, in.ID, "equip on the mercenary of %s stopped at step %q (order %s)", in.Name, in.Step, in.Order)
	}
	var ef equipFile
	readJSON(filepath.Join(dir, checkEquip), &ef)
	for _, o := range ef.Orders {
		switch {
		case o.Direct && o.active() && len(o.Removed) > 0:
			add(FindOrphanIntent, o.ID, "direct equip %s of %s: %d displaced item(s) are only in the order file", o.Order, o.Name, len(o.Removed))
		case o.active():
			add(FindOrphanIntent, o.ID, "equip order of %s is %s", o.Name, o.State)
		}
	}
	seen := map[string]bool{}
	for _, h := range hero {
		h = strings.ToLower(strings.TrimSpace(h))
		if id, ok := live[h]; ok && h != "" && !seen[id] {
			seen[id] = true
			add(FindDuplicate, id, "the hero holds an item with the bytes of this live record (%s); two identical items the player owns look the same", by[id].Name)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out, nil
}
