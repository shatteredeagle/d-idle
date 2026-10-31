package d2

import (
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/mem"
)

// chaosMem is an endless synthetic address space for robustness tests. Every aligned word is derived from
// (seed, address) so reads are repeatable, and the mode decides what the words look like.
type chaosMem struct {
	seed  uint64
	mode  int
	reads atomic.Int64
	limit int64
	// Hole makes some pages unreadable, like a game that unmaps memory during a level change.
	holes bool
}

const (
	modeRandom  = iota // uniform random words
	modeZero           // all zero
	modePtrLoop        // every word points at its own address (self loops)
	modePtrDens        // plausible pointers into the same space, small counts mixed in
	modeHuge           // counts and sizes at the extremes
	modeFF             // 0xFFFFFFFF
	numModes
)

var errChaosLimit = fmt.Errorf("chaos: read budget exhausted")

func mix(x uint64) uint64 {
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 33
	x *= 0xc4ceb9fe1a85ec53
	x ^= x >> 33
	return x
}

func (c *chaosMem) word(a uint32) uint32 {
	h := mix(c.seed ^ uint64(a)*0x9e3779b97f4a7c15)
	switch c.mode {
	case modeZero:
		return 0
	case modeFF:
		return 0xffffffff
	case modePtrLoop:
		return a &^ 3
	case modePtrDens:
		switch h % 8 {
		case 0:
			return 0
		case 1, 2:
			return uint32(h>>8)%0x40 + 1
		case 3:
			return a &^ 3
		default:
			return 0x400000 + uint32(h>>16)%0x2000000&^3
		}
	case modeHuge:
		switch h % 4 {
		case 0:
			return 0x7ffffff0
		case 1:
			return 0xffff
		case 2:
			return 0x10000 + uint32(h>>20)&^3
		default:
			return uint32(h >> 12)
		}
	}
	return uint32(h >> 11)
}

func (c *chaosMem) ReadAt(p []byte, addr uint32) error {
	if n := c.reads.Add(1); c.limit > 0 && n > c.limit {
		return errChaosLimit
	}
	if uint64(addr)+uint64(len(p)) > 1<<32 {
		return &mem.ReadError{Addr: addr, Len: len(p), Err: mem.ErrUnmapped}
	}
	if c.holes && mix(c.seed^uint64(addr>>12))%7 == 0 {
		return &mem.ReadError{Addr: addr, Len: len(p), Err: mem.ErrUnmapped}
	}
	for i := 0; i < len(p); {
		a := addr + uint32(i)
		var w [4]byte
		binary.LittleEndian.PutUint32(w[:], c.word(a&^3))
		i += copy(p[i:], w[a&3:])
	}
	return nil
}

func (c *chaosMem) Regions() []Region { return []Region{{0x400000, 0x480000}} }

// exercise calls every public Reader entry point once. It returns the first panic as an error.
func exercise(r *Reader, seed uint64) (err error) {
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("panic: %v", x)
		}
	}()
	a := uint32(mix(seed)>>8)&^3 | 0x10000
	r.Snapshot()
	r.State()
	r.UI()
	r.Mouse()
	r.ScreenSize()
	r.Viewport()
	r.Hovered()
	r.Inventory()
	r.GroundItems()
	for t := -1; t <= 6; t++ {
		r.Units(t)
	}
	r.MissileUnits()
	if p, e := r.Player(); e == nil && p != nil {
		r.StaffTombLevel(p.Addr)
		r.FindServerPlayer(p)
		r.ServerPlayerOK(a, p)
	}
	r.PlayerIdent()
	r.AreaOf(a)
	r.UnitStates(a)
	r.StatListStates(a)
	r.RawStatList(a, 64)
	r.ReadStatList(a)
	r.ReadBaseStatList(a)
	r.ReadStates(a)
	r.UnitStats(a)
	r.ReadProgress(a)
	if s, e := r.Snapshot(); e == nil && s != nil {
		FindMerc(s)
		if s.Player != nil {
			s.Player.LifePct()
			s.Player.ManaPct()
			s.Player.InTown()
			s.Player.Poisoned()
			s.Player.Curses(DefaultCurseIDs)
			WorldToScreen(s.Player.X, s.Player.Y, s.Viewport)
		}
		for i := range s.Monsters {
			s.Monsters[i].Alive()
		}
	}
	return nil
}

// runBounded runs fn and fails the test if it takes longer than d (a hang) or panics.
func runBounded(t testing.TB, d time.Duration, name string, fn func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
	case <-time.After(d):
		t.Fatalf("%s: hang (> %v)", name, d)
	}
}

func newChaosReader(seed uint64, mode int, holes bool) (*Reader, *chaosMem) {
	c := &chaosMem{seed: seed, mode: mode, holes: holes, limit: 3_000_000}
	r := New(c)
	r.Sleep = func(time.Duration) {}
	return r, c
}

func TestChaosReaderTable(t *testing.T) {
	for mode := 0; mode < numModes; mode++ {
		for seed := uint64(1); seed <= 12; seed++ {
			for _, holes := range []bool{false, true} {
				name := fmt.Sprintf("mode%d/seed%d/holes%v", mode, seed, holes)
				r, c := newChaosReader(seed, mode, holes)
				runBounded(t, 20*time.Second, name, func() error { return exercise(r, seed) })
				if c.reads.Load() > c.limit {
					t.Errorf("%s: %d reads, unbounded work (budget %d)", name, c.reads.Load(), c.limit)
				}
			}
		}
	}
}

