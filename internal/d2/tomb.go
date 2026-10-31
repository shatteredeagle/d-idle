package d2

// Act and ActMisc offsets for the real Tal Rasha tomb. All S (D2BS D2Structs.h, same layout as 1.13c per
// docs/d2-1.14d-memory.md 3.2), UNVERIFIED on 1.14d until a live read is compared with the tomb holding object 152.
const (
	actPMisc         = 0x48 // Act.pMisc (ActMisc*)
	miscStaffTombLvl = 0x94 // ActMisc.dwStaffTombLevel
	tombLevelFirst   = 66
	tombLevelLast    = 72
)

// StaffTombLevel returns the area id (66..72) of the real Tal Rasha tomb of the player's act, following
// player (unit +0x1C pAct, S) -> Act +0x48 pMisc -> ActMisc +0x94. Read-only. A value outside 66..72 (not act 2,
// or not yet set) is returned as 0 with a nil error; broken pointers are errors.
func (r *Reader) StaffTombLevel(player uint32) (int, error) {
	act, err := r.ptr(player + uoPActObj)
	if err != nil {
		return 0, err
	}
	misc, err := r.ptr(act + actPMisc)
	if err != nil {
		return 0, err
	}
	v, err := r.u32(misc + miscStaffTombLvl)
	if err != nil {
		return 0, err
	}
	if v < tombLevelFirst || v > tombLevelLast {
		return 0, nil
	}
	return int(v), nil
}
