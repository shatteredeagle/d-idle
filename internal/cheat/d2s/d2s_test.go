package d2s

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// synth builds a minimal valid v96 file by hand (no real save data): zeroed header with the section
// markers, the given stats, a skill block and a tiny "JM" item list.
func synth(t testing.TB, class byte, name string, stats []Stat) []byte {
	t.Helper()
	head := make([]byte, offStats)
	binary.LittleEndian.PutUint32(head[offSignature:], Signature)
	binary.LittleEndian.PutUint32(head[offVersion:], Version)
	copy(head[offName:], name)
	head[offStatus] = FlagExpansion
	head[offClass] = class
	head[offLevel] = 1
	head[offDifficulty] = 0x80
	copy(head[offQuests:], "Woo!")
	copy(head[offWaypoints:], "WS")
	copy(head[offNPC:], "w4")
	f := &File{head: head, Stats: stats}
	f.tail = append([]byte("if"), make([]byte, 30)...)
	f.tail = append(f.tail, 'J', 'M', 0, 0, 'j', 'f', 0, 0, 'k', 'f', 0)
	return f.Bytes()
}

var baseStats = []Stat{{0, 10}, {1, 35}, {2, 25}, {3, 10}, {6, 10240}, {7, 10240}, {8, 8960}, {9, 8960}, {10, 18944}, {11, 18944}, {12, 1}}

func TestRoundTripSynthetic(t *testing.T) {
	b := synth(t, 1, "Testchar", baseStats)
	f, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.Bytes(), b) {
		t.Fatal("re-encode differs")
	}
	if f.Name() != "Testchar" || f.Class() != 1 || f.Level() != 1 {
		t.Fatalf("header %s %d %d", f.Name(), f.Class(), f.Level())
	}
}

func TestEditKeepsItemsAndChecksum(t *testing.T) {
	b := synth(t, 1, "Testchar", baseStats)
	f, _ := Parse(b)
	items := append([]byte(nil), f.ItemSection()...)
	if err := f.SetLevel(25); err != nil {
		t.Fatal(err)
	}
	f.SetStat(StatExp, 1640359)
	f.SetStat(StatGold, 990000)
	f.SetStat(StatStashGold, 2500000)
	f.SetStat(StatPoints, 120)
	f.SetStat(StatSkillPoints, 24)
	f.SetSkillLevel(36, 20)
	f.SetWaypoints(0, AllWaypoints)
	f.SetActive(1, 2)
	out := f.Bytes()
	g, err := Parse(out) // verifies size + checksum
	if err != nil {
		t.Fatal(err)
	}
	if g.Level() != 25 || g.SkillLevel(36) != 20 || g.WaypointMask(0) != AllWaypoints || g.Difficulty(1) != 0x82 || g.Difficulty(0)&0x80 != 0 {
		t.Fatal("edit lost")
	}
	for id, want := range map[int]uint32{StatExp: 1640359, StatGold: 990000, StatStashGold: 2500000, StatPoints: 120, StatSkillPoints: 24, StatLevel: 25} {
		if v, ok := g.Stat(id); !ok || v != want {
			t.Errorf("stat %d = %d,%v want %d", id, v, ok, want)
		}
	}
	if !bytes.Equal(g.ItemSection(), items) {
		t.Fatal("item section changed")
	}
	// ascending ids
	for i := 1; i < len(g.Stats); i++ {
		if g.Stats[i].ID <= g.Stats[i-1].ID {
			t.Fatal("stat order")
		}
	}
	// removing zero stats shrinks the file again
	g.SetStat(StatGold, 0)
	if _, ok := g.Stat(StatGold); ok {
		t.Fatal("zero gold kept")
	}
}

func TestRejects(t *testing.T) {
	b := synth(t, 1, "T", baseStats)
	bad := append([]byte(nil), b...)
	bad[offLevel] ^= 1 // checksum now wrong
	if _, err := Parse(bad); err == nil {
		t.Error("checksum not enforced")
	}
	v := append([]byte(nil), b...)
	binary.LittleEndian.PutUint32(v[offVersion:], 0x61)
	if _, err := Parse(v); err == nil {
		t.Error("D2R version accepted")
	}
	f, _ := Parse(b)
	if f.SetStat(StatLevel, 200) == nil {
		t.Error("overflow accepted")
	}
	if f.SetLevel(100) == nil {
		t.Error("level 100 accepted")
	}
}

// TestRealSaves round-trips copies of real saves when D2S_REAL points at a folder holding some. The
// files are never committed; copy them from the game first (docs/cheats.md).
func TestRealSaves(t *testing.T) {
	dir := os.Getenv("D2S_REAL")
	if dir == "" {
		t.Skip("set D2S_REAL to a folder with copies of .d2s files")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.d2s"))
	if len(files) == 0 {
		t.Skip("no .d2s files")
	}
	for _, p := range files {
		b, _ := os.ReadFile(p)
		f, err := Parse(b)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		if !bytes.Equal(f.Bytes(), b) {
			t.Errorf("%s: re-encode differs", p)
		}
		t.Logf("%s: %s class %d level %d stats %v", filepath.Base(p), f.Name(), f.Class(), f.Level(), f.Stats)
	}
}

func TestQuestWaypointAccessors(t *testing.T) {
	f, _ := Parse(synth(t, 0, "Testchar", baseStats))
	items := append([]byte(nil), f.ItemSection()...)
	orig := f.Clone()
	f.SetQuestWord(1, 47, 0xBEEF)
	f.SetQuestWord(1, 0, 0x0001)
	f.SetWaypoints(2, 0b101)
	f.SetWaypointMask(2, 0b100)
	g, err := Parse(f.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if w, _ := g.QuestWord(1, 47); w != 0xBEEF {
		t.Fatalf("%#x", w)
	}
	if q := g.Quests(1); q[94] != 0xEF || q[95] != 0xBE || q[0] != 1 {
		t.Fatal("raw bytes")
	}
	if w, _ := g.QuestWord(0, 47); w != 0 {
		t.Fatal("neighbour difficulty touched")
	}
	if g.WaypointMask(2) != 0b100 || g.WaypointMask(0) != 0 {
		t.Fatalf("mask %b", g.WaypointMask(2))
	}
	g.ClearQuests(1)
	if w, _ := g.QuestWord(1, 47); w != 0 {
		t.Fatal("not cleared")
	}
	if !bytes.Equal(g.ItemSection(), items) || !bytes.Equal(orig.Bytes(), synth(t, 0, "Testchar", baseStats)) {
		t.Fatal("items changed or Clone shares state")
	}
	if _, err := g.QuestWord(3, 0); err == nil {
		t.Fatal("difficulty 3 accepted")
	}
	if g.SetQuestWord(0, 48, 1) == nil {
		t.Fatal("word 48 accepted")
	}
}
