package bot

import (
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

func TestLevelGapRefusalOnlyAfterNormal(t *testing.T) {
	if levelGapRefusal("normal", 1, 99) != "" {
		t.Fatal("Normal must never be refused by the gap rule")
	}
	if levelGapRefusal("nightmare", 40, 36) != "" || levelGapRefusal("nightmare", 31, 36) != "" {
		t.Fatal("nightmare margin 5: 31 vs 36 is fine")
	}
	if levelGapRefusal("nightmare", 30, 40) == "" {
		t.Fatal("nightmare: 10 below must be refused")
	}
	if levelGapRefusal("hell", 71, 79) != "" || levelGapRefusal("hell", 70, 79) == "" {
		t.Fatal("hell margin 8: 71 vs 79 passes, 70 does not")
	}
	if levelGapRefusal("hell", 60, 70) == "" {
		t.Fatal("hell: 10 or more below is refused")
	}
	if levelGapRefusal("hell", 60, 0) != "" {
		t.Fatal("unknown monster level passes")
	}
}

func TestAntidotesWantedHellOnly(t *testing.T) {
	g := newRig(t)
	mk := func(d d2.Difficulty, poison int) {
		setSnap(g.r, &d2.Snapshot{Player: &d2.Player{Class: d2.Sorceress, Level: 60, Difficulty: d, Resist: d2.Resists{Poison: poison}}})
	}
	mk(d2.Hell, 0)
	if n := g.r.antidotesWanted(); n != 2 {
		t.Fatalf("hell, poison 0: %d antidotes, want 2", n)
	}
	mk(d2.Nightmare, 0)
	if n := g.r.antidotesWanted(); n != 0 {
		t.Fatalf("nightmare: %d, want 0", n)
	}
	mk(d2.Normal, 0)
	g.r.killMu.Lock()
	g.r.killedNames = map[string]bool{"andariel": true}
	g.r.killMu.Unlock()
	if n := g.r.antidotesWanted(); n != 0 {
		t.Fatalf("normal after Andariel: %d, want 0 (unchanged)", n)
	}
}

type wSetter struct{ got combat.DeathWeights }

func (w *wSetter) SetDeathWeights(d combat.DeathWeights) { w.got = d }

func TestDeathWeightsNightmareHellOnly(t *testing.T) {
	g := newRig(t)
	r := g.r
	ws := &wSetter{}
	r.d.Fighter = struct {
		Fighter
		*wSetter
	}{r.d.Fighter, ws}
	di := combat.DeathInfo{Difficulty: d2.Normal, Affixes: combat.AffixCursed}
	r.noteDeathWeights(di, nil)
	if ws.got.Affix != nil || r.deathW.Affix != nil {
		t.Fatal("a Normal death must add no weight")
	}
	di.Difficulty = d2.Hell
	r.noteDeathWeights(di, nil)
	if ws.got.Affix["cursed"] != 1 {
		t.Fatalf("hell death weights %+v", ws.got)
	}
	r.noteDeathWeights(di, nil)
	if ws.got.Affix["cursed"] != 2 {
		t.Fatalf("weights must accumulate: %+v", ws.got)
	}
}
