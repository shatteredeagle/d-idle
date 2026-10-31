package input

import (
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/mem"
)

// Game.exe RVAs used for input verification. All VERIFIED LIVE (docs/input.md, section 1).
const (
	rvaPlayerUnit = 0x3A6A70
	rvaMouseX     = 0x3A6AB0
	rvaMouseY     = 0x3A6AAC
	rvaViewportX  = 0x3A520C
	rvaViewportY  = 0x3A5208
)

// Reader reads the few game variables that input verification needs. It is deliberately small:
// the full typed readers live in internal/d2; this exists so internal/input has no dependency on it.
type Reader struct{ Src mem.Source }

// PlayerState is the player's position and movement target.
type PlayerState struct {
	X, Y       float64 // subtiles incl. the sub-subtile fraction
	TargetX    int     // Path xTarget/yTarget (subtiles)
	TargetY    int
	Mode       uint32
	Unit, Path uint32
}

// Player reads the local player. It fails when no game is in progress (menu, loading).
func (r Reader) Player() (PlayerState, error) {
	var s PlayerState
	u, err := mem.U32(r.Src, mem.Base+rvaPlayerUnit)
	if err != nil {
		return s, err
	}
	if u == 0 {
		return s, fmt.Errorf("input: no player unit (not in game)")
	}
	p, err := mem.U32(r.Src, u+0x2C)
	if err != nil {
		return s, err
	}
	if p == 0 {
		return s, fmt.Errorf("input: player has no path")
	}
	var b [0x14]byte
	if err := r.Src.ReadAt(b[:], p); err != nil {
		return s, err
	}
	le16 := func(o int) uint16 { return uint16(b[o]) | uint16(b[o+1])<<8 }
	s.X = float64(le16(0x02)) + float64(le16(0x00))/65536
	s.Y = float64(le16(0x06)) + float64(le16(0x04))/65536
	s.TargetX, s.TargetY = int(le16(0x10)), int(le16(0x12))
	s.Mode, _ = mem.U32(r.Src, u+0x10)
	s.Unit, s.Path = u, p
	return s, nil
}

// Mouse is the game's own idea of the pointer (client pixels); it follows posted WM_MOUSEMOVE exactly.
func (r Reader) Mouse() (x, y int, err error) {
	a, err := mem.I32(r.Src, mem.Base+rvaMouseX)
	if err != nil {
		return 0, 0, err
	}
	b, err := mem.I32(r.Src, mem.Base+rvaMouseY)
	return int(a), int(b), err
}

// Viewport returns the scroll offsets ViewportX/Y (docs 4.1).
func (r Reader) Viewport() (x, y int, err error) {
	a, err := mem.I32(r.Src, mem.Base+rvaViewportX)
	if err != nil {
		return 0, 0, err
	}
	b, err := mem.I32(r.Src, mem.Base+rvaViewportY)
	return int(a), int(b), err
}

// Skills returns the selected left and right skill ids (UnitAny.pInfo -> pLeftSkill/pRightSkill ->
// pSkillInfo -> wSkillId, docs 2.8). UNVERIFIED until tested live, see docs/input.md.
func (r Reader) Skills() (left, right int, err error) {
	u, err := mem.U32(r.Src, mem.Base+rvaPlayerUnit)
	if err != nil || u == 0 {
		return 0, 0, fmt.Errorf("input: no player unit: %v", err)
	}
	info, err := mem.U32(r.Src, u+0xA8)
	if err != nil || info == 0 {
		return 0, 0, fmt.Errorf("input: no skill info: %v", err)
	}
	id := func(off uint32) (int, error) {
		sk, err := mem.U32(r.Src, info+off)
		if err != nil || sk == 0 {
			return 0, fmt.Errorf("input: no skill at +%#x: %v", off, err)
		}
		si, err := mem.U32(r.Src, sk)
		if err != nil || si == 0 {
			return 0, err
		}
		v, err := mem.U16(r.Src, si)
		return int(v), err
	}
	if left, err = id(0x08); err != nil {
		return
	}
	right, err = id(0x0C)
	return
}

const rvaUIVars = 0x3A27C0 // array read by GetUiVar (RVA 0x538D0: mov eax,[ecx*4+0x7A27C0]); VERIFIED LIVE, docs/input.md

// UIOpen reports whether any Esc-closable panel, dialog or menu is open. Which flags count is decided in one place,
// d2.UIState (internal/d2/ui.go).
func (r Reader) UIOpen() (bool, error) {
	var u d2.UIState
	for i := range u {
		v, err := mem.U32(r.Src, mem.Base+rvaUIVars+uint32(i)*4)
		if err != nil {
			return false, err
		}
		u[i] = v
	}
	return u.EscClosableOpen(), nil
}

// Hovered reads the selected-unit pair used by GetSelectedUnit (RVA 0x67A10): id at 0x3A6A78, type at
// 0x3A6A8C, valid flag at 0x3A6A94. UNVERIFIED: see docs/input.md for the test result.
func (r Reader) Hovered() (typ, id uint32, err error) {
	flag, err := mem.U32(r.Src, mem.Base+0x3A6A94)
	if err != nil || flag == 0 {
		return 0, 0, err
	}
	id, _ = mem.U32(r.Src, mem.Base+0x3A6A78)
	typ, err = mem.U32(r.Src, mem.Base+0x3A6A8C)
	return typ, id, err
}

// Area returns the current level id via player -> Path -> Room1 -> Room2 -> Level (VERIFIED LIVE in the removed cmd/spike).
// Rogue Encampment is 1.
func (r Reader) Area() (int, error) {
	ptr := func(base, off uint32) (uint32, error) {
		v, err := mem.U32(r.Src, base+off)
		if err == nil && v < 0x10000 {
			err = fmt.Errorf("input: null pointer at %#x+%#x", base, off)
		}
		return v, err
	}
	u, err := ptr(mem.Base+rvaPlayerUnit, 0)
	if err != nil {
		return 0, err
	}
	for _, off := range []uint32{0x2C, 0x1C, 0x10, 0x58} {
		if u, err = ptr(u, off); err != nil {
			return 0, err
		}
	}
	n, err := mem.U32(r.Src, u+0x1D0)
	return int(n), err
}
