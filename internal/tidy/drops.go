package tidy

import (
	"sync"
	"time"
)

// Drops remembers what the Tidy chore threw on the ground so the looter does not fetch it again (no drop-pickup loop).
// It matches by unit id plus print, or by the same print near the drop spot (a dropped item may get a new unit id, U
// live); entries expire after TTL (a new game gets new items anyway) and at most Max are kept.
type Drops struct {
	TTL  time.Duration
	Max  int
	Near int // subtiles
	Now  func() time.Time

	mu   sync.Mutex
	list []drop
}

type drop struct {
	id    uint32
	print string
	x, y  int
	at    time.Time
}

// NewDrops returns a memory with the defaults of the tidy chore: 2 h, 64 entries, 12 subtiles.
func NewDrops(now func() time.Time) *Drops {
	return &Drops{TTL: 2 * time.Hour, Max: 64, Near: 12, Now: now}
}

// Note records a drop of the item (id, print) at the hero's position (x, y).
func (d *Drops) Note(id uint32, print string, x, y int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := d.Now()
	kept := d.list[:0:0]
	for _, e := range d.list {
		if now.Sub(e.at) <= d.TTL {
			kept = append(kept, e)
		}
	}
	kept = append(kept, drop{id, print, x, y, now})
	if len(kept) > d.Max {
		kept = kept[len(kept)-d.Max:]
	}
	d.list = kept
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// Ignored is the looter's skip test for a ground item.
func (d *Drops) Ignored(id uint32, print string, x, y int) bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	now := d.Now()
	for _, e := range d.list {
		if now.Sub(e.at) > d.TTL || e.print != print {
			continue
		}
		if e.id == id || max(abs(e.x-x), abs(e.y-y)) <= d.Near {
			return true
		}
	}
	return false
}
