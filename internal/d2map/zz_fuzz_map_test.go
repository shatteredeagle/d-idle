package d2map

import (
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/mem"
)

func mix(x uint64) uint64 {
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 33
	x *= 0xc4ceb9fe1a85ec53
	x ^= x >> 33
	return x
}

// poison overlays a source with byte flips, extreme words and unreadable pages (torn memory during a level change).
type poison struct {
	mem.Source
	seed                 uint64
	flip, word, pageRate uint64
	reads                atomic.Int64
}

func (p *poison) ReadAt(b []byte, a uint32) error {
	p.reads.Add(1)
	if p.pageRate > 0 && len(b) > 0 {
		for pg := a >> 12; pg <= (a+uint32(len(b))-1)>>12; pg++ {
			if mix(p.seed^uint64(pg)*31)%p.pageRate == 0 {
				return &mem.ReadError{Addr: a, Len: len(b), Err: mem.ErrUnmapped}
			}
		}
	}
	if err := p.Source.ReadAt(b, a); err != nil {
		return err
	}
	for i := range b {
		ad := a + uint32(i)
		if p.flip > 0 && mix(p.seed^uint64(ad)*7)%p.flip == 0 {
			b[i] ^= byte(mix(p.seed+uint64(ad)) | 1)
		}
		if p.word > 0 && mix(p.seed^uint64(ad&^3)*13)%p.word == 0 {
			h := mix(p.seed + uint64(ad&^3))
			w := [5]uint32{0xffffffff, 0x7ffffff0, ad &^ 3, 0, uint32(h >> 20)}[h%5]
			b[i] = byte(w >> (8 * (ad & 3)))
		}
	}
	return nil
}

func runTracker(t testing.TB, src mem.Source, name string) {
	t.Helper()
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		tr := NewTracker(src)
		for i := 0; i < 6; i++ {
			tr.Update()
			if lm := tr.Current(); lm != nil {
				lm.Waypoints()
				lm.BorderCells(0)
				lm.Grid.Bounds()
				lm.Grid.ASCII(Rect{0, 0, 4, 4}, nil)
			}
			if i == 3 {
				tr.Reset()
			}
		}
		r := Reader{src}
		pc, _ := r.Player()
		act, _ := r.Act(pc.Act)
		for _, l := range r.Levels(act) {
			for _, rm := range r.LevelRoom2s(l) {
				r.Presets(rm)
				r.RoomTiles(rm)
			}
		}
	}()
	select {
	case x := <-done:
		if x != nil {
			t.Fatalf("%s: panic: %v", name, x)
		}
	case <-time.After(30 * time.Second):
		t.Fatalf("%s: hang", name)
	}
}

func TestMapPoisonedFixture(t *testing.T) {
	for _, f := range []string{"rogue_encampment.d2mi", "rogue_encampment_b.d2mi"} {
		fh, err := os.Open("testdata/" + f)
		if err != nil {
			t.Fatal(err)
		}
		img, err := LoadImage(fh)
		fh.Close()
		if err != nil {
			t.Fatal(err)
		}
		for seed := uint64(1); seed <= 120; seed++ {
			p := &poison{Source: img, seed: seed}
			switch seed % 4 {
			case 0:
				p.flip = 3000
			case 1:
				p.word = 400
			case 2:
				p.pageRate = 4
			default:
				p.word, p.flip = 50, 700
			}
			runTracker(t, p, fmt.Sprintf("%s/seed%d", f, seed))
		}
	}
}

func FuzzMapPoison(f *testing.F) {
	f.Add(uint64(1), uint16(3000), uint16(400), uint8(4))
	f.Add(uint64(7), uint16(0), uint16(20), uint8(0))
	fh, err := os.Open("testdata/rogue_encampment.d2mi")
	if err != nil {
		f.Fatal(err)
	}
	img, err := LoadImage(fh)
	fh.Close()
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, seed uint64, flip, word uint16, page uint8) {
		runTracker(t, &poison{Source: img, seed: seed, flip: uint64(flip), word: uint64(word), pageRate: uint64(page)}, "fuzz")
	})
}

// A coherent but absurd room (torn read of a freed Room1) must not make the grid allocate gigabytes.
func TestGridFarApartBlocks(t *testing.T) {
	g := NewGrid()
	g.SetBlock(Rect{0, 0, 4, 4}, make([]uint16, 16))
	g.SetBlock(Rect{1 << 30, -(1 << 30), 4, 4}, make([]uint16, 16))
	g.SetBlock(Rect{-(1 << 31), 5, 256, 256}, make([]uint16, 256*256))
	if b := g.Bounds(); b.W*b.H > 1<<25 {
		t.Fatalf("grid grew to %dx%d cells", b.W, b.H)
	}
}
