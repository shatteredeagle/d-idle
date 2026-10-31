package data

import (
	"encoding/binary"
	"errors"
	"strings"
)

// ParseTBL parses a D2 string table (.tbl) into key -> text. Layout (all
// little-endian): u16 crc, u16 numElements, u32 hashSize, u8 version,
// u32 dataStart, u32 maxLoop, u32 fileSize, u16 index[numElements], then
// hashSize 17-byte nodes {u8 used; u16 idx; u32 hash; u32 keyOff; u32 valOff;
// u16 valLen}. Offsets are from the start of the file.
func ParseTBL(b []byte) (map[string]string, error) {
	if len(b) < 21 {
		return nil, errors.New("tbl: too short")
	}
	n := int(binary.LittleEndian.Uint16(b[2:]))
	hs := int(binary.LittleEndian.Uint32(b[4:]))
	nodes := 21 + n*2
	if hs < 0 || nodes+hs*17 > len(b) {
		return nil, errors.New("tbl: bad sizes")
	}
	out := make(map[string]string, n)
	for i := 0; i < hs; i++ {
		e := b[nodes+i*17:]
		if e[0] == 0 {
			continue
		}
		ko := int(binary.LittleEndian.Uint32(e[7:]))
		vo := int(binary.LittleEndian.Uint32(e[11:]))
		if ko >= len(b) || vo >= len(b) {
			continue
		}
		out[cstr(b[ko:])] = cstr(b[vo:])
	}
	return out, nil
}

func cstr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// CleanString strips D2 colour codes (0xFF 'c' digit, or U+00FF 'c' digit in
// UTF-8) and newlines' trailing markers, leaving plain display text.
func CleanString(s string) string {
	for _, pre := range []string{"ÿ", "\xff"} {
		for {
			i := strings.Index(s, pre+"c")
			if i < 0 || i+len(pre)+2 > len(s) {
				break
			}
			s = s[:i] + s[i+len(pre)+2:]
		}
	}
	return s
}
