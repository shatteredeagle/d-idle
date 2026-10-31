package d2

import "d2lod-bot/internal/mem"

// State is the coarse client state.
type State int

const (
	// StateLoading covers loading screens, torn reads and anything ambiguous: do not act.
	StateLoading State = iota
	StateMenu
	StateInGame
)

func (s State) String() string {
	switch s {
	case StateMenu:
		return "menu"
	case StateInGame:
		return "ingame"
	}
	return "loading"
}

// State applies the canary rule of docs section 2.1 (VERIFIED LIVE: in game player!=0 and
// FirstControl==0). In game additionally requires the area chain to be complete and no pending
// pUpdateUnit.
func (r *Reader) State() State {
	player, err1 := r.u32(mem.Base + rvaPlayerUnit)
	first, err2 := r.u32(mem.Base + rvaFirstControl)
	if err1 != nil || err2 != nil {
		return StateLoading
	}
	switch {
	case player == 0 && first != 0:
		return StateMenu
	case player != 0 && first == 0 && okPtr(player):
		if upd, err := r.u32(player + uoUpdateUnit); err != nil || upd != 0 {
			return StateLoading
		}
		if _, err := r.AreaOf(player); err != nil {
			return StateLoading
		}
		return StateInGame
	}
	return StateLoading
}

// Chain offsets VERIFIED LIVE (spike): unit->path 0x2C, path->room1 0x1C, room1->room2 0x10,
// room2->level 0x58, level->levelNo 0x1D0.
const (
	uoType       = 0x00
	uoClass      = 0x04
	uoID         = 0x0C
	uoMode       = 0x10
	uoData       = 0x14
	uoAct        = 0x18
	uoPActObj    = 0x1C
	uoPath       = 0x2C
	uoStats      = 0x5C
	uoInventory  = 0x60
	uoUpdateUnit = 0x70
	uoOwnerType  = 0x94
	uoOwnerID    = 0x98
	uoInfo       = 0xA8
	uoFlags      = 0xC4
	uoFlags2     = 0xC8
	uoListNext   = 0xE4
	uoRoomNext   = 0xE8
)

// AreaOf follows unit -> path -> room1 -> room2 -> level -> levelNo.
func (r *Reader) AreaOf(unit uint32) (int, error) {
	path, err := r.ptr(unit + uoPath)
	if err != nil {
		return 0, err
	}
	room1, err := r.ptr(path + 0x1C)
	if err != nil {
		return 0, err
	}
	room2, err := r.ptr(room1 + 0x10)
	if err != nil {
		return 0, err
	}
	lvl, err := r.ptr(room2 + 0x58)
	if err != nil {
		return 0, err
	}
	n, err := r.u32(lvl + 0x1D0)
	if err != nil {
		return 0, err
	}
	if n == 0 || n > 200 {
		return 0, ErrNotInGame
	}
	return int(n), nil
}
