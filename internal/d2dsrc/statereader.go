package d2dsrc

import "d2lod-bot/internal/d2"

var _ d2.StateReader = (*Source)(nil)

// The methods below complete d2.StateReader. Whatever a snapshot already holds is derived from one Snapshot call;
// everything address based answers d2.ErrUnsupported (d2d has no unit addresses), never a made-up value.

// Player is the snapshot's player; a game that is not in a game gives d2.ErrNotInGame like the memory reader.
func (s *Source) Player() (*d2.Player, error) {
	snap, err := s.Snapshot()
	if err != nil {
		return nil, err
	}
	if snap.State != d2.StateInGame || snap.Player == nil {
		return nil, d2.ErrNotInGame
	}
	return snap.Player, nil
}

// UI is the snapshot's panel state.
func (s *Source) UI() (d2.UIState, error) {
	snap, err := s.Snapshot()
	if err != nil {
		return d2.UIState{}, err
	}
	return snap.UI, nil
}

// Units answers monsters, objects and missiles from the snapshot. Players and items (inventory, ground by address) are
// unsupported: the snapshot carries them in Player and Inventory, not as units.
func (s *Source) Units(typ int) ([]d2.Unit, error) {
	var pick func(*d2.Snapshot) []d2.Unit
	switch typ {
	case d2.TypeMonster:
		pick = func(sn *d2.Snapshot) []d2.Unit { return sn.Monsters }
	case d2.TypeObject:
		pick = func(sn *d2.Snapshot) []d2.Unit { return sn.Objects }
	case d2.TypeMissile:
		pick = func(sn *d2.Snapshot) []d2.Unit { return sn.Missiles }
	default:
		return nil, d2.ErrUnsupported
	}
	snap, err := s.Snapshot()
	if err != nil {
		return nil, err
	}
	return pick(snap), nil
}

func (s *Source) UnitStats(uint32) (d2.Stats, error)        { return nil, d2.ErrUnsupported }
func (s *Source) ReadStatList(uint32) (d2.Stats, error)     { return nil, d2.ErrUnsupported }
func (s *Source) ReadBaseStatList(uint32) (d2.Stats, error) { return nil, d2.ErrUnsupported }
func (s *Source) RawStatList(uint32, int) ([]byte, error)   { return nil, d2.ErrUnsupported }
