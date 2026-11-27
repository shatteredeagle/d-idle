package main

import (
	"fmt"
	"hash/fnv"
	"strings"
	"sync"

	"d2lod-bot/internal/advisor"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/vault"
)

// The runeword and cube advisor on the Return screen (docs/design/advisor-r14.md, knob items.advisor). TEXT ONLY: the
// source reads the hero snapshot and the vault and returns lines; it never moves, sockets or cubes an item (owner rule:
// inventory management beyond starter gear and potions is the player's).

// adviceRow is one item base from the game tables: its code, display name and type chain (most specific first).
type adviceRow func(class int) (code, name string, chain []string, ok bool)

// adviceQuality words the d2 quality numbers the way advisor.Base expects them.
var adviceQuality = map[int]string{1: "low", 2: "normal", 3: "superior", 4: "magic", 5: "set", 6: "rare", 7: "unique", 8: "crafted"}

// rowsFromDB reads adviceRow from the item tables; nil db: nil.
func rowsFromDB(db *data.DB) adviceRow {
	if db == nil {
		return nil
	}
	return func(class int) (string, string, []string, bool) {
		if class < 0 || class >= len(db.Items) || db.Items[class] == nil {
			return "", "", nil, false
		}
		b := db.Items[class]
		return b.Code, b.Display, typeChain(db, b.Type), true
	}
}

// typeChain is typ followed by every type it inherits from (breadth first over Equiv1 and Equiv2).
func typeChain(db *data.DB, typ string) []string {
	var out []string
	seen := map[string]bool{}
	for q := []string{typ}; len(q) > 0; q = q[1:] {
		c := q[0]
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
		if t := db.ItemType(c); t != nil {
			q = append(q, t.Equiv[0], t.Equiv[1])
		}
	}
	return out
}

// heroStock gathers the pack, cube and stash of the snapshot. sockets reads the socket count of one item (the base
// table only knows the maximum; nil: no base is a candidate, runes and gems still count). Filled sockets are not read
// (U): the d2d socket count is the total, so a base that already holds runes may still be offered.
func heroStock(items []d2.Unit, row adviceRow, sockets func(id uint32) int) advisor.Stock {
	var s advisor.Stock
	if row == nil {
		return s
	}
	for _, u := range items {
		if u.Item == nil {
			continue
		}
		where := ""
		switch u.Item.Where {
		case d2.ContInventory, d2.ContCube:
			where = "pack"
		case d2.ContStash:
			where = "stash"
		default:
			continue
		}
		code, name, chain, ok := row(u.Class)
		if !ok {
			continue
		}
		n := 0
		if sockets != nil {
			n = sockets(u.ID)
		}
		s.Add(code, name, chain, adviceQuality[u.Item.Quality], n, 0, where)
	}
	return s
}

// adviceSource builds the advice lines and caches them by a fingerprint of what the hero and the vault hold, so the
// advisor runs when the stock changed and not on every page build.
type adviceSource struct {
	on      func() bool // items.advisor
	ladder  func() bool // items.ladderRunewords
	class   func() string
	snap    func() *d2.Snapshot
	row     adviceRow
	sockets func(id uint32) int
	store   func() *vault.Store // nil or nil result: no vault stock

	mu    sync.Mutex
	key   string
	lines []string
	runs  int // advisor runs, for the tests
}

// Lines is watch.FunConfig.Advice. Knob 0: nil, so the section hides.
func (a *adviceSource) Lines() []string {
	if a.on == nil || !a.on() || a.snap == nil || a.row == nil {
		return nil
	}
	class := strings.ToLower(a.class())
	snap := a.snap()
	if class == "" || snap == nil || snap.Inventory == nil {
		return nil
	}
	var recs []vault.Rec
	if a.store != nil {
		if st := a.store(); st != nil {
			recs, _ = st.Query(vault.Filter{})
		}
	}
	ladder := a.ladder != nil && a.ladder()
	key := adviceKey(class, ladder, snap.Inventory.Items, recs)
	a.mu.Lock()
	defer a.mu.Unlock()
	if key == a.key {
		return a.lines
	}
	st := heroStock(snap.Inventory.Items, a.row, a.sockets)
	st.AddVault(recs)
	lines, err := advisor.Lines(st, advisor.Options{Class: class, Ladder: ladder})
	if err != nil {
		lines = nil // a class the table does not know: no advice, never a guess
	}
	a.key, a.lines = key, lines
	a.runs++
	return lines
}

// adviceKey fingerprints the inputs: item ids, bases and places of the hero, the vault records and their update times.
func adviceKey(class string, ladder bool, items []d2.Unit, recs []vault.Rec) string {
	h := fnv.New64a()
	fmt.Fprint(h, class, ladder)
	for _, u := range items {
		if u.Item != nil {
			fmt.Fprint(h, "|", u.ID, u.Class, u.Item.Where, u.Item.Flags)
		}
	}
	for i := range recs {
		fmt.Fprint(h, "|", recs[i].ID, recs[i].State, recs[i].Updated.UnixNano())
	}
	return fmt.Sprintf("%x", h.Sum64())
}
