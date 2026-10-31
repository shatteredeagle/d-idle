package d2

import "fmt"

// Stat ids (D2BS Constants.h; 0-12 VERIFIED LIVE present in the player's list, the rest UNVERIFIED).
const (
	StatStrength   = 0
	StatEnergy     = 1
	StatDexterity  = 2
	StatVitality   = 3
	StatPoints     = 4
	StatSkillPts   = 5
	StatLife       = 6 // 24.8 fixed point, VERIFIED LIVE (raw 10240 for 40 life)
	StatMaxLife    = 7
	StatMana       = 8
	StatMaxMana    = 9
	StatStamina    = 10
	StatMaxStamina = 11
	StatLevel      = 12
	StatExp        = 13
	StatGold       = 14
	StatStashGold  = 15
	StatFireRes    = 39
	StatLightRes   = 41
	StatColdRes    = 43
	StatPoisonRes  = 45
	StatMagicRes   = 37
	StatDmgReduce  = 36
	StatMagicFind  = 80
)

// Stat list layout. VERIFIED LIVE (dump of the player's list and an item's list): the vector is at
// +0x24 {Stat*, WORD count, WORD cap}; each Stat is 8 bytes {WORD sub, WORD id, int32 value}.
const (
	slOwnerType = 0x08
	slOwnerID   = 0x0C
	slFlags     = 0x10 // 0x80000000 on the player's list (the "extended" list), see docs/mem.md Q4
	slVec       = 0x24 // base vector
	slCount     = 0x28
	slFlagEx    = 0x80000000
	slVecEx     = 0x48 // effective (merged) vector when slFlagEx is set
	slCountEx   = 0x4C
	maxStats    = 512
)

// Stats is a decoded stat vector. Key is sub<<16 | id (sub is 0 for almost every stat).
type Stats map[uint32]int32

// Get returns the raw value of stat id (sub index 0).
func (s Stats) Get(id int) int32 { return s[uint32(id)] }

// Fixed returns a 24.8 fixed-point stat as a whole number (life, mana, stamina).
func (s Stats) Fixed(id int) int { return int(s[uint32(id)] >> 8) }

// Has reports whether the stat is present in the vector.
func (s Stats) Has(id int) bool { _, ok := s[uint32(id)]; return ok }

// ReadStatList decodes the effective stat vector of a StatList*, the one GetUnitStat reads: when the
// list has flag 0x80000000 (players, monsters) that is the merged vector at +0x48/+0x4C, otherwise the
// vector at +0x24/+0x28. Verified by decoding the code at RVA 0x224F60 (the lookup helper); see docs/mem.md.
func (r *Reader) ReadStatList(sl uint32) (Stats, error) {
	if !okPtr(sl) {
		return nil, fmt.Errorf("%w: statlist 0x%x", ErrBadPointer, sl)
	}
	flags, err := r.u32(sl + slFlags)
	if err != nil {
		return nil, err
	}
	if flags&slFlagEx != 0 {
		return r.readVec(sl+slVecEx, sl+slCountEx)
	}
	return r.readVec(sl+slVec, sl+slCount)
}

// ReadBaseStatList decodes the base vector at +0x24 regardless of flags (for comparison/debugging).
func (r *Reader) ReadBaseStatList(sl uint32) (Stats, error) {
	if !okPtr(sl) {
		return nil, fmt.Errorf("%w: statlist 0x%x", ErrBadPointer, sl)
	}
	return r.readVec(sl+slVec, sl+slCount)
}

func (r *Reader) readVec(vecAt, cntAt uint32) (Stats, error) {
	vec, err := r.ptr(vecAt)
	if err != nil {
		return nil, err
	}
	cnt, err := r.u16(cntAt)
	if err != nil {
		return nil, err
	}
	if cnt > maxStats {
		return nil, fmt.Errorf("d2: implausible stat count %d", cnt)
	}
	b, err := r.bytes(vec, int(cnt)*8)
	if err != nil {
		return nil, err
	}
	st := make(Stats, cnt)
	for i := 0; i < int(cnt); i++ {
		sub := uint32(b[i*8]) | uint32(b[i*8+1])<<8
		id := uint32(b[i*8+2]) | uint32(b[i*8+3])<<8
		v := int32(uint32(b[i*8+4]) | uint32(b[i*8+5])<<8 | uint32(b[i*8+6])<<16 | uint32(b[i*8+7])<<24)
		st[sub<<16|id] = v
	}
	return st, nil
}

// UnitStats reads the stat list of a unit (UnitAny+0x5C).
func (r *Reader) UnitStats(unit uint32) (Stats, error) {
	sl, err := r.ptr(unit + uoStats)
	if err != nil {
		return nil, err
	}
	return r.ReadStatList(sl)
}

// Child stat-list chain. V for the player (aura state, docs/classes/smoke-paladin.md finding 6) and for monsters
// (curse state 9 and minion state 105 read this way, smoke-necromancer.md findings 1 and 4): StatList+0x3c is the newest
// child, +0x2c links to the previous one, +0x14 holds the state number.
const (
	slChild      = 0x3c
	slChildPrev  = 0x2c
	slChildState = 0x14
	maxChildren  = 24
)

// ReadStates lists the state numbers of the child lists of a StatList*. It never fails: an unreadable or bad
// chain just yields what was read so far (nil for none).
func (r *Reader) ReadStates(sl uint32) []uint8 {
	if !okPtr(sl) {
		return nil
	}
	child, err := r.u32(sl + slChild)
	var out []uint8
	for n := 0; err == nil && okPtr(child) && n < maxChildren; n++ {
		st, e := r.u32(child + slChildState)
		if e != nil {
			break
		}
		if st != 0 && st < 256 {
			out = append(out, uint8(st))
		}
		child, err = r.u32(child + slChildPrev)
	}
	return out
}
