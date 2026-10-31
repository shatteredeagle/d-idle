package d2

import (
	"errors"
	"time"
)

// Snapshot is one consistent read of everything the bot loop needs. When State is not StateInGame
// only State, UI, Mouse and Screen are filled.
type Snapshot struct {
	State     State
	Player    *Player
	Inventory *Inventory
	Monsters  []Unit // includes town NPCs; filter with Alive and class tables
	Objects   []Unit
	Missiles  []Unit
	Ground    []Unit // items on the ground
	UI        UIState
	Mouse     Point
	Screen    Point
	Viewport  Point
	Hovered   Hover
}

// Snapshot reads the whole state. It is consistent in the sense that the player (address, id, area)
// is identical before and after the unit reads and no read failed; otherwise it retries up to
// Reader.Retries times and finally reports StateLoading.
func (r *Reader) Snapshot() (*Snapshot, error) {
	var lastErr error
	for try := 0; try <= r.Retries; try++ {
		if try > 0 && r.Sleep != nil {
			r.Sleep(15 * time.Millisecond)
		}
		s, err := r.snapshotOnce()
		if err == nil {
			return s, nil
		}
		lastErr = err
	}
	if errors.Is(lastErr, ErrNotInGame) || errors.Is(lastErr, ErrBadPointer) || isTorn(lastErr) {
		return &Snapshot{State: StateLoading}, nil
	}
	return nil, lastErr
}

var errTorn = errors.New("d2: torn read")

func isTorn(err error) bool { return errors.Is(err, errTorn) }

func (r *Reader) snapshotOnce() (*Snapshot, error) {
	st := r.State()
	s := &Snapshot{State: st}
	var err error
	if s.UI, err = r.UI(); err != nil {
		return nil, err
	}
	if s.Mouse, err = r.Mouse(); err != nil {
		return nil, err
	}
	if s.Screen, err = r.ScreenSize(); err != nil {
		return nil, err
	}
	if st != StateInGame {
		return s, nil
	}
	if s.Viewport, err = r.Viewport(); err != nil {
		return nil, err
	}
	if s.Player, err = r.Player(); err != nil {
		return nil, err
	}
	if s.Inventory, err = r.Inventory(); err != nil {
		return nil, err
	}
	if s.Monsters, err = r.Units(TypeMonster); err != nil {
		return nil, err
	}
	if s.Objects, err = r.Units(TypeObject); err != nil {
		return nil, err
	}
	if s.Missiles, err = r.Units(TypeMissile); err != nil {
		return nil, err
	}
	if s.Ground, err = r.GroundItems(); err != nil {
		return nil, err
	}
	if s.Hovered, err = r.Hovered(); err != nil {
		return nil, err
	}
	p2, err := r.Player()
	if err != nil {
		return nil, err
	}
	if p2.Addr != s.Player.Addr || p2.ID != s.Player.ID || p2.Area != s.Player.Area || r.State() != StateInGame {
		return nil, errTorn
	}
	return s, nil
}
