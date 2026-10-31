package d2

import (
	"encoding/binary"
	"errors"
	"testing"

	"d2lod-bot/internal/mem"
)

// One page holding a server-like unit: PlayerData with three quest buffers and three waypoint structs.
func progressPage() *mem.Fake {
	const base = 0x500000
	f := mem.NewFake()
	pg := make([]byte, 0x1000)
	p32 := func(o int, v uint32) { binary.LittleEndian.PutUint32(pg[o:], v) }
	p32(uoData, base+0x400)
	copy(pg[0x400:], "Multi")
	for d := 0; d < 3; d++ {
		p32(0x400+pdQuest+4*d, base+0x700+0x20*uint32(d))
		p32(0x700+0x20*d, base+0x800+0x80*uint32(d))
		binary.LittleEndian.PutUint16(pg[0x800+0x80*d+2*(d+1):], uint16(0x10*(d+1)+1))
		p32(0x400+pdWaypoint+4*d, base+0xA00+0x10*uint32(d))
		copy(pg[0xA00+0x10*d:], []byte{2, 1, byte(1 + d), 0x80, 0, 0, 0x01})
	}
	f.Put(base, pg)
	return f
}

func TestReadProgress(t *testing.T) {
	r := New(progressPage())
	p, err := r.ReadProgress(0x500000)
	if err != nil {
		t.Fatal(err)
	}
	for d := 0; d < 3; d++ {
		if got := p.Quest[d][d+1]; got != uint16(0x10*(d+1)+1) {
			t.Errorf("diff %d word %#x", d, got)
		}
		want := uint64(1+d) | 0x80<<8 | 0x01<<32
		if p.Waypoints[d] != want {
			t.Errorf("diff %d waypoints %#x want %#x", d, p.Waypoints[d], want)
		}
	}
}

func TestReadProgressBroken(t *testing.T) {
	f := progressPage()
	f.WriteAt([]byte{0, 0, 0, 0}, 0x500000+0x700) // QuestInfo.buf NULL
	if _, err := New(f).ReadProgress(0x500000); !errors.Is(err, ErrBadPointer) {
		t.Fatalf("err %v", err)
	}
	if _, err := New(mem.NewFake()).ReadProgress(0x500000); err == nil {
		t.Fatal("unmapped unit read")
	}
}

func TestFindServerPlayerNeedsRegions(t *testing.T) {
	if _, err := New(mem.NewFake()).FindServerPlayer(&Player{Name: "x"}); err == nil {
		t.Fatal("no region source, no error")
	}
}
