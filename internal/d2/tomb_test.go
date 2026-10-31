package d2

import (
	"encoding/binary"
	"errors"
	"testing"

	"d2lod-bot/internal/mem"
)

func tombPage(level uint32) *mem.Fake {
	const base = 0x500000
	f := mem.NewFake()
	pg := make([]byte, 0x1000)
	binary.LittleEndian.PutUint32(pg[uoPActObj:], base+0x200)
	binary.LittleEndian.PutUint32(pg[0x200+actPMisc:], base+0x400)
	binary.LittleEndian.PutUint32(pg[0x400+miscStaffTombLvl:], level)
	f.Put(base, pg)
	return f
}

func TestStaffTombLevel(t *testing.T) {
	for in, want := range map[uint32]int{68: 68, 66: 66, 72: 72, 0: 0, 65: 0, 73: 0} {
		got, err := New(tombPage(in)).StaffTombLevel(0x500000)
		if err != nil || got != want {
			t.Errorf("%d: got %d %v want %d", in, got, err, want)
		}
	}
	f := tombPage(68)
	f.WriteAt([]byte{0, 0, 0, 0}, 0x500000+0x200+actPMisc)
	if _, err := New(f).StaffTombLevel(0x500000); !errors.Is(err, ErrBadPointer) {
		t.Fatalf("err %v", err)
	}
}
