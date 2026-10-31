package d2

// Skill is one entry of the unit's skill list. Info layout VERIFIED LIVE: UnitAny+0xA8 -> Info
// {+4 first skill, +8 left skill, +0xC right skill}; Skill {+0 SkillInfo* (WORD id at +0), +4 next,
// +0x28 base level}. The level excludes +skills from items; e.g. a new Sorceress reads Fire Bolt 0.
type Skill struct {
	ID    int
	Level int // base points (+0x28)
	// Bonus is the +skill from items/other sources: stat 107 (item_singleskill) with sub index = skill id
	// in the effective stat vector. VERIFIED LIVE: the starting staff gives Fire Bolt +1 (107/36 = 1)
	// while its base level reads 0. Class/tab bonuses (stats 83, 188) are not added.
	Bonus int
}

// Effective returns Level + Bonus.
func (s Skill) Effective() int { return s.Level + s.Bonus }

const (
	infoFirst = 0x04
	infoLeft  = 0x08
	infoRight = 0x0C
	skNext    = 0x04
	skLevel   = 0x28
	maxSkills = 128
)

// Well-known skill ids (VERIFIED LIVE for 0, 1, 2, 3, 36, 217-220 present in the list of a Sorceress).
const (
	SkillAttack   = 0
	SkillKick     = 1
	SkillThrow    = 2
	SkillUnsummon = 3
	SkillFireBolt = 36
	SkillTeleport = 54
)

func (r *Reader) skillID(sk uint32) (int, error) {
	si, err := r.ptr(sk)
	if err != nil {
		return 0, err
	}
	id, err := r.u16(si)
	return int(id), err
}

func (r *Reader) skills(unit uint32) (all []Skill, left, right int, err error) {
	info, err := r.ptr(unit + uoInfo)
	if err != nil {
		return nil, 0, 0, err
	}
	seen := map[uint32]bool{}
	first, err := r.optPtr(info + infoFirst)
	if err != nil {
		return nil, 0, 0, err
	}
	for sk := first; sk != 0 && len(all) < maxSkills; {
		if seen[sk] {
			break
		}
		seen[sk] = true
		id, err := r.skillID(sk)
		if err != nil {
			return nil, 0, 0, err
		}
		lvl, err := r.u32(sk + skLevel)
		if err != nil {
			return nil, 0, 0, err
		}
		all = append(all, Skill{ID: id, Level: int(lvl)})
		if sk, err = r.optPtr(sk + skNext); err != nil {
			return nil, 0, 0, err
		}
	}
	if len(all) == 0 {
		return nil, 0, 0, ErrNotInGame
	}
	for i, off := range []uint32{infoLeft, infoRight} {
		p, err := r.ptr(info + off)
		if err != nil {
			return nil, 0, 0, err
		}
		id, err := r.skillID(p)
		if err != nil {
			return nil, 0, 0, err
		}
		if i == 0 {
			left = id
		} else {
			right = id
		}
	}
	return all, left, right, nil
}
