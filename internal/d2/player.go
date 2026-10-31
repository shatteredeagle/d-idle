package d2

import (
	"fmt"

	"d2lod-bot/internal/mem"
)

// Class is the player class (UnitAny.dwTxtFileNo of the player). VERIFIED LIVE: 1 = Sorceress.
type Class int

const (
	Amazon Class = iota
	Sorceress
	Necromancer
	Paladin
	Barbarian
	Druid
	Assassin
)

var classNames = [...]string{"Amazon", "Sorceress", "Necromancer", "Paladin", "Barbarian", "Druid", "Assassin"}

func (c Class) String() string {
	if c >= 0 && int(c) < len(classNames) {
		return classNames[c]
	}
	return fmt.Sprintf("Class(%d)", int(c))
}

// Difficulty: byte at RVA 0x3A060C (found by decoding GetDifficulty, value 0 = Normal VERIFIED LIVE;
// 1 Nightmare and 2 Hell are UNVERIFIED).
type Difficulty int

const (
	Normal Difficulty = iota
	Nightmare
	Hell
)

const rvaDifficulty = 0x3A060C

// Resists are the resistance stats as stored (before any display clamping). Normal difficulty has no
// penalty, so a fresh character reads 0. Stat ids UNVERIFIED (absent=0 at level 1).
type Resists struct{ Fire, Cold, Lightning, Poison, Magic int }

// Player is the local character.
type Player struct {
	Addr  uint32
	ID    uint32
	Class Class
	Name  string // PlayerData.szName (+0x00), VERIFIED LIVE ("Testy")
	X, Y  int    // subtiles
	Mode  int
	Act   int // 0-based (act I = 0), VERIFIED LIVE
	Area  int // level id, VERIFIED LIVE (1 = Rogue Encampment)

	Life, MaxLife       int
	Mana, MaxMana       int
	Stamina, MaxStamina int
	Level               int
	Exp                 uint32
	Gold, StashGold     int
	Strength, Dexterity int
	Vitality, Energy    int
	StatPoints          int // unspent
	SkillPoints         int // unspent
	Resist              Resists
	Stats               Stats
	States              StateSet // states.txt bit array (poison, buffs); empty when unreadable

	Skills     []Skill
	LeftSkill  int // skill id on the left hand
	RightSkill int // skill id on the right hand
	Difficulty Difficulty
}

// InTown reports whether the area is a town (act I-V town level ids).
func (p *Player) InTown() bool {
	switch p.Area {
	case 1, 40, 75, 103, 109:
		return true
	}
	return false
}

// Poisoned reports whether the poison state is on the character.
func (p *Player) Poisoned() bool { return p.States.Has(StatePoison) }

// LifePct returns life as 0-100.
func (p *Player) LifePct() int {
	if p.MaxLife <= 0 {
		return 0
	}
	return p.Life * 100 / p.MaxLife
}

// ManaPct returns mana as 0-100.
func (p *Player) ManaPct() int {
	if p.MaxMana <= 0 {
		return 0
	}
	return p.Mana * 100 / p.MaxMana
}

// playerAddr reads the PlayerUnit variable and validates the type field.
func (r *Reader) playerAddr() (uint32, error) {
	pa, err := r.u32(mem.Base + rvaPlayerUnit)
	if err != nil {
		return 0, err
	}
	if !okPtr(pa) {
		return 0, ErrNotInGame
	}
	t, err := r.u32(pa + uoType)
	if err != nil {
		return 0, err
	}
	if t != TypePlayer {
		return 0, fmt.Errorf("d2: player unit type %d", t)
	}
	return pa, nil
}

// Player reads the local character. Returns ErrNotInGame in menus and loading.
func (r *Reader) Player() (*Player, error) {
	pa, err := r.playerAddr()
	if err != nil {
		return nil, err
	}
	p := &Player{Addr: pa}
	id, err := r.u32(pa + uoID)
	if err != nil {
		return nil, err
	}
	class, err := r.u32(pa + uoClass)
	if err != nil {
		return nil, err
	}
	mode, err := r.u32(pa + uoMode)
	if err != nil {
		return nil, err
	}
	act, err := r.u32(pa + uoAct)
	if err != nil {
		return nil, err
	}
	if class > 6 || act > 4 {
		return nil, fmt.Errorf("d2: implausible player class %d act %d", class, act)
	}
	p.ID, p.Class, p.Mode, p.Act = id, Class(class), int(mode), int(act)
	if p.X, p.Y, err = r.pathPos(pa); err != nil {
		return nil, err
	}
	if p.Area, err = r.AreaOf(pa); err != nil {
		return nil, err
	}
	pd, err := r.ptr(pa + uoData)
	if err != nil {
		return nil, err
	}
	if p.Name, err = r.cstr(pd, 16); err != nil {
		return nil, err
	}
	st, err := r.UnitStats(pa)
	if err != nil {
		return nil, err
	}
	p.Stats = st
	if ss, err := r.UnitStates(pa); err == nil {
		p.States = ss
	}
	p.Life, p.MaxLife = st.Fixed(StatLife), st.Fixed(StatMaxLife)
	p.Mana, p.MaxMana = st.Fixed(StatMana), st.Fixed(StatMaxMana)
	p.Stamina, p.MaxStamina = st.Fixed(StatStamina), st.Fixed(StatMaxStamina)
	p.Level = int(st.Get(StatLevel))
	p.Exp = uint32(st.Get(StatExp))
	p.Gold, p.StashGold = int(st.Get(StatGold)), int(st.Get(StatStashGold))
	p.Strength, p.Dexterity = int(st.Get(StatStrength)), int(st.Get(StatDexterity))
	p.Vitality, p.Energy = int(st.Get(StatVitality)), int(st.Get(StatEnergy))
	p.StatPoints, p.SkillPoints = int(st.Get(StatPoints)), int(st.Get(StatSkillPts))
	p.Resist = Resists{
		Fire: int(st.Get(StatFireRes)), Cold: int(st.Get(StatColdRes)), Lightning: int(st.Get(StatLightRes)),
		Poison: int(st.Get(StatPoisonRes)), Magic: int(st.Get(StatMagicRes)),
	}
	if p.Skills, p.LeftSkill, p.RightSkill, err = r.skills(pa); err != nil {
		return nil, err
	}
	for i := range p.Skills {
		p.Skills[i].Bonus = int(st[uint32(p.Skills[i].ID)<<16|107])
	}
	if d, err := r.u8(mem.Base + rvaDifficulty); err == nil && d <= 2 {
		p.Difficulty = Difficulty(d)
	}
	if p.Level < 1 || p.Level > 99 {
		return nil, fmt.Errorf("d2: implausible level %d", p.Level)
	}
	return p, nil
}

// pathPos reads Path.xPos/yPos (WORD at +2/+6), VERIFIED LIVE for players and monsters.
func (r *Reader) pathPos(unit uint32) (x, y int, err error) {
	path, err := r.ptr(unit + uoPath)
	if err != nil {
		return 0, 0, err
	}
	xw, err := r.u16(path + 2)
	if err != nil {
		return 0, 0, err
	}
	yw, err := r.u16(path + 6)
	if err != nil {
		return 0, 0, err
	}
	return int(xw), int(yw), nil
}
