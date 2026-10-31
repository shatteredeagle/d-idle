package d2map

import (
	"bytes"
	"strings"
	"testing"

	"d2lod-bot/internal/mem"
)

func TestGridGrowRegionASCII(t *testing.T) {
	g := ParseASCII(10, 20,
		"####",
		"#..#",
		"#.?#", // '?' stays unknown
		"####")
	if g.Bounds() != (Rect{10, 20, 4, 4}) || g.KnownCount() != 15 {
		t.Fatalf("%+v %d", g.Bounds(), g.KnownCount())
	}
	if g.Known(12, 22) || !g.Walkable(11, 21) || g.Walkable(10, 20) {
		t.Fatal("cells")
	}
	v := g.Version()
	g.SetBlock(Rect{-5, 100, 2, 1}, []uint16{0, ClosedDoor}) // grows in both directions
	if g.Version() == v || !g.Walkable(-5, 100) || !g.Known(-4, 100) || !g.Walkable(11, 21) {
		t.Fatal("grow lost data")
	}
	if g.SetBlock(Rect{-5, 100, 2, 1}, []uint16{0, ClosedDoor}) {
		t.Fatal("identical block reported as change")
	}
	r := g.Region(10, 20, 13, 22)
	if got := r.ASCII(Rect{}, nil); got != "###\n#..\n" {
		t.Fatalf("region %q", got)
	}
	if g.Region(500, 500, 600, 600).KnownCount() != 0 {
		t.Fatal("empty region")
	}
	c := g.Clone()
	c.SetBlock(Rect{10, 20, 1, 1}, []uint16{0})
	if g.Walkable(10, 20) {
		t.Fatal("clone shares storage")
	}
	out := g.ASCII(Rect{10, 20, 4, 3}, map[Point]rune{{11, 21}: '@'})
	if out != "####\n#@.#\n#. #\n" {
		t.Fatalf("ascii %q", out)
	}
	if !strings.Contains(g.ASCII(Rect{-5, 100, 2, 1}, nil), ".+") {
		t.Fatal("door glyph")
	}
}

func TestSetBlockRejectsBadInput(t *testing.T) {
	g := NewGrid()
	if g.SetBlock(Rect{0, 0, 2, 2}, []uint16{1}) || g.SetBlock(Rect{0, 0, 0, 5}, nil) {
		t.Fatal("bad blocks accepted")
	}
	if g.Bounds() != (Rect{}) || g.Walkable(0, 0) {
		t.Fatal("empty grid")
	}
}

func TestImageRoundTripAndRecorder(t *testing.T) {
	f := mem.NewFake()
	f.PutU32(0x1000, 1, 2, 3)
	f.PutU32(0x1008, 99) // overlaps the first segment; Fake prefers the later start
	rec := NewRecorder(f)
	for _, a := range []uint32{0x1000, 0x1004, 0x1008, 0x1010} {
		_, _ = mem.U32(rec, a) // the last one fails and must not be recorded
	}
	img := rec.Image()
	var buf bytes.Buffer
	if err := img.Save(&buf); err != nil {
		t.Fatal(err)
	}
	img2, err := LoadImage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if v, err := mem.U32(img2, 0x1004); err != nil || v != 2 {
		t.Fatalf("%d %v", v, err)
	}
	if v, err := mem.U32(img2, 0x1008); err != nil || v != 99 {
		t.Fatalf("%d %v", v, err)
	}
	if _, err := mem.U32(img2, 0x1010); err == nil {
		t.Fatal("unrecorded read succeeded")
	}
	if _, err := LoadImage(strings.NewReader("junk")); err == nil {
		t.Fatal("junk loaded")
	}
}
