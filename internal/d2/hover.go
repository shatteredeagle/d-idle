package d2

import "d2lod-bot/internal/mem"

// Hover is the engine's selected (hovered) unit. Decoded from GetSelectedUnit (RVA 0x67A10):
//
//	if [0x7a6a94] == 0 return 0
//	unit = GetServerUnit(id=[0x7a6a78], type=[0x7a6a8c])   ; helper at RVA 0x63990 indexes
//	                                                       ; table 0x7a5e70 + type*0x200, bucket id&0x7f
//
// So: Active = dword at RVA 0x3A6A94, ID = RVA 0x3A6A78, Type = RVA 0x3A6A8C. The code decoding is
// VERIFIED LIVE (bytes read from the process); the stored (type=1,id=9) matched a real NPC (Kashya) after
// earlier mouse movement. That the flag turns on while hovering a monster is UNVERIFIED: no selectable
// unit was on screen during the session (town guards did not select; a pointer sweep changed nothing).
type Hover struct {
	Active bool
	Type   int
	ID     uint32
}

const (
	rvaSelActive = 0x3A6A94
	rvaSelID     = 0x3A6A78
	rvaSelType   = 0x3A6A8C
)

func (r *Reader) Hovered() (Hover, error) {
	a, err := r.u32(mem.Base + rvaSelActive)
	if err != nil {
		return Hover{}, err
	}
	id, err := r.u32(mem.Base + rvaSelID)
	if err != nil {
		return Hover{}, err
	}
	t, err := r.u32(mem.Base + rvaSelType)
	if err != nil {
		return Hover{}, err
	}
	return Hover{Active: a != 0 && t <= 5, Type: int(t), ID: id}, nil
}
