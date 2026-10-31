package mpq

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"sort"
)

// Build serialises files into a minimal MPQ v1 image: each file is a single
// unit, zlib-compressed when that is smaller. It exists so tests (here and in
// internal/data) can make small archives from hand-written content; it is not
// used on the extraction path.
func Build(files map[string][]byte) []byte {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)

	var body bytes.Buffer
	body.Write(make([]byte, 32))
	blocks := make([]blockEntry, len(names))
	for i, n := range names {
		raw := files[n]
		stored, flags := raw, uint32(flagSingleUnit|flagExists)
		if len(raw) > 0 {
			var zb bytes.Buffer
			zb.WriteByte(cmpZlib)
			w := zlib.NewWriter(&zb)
			w.Write(raw)
			w.Close()
			if zb.Len() < len(raw) {
				stored, flags = zb.Bytes(), flags|flagCompress
			}
		}
		blocks[i] = blockEntry{uint32(body.Len()), uint32(len(stored)), uint32(len(raw)), flags}
		body.Write(stored)
	}
	hsize := 4
	for hsize < len(names)*2 {
		hsize *= 2
	}
	hashes := make([]hashEntry, hsize)
	for i := range hashes {
		hashes[i] = hashEntry{0xFFFFFFFF, 0xFFFFFFFF, 0xFFFF, 0xFFFF, hashEmpty}
	}
	for i, n := range names {
		h := hashString(n, 0) & uint32(hsize-1)
		for hashes[h].block != hashEmpty {
			h = (h + 1) & uint32(hsize-1)
		}
		hashes[h] = hashEntry{hashString(n, 1), hashString(n, 2), 0, 0, uint32(i)}
	}
	hpos := body.Len()
	hb := make([]byte, hsize*16)
	for i, h := range hashes {
		o := hb[i*16:]
		binary.LittleEndian.PutUint32(o, h.nameA)
		binary.LittleEndian.PutUint32(o[4:], h.nameB)
		binary.LittleEndian.PutUint16(o[8:], h.locale)
		binary.LittleEndian.PutUint16(o[10:], h.platform)
		binary.LittleEndian.PutUint32(o[12:], h.block)
	}
	encryptBlock(hb, hashString("(hash table)", 3))
	body.Write(hb)
	bpos := body.Len()
	bb := make([]byte, len(blocks)*16)
	for i, b := range blocks {
		o := bb[i*16:]
		binary.LittleEndian.PutUint32(o, b.pos)
		binary.LittleEndian.PutUint32(o[4:], b.csize)
		binary.LittleEndian.PutUint32(o[8:], b.fsize)
		binary.LittleEndian.PutUint32(o[12:], b.flags)
	}
	encryptBlock(bb, hashString("(block table)", 3))
	body.Write(bb)

	out := body.Bytes()
	copy(out, "MPQ\x1a")
	binary.LittleEndian.PutUint32(out[4:], 32)
	binary.LittleEndian.PutUint32(out[8:], uint32(len(out)))
	binary.LittleEndian.PutUint16(out[14:], 3)
	binary.LittleEndian.PutUint32(out[16:], uint32(hpos))
	binary.LittleEndian.PutUint32(out[20:], uint32(bpos))
	binary.LittleEndian.PutUint32(out[24:], uint32(hsize))
	binary.LittleEndian.PutUint32(out[28:], uint32(len(blocks)))
	return out
}

func encryptBlock(data []byte, key uint32) {
	cryptOnce.Do(initCrypt)
	var seed2 uint32 = 0xEEEEEEEE
	for i := 0; i+4 <= len(data); i += 4 {
		seed2 += cryptTable[0x400+(key&0xFF)]
		plain := binary.LittleEndian.Uint32(data[i:])
		binary.LittleEndian.PutUint32(data[i:], plain^(key+seed2))
		key = (^key<<21 + 0x11111111) | key>>11
		seed2 = plain + seed2 + seed2<<5 + 3
	}
}
