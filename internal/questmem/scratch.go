package questmem

import (
	"encoding/binary"

	"d2lod-bot/internal/cheat/d2s"
)

// scratchSave builds a minimal valid empty .d2s (same layout as the saves tests' builder) so the quest and
// waypoint tables of internal/saves can be queried without a real file. It never touches disk.
func scratchSave() []byte {
	h := make([]byte, 0x2FD)
	binary.LittleEndian.PutUint32(h[0:], d2s.Signature)
	binary.LittleEndian.PutUint32(h[4:], d2s.Version)
	copy(h[0x14:], "Scratch")
	h[0x24] = d2s.FlagExpansion
	h[0x28] = 1
	h[0x2B] = 1
	h[0xA8] = 0x80
	copy(h[0x14F:], "Woo!")
	copy(h[0x279:], "WS")
	copy(h[0x2CA:], "w4")
	b := append(h, 'g', 'f', 0xFF, 0x01)
	b = append(b, 'i', 'f')
	b = append(b, make([]byte, 30)...)
	b = append(b, 'J', 'M', 0, 0)
	binary.LittleEndian.PutUint32(b[8:], uint32(len(b)))
	binary.LittleEndian.PutUint32(b[0xC:], d2s.Checksum(b))
	return b
}
