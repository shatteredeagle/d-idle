package task

import "testing"

// The Seven Tombs opens on the Staff made. The setup word 0x0c78 reads back as 0x0c08 after the join (bits 4 to 6
// cleared, V lab rerun 2026-11-27); bit 11, the Cube's mark, survives and alone says the Staff is made.
func TestStaffMadeFromTheQuestWord(t *testing.T) {
	for _, c := range []struct {
		raw  uint16
		want Stage
	}{
		{0x0c08, StageCleared},
		{0x0800, StageCleared},
		{0x0008, StageStarted},
	} {
		w := World{Quests: map[string]QuestState{QStaff: {Stage: StageStarted, Ev: EvMemory, Raw: c.raw}}}
		if got := w.Quest(QStaff).Stage; got != c.want {
			t.Errorf("word %#04x: stage %v, want %v", c.raw, got, c.want)
		}
	}
	w := World{Items: map[string]int{ItemStaff: 1}}
	if got := w.Quest(QStaff).Stage; got != StageCleared {
		t.Errorf("Staff in the pack: stage %v, want cleared", got)
	}
}
