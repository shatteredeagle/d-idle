// Package d2s reads and edits the header, stat and skill sections of a classic Diablo II LoD character
// file (.d2s, version 0x60 = 96, patches 1.10 to 1.14d). It does NOT understand the item section: every
// byte from the skill block on is copied through untouched. Only for the throwaway test characters
// (see internal/cheat); never for real data.
//
// Sources: Phrozen Keep "Diablo II Character File Format" (1.10+ layout, the d2s header offsets, the "gf"
// stat stream with 9-bit ids and per-stat bit widths, the checksum algorithm) and the open-source parsers
// nokka/d2s (Go) and dschu012/d2s (JS), whose version-96 branches match. The D2R (version 0x61 / 105)
// layout is different and is rejected. Layout facts below are tagged VERIFIED when a real 1.14d save
// (Multi/Botty/Testy of this project) parsed and re-encoded byte-identically, UNVERIFIED otherwise.
package d2s

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Header offsets (v96). VERIFIED against three real saves: signature, version, size, checksum, name,
// status, class, level, difficulty bytes and the section markers all sat where listed.
const (
	offSignature   = 0x00
	offVersion     = 0x04
	offSize        = 0x08
	offChecksum    = 0x0C
	offName        = 0x14 // 16 bytes, NUL padded
	offStatus      = 0x24 // bit 2 hardcore, 3 died, 5 expansion, 6 ladder
	offProgression = 0x25
	offClass       = 0x28
	offLevel       = 0x2B
	offLastPlayed  = 0x30
	offDifficulty  = 0xA8 // 3 bytes (normal, nightmare, hell): bit 7 active, low bits = act (0-4)
	offMapID       = 0xAB
	offQuests      = 0x14F // "Woo!" + version + size, then 3 x 96 quest bytes
	offWaypoints   = 0x279 // "WS" + version + size, then 3 x 24 bytes
	offNPC         = 0x2CA // "w4" at 714
	offStats       = 0x2FD // "gf" at 765
)

const (
	Signature = 0xAA55AA55
	Version   = 0x60
)

// Status flag bits at offStatus.
const (
	FlagHardcore  = 0x04
	FlagDied      = 0x08
	FlagExpansion = 0x20
	FlagLadder    = 0x40
)

// Stat ids stored in the "gf" block (the character section only holds ids 0-15).
const (
	StatStrength, StatEnergy, StatDexterity, StatVitality = 0, 1, 2, 3
	StatPoints, StatSkillPoints                           = 4, 5
	StatLife, StatMaxLife, StatMana, StatMaxMana          = 6, 7, 8, 9
	StatStamina, StatMaxStamina                           = 10, 11
	StatLevel, StatExp, StatGold, StatStashGold           = 12, 13, 14, 15
)

// statBits are the csvbits of ItemStatCost for ids 0-15 (PK). VERIFIED: the 1.14d saves decode with them
// and re-encode identically. Life/mana/stamina values carry 8 fractional bits (10240 = 40 life).
var statBits = [16]uint{10, 10, 10, 10, 10, 8, 21, 21, 21, 21, 21, 21, 7, 32, 25, 25}

var (
	ErrFormat   = errors.New("d2s: not a version 96 (1.10-1.14d) character file")
	ErrChecksum = errors.New("d2s: checksum mismatch")
)

// Stat is one entry of the character stat block.
type Stat struct {
	ID    int
	Value uint32
}

// File is a parsed character. Edit the exported fields through the methods; Bytes re-encodes.
type File struct {
	head  []byte // bytes [0:offStats): header, quests, waypoints, NPC data
	Stats []Stat
	tail  []byte // from the "if" skill block to the end: skills + items + corpse + golem, untouched
}

// Checksum computes the d2s checksum of b as if the checksum field were zero: for each byte,
// sum = rotl(sum, 1) + byte (PK). VERIFIED on three real saves.
func Checksum(b []byte) uint32 {
	var sum uint32
	for i, c := range b {
		if i >= offChecksum && i < offChecksum+4 {
			c = 0
		}
		sum = sum<<1 | sum>>31
		sum += uint32(c)
	}
	return sum
}

