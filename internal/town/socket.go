package town

import (
	"context"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/gear"
)

// Larzuk's socket reward (Act 5, Siege on Harrogath, once per difficulty). U live: see SocketFlow.

// ExpectedSockets is how many sockets Larzuk should give a base: its gemsockets, capped by the item type's limit for the
// item level (itemtypes MaxSock1/25/40 columns, V tables; the "Larzuk gives the maximum" rule is S, docs/items.md 1.5).
func ExpectedSockets(db *data.DB, b *data.ItemBase, ilvl int) int {
	n := b.Sockets
	if ty := db.ItemType(b.Type); ty != nil {
		tier := 0
		switch {
		case ilvl >= 40:
			tier = 2
		case ilvl >= 25:
			tier = 1
		}
		if m := ty.MaxSock[tier]; m > 0 && m < n {
			n = m
		}
	}
	return n
}

// bestRuneword is the best gear.RunewordScore (and its name) among the runewords that fit the base once socketed: type
// allowed, rune count equal to the expected sockets, available offline, in want when want is non-empty. 0: none.
func bestRuneword(db *data.DB, u d2.Unit, want []string) (float64, string) {
	if u.Item == nil || u.Class < 0 || u.Class >= len(db.Items) {
		return 0, ""
	}
	b := db.Items[u.Class]
	n := ExpectedSockets(db, b, u.Item.ILvl)
	allow := map[string]bool{}
	for _, w := range want {
		allow[w] = true
	}
	var bestV float64
	var bestN string
	for _, rw := range db.Runewords {
		if len(rw.Runes) != n || (len(allow) > 0 && !allow[rw.Name]) || !db.RunewordAvailableOffline(rw.Name) || !typeAllowed(db, b.Type, rw) {
			continue
		}
		if v := gear.RunewordScore(rw.Name); v > bestV {
			bestV, bestN = v, rw.Name
		}
	}
	return bestV, bestN
}

func typeAllowed(db *data.DB, typ string, rw *data.Runeword) bool {
	for _, ex := range rw.ExcludeTypes {
		if db.IsA(typ, ex) {
			return false
		}
	}
	for _, it := range rw.ItemTypes {
		if db.IsA(typ, it) {
			return true
		}
	}
	return false
}

// PickSocket chooses the item Larzuk should socket: the plain base that carries the most valuable runeword at the
// socket count it will get (docs/items.md 3.2: "the best white base you hold"), else the plain base with the best
// score; ties go to the worn one, then the lower id. Pure over the snapshot.
func PickSocket(db *data.DB, p *d2.Player, inv *d2.Inventory, stats StatsFn, want []string) (Pick, bool) {
	cands, _ := candidates(db, p, inv, stats)
	var best Pick
	found := false
	for _, c := range cands {
		v, rw := bestRuneword(db, c.Unit, want)
		r := v*1000 + c.Score
		worn := c.Unit.Item.Where == d2.ContEquipment
		if found && (r < best.Score || r == best.Score && !(worn && !best.Worn)) {
			continue
		}
		why := fmt.Sprintf("best plain base by score %.0f", c.Score)
		if rw != "" {
			why = fmt.Sprintf("base for %s with %d sockets", rw, ExpectedSockets(db, db.Items[c.Unit.Class], c.Unit.Item.ILvl))
		}
		best, found = Pick{Unit: c.Unit, Name: c.Name, Score: r, Worn: worn, Why: why}, true
	}
	return best, found
}

// socketedProof: the item (same id: sockets do not recreate it, U) carries the socketed flag or a socket count.
func (t *Town) socketedProof(before d2.Unit, s *d2.Snapshot) bool {
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if u.ID == before.ID && (u.Item.Socketed() || t.statsOf(u).Get(statSockets) > 0) {
			return true
		}
	}
	return false
}

// Socket talks to Larzuk and picks the item.
func (t *Town) Socket(ctx context.Context, w SocketWant) (ServiceReport, error) {
	return t.itemService(ctx, SocketFlow, func(s *d2.Snapshot) (Pick, error) {
		if w.Item != 0 {
			u, ok := findIn(s.Inventory, w.Item)
			if !ok {
				return Pick{}, fmt.Errorf("%w: item %d", ErrNoTarget, w.Item)
			}
			return Pick{Unit: u, Name: t.nameOf(u), Worn: u.Item.Where == d2.ContEquipment, Why: "asked for"}, nil
		}
		pk, ok := PickSocket(t.e.Data, s.Player, s.Inventory, t.statsOf, w.Runewords)
		if !ok {
			return Pick{}, ErrNoTarget
		}
		return pk, nil
	}, t.socketedProof)
}
