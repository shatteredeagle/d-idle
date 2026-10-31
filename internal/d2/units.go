package d2

import (
	"d2lod-bot/internal/mem"
)

// Unit modes. Monster/NPC modes per D2BS Constants.h (UNVERIFIED except 1 = stand, seen on town NPCs).
const (
	MonModeDeath = 0
	MonModeStand = 1
	MonModeDead  = 12
)

// Monster flag bits (MonsterData+0x16), UNVERIFIED (only town NPCs with all-zero flags were available).
const (
	monFlagNormal = 1 << 1
	monFlagChamp  = 1 << 2
	monFlagBoss   = 1 << 3
	monFlagMinion = 1 << 4
)

// MonsterInfo is the MonsterData block. UNVERIFIED offsets from D2BS (+0x16 flags, +0x1C enchants[9],
// +0x26 unique no, +0x2C pointer to the wide name); live town NPCs read all zero.
type MonsterInfo struct {
	Flags    uint8
	Enchants [9]uint8
	UniqueNo uint16
	Name     string // set for uniques/bosses
	// LifePct is current/max stat 6/7 * 100. VERIFIED LIVE only that town NPCs read 128/128 (raw 0x8000);
	// the client appears to scale monster life to 128, so a ratio is the safe interpretation. UNVERIFIED for damaged monsters.
	LifePct int
	// Life/MaxLife are stats 6/7 as whole points and Level is stat 12 (merc level; U: unverified for monsters).
	Life, MaxLife, Level int
	// States are the state numbers of the unit's child stat lists (curses, auras, pet marks); see ReadStates.
	States []uint8
}

// StateMinion is the child state carried by a Necromancer's own minions (V docs/classes/smoke-necromancer.md finding 1:
// clay golem and two skeletons had it, a live hostile Zombie and Quill Rat did not). What the state means is UNVERIFIED,
// and it is U whether other classes' summons carry it, so the class and AI tables in friendly.go back it up.
const StateMinion = 105

// HasState reports whether a child state with that number is present.
func (m *MonsterInfo) HasState(n int) bool {
	if m == nil {
		return false
	}
	for _, s := range m.States {
		if int(s) == n {
			return true
		}
	}
	return false
}

func (m *MonsterInfo) IsChampion() bool { return m.Flags&monFlagChamp != 0 }

// IsUnique is a rare or superunique: the boss bit without the champion bit. A champion reads 0x0c (both bits, V-orig
// census, docs/orig-spawns.md), so the boss bit alone would call every champion unique.
func (m *MonsterInfo) IsUnique() bool { return m.Flags&monFlagBoss != 0 && m.Flags&monFlagChamp == 0 }
func (m *MonsterInfo) IsMinion() bool { return m.Flags&monFlagMinion != 0 }

// Unit is one entry of the unit tables.
type Unit struct {
	Addr      uint32
	Type      int
	Class     int // dwTxtFileNo: monster id, object id, item base row, missile id
	ID        uint32
	Mode      int
	X, Y      int // subtiles; for items in a container this is the grid cell, see Item
	Flags     uint32
	OwnerType uint32
	OwnerID   uint32

	Monster *MonsterInfo // Type == TypeMonster
	Item    *Item        // Type == TypeItem
}

// Alive reports whether a monster is neither dying nor dead.
func (u *Unit) Alive() bool { return u.Mode != MonModeDeath && u.Mode != MonModeDead }

const (
	hashBuckets = 128
	maxChain    = 256
	maxUnits    = 4096
)

