package launch

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// Version is a four-part Windows file version, e.g. {1,14,3,71}.
type Version [4]uint16

func (v Version) String() string { return fmt.Sprintf("%d.%d.%d.%d", v[0], v[1], v[2], v[3]) }

// Want114d is the file version of Game.exe in Diablo II LoD 1.14d. VERIFIED on the live install.
var Want114d = Version{1, 14, 3, 71}

// FileVersion reads the numeric file version (VS_FIXEDFILEINFO.dwFileVersion) from the RT_VERSION
// resource of a PE file, in pure Go. Used to verify that the patch really brought Game.exe to 1.14d.
func FileVersion(path string) (Version, error) {
	f, err := pe.Open(path)
	if err != nil {
		return Version{}, err
	}
	defer f.Close()
	raw, err := os.ReadFile(path)
	if err != nil {
		return Version{}, err
	}
	return fileVersion(f, raw)
}

func fileVersion(f *pe.File, raw []byte) (Version, error) {
	var rsrcRVA uint32
	switch oh := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		rsrcRVA = oh.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress
	case *pe.OptionalHeader64:
		rsrcRVA = oh.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress
	}
	if rsrcRVA == 0 {
		return Version{}, errors.New("no resource directory")
	}
	rvaToOff := func(rva uint32) (int, bool) {
		for _, s := range f.Sections {
			if rva >= s.VirtualAddress && rva < s.VirtualAddress+max(s.VirtualSize, s.Size) {
				return int(rva-s.VirtualAddress) + int(s.Offset), true
			}
		}
		return 0, false
	}
	base, ok := rvaToOff(rsrcRVA)
	if !ok || base >= len(raw) {
		return Version{}, errors.New("resource directory outside file")
	}
	u16 := func(o int) int { return int(binary.LittleEndian.Uint16(raw[o:])) }
	u32 := func(o int) uint32 { return binary.LittleEndian.Uint32(raw[o:]) }
	// entry returns the offset (relative to base) of the n-th entry of the directory at dir whose ID is
	// want (want < 0: the first entry), and whether it is a subdirectory.
	entry := func(dir int, want int) (off int, sub bool, ok bool) {
		o := base + dir
		if o+16 > len(raw) {
			return 0, false, false
		}
		n := u16(o+12) + u16(o+14)
		for i := 0; i < n; i++ {
			e := o + 16 + 8*i
			if e+8 > len(raw) {
				return 0, false, false
			}
			id, to := int(u32(e)), u32(e+4)
			if want >= 0 && id != want {
				continue
			}
			return int(to & 0x7fffffff), to&0x80000000 != 0, true
		}
		return 0, false, false
	}
	const rtVersion = 16
	d1, sub, ok := entry(0, rtVersion)
	if !ok || !sub {
		return Version{}, errors.New("no RT_VERSION resource")
	}
	d2, sub, ok := entry(d1, -1)
	if !ok || !sub {
		return Version{}, errors.New("bad RT_VERSION name level")
	}
	de, sub, ok := entry(d2, -1)
	if !ok || sub || base+de+8 > len(raw) {
		return Version{}, errors.New("bad RT_VERSION language level")
	}
	dataRVA, size := u32(base+de), int(u32(base+de+4))
	doff, ok := rvaToOff(dataRVA)
	if !ok || doff+size > len(raw) {
		return Version{}, errors.New("version data outside file")
	}
	blob := raw[doff : doff+size]
	k := bytes.Index(blob, []byte{0xBD, 0x04, 0xEF, 0xFE}) // VS_FIXEDFILEINFO.dwSignature 0xFEEF04BD
	if k < 0 || k+16 > len(blob) {
		return Version{}, errors.New("VS_FIXEDFILEINFO not found")
	}
	ms, ls := binary.LittleEndian.Uint32(blob[k+8:]), binary.LittleEndian.Uint32(blob[k+12:])
	return Version{uint16(ms >> 16), uint16(ms), uint16(ls >> 16), uint16(ls)}, nil
}
