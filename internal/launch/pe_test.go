package launch

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// buildPE makes a minimal PE32 with one .rsrc section holding a VS_VERSION_INFO for version v.
func buildPE(v Version) []byte {
	le := binary.LittleEndian
	const (
		fileAlign = 0x200
		rsrcRVA   = 0x1000
	)
	// resource tree: root -> type 16 -> name 1 -> lang 0x409 -> data entry
	rsrc := make([]byte, 0x100)
	dir := func(off int, id uint32, childOff uint32, sub bool) {
		le.PutUint16(rsrc[off+14:], 1) // one ID entry
		le.PutUint32(rsrc[off+16:], id)
		if sub {
			childOff |= 0x80000000
		}
		le.PutUint32(rsrc[off+20:], childOff)
	}
	dir(0x00, 16, 0x18, true)
	dir(0x18, 1, 0x30, true)
	dir(0x30, 0x409, 0x48, false)
	// data entry at 0x48: RVA, size
	le.PutUint32(rsrc[0x48:], rsrcRVA+0x60)
	le.PutUint32(rsrc[0x4c:], 0x40)
	// VS_VERSION_INFO blob at 0x60: header (6 bytes) + "VS_VERSION_INFO\0" utf16 (32) -> padded to 0x28, then FFI
	ffi := 0x60 + 0x28
	le.PutUint32(rsrc[ffi:], 0xFEEF04BD)
	le.PutUint32(rsrc[ffi+4:], 0x10000)
	le.PutUint32(rsrc[ffi+8:], uint32(v[0])<<16|uint32(v[1]))
	le.PutUint32(rsrc[ffi+12:], uint32(v[2])<<16|uint32(v[3]))

	b := make([]byte, 0x400+fileAlign)
	copy(b, "MZ")
	le.PutUint32(b[0x3c:], 0x80)
	p := 0x80
	copy(b[p:], "PE\x00\x00")
	le.PutUint16(b[p+4:], 0x14c) // i386
	le.PutUint16(b[p+6:], 1)     // sections
	le.PutUint16(b[p+20:], 0xE0) // SizeOfOptionalHeader
	le.PutUint16(b[p+22:], 0x102)
	o := p + 24
	le.PutUint16(b[o:], 0x10b)
	le.PutUint32(b[o+28:], 0x400000) // ImageBase
	le.PutUint32(b[o+32:], 0x1000)   // SectionAlignment
	le.PutUint32(b[o+36:], fileAlign)
	le.PutUint32(b[o+56:], 0x2000) // SizeOfImage
	le.PutUint32(b[o+60:], 0x200)  // SizeOfHeaders
	le.PutUint32(b[o+92:], 16)     // NumberOfRvaAndSizes
	le.PutUint32(b[o+96+2*8:], rsrcRVA)
	le.PutUint32(b[o+96+2*8+4:], 0x100)
	s := o + 0xE0
	copy(b[s:], ".rsrc")
	le.PutUint32(b[s+8:], 0x100)
	le.PutUint32(b[s+12:], rsrcRVA)
	le.PutUint32(b[s+16:], 0x200)
	le.PutUint32(b[s+20:], 0x200)
	copy(b[0x200:], rsrc)
	return b[:0x400]
}

func TestFileVersionSynthetic(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.exe")
	os.WriteFile(p, buildPE(Want114d), 0o644)
	v, err := FileVersion(p)
	if err != nil || v != Want114d {
		t.Fatalf("v=%v err=%v", v, err)
	}
	if v.String() != "1.14.3.71" {
		t.Fatal(v.String())
	}
}

// TestFileVersionRealGameExe runs against a real Game.exe copied OUTSIDE the repo:
// D2_GAME_EXE=/path/to/Game.exe go test ./internal/launch -run Real
func TestFileVersionRealGameExe(t *testing.T) {
	p := os.Getenv("D2_GAME_EXE")
	if p == "" {
		t.Skip("D2_GAME_EXE not set")
	}
	v, err := FileVersion(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Game.exe file version %s", v)
	if v != Want114d {
		t.Fatalf("got %s want %s", v, Want114d)
	}
}
