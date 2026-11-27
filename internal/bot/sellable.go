package bot

import (
	"sync"

	"d2lod-bot/internal/d2"
)

// sellMemo keeps the last sell count with the snapshot it was judged on (sellsnap: resInput runs many times a tick and
// the watch API's guard world on every request; the count judged every pack item each time).
type sellMemo struct {
	mu   sync.Mutex
	snap *d2.Snapshot
	fp   uint64
	n    int
	ok   bool
}

// sellFingerprint changes whenever what the sell count reads of a snapshot changes: the pack items (id, cell,
// container, flags), level and gold. A snapshot mutated in place (test rigs) therefore misses the memo too.
func sellFingerprint(s *d2.Snapshot) uint64 {
	const prime = 1099511628211
	h := uint64(14695981039346656037)
	mix := func(v uint64) { h = (h ^ v) * prime }
	if s.Player != nil {
		mix(uint64(s.Player.Level))
		mix(uint64(s.Player.Gold))
	}
	if s.Inventory != nil {
		mix(uint64(len(s.Inventory.Items)))
		for i := range s.Inventory.Items {
			u := &s.Inventory.Items[i]
			mix(uint64(u.ID))
			mix(uint64(u.X)<<32 | uint64(uint32(u.Y)))
			if u.Item != nil {
				mix(uint64(u.Item.Where)<<32 | uint64(u.Item.Flags))
			}
		}
	}
	return h
}

// sellableOnce is r.sellable computed at most once per snapshot.
func (r *Runner) sellableOnce(s *d2.Snapshot) int {
	if s == nil {
		return r.sellable(s)
	}
	fp := sellFingerprint(s)
	m := &r.sellMemo
	m.mu.Lock()
	if m.ok && m.snap == s && m.fp == fp {
		n := m.n
		m.mu.Unlock()
		return n
	}
	m.mu.Unlock()
	n := r.sellable(s)
	m.mu.Lock()
	m.snap, m.fp, m.n, m.ok = s, fp, n, true
	m.mu.Unlock()
	return n
}

// sellableCached is the last count the bot loop computed, whatever snapshot it was for (0 before the first). Readers
// off the bot loop (the watch API's readiness, risk and propose pages through guardWorld) use it: they must not run
// the town's sell judgement, which reads the game and takes the backend's client lock.
func (r *Runner) sellableCached() int {
	m := &r.sellMemo
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.n
}
