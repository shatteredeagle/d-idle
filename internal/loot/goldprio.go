package loot

import (
	"math"

	"d2lod-bot/internal/d2"
)

// Gold priority (docs/goldprio.md, knob loot.goldPriority): gold piles are the one pickup that is worth interrupting
// the day for, so the looter keeps a ledger of them, retries a failed pile over several calls and answers "is there a
// pile to take now" for the bot's mid-fight grab and its roam stop.

// GoldLedger counts the piles the looter saw since the game started.
type GoldLedger struct {
	Seen      int // piles judged worth taking when first seen
	Picked    int // piles taken
	Gold      int // coins gained by piles
	Abandoned int // piles still remembered when their area was left
}

// GoldLedger returns a copy of the ledger.
func (l *Looter) GoldLedger() GoldLedger {
	l.mem.mu.Lock()
	defer l.mem.mu.Unlock()
	return l.mem.gold
}

func (l *Looter) isGold(u d2.Unit) bool {
	b := l.base(u.Class)
	return b != nil && b.Type == "gold"
}

func (l *Looter) notePicked(u d2.Unit, gold int) {
	if !l.isGold(u) {
		return
	}
	l.mem.mu.Lock()
	l.mem.init()
	l.mem.gold.Picked++
	l.mem.gold.Gold += gold
	delete(l.mem.gfail, u.ID)
	l.mem.mu.Unlock()
}

// goldFailed records a failed pickup round of a gold pile and reports whether the pile stays wanted (priority on and
// retries left). Piles of other kinds, or with the priority off, keep the old rule (forgotten after two rounds).
func (l *Looter) goldFailed(u d2.Unit) bool {
	t := &l.e.Tune.Loot
	if t.GoldPriority == 0 || !l.isGold(u) {
		return false
	}
	l.mem.mu.Lock()
	defer l.mem.mu.Unlock()
	l.mem.init()
	l.mem.gfail[u.ID]++
	return l.mem.gfail[u.ID] < max(t.GoldRetries, 1)
}

// goldGivenUp reports whether a gold pile failed GoldRetries rounds already.
func (l *Looter) goldGivenUp(u d2.Unit) bool {
	t := &l.e.Tune.Loot
	if t.GoldPriority == 0 {
		return false
	}
	l.mem.mu.Lock()
	n := l.mem.gfail[u.ID]
	l.mem.mu.Unlock()
	return n >= max(t.GoldRetries, 1) && l.isGold(u)
}

// GoldNear reports whether a gold pile the pickit takes lies within dist subtiles of the hero and has retries left.
// It is cheap (no stat read beyond the pile size, no click) so the bot can poll it every tick.
func (l *Looter) GoldNear(s *d2.Snapshot, dist int) bool {
	t := &l.e.Tune.Loot
	if t.GoldPriority == 0 || s == nil || s.Player == nil || s.Player.InTown() {
		return false
	}
	var c Ctx
	haveCtx := false
	for _, u := range s.Ground {
		if u.Item == nil || u.Mode != d2.ItemModeGround || !l.isGold(u) || l.ignored(u) || l.goldGivenUp(u) {
			continue
		}
		if math.Hypot(float64(u.X-s.Player.X), float64(u.Y-s.Player.Y)) > float64(dist) {
			continue
		}
		if !haveCtx {
			c, haveCtx = l.ctxFrom(s), true
		}
		cc := c
		if l.stats != nil {
			if st, err := l.stats(u.Addr); err == nil {
				cc.Stats = st
			}
		}
		v := l.EvaluateWith(u, cc)
		if !passes(v, t.PickStrictness, s.Player.Gold < t.LowGoldBelow) {
			continue
		}
		if cc.Stats != nil && int(cc.Stats.Get(d2.StatGold)) < l.goldBar(s) && s.Player.Gold >= l.poorBelow() {
			continue
		}
		return true
	}
	return false
}

// noteAreaLeft books the piles still remembered in the area the hero just left as abandoned. m.mu is held.
func (m *lootMem) noteAreaLeft(area int) {
	for _, it := range m.items {
		if it.area == area && it.gold > 0 {
			m.gold.Abandoned++
		}
	}
}
