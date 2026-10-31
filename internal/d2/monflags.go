package d2

// MonKind is what the MonsterData flag byte says about a monster.
type MonKind int

const (
	MonKindUnknown MonKind = iota // flags unreadable or all zero: not known to be anything
	MonKindNormal
	MonKindChampion
	MonKindUnique
	MonKindMinion
)

func (k MonKind) String() string {
	return [...]string{"unknown", "normal", "champion", "unique", "minion"}[k]
}

// maxMonumod is the highest row id of monumod.txt (V: real file, 30 aura enchanted is the last used id). Enchant bytes
// above it are treated as garbage from a wrong offset.
const maxMonumod = 40

// Kind classifies the monster from MonsterData+0x16. This and EnchantIDs are the ONLY place that interprets the
// flag bits and the enchant array: the offsets (+0x16 flags, +0x1C enchants) and bit meanings (2 normal, 4 champion,
// 8 unique/boss, 16 minion) come from D2BS and are UNVERIFIED live (only town NPCs, all zero, were read). Until a
// live census confirms them, a result of MonKindUnknown means "do not conclude anything", never "normal".
func (m *MonsterInfo) Kind() MonKind {
	if m == nil {
		return MonKindUnknown
	}
	switch f := m.Flags; {
	case f&monFlagBoss != 0:
		return MonKindUnique
	case f&monFlagChamp != 0:
		return MonKindChampion
	case f&monFlagMinion != 0:
		return MonKindMinion
	case f&monFlagNormal != 0:
		return MonKindNormal
	}
	return MonKindUnknown
}

// EnchantIDs returns the monumod ids of the monster, or nil when they are not trustworthy: the monster is not known
// to be a champion/unique/minion (UNVERIFIED offset, see Kind) or any byte is out of the monumod range. Zero bytes
// are empty slots. ok is false when the data is unknown, as opposed to a known monster without modifiers.
func (m *MonsterInfo) EnchantIDs() (ids []uint8, ok bool) {
	if m == nil {
		return nil, false
	}
	switch m.Kind() {
	case MonKindUnknown:
		return nil, false
	case MonKindNormal:
		return nil, true
	}
	for _, id := range m.Enchants {
		if id == 0 {
			continue
		}
		if id > maxMonumod {
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}
