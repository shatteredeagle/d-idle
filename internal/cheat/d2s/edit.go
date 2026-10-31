package d2s

import (
	"encoding/binary"
	"fmt"
)

// Header accessors. Offsets VERIFIED (see the package comment) unless a method says otherwise.

func (f *File) Name() string {
	n := f.head[offName : offName+16]
	for i, c := range n {
		if c == 0 {
			return string(n[:i])
		}
	}
	return string(n)
}

func (f *File) Class() int            { return int(f.head[offClass]) }
func (f *File) Level() int            { return int(f.head[offLevel]) }
func (f *File) Status() byte          { return f.head[offStatus] }
func (f *File) Progression() byte     { return f.head[offProgression] }
func (f *File) SetProgression(v byte) { f.head[offProgression] = v }

// SetLevel writes the header level byte and the level stat together (the game reads both).
func (f *File) SetLevel(n int) error {
	if n < 1 || n > 99 {
		return fmt.Errorf("d2s: level %d out of range 1-99", n)
	}
	f.head[offLevel] = byte(n)
	return f.SetStat(StatLevel, uint32(n))
}

func (f *File) MapID() uint32 { return binary.LittleEndian.Uint32(f.head[offMapID:]) }

// Difficulty returns the byte of difficulty d (0 normal, 1 nightmare, 2 hell): bit 7 set = the active
// one, low 5 bits = the act (0-4) the character stands in.
func (f *File) Difficulty(d int) byte { return f.head[offDifficulty+d] }

// SetActive marks difficulty d active in act (0-4) and clears the flag on the others.
// Whether the game honors a difficulty the character has not unlocked is tested live (docs/cheats.md).
func (f *File) SetActive(d, act int) error {
	if d < 0 || d > 2 || act < 0 || act > 4 {
		return fmt.Errorf("d2s: difficulty %d act %d out of range", d, act)
	}
	for i := 0; i < 3; i++ {
		f.head[offDifficulty+i] &^= 0x80
	}
	f.head[offDifficulty+d] = 0x80 | byte(act)
	return nil
}

// Waypoints: 3 blocks of 24 bytes after the 8-byte "WS" header (VERIFIED position and size: the block ends
// at the "w4" marker). Per block: 2 header bytes, then a bit field where bit i is waypoint i of the 39 in the
// order Act I (9), II (9), III (9), IV (3), V (9). The waypoint NAME per bit is UNVERIFIED (PK table).
const wpBlock = 24

func (f *File) wpOff(d int) int { return offWaypoints + 8 + d*wpBlock + 2 }

// WaypointMask returns the 39 waypoint bits of difficulty d as a uint64.
func (f *File) WaypointMask(d int) uint64 {
	var m uint64
	for i := 0; i < 5; i++ {
		m |= uint64(f.head[f.wpOff(d)+i]) << (8 * i)
	}
	return m & (1<<39 - 1)
}

// SetWaypoints ORs mask (39 bits) into difficulty d; use AllWaypoints to open them all.
func (f *File) SetWaypoints(d int, mask uint64) error {
	if d < 0 || d > 2 {
		return fmt.Errorf("d2s: difficulty %d", d)
	}
	m := f.WaypointMask(d) | mask&(1<<39-1)
	for i := 0; i < 5; i++ {
		f.head[f.wpOff(d)+i] = byte(m >> (8 * i))
	}
	return nil
}

const AllWaypoints uint64 = 1<<39 - 1

// Skills: the 30-byte block after "if" holds the skill levels in the class's skill order (skills.txt
// order within the class). VERIFIED position/length; per-class first skill ids are standard.
var classFirstSkill = [7]int{6, 36, 66, 96, 126, 221, 251}

// SkillLevel returns the hard points in the skill with game id skillID (0 when it is not of this class).
func (f *File) SkillLevel(skillID int) int {
	i := skillID - classFirstSkill[f.Class()%7]
	if i < 0 || i >= 30 {
		return 0
	}
	return int(f.tail[2+i])
}

// SetSkillLevel sets the hard points of a skill of this class.
func (f *File) SetSkillLevel(skillID, level int) error {
	i := skillID - classFirstSkill[f.Class()%7]
	if i < 0 || i >= 30 || level < 0 || level > 255 {
		return fmt.Errorf("d2s: skill %d (level %d) is not a skill of class %d", skillID, level, f.Class())
	}
	f.tail[2+i] = byte(level)
	return nil
}

// SkillBlock returns a copy of the 30 skill bytes.
func (f *File) SkillBlock() [30]byte {
	var b [30]byte
	copy(b[:], f.tail[2:32])
	return b
}

// ItemSection returns the bytes after the skill block (unchanged by every edit), for tests.
func (f *File) ItemSection() []byte { return f.tail[32:] }

// Quests returns a copy of the 96 quest bytes of difficulty d (raw; their meaning per act is not decoded here).
func (f *File) Quests(d int) []byte {
	o := offQuests + 10 + d*96
	return append([]byte(nil), f.head[o:o+96]...)
}

// Quest words: 48 little-endian uint16 per difficulty after the 10-byte "Woo!" header. The block position
// and size are VERIFIED (real saves); what a given word index or bit means is NOT decided here (see
// internal/saves, docs/saves.md: tags S/U). These accessors are raw on purpose.
const (
	QuestWords = 48
	questBlock = 96
)

func questOff(d, i int) int { return offQuests + 10 + d*questBlock + 2*i }

func checkQuest(d, i int) error {
	if d < 0 || d > 2 || i < 0 || i >= QuestWords {
		return fmt.Errorf("d2s: quest word %d of difficulty %d out of range", i, d)
	}
	return nil
}

// QuestWord returns word i (0-47) of difficulty d.
func (f *File) QuestWord(d, i int) (uint16, error) {
	if err := checkQuest(d, i); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(f.head[questOff(d, i):]), nil
}

// SetQuestWord overwrites word i of difficulty d.
func (f *File) SetQuestWord(d, i int, w uint16) error {
	if err := checkQuest(d, i); err != nil {
		return err
	}
	binary.LittleEndian.PutUint16(f.head[questOff(d, i):], w)
	return nil
}

// ClearQuests zeroes all 48 words of difficulty d.
func (f *File) ClearQuests(d int) error {
	if d < 0 || d > 2 {
		return fmt.Errorf("d2s: difficulty %d", d)
	}
	o := questOff(d, 0)
	clear(f.head[o : o+questBlock])
	return nil
}

// SetWaypointMask replaces the 39 waypoint bits of difficulty d exactly (SetWaypoints only adds).
func (f *File) SetWaypointMask(d int, mask uint64) error {
	if d < 0 || d > 2 {
		return fmt.Errorf("d2s: difficulty %d", d)
	}
	for i := 0; i < 5; i++ {
		f.head[f.wpOff(d)+i] = byte(mask >> (8 * i))
	}
	return nil
}

// Clone returns an independent copy (edits to one never show in the other).
func (f *File) Clone() *File {
	return &File{head: append([]byte(nil), f.head...), Stats: append([]Stat(nil), f.Stats...), tail: append([]byte(nil), f.tail...)}
}
