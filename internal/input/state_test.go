package input

import (
	"testing"

	"d2lod-bot/internal/mem"
)

func TestReaderOnFake(t *testing.T) {
	f := mem.NewFake()
	const unit, path = 0x1000000, 0x2000000
	f.PutU32(mem.Base+rvaPlayerUnit, unit)
	f.PutU32(unit+0x2C, path)
	// Path: xOffset=0x8000 xPos=6051 yOffset=0x8000 yPos=4963, target at +0x10
	f.Put(path, []byte{0x00, 0x80, 0xA3, 0x17, 0x00, 0x80, 0x63, 0x13, 0, 0, 0, 0, 0, 0, 0, 0, 0xA0, 0x17, 0x60, 0x13})
	r := Reader{Src: f}
	p, err := r.Player()
	if err != nil || p.X != 6051.5 || p.Y != 4963.5 || p.TargetX != 6048 || p.TargetY != 4960 {
		t.Fatalf("%+v %v", p, err)
	}
	f.PutU32(mem.Base+rvaUIVars, make([]uint32, 0x26)...)
	if open, err := r.UIOpen(); err != nil || open {
		t.Fatal(open, err)
	}
	f.PutU32(mem.Base+rvaUIVars+8*4, 1) // NPC menu
	if open, _ := r.UIOpen(); !open {
		t.Fatal("NPC menu not detected")
	}
	f.PutU32(mem.Base+0x3A6A94, 1)
	f.PutU32(mem.Base+0x3A6A78, 9)
	f.PutU32(mem.Base+0x3A6A8C, 1)
	if typ, id, err := r.Hovered(); err != nil || typ != 1 || id != 9 {
		t.Fatal(typ, id, err)
	}
}
