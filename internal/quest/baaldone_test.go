package quest

import "testing"

func TestBaalDead(t *testing.T) {
	src := &MapSource{}
	src.Set(0, BaalQuest, 0x0001) // done, normal
	src.Set(1, BaalQuest, 0x0010) // started only, nightmare
	r := NewReader(src)
	tests := []struct {
		diff        int
		dead, known bool
	}{
		{0, true, true},
		{1, false, true},
		{2, false, false}, // no word: unknown, not alive
	}
	for _, tc := range tests {
		got, err := BaalDead(r, tc.diff)
		if err != nil || got.Dead != tc.dead || got.Known != tc.known {
			t.Errorf("diff %d: got %+v err %v, want dead=%v known=%v", tc.diff, got, err, tc.dead, tc.known)
		}
	}
	if s, _ := BaalDead(r, 0); s.Evidence != EvMemory {
		t.Errorf("evidence %v", s.Evidence)
	}
	// 0x8001 (completed in a previous game) also counts
	src.Set(2, BaalQuest, 0x8001)
	if s, _ := BaalDead(r, 2); !s.Dead {
		t.Error("0x8001 must read dead")
	}
}

// Must fail: a difficulty outside 0..2 is an error, never a silent "not dead".
func TestBaalDeadBadDifficulty(t *testing.T) {
	for _, d := range []int{-1, 3} {
		if _, err := BaalDead(NewReader(), d); err == nil {
			t.Errorf("difficulty %d must error", d)
		}
	}
}
