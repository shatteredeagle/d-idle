package loot

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Loot memory (docs/lootmemory.md): drops the hero saw but was too far from to pick up (a pack chased, a leg walked)
// are remembered per game and fetched at quiet moments by WalkBack. PickUp itself is unchanged and does the fetching.

// memVisible is how far from the hero an absent ground unit still counts as gone. Farther away the client may simply
// not have the room loaded (U), so the entry stays.
const memVisible = 50

// memAreaMargin is the subtile reach around a remembered spot given to PickUp (the item may have been nudged).
const memAreaMargin = 6

type memItem struct {
	id     uint32
	class  int
	name   string
	x, y   int
	area   int
	seen   time.Time
	gold   int // pile size, 0 for items
	value  int // estimated worth: gold amount, or the sell value of an item
	wanted bool
	fails  int
}

type lootMem struct {
	mu     sync.Mutex
	items  map[uint32]*memItem
	judged map[uint32]bool // ids already evaluated (kept or rejected), so a tick costs nothing for known units
	area   int             // area of the last snapshot
	trips  map[int]int     // walk-back trips since the hero entered the area
}

func (m *lootMem) init() {
	if m.items == nil {
		m.items, m.judged, m.trips = map[uint32]*memItem{}, map[uint32]bool{}, map[int]int{}
	}
}

// ResetMemory forgets everything: a new game starts, unit ids are not comparable across games.
func (l *Looter) ResetMemory() {
	l.mem.mu.Lock()
	l.mem.items, l.mem.judged, l.mem.trips, l.mem.area = nil, nil, nil, 0
	l.mem.mu.Unlock()
}

// Remembered is one remembered drop.
type Remembered struct {
	ID         uint32
	Name       string
	X, Y, Area int
	Value      int
	Gold       int
}

