package fidprobe

import (
	"context"
	"fmt"
	"os"

	"d2lod-bot/internal/cheat/d2s"
)

// FileSource reads the character file itself (the third reference next to game.exe and d2d). It parses
// what internal/cheat/d2s understands: the header, the character stat block (ids 0 to 15), the skill
// block, quest words and waypoints. It does NOT parse items, so items.* is absent (UNREADABLE in a diff).
// Read-only: the file is read once and never written. Layout VERIFIED by internal/cheat/d2s against real
// 1.14d saves (byte-identical re-encode); the meaning of the stat values VERIFIED LIVE in run 1 (the file
// explained current life and mana at load, fidprobe.md).
type FileSource struct{ Path string }

// Name implements Source.
func (f FileSource) Name() string { return "file" }

// Read implements Source. A file is not a Snapshot, so ReadStable's FromInput path is bypassed through
// SheetSource.
func (f FileSource) Read(context.Context) (Input, error) {
	return Input{}, fmt.Errorf("fidprobe: file source has no snapshot; use Sheet()")
}

// SheetSource is a Source that builds the Sheet directly (the file reader).
type SheetSource interface {
	Source
	Sheet() (*Sheet, error)
}

// Sheet reads the file into a sheet.
func (f FileSource) Sheet() (*Sheet, error) {
	b, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}
	return SheetFromD2S(b)
}

// SheetFromD2S flattens a v96 character file to a Sheet. Fields the file cannot give are not added.
func SheetFromD2S(b []byte) (*Sheet, error) {
	rl, err := DefaultRules()
	if err != nil {
		return nil, err
	}
	d, err := d2s.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("fidprobe: file: %w", err)
	}
	s := NewSheet("file")
	st := func(id int) int64 { v, _ := d.Stat(id); return int64(v) }
	s.Int("base.class", int64(d.Class()))
	s.Int("base.level", int64(d.Level()))
	for i, k := range []string{"base.str", "base.ene", "base.dex", "base.vit", "base.stat_points", "base.skill_points"} {
		s.Int(k, st(i))
	}
	s.Int("base.exp", st(d2s.StatExp))
	s.Int("base.gold", st(d2s.StatGold))
	s.Int("base.stash_gold", st(d2s.StatStashGold))
	diff, act := -1, 0
	for i := 0; i < 3; i++ {
		if v := d.Difficulty(i); v&0x80 != 0 {
			diff, act = i, int(v&0x7f)
		}
	}
	if diff < 0 {
		s.Unreadable("base.difficulty", "no difficulty byte has the active bit")
		s.Unreadable("base.act", "no difficulty byte has the active bit")
	} else {
		s.Int("base.difficulty", int64(diff))
		s.Int("base.act", int64(act))
	}
	// Integers as game.exe shows them (the 8 fractional bits dropped) and the exact values for load.*.
	for i, n := range []string{"life", "max_life", "mana", "max_mana", "stamina", "max_stamina"} {
		raw := st(d2s.StatLife + i)
		s.Float("load."+n, float64(raw)/256)
		s.Int("derived."+n, raw>>8)
	}
	var sk []string
	first := rl.ClassSkillFirst[d.Class()%len(rl.ClassSkillFirst)]
	for i := 0; i < rl.ClassSkillCount; i++ {
		if lv := d.SkillLevel(first + i); lv != 0 {
			sk = append(sk, fmt.Sprintf("%d=%d", first+i, lv))
		}
	}
	s.Set("skills.base", sk)
	// The character block as raw values, nonzero only (the same normal form as stats.vector).
	var cv []string
	for _, x := range d.Stats {
		if x.Value != 0 {
			cv = append(cv, fmt.Sprintf("0/%d=%d", x.ID, x.Value))
		}
	}
	s.Set("statsfile.vector", cv)
	for dd := 0; dd < 3; dd++ {
		var wp, qw []string
		m := d.WaypointMask(dd)
		for bit := 0; bit < 64; bit++ {
			if m>>uint(bit)&1 != 0 {
				wp = append(wp, fmt.Sprint(bit))
			}
		}
		for n := 0; n < d2s.QuestWords; n++ {
			if w, err := d.QuestWord(dd, n); err == nil && w != 0 {
				qw = append(qw, fmt.Sprintf("w%d=%#x", n, w))
			}
		}
		s.Set(fmt.Sprintf("waypoints.d%d", dd), wp)
		s.Set(fmt.Sprintf("quests.d%d", dd), qw)
	}
	return s, nil
}
