package cheat

import (
	"encoding/binary"
	"testing"
)

// synthFile hand-builds a minimal v96 save (zero header + markers, level-1 sorceress stats, empty items).
// Only the offsets in internal/cheat/d2s are used; no real save is involved.
func synthFile(t *testing.T, name string) []byte {
	t.Helper()
	const offStats = 0x2FD
	head := make([]byte, offStats)
	binary.LittleEndian.PutUint32(head[0:], 0xAA55AA55)
	binary.LittleEndian.PutUint32(head[4:], 0x60)
	copy(head[0x14:], name)
	head[0x24] = 0x20
	head[0x28] = 1
	head[0x2B] = 1
	head[0xA8] = 0x80
	copy(head[0x14F:], "Woo!")
	copy(head[0x279:], "WS")
	copy(head[0x2CA:], "w4")
	// stat block bits: ids 0,1,2,3 (10 bits), 6,7,8,9,10,11 (21 bits), 12 (7 bits), end 0x1FF
	type st struct {
		id   uint32
		bits uint
		v    uint32
	}
	stats := []st{{0, 10, 10}, {1, 10, 35}, {2, 10, 25}, {3, 10, 10}, {6, 21, 10240}, {7, 21, 10240}, {8, 21, 8960}, {9, 21, 8960}, {10, 21, 18944}, {11, 21, 18944}, {12, 7, 1}}
	var bits []byte
	pos := uint(0)
	put := func(v uint32, n uint) {
		for i := uint(0); i < n; i++ {
			if pos%8 == 0 {
				bits = append(bits, 0)
			}
			if v>>i&1 == 1 {
				bits[len(bits)-1] |= 1 << (pos % 8)
			}
			pos++
		}
	}
	for _, s := range stats {
		put(s.id, 9)
		put(s.v, s.bits)
	}
	put(0x1FF, 9)
	out := append(head, 'g', 'f')
	out = append(out, bits...)
	out = append(out, 'i', 'f')
	out = append(out, make([]byte, 30)...)
	out = append(out, 'J', 'M', 0, 0, 'j', 'f', 0, 0, 'k', 'f', 0)
	binary.LittleEndian.PutUint32(out[8:], uint32(len(out)))
	var sum uint32
	for i, c := range out {
		if i >= 12 && i < 16 {
			c = 0
		}
		sum = sum<<1 | sum>>31
		sum += uint32(c)
	}
	binary.LittleEndian.PutUint32(out[12:], sum)
	return out
}
