package d2map

import (
	"testing"

	"d2lod-bot/internal/mem"
)

func TestResetForgetsLevels(t *testing.T) {
	tr := NewTracker(mem.NewFake())
	tr.levels[2] = &LevelMap{No: 2}
	tr.byAddr[1] = Level{No: 2}
	tr.Update() // not in a game: fails, nothing reset
	if len(tr.levels) != 1 {
		t.Fatal("levels dropped without a reset request")
	}
	tr.Reset()
	tr.Update()
	if len(tr.levels) != 0 || len(tr.byAddr) != 0 {
		t.Fatalf("reset not applied: %d levels, %d addr", len(tr.levels), len(tr.byAddr))
	}
}
