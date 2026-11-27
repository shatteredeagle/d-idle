package env

import "testing"

func TestBoardLegacyFollowsKnob(t *testing.T) {
	q := DefaultTuning().Quests
	if !q.BoardLegacy() {
		t.Fatal("default knob 0 must be the legacy board")
	}
	q.BoardSM = 1
	if q.BoardLegacy() {
		t.Fatal("knob 1 must not be legacy")
	}
}