func FuzzChaosReader(f *testing.F) {
	for m := 0; m < numModes; m++ {
		f.Add(uint64(m+1), uint8(m), false)
	}
	f.Add(uint64(99), uint8(modePtrDens), true)
	f.Fuzz(func(t *testing.T, seed uint64, mode uint8, holes bool) {
		r, c := newChaosReader(seed, int(mode)%numModes, holes)
		runBounded(t, 20*time.Second, "chaos", func() error { return exercise(r, seed) })
		if c.reads.Load() > c.limit {
			t.Fatalf("%d reads: unbounded work", c.reads.Load())
		}
	})
}

// A random-corruption pass over a Fake that holds a valid-looking world is covered by the fixture tests; here
// we flip bytes of the zero and dense spaces through a mutating wrapper.
type flipMem struct {
	mem.Source
	seed uint64
	rate uint64
}

func (f flipMem) ReadAt(p []byte, a uint32) error {
	if err := f.Source.ReadAt(p, a); err != nil {
		return err
	}
	for i := range p {
		if mix(f.seed^uint64(a)+uint64(i))%f.rate == 0 {
			p[i] ^= byte(mix(f.seed + uint64(a) + uint64(i)))
		}
	}
	return nil
}

func TestTornReadsOnFakeFixture(t *testing.T) {
	for seed := uint64(1); seed <= 20; seed++ {
		base := &chaosMem{seed: seed, mode: modePtrDens, limit: 3_000_000}
		r := New(flipMem{base, seed, 5})
		r.Sleep = func(time.Duration) {}
		runBounded(t, 20*time.Second, fmt.Sprint("seed", seed), func() error { return exercise(r, seed) })
	}
}

// poisonMem overlays a recorded game snapshot with damage: random byte flips, extreme or self-pointing words
// and unreadable pages. It models half-written structs and unmapped memory during a loading screen.
type poisonMem struct {
	mem.Source
	seed      uint64
	flipRate  uint64 // one in N bytes flipped (0 = off)
	wordRate  uint64 // one in N aligned words replaced (0 = off)
	pageRate  uint64 // one in N pages unreadable (0 = off)
	reads     atomic.Int64
	readLimit int64
}

func (p *poisonMem) ReadAt(b []byte, a uint32) error {
	if p.readLimit > 0 && p.reads.Add(1) > p.readLimit {
		return errChaosLimit
	}
	if p.pageRate > 0 {
		for pg := a >> 12; pg <= (a+uint32(len(b))-1)>>12 && len(b) > 0; pg++ {
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
		if p.flipRate > 0 && mix(p.seed^uint64(ad)*7)%p.flipRate == 0 {
			b[i] ^= byte(mix(p.seed+uint64(ad)) | 1)
		}
		if p.wordRate > 0 && mix(p.seed^uint64(ad&^3)*13)%p.wordRate == 0 {
			h := mix(p.seed + uint64(ad&^3))
			var w uint32
			switch h % 5 {
			case 0:
				w = 0xffffffff
			case 1:
				w = 0x7ffffff0
			case 2:
				w = ad &^ 3
			case 3:
				w = 0
			default:
				w = uint32(h >> 20)
			}
			b[i] = byte(w >> (8 * (ad & 3)))
		}
	}
	return nil
}

func TestPoisonedFixtures(t *testing.T) {
	for _, name := range []string{"town.snap", "town_inv.snap"} {
		base, err := mem.LoadSnapshot("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		for seed := uint64(1); seed <= 150; seed++ {
			p := &poisonMem{Source: base, seed: seed, readLimit: 2_000_000}
			switch seed % 5 {
			case 0:
				p.flipRate = 2000
			case 1:
				p.wordRate = 300
			case 2:
				p.pageRate = 3
			case 3:
				p.wordRate, p.flipRate = 60, 500
			default:
				p.wordRate, p.pageRate = 30, 9
			}
			r := New(p)
			r.Sleep = func(time.Duration) {}
			runBounded(t, 20*time.Second, fmt.Sprintf("%s/seed%d", name, seed), func() error { return exercise(r, seed) })
			if p.reads.Load() > p.readLimit {
				t.Errorf("%s/seed%d: %d reads, unbounded work", name, seed, p.reads.Load())
			}
		}
	}
}

func FuzzPoisonedFixture(f *testing.F) {
	f.Add(uint64(1), uint16(2000), uint16(300), uint8(0), false)
	f.Add(uint64(2), uint16(0), uint16(30), uint8(9), true)
	f.Fuzz(func(t *testing.T, seed uint64, flip, word uint16, page uint8, inv bool) {
		name := "town.snap"
		if inv {
			name = "town_inv.snap"
		}
		base, err := mem.LoadSnapshot("testdata/" + name)
		if err != nil {
			t.Skip(err)
		}
		p := &poisonMem{Source: base, seed: seed, flipRate: uint64(flip), wordRate: uint64(word), pageRate: uint64(page), readLimit: 2_000_000}
		r := New(p)
		r.Sleep = func(time.Duration) {}
		runBounded(t, 20*time.Second, "poison", func() error { return exercise(r, seed) })
		if p.reads.Load() > p.readLimit {
			t.Fatalf("%d reads: unbounded work", p.reads.Load())
		}
	})
}
