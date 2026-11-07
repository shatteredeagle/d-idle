package fidprobe

import (
	"encoding/binary"
	"testing"

	"d2lod-bot/internal/cheat/d2s"
	"d2lod-bot/internal/d2"
)

// Header offsets of the v96 layout (d2s package: VERIFIED against real saves). The fixture below is
// synthesized byte by byte: no Blizzard data and no save of a real character.
const (
	tOffName, tOffStatus, tOffClass, tOffLevel = 0x14, 0x24, 0x28, 0x2B
	tOffDiff, tOffQuests, tOffWP, tOffNPC      = 0xA8, 0x14F, 0x279, 0x2CA
	tOffStats                                  = 0x2FD
)

type bits struct {
	b []byte
	n uint
}

func (w *bits) put(v uint32, n uint) {
	for i := uint(0); i < n; i++ {
		if w.n%8 == 0 {
			w.b = append(w.b, 0)
		}
		if v>>i&1 != 0 {
			w.b[len(w.b)-1] |= 1 << (w.n % 8)
		}
		w.n++
	}
}

// synthD2S builds a Sorceress-class file with the given character stats (id, value), skill points in
// the first skill slots and one quest word and waypoint in difficulty 0.
func synthD2S(class byte, stats [][2]uint32, skills []byte) []byte {
	h := make([]byte, tOffStats)
	binary.LittleEndian.PutUint32(h[0:], d2s.Signature)
	binary.LittleEndian.PutUint32(h[4:], d2s.Version)
	copy(h[tOffName:], "Synth")
	h[tOffStatus] = d2s.FlagExpansion
	h[tOffClass] = class
	h[tOffLevel] = 1
	h[tOffDiff] = 0x80
	copy(h[tOffQuests:], "Woo!")
	copy(h[tOffWP:], "WS")
	copy(h[tOffNPC:], "w4")
	h[tOffQuests+10+2*3] = 0x11 // quest word 3 of difficulty 0
	h[tOffWP+8+2] = 0x01        // waypoint bit 0 of difficulty 0
	widths := [16]uint{10, 10, 10, 10, 10, 8, 21, 21, 21, 21, 21, 21, 7, 32, 25, 25}
	w := &bits{}
	for _, s := range stats {
		w.put(s[0], 9)
		w.put(s[1], widths[s[0]])
	}
	w.put(0x1FF, 9)
	out := append(h, 'g', 'f')
	out = append(out, w.b...)
	out = append(out, 'i', 'f')
	sk := make([]byte, 30)
	copy(sk, skills)
	out = append(out, sk...)
	out = append(out, 'J', 'M', 0, 0)
	binary.LittleEndian.PutUint32(out[8:], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[0xC:], d2s.Checksum(out))
	return out
}

func TestSheetFromD2S(t *testing.T) {
	// life 89.5 of 91, mana 24.4 of 45 (8 fractional bits, as in the first live run's Freshe).
	raw := synthD2S(1, [][2]uint32{{0, 10}, {1, 35}, {2, 25}, {3, 10}, {6, 89*256 + 128}, {7, 91 * 256}, {8, 6246}, {9, 45 * 256}, {12, 6}, {14, 500}}, []byte{1, 0, 3})
	s, err := SheetFromD2S(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"base.class": "1", "base.level": "1", "base.str": "10", "base.ene": "35", "base.gold": "500",
		"derived.life": "89", "derived.max_mana": "45", "load.life": "89.5", "load.max_life": "91", "load.mana": "24.3984375",
		"base.difficulty": "0", "base.act": "0"}
	for k, v := range want {
		if got := s.Fields[k].String(); got != v {
			t.Errorf("%s = %s, want %s", k, got, v)
		}
	}
	if sk := s.Fields["skills.base"]; len(sk.Set) != 2 || sk.Set[0] != "36=1" || sk.Set[1] != "38=3" {
		t.Errorf("skills.base %v", sk.Set)
	}
	if q := s.Fields["quests.d0"]; len(q.Set) != 1 || q.Set[0] != "w3=0x11" {
		t.Errorf("quests %v", q.Set)
	}
	if w := s.Fields["waypoints.d0"]; len(w.Set) != 1 || w.Set[0] != "0" {
		t.Errorf("waypoints %v", w.Set)
	}
	if _, ok := s.Fields["items.inventory"]; ok {
		t.Error("the file reader must not claim items")
	}
}

// A second build: a Paladin file reads its own skill range (build flexibility), and a corrupt file errors.
func TestSheetFromD2SPaladinAndBad(t *testing.T) {
	raw := synthD2S(3, [][2]uint32{{12, 4}}, []byte{2})
	s, err := SheetFromD2S(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sk := s.Fields["skills.base"]; len(sk.Set) != 1 || sk.Set[0] != "96=2" {
		t.Errorf("paladin skills %v", sk.Set)
	}
	raw[tOffStats+5] ^= 0xff
	if _, err := SheetFromD2S(raw); err == nil {
		t.Error("bad checksum must be refused")
	}
}

// File against a game.exe-style sheet: absent groups are UNREADABLE, never EQUAL, and a healed d2d
// value shows as MISMATCH on load.life.
func TestFileVsGameLoadLife(t *testing.T) {
	raw := synthD2S(1, [][2]uint32{{6, 89*256 + 128}, {7, 91 * 256}, {8, 6246}, {9, 45 * 256}}, nil)
	f, _ := SheetFromD2S(raw)
	g := NewSheet("d2d")
	g.Float("load.life", 91)
	g.Float("load.max_life", 91)
	g.Float("load.mana", 45)
	g.Float("load.max_mana", 45)
	r := Diff(f, g, []string{"load"}, rules(t))
	if classOf(r, "load.life") != Mismatch || classOf(r, "load.max_life") != Equal || classOf(r, "load.mana") != Mismatch {
		t.Fatalf("%s", r.Text(true))
	}
}

func TestNormalisation(t *testing.T) {
	sa := snap(1)
	sa.Player.Skills = append(sa.Player.Skills, skillAt(0, 1), skillAt(217, 1))
	sa.Player.Stats = map[uint32]int32{0: 20, 13: 0, 31: 50, 172: 1, 6: 89*256 + 128}
	sb := snap(1)
	sb.Player.Stats = map[uint32]int32{0: 20, 14: 0, 6: 91 * 256}
	a, b := sheet(t, "gameexe", sa), sheet(t, "d2d", sb)
	r := Diff(a, b, []string{"skills", "stats"}, rules(t))
	if c := classOf(r, "skills.base"); c != Equal {
		t.Errorf("skills.base %s\n%s", c, r.Text(true))
	}
	if c := classOf(r, "stats.vector"); c != Mismatch { // only stat 6 differs now: 89.5 vs 91
		t.Errorf("stats.vector %s\n%s", c, r.Text(true))
	}
	if x := a.Fields["statsx.vector"]; len(x.Set) != 2 {
		t.Errorf("scope-only stats %v", x.Set)
	}
}

func skillAt(id, lv int) d2.Skill { return d2.Skill{ID: id, Level: lv} }
