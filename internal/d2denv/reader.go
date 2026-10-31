package d2denv

import (
	"context"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dsrc"
)

// Item stat ids of the classic stat table (S: standard D2 itemstatcost order, not re-read from our tables here).
const (
	statQuantity   = 70
	statDurability = 72
	statMaxDur     = 73
	statSockets    = 194
)

// Reader is the d2.StateReader of the d2d backend: a d2dsrc.Source plus two things the decision code expects of every
// reader. (1) Unit.Addr is d2d's unit id, so code that keys stat reads on Addr (loot, town, combat gearcheck) keeps
// working: ids are unique among the unit lists U (they come from one world table), and Addr 0 stays "none". (2) Stat
// lists by unit id: UnitStats answers from the last snapshot's item facts (quantity, durability, sockets), so the
// decision code reads what d2d showed this tick instead of getting ErrUnsupported. Monsters, objects and raw lists
// stay ErrUnsupported (never a made-up zero); callers already degrade on it.
type Reader struct {
	*d2dsrc.Source

	mu   sync.Mutex
	last *d2dsrc.Result
}

var _ d2.StateReader = (*Reader)(nil)

// NewReader wraps src.
func NewReader(src *d2dsrc.Source) *Reader { return &Reader{Source: src} }

// Snapshot is one d2dsrc read with Addr filled in.
func (r *Reader) Snapshot() (*d2.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := r.Source.Read(ctx)
	if err != nil {
		return nil, err
	}
	addrFromID(res.Snap)
	r.mu.Lock()
	r.last = res
	r.mu.Unlock()
	return res.Snap, nil
}

func addrFromID(s *d2.Snapshot) {
	set := func(us []d2.Unit) {
		for i := range us {
			us[i].Addr = us[i].ID
		}
	}
	set(s.Monsters)
	set(s.Objects)
	set(s.Missiles)
	set(s.Ground)
	if s.Inventory != nil {
		set(s.Inventory.Items)
		if s.Inventory.Cursor != nil {
			s.Inventory.Cursor.Addr = s.Inventory.Cursor.ID
		}
	}
	if s.Player != nil && s.Player.Addr == 0 {
		s.Player.Addr = s.Player.ID
	}
}

func (r *Reader) Player() (*d2.Player, error) {
	s, err := r.Snapshot()
	if err != nil {
		return nil, err
	}
	if s.State != d2.StateInGame || s.Player == nil {
		return nil, d2.ErrNotInGame
	}
	return s.Player, nil
}

func (r *Reader) UI() (d2.UIState, error) {
	s, err := r.Snapshot()
	if err != nil {
		return d2.UIState{}, err
	}
	return s.UI, nil
}

func (r *Reader) Units(typ int) ([]d2.Unit, error) {
	var pick func(*d2.Snapshot) []d2.Unit
	switch typ {
	case d2.TypeMonster:
		pick = func(s *d2.Snapshot) []d2.Unit { return s.Monsters }
	case d2.TypeObject:
		pick = func(s *d2.Snapshot) []d2.Unit { return s.Objects }
	case d2.TypeMissile:
		pick = func(s *d2.Snapshot) []d2.Unit { return s.Missiles }
	default:
		return nil, d2.ErrUnsupported
	}
	s, err := r.Snapshot()
	if err != nil {
		return nil, err
	}
	return pick(s), nil
}

// UnitStats gives the stat vector of a unit of the LAST snapshot. Items: quantity, durability, max durability and
// socket count where d2d reported them; the player id (or 0): the player's own Stats. Anything else is
// d2.ErrUnsupported.
func (r *Reader) UnitStats(id uint32) (d2.Stats, error) {
	r.mu.Lock()
	last := r.last
	r.mu.Unlock()
	if last == nil || last.Snap == nil || id == 0 { // 0 is "no unit", never a key
		return nil, d2.ErrUnsupported
	}
	if p := last.Snap.Player; p != nil && (id == p.ID || id == p.Addr) {
		if len(p.Stats) == 0 { // stock d2d without the dump verb: the stat vector is unknown
			return nil, d2.ErrUnsupported
		}
		out := make(d2.Stats, len(p.Stats))
		for k, v := range p.Stats {
			out[k] = v
		}
		return out, nil
	}
	ex, ok := last.Extras.Items[id]
	if !ok {
		return nil, d2.ErrUnsupported
	}
	st := d2.Stats{}
	if ex.Qty > 0 {
		st[statQuantity] = int32(ex.Qty)
	}
	if ex.MaxDur > 0 {
		st[statDurability], st[statMaxDur] = int32(ex.Dur), int32(ex.MaxDur)
	}
	if ex.Sockets >= 0 {
		st[statSockets] = int32(ex.Sockets)
	}
	return st, nil
}

// ReadStatList and ReadBaseStatList take a stat list pointer in the memory backend; here the pointer is the unit id.
func (r *Reader) ReadStatList(id uint32) (d2.Stats, error)     { return r.UnitStats(id) }
func (r *Reader) ReadBaseStatList(id uint32) (d2.Stats, error) { return r.UnitStats(id) }
