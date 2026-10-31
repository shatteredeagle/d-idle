// Package fake is an in-memory d2.StateReader for tests of code that only needs snapshots and unit lists.
package fake

import "d2lod-bot/internal/d2"

// Reader serves fixed data. Address based reads answer from the maps (nil map: d2.ErrUnsupported).
type Reader struct {
	Snap  *d2.Snapshot
	Err   error // returned by every call when set
	Stats map[uint32]d2.Stats
}

var _ d2.StateReader = (*Reader)(nil)

func (r *Reader) Snapshot() (*d2.Snapshot, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	if r.Snap == nil {
		return &d2.Snapshot{State: d2.StateLoading}, nil
	}
	return r.Snap, nil
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
	s, err := r.Snapshot()
	if err != nil {
		return nil, err
	}
	switch typ {
	case d2.TypeMonster:
		return s.Monsters, nil
	case d2.TypeObject:
		return s.Objects, nil
	case d2.TypeMissile:
		return s.Missiles, nil
	case d2.TypeItem:
		return s.Ground, nil
	}
	return nil, d2.ErrUnsupported
}

func (r *Reader) stats(a uint32) (d2.Stats, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	if st, ok := r.Stats[a]; ok {
		return st, nil
	}
	return nil, d2.ErrUnsupported
}

func (r *Reader) UnitStats(unit uint32) (d2.Stats, error)      { return r.stats(unit) }
func (r *Reader) ReadStatList(sl uint32) (d2.Stats, error)     { return r.stats(sl) }
func (r *Reader) ReadBaseStatList(sl uint32) (d2.Stats, error) { return r.stats(sl) }
func (r *Reader) RawStatList(uint32, int) ([]byte, error)      { return nil, d2.ErrUnsupported }