// Units lists all units of one type (TypeMonster..TypeTile). Monsters, objects and items come from the
// "server side" table, missiles from the client table (D2BS Unit.cpp:46-55). Server table VERIFIED LIVE;
// the client table for missiles is UNVERIFIED (none in flight during recording).
// Units whose type field does not match are dropped (torn read).
func (r *Reader) Units(typ int) ([]Unit, error) {
	if typ < 0 || typ > 5 {
		return nil, ErrBadPointer
	}
	base := uint32(mem.Base + rvaServerUnits)
	if typ == TypeMissile {
		base = mem.Base + rvaClientUnits
	}
	tbl := base + uint32(typ)*0x200
	raw, err := r.bytes(tbl, hashBuckets*4)
	if err != nil {
		return nil, err
	}
	var out []Unit
	seen := map[uint32]bool{}
	for i := 0; i < hashBuckets; i++ {
		head := le32(raw[i*4:])
		for n := 0; head != 0 && n < maxChain && len(out) < maxUnits; n++ {
			if !okPtr(head) || seen[head] {
				break
			}
			seen[head] = true
			u, err := r.readUnit(head, typ)
			if err != nil {
				return nil, err
			}
			if u != nil {
				out = append(out, *u)
			}
			head, err = r.u32(head + uoListNext)
			if err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// readUnit decodes a UnitAny. A type mismatch returns (nil, nil).
func (r *Reader) readUnit(a uint32, typ int) (*Unit, error) {
	b, err := r.bytes(a, 0xF0)
	if err != nil {
		return nil, err
	}
	if int(le32(b[uoType:])) != typ {
		return nil, nil
	}
	u := &Unit{
		Addr:      a,
		Type:      typ,
		Class:     int(le32(b[uoClass:])),
		ID:        le32(b[uoID:]),
		Mode:      int(le32(b[uoMode:])),
		Flags:     le32(b[uoFlags:]),
		OwnerType: le32(b[uoOwnerType:]),
		OwnerID:   le32(b[uoOwnerID:]),
	}
	data, path := le32(b[uoData:]), le32(b[uoPath:])
	if !okPtr(path) {
		return nil, nil
	}
	switch typ {
	case TypePlayer, TypeMonster, TypeMissile:
		// Path.xPos/yPos WORDs at +2/+6: VERIFIED LIVE for players and monsters.
		pb, err := r.bytes(path, 8)
		if err != nil {
			return nil, err
		}
		u.X, u.Y = int(pb[2])|int(pb[3])<<8, int(pb[6])|int(pb[7])<<8
	case TypeObject, TypeItem, TypeTile:
		// ObjectPath/ItemPath dwPosX/dwPosY at +0x0C/+0x10: VERIFIED LIVE for objects and for
		// belt/inventory items (grid cell); ground items UNVERIFIED (none available).
		pb, err := r.bytes(path+0x0C, 8)
		if err != nil {
			return nil, err
		}
		u.X, u.Y = int(le32(pb)), int(le32(pb[4:]))
	}
	switch typ {
	case TypeMonster:
		if okPtr(data) {
			m, err := r.monsterInfo(data, le32(b[uoStats:]))
			if err != nil {
				return nil, err
			}
			u.Monster = m
		}
	case TypeItem:
		if okPtr(data) {
			it, err := r.itemInfo(u, data)
			if err != nil {
				return nil, err
			}
			u.Item = it
		}
	}
	return u, nil
}

func (r *Reader) monsterInfo(data, statList uint32) (*MonsterInfo, error) {
	b, err := r.bytes(data, 0x30)
	if err != nil {
		return nil, err
	}
	m := &MonsterInfo{Flags: b[0x16], UniqueNo: uint16(b[0x26]) | uint16(b[0x27])<<8}
	copy(m.Enchants[:], b[0x1C:0x25])
	if m.Flags&monFlagBoss != 0 {
		// +0x2C is a POINTER to the wide name, not the name itself (VERIFIED LIVE i4 2026-10-28: read inline it gave
		// the pointer bytes as CJK garbage for Blood Raven). A bad pointer leaves the name empty.
		if np := le32(b[0x2C:]); okPtr(np) {
			for _, n := range []int{40, 20, 10} { // a short string may end a page: retry with smaller reads
				if name, werr := r.wstr(np, n); werr == nil {
					m.Name = name
					break
				}
			}
		}
	}
	if okPtr(statList) {
		m.States = r.ReadStates(statList)
		if st, err := r.ReadStatList(statList); err == nil {
			if mx := st.Get(StatMaxLife); mx > 0 {
				m.LifePct = int(int64(st.Get(StatLife)) * 100 / int64(mx))
			}
			m.Life, m.MaxLife, m.Level = st.Fixed(StatLife), st.Fixed(StatMaxLife), int(st.Get(StatLevel))
		}
	}
	return m, nil
}

// Nearest returns the unit closest to (x, y) by Chebyshev-ish subtile distance (max of dx, dy), or nil.
func Nearest(us []Unit, x, y int, keep func(*Unit) bool) *Unit {
	var best *Unit
	bd := 1 << 30
	for i := range us {
		u := &us[i]
		if keep != nil && !keep(u) {
			continue
		}
		d := abs(u.X-x) + abs(u.Y-y)
		if d < bd {
			best, bd = u, d
		}
	}
	return best
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