// Remembered lists the remembered drops (copy), for tests and the log.
func (l *Looter) Remembered() []Remembered {
	l.mem.mu.Lock()
	defer l.mem.mu.Unlock()
	out := make([]Remembered, 0, len(l.mem.items))
	for _, it := range l.mem.items {
		out = append(out, Remembered{ID: it.id, Name: it.name, X: it.x, Y: it.y, Area: it.area, Value: it.value, Gold: it.gold})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ObserveGround is called on every snapshot (the bot observer). It records new ground items the pickit takes and drops
// the ones that left the ground. Cheap for known units: only an unseen id is evaluated.
func (l *Looter) ObserveGround(s *d2.Snapshot) {
	t := &l.e.Tune.Loot
	if t.MemEnabled == 0 || s == nil || s.State != d2.StateInGame || s.Player == nil || s.Player.InTown() {
		return
	}
	m := &l.mem
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	area := s.Player.Area
	if area != m.area {
		m.area = area
		delete(m.trips, area) // a new visit of this area
	}
	onGround := make(map[uint32]bool, len(s.Ground))
	var c Ctx
	haveCtx := false
	for _, u := range s.Ground {
		if u.Item == nil || u.Mode != d2.ItemModeGround {
			continue
		}
		onGround[u.ID] = true
		if m.judged[u.ID] {
			continue
		}
		m.judged[u.ID] = true
		if l.ignored(u) {
			continue
		}
		if !haveCtx {
			c, haveCtx = l.ctxFrom(s), true
		}
		cc := c
		b := l.base(u.Class)
		isGold := b != nil && b.Type == "gold"
		if l.stats != nil && (b == nil || isGold) {
			if st, err := l.stats(u.Addr); err == nil {
				cc.Stats = st
			}
		}
		v := l.EvaluateWith(u, cc)
		if !passes(v, t.PickStrictness, s.Player.Gold < t.LowGoldBelow) {
			continue
		}
		it := &memItem{id: u.ID, class: u.Class, name: l.itemName(u), x: u.X, y: u.Y, area: area, seen: l.e.Clock.Now(), wanted: v.Wanted()}
		if isGold {
			it.gold = int(cc.Stats.Get(d2.StatGold))
			it.value = it.gold
			if cc.Stats != nil && it.gold < l.goldBar(s) {
				continue
			}
		} else {
			it.value = SellValue(l.e.Data, b, u.Item, MaxBuyOf(s.Player))
			if v.Prio >= t.ValuableMinPrio {
				it.value = max(it.value, t.ValuableMinGold) // runes, gems, uniques: always worth it
			}
		}
		m.items[u.ID] = it
		l.evict()
	}
	for id, it := range m.items {
		if it.area != area || onGround[id] {
			continue
		}
		if math.Hypot(float64(it.x-s.Player.X), float64(it.y-s.Player.Y)) <= memVisible {
			delete(m.items, id) // picked by us, the merc or a pursuit, or despawned
		}
	}
	if len(m.judged) > 8*max(t.MemCap, 1)+len(onGround) { // keep the evaluated set bounded
		for id := range m.judged {
			if !onGround[id] && m.items[id] == nil {
				delete(m.judged, id)
			}
		}
	}
}

// evict drops the lowest-value (then oldest) entries above MemCap. m.mu is held.
func (l *Looter) evict() {
	m := &l.mem
	for len(m.items) > max(l.e.Tune.Loot.MemCap, 1) {
		var worst *memItem
		for _, it := range m.items {
			if worst == nil || it.value < worst.value || it.value == worst.value && it.seen.Before(worst.seen) {
				worst = it
			}
		}
		delete(m.items, worst.id)
	}
}

func (l *Looter) forget(id uint32) {
	l.mem.mu.Lock()
	delete(l.mem.items, id)
	l.mem.mu.Unlock()
}

// worth reports whether a remembered drop justifies a walk at this character level.
func (l *Looter) worth(it *memItem, clvl int) bool {
	t := &l.e.Tune.Loot
	if it.gold > 0 {
		return float64(it.gold) >= float64(t.MemMinGold)*(1+0.1*float64(max(clvl-1, 0)))
	}
	return it.wanted || it.value >= t.ValuableMinGold
}

// WalkBack walks to remembered drops beyond the pickup radius, nearest first, and picks them up with PickUp (so the
// room, ignore and safety rules apply). abort is polled between trips and inside PickUp: the bot passes "a hostile is
// near". why is told once with what the first trip is for. Returns the trips made.
func (l *Looter) WalkBack(ctx context.Context, abort func() bool, why func(n, gold int, first string)) (int, error) {
	t := &l.e.Tune.Loot
	if t.MemEnabled == 0 {
		return 0, nil
	}
	trips, told, picked, gold := 0, false, 0, 0
	for {
		if err := ctx.Err(); err != nil {
			return trips, err
		}
		if abort != nil && abort() {
			break
		}
		s, err := l.hand.Fresh()
		if err != nil || s.Player == nil || s.Player.InTown() {
			return trips, err
		}
		area := s.Player.Area
		l.mem.mu.Lock()
		l.mem.init()
		if l.mem.trips[area] >= t.MemMaxTrips {
			l.mem.mu.Unlock()
			break
		}
		var best *memItem
		bestD := math.MaxFloat64
		n, g := 0, 0
		for _, it := range l.mem.items {
			if it.area != area || it.fails >= 2 || !l.worth(it, s.Player.Level) {
				continue
			}
			d := math.Hypot(float64(it.x-s.Player.X), float64(it.y-s.Player.Y))
			if d <= float64(t.PickRadius) || d > float64(t.MemMaxDist) {
				continue
			}
			n++
			g += it.gold
			if d < bestD {
				best, bestD = it, d
			}
		}
		if best == nil {
			l.mem.mu.Unlock()
			break
		}
		cp := *best
		l.mem.trips[area]++
		l.mem.mu.Unlock()

		if !told && why != nil {
			told = true
			why(n, g, cp.name)
		}
		trips++
		ctr := input.Point{X: cp.x, Y: cp.y}
		rep, err := l.PickUp(ctx, PickOpts{Center: &ctr, Radius: memAreaMargin, Abort: abort})
		picked += len(rep.Picked)
		gold += rep.GoldGained
		if err != nil {
			return trips, err
		}
		if after, e2 := l.hand.Fresh(); e2 == nil {
			if _, still := groundItem(after, cp.id); !still {
				l.forget(cp.id)
			} else {
				l.mem.mu.Lock()
				if it := l.mem.items[cp.id]; it != nil {
					it.fails++
					if it.fails >= 2 {
						delete(l.mem.items, cp.id)
					}
				}
				l.mem.mu.Unlock()
			}
		}
		if rep.Aborted {
			break
		}
	}
	if trips > 0 {
		l.e.Log.Info(fmt.Sprintf("loot_memory: %d trips, picked %d, gold +%d", trips, picked, gold))
	}
	return trips, nil
}