// Parse decodes a v96 save. It verifies signature, version, size and checksum.
func Parse(b []byte) (*File, error) {
	if len(b) < offStats+4+2+32 {
		return nil, fmt.Errorf("%w: only %d bytes", ErrFormat, len(b))
	}
	if binary.LittleEndian.Uint32(b[offSignature:]) != Signature {
		return nil, fmt.Errorf("%w: bad signature", ErrFormat)
	}
	if v := binary.LittleEndian.Uint32(b[offVersion:]); v != Version {
		return nil, fmt.Errorf("%w: version 0x%x", ErrFormat, v)
	}
	if sz := binary.LittleEndian.Uint32(b[offSize:]); int(sz) != len(b) {
		return nil, fmt.Errorf("%w: header says %d bytes, file has %d", ErrFormat, sz, len(b))
	}
	if got, want := Checksum(b), binary.LittleEndian.Uint32(b[offChecksum:]); got != want {
		return nil, fmt.Errorf("%w: stored 0x%x computed 0x%x", ErrChecksum, want, got)
	}
	for _, m := range []struct {
		off int
		s   string
	}{{offQuests, "Woo!"}, {offWaypoints, "WS"}, {offStats, "gf"}} {
		if string(b[m.off:m.off+len(m.s)]) != m.s {
			return nil, fmt.Errorf("%w: section %q not at 0x%x", ErrFormat, m.s, m.off)
		}
	}
	f := &File{head: append([]byte(nil), b[:offStats]...)}
	br := bitReader{b: b[offStats+2:]}
	for {
		id, ok := br.read(9)
		if !ok {
			return nil, fmt.Errorf("%w: stat block runs off the file", ErrFormat)
		}
		if id == 0x1FF {
			break
		}
		if id >= uint32(len(statBits)) {
			return nil, fmt.Errorf("%w: unsupported stat id %d in the character block", ErrFormat, id)
		}
		v, ok := br.read(statBits[id])
		if !ok {
			return nil, fmt.Errorf("%w: truncated stat %d", ErrFormat, id)
		}
		f.Stats = append(f.Stats, Stat{int(id), v})
	}
	end := offStats + 2 + br.bytesUsed()
	if end+2 > len(b) || string(b[end:end+2]) != "if" {
		return nil, fmt.Errorf("%w: skill block \"if\" not after the stat block (at %d)", ErrFormat, end)
	}
	f.tail = append([]byte(nil), b[end:]...)
	return f, nil
}

// Bytes re-encodes the file: header + stat block + untouched tail, with the size and checksum recomputed.
func (f *File) Bytes() []byte {
	bw := bitWriter{}
	for _, s := range f.Stats {
		bw.write(uint32(s.ID), 9)
		bw.write(s.Value, statBits[s.ID])
	}
	bw.write(0x1FF, 9)
	out := append([]byte(nil), f.head...)
	out = append(out, 'g', 'f')
	out = append(out, bw.bytes()...)
	out = append(out, f.tail...)
	binary.LittleEndian.PutUint32(out[offSize:], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[offChecksum:], Checksum(out))
	return out
}

// Stat returns the value of stat id and whether it is present (absent stats are zero in the game).
func (f *File) Stat(id int) (uint32, bool) {
	for _, s := range f.Stats {
		if s.ID == id {
			return s.Value, true
		}
	}
	return 0, false
}

// SetStat sets or inserts a stat, keeping the ids ascending (the order the game writes). A value of 0 for
// a stat the game omits when zero (everything except the attributes, life/mana/stamina and level) removes it.
func (f *File) SetStat(id int, v uint32) error {
	if id < 0 || id >= len(statBits) {
		return fmt.Errorf("d2s: stat id %d not in the character block (0-15)", id)
	}
	if max := uint64(1)<<statBits[id] - 1; uint64(v) > max {
		return fmt.Errorf("d2s: stat %d value %d exceeds %d bits", id, v, statBits[id])
	}
	for i, s := range f.Stats {
		if s.ID == id {
			if v == 0 && omitWhenZero(id) {
				f.Stats = append(f.Stats[:i], f.Stats[i+1:]...)
			} else {
				f.Stats[i].Value = v
			}
			return nil
		}
		if s.ID > id {
			if v == 0 && omitWhenZero(id) {
				return nil
			}
			f.Stats = append(f.Stats[:i], append([]Stat{{id, v}}, f.Stats[i:]...)...)
			return nil
		}
	}
	if v != 0 || !omitWhenZero(id) {
		f.Stats = append(f.Stats, Stat{id, v})
	}
	return nil
}

func omitWhenZero(id int) bool {
	switch id {
	case StatPoints, StatSkillPoints, StatExp, StatGold, StatStashGold:
		return true
	}
	return false
}
