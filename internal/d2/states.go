package d2

// Unit states (states.txt ids): the stat list of a unit carries a bit array of the states that are on it.
// StatePoison is states.txt "poison" (id 2); StateFrozenArmor (10) is the easiest buff to switch on for a live probe.
const (
	StatePoison      = 2
	StateFrozenArmor = 10
	stateWords       = 6 // 192 state bits
)

// slStateFlags is the offset inside an extended StatList (flag 0x80000000) of a POINTER to the state bit array (DWORD[6]).
// VERIFIED LIVE (i4, 2026-10-28, cmd/d2bot --probe-states): +0x58 holds a heap pointer and +0x5C a code address in Game.exe
// (0x460270), so +0x58 is the flags pointer, not the array itself. That bit n of the array is state n is S (D2MOO) and
// UNVERIFIED live until a known state (Frozen Armor, an antidote potion, poison) was seen switching its bit.
var slStateFlags uint32 = 0x58

// StateSet is a unit's state bit array: bit n is states.txt state n.
type StateSet [stateWords]uint32

// Has reports whether state n is on.
func (s StateSet) Has(n int) bool {
	return n >= 0 && n < stateWords*32 && s[n>>5]>>(uint(n)&31)&1 == 1
}

// UnitStates reads the state bit array of a unit's stat list. Players and monsters have the extended list; others
// answer an empty set.
func (r *Reader) UnitStates(unit uint32) (StateSet, error) {
	var out StateSet
	sl, err := r.ptr(unit + uoStats)
	if err != nil {
		return out, err
	}
	return r.StatListStates(sl)
}

// StatListStates is UnitStates for a StatList*.
func (r *Reader) StatListStates(sl uint32) (StateSet, error) {
	var out StateSet
	if !okPtr(sl) {
		return out, ErrBadPointer
	}
	flags, err := r.u32(sl + slFlags)
	if err != nil || flags&slFlagEx == 0 {
		return out, err
	}
	arr, err := r.ptr(sl + slStateFlags)
	if err != nil {
		return out, err
	}
	b, err := r.bytes(arr, stateWords*4)
	if err != nil {
		return out, err
	}
	for i := range out {
		out[i] = le32(b[i*4:])
	}
	return out, nil
}

// RawStatList reads n bytes of a unit's stat list struct from its start (layout probing).
func (r *Reader) RawStatList(unit uint32, n int) ([]byte, error) {
	sl, err := r.ptr(unit + uoStats)
	if err != nil {
		return nil, err
	}
	return r.bytes(sl, n)
}
