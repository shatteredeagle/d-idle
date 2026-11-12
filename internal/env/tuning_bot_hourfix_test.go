package env

import "testing"

// docs/hourfix.md: the first-hour defaults. Changing one needs the doc and a new hour's evidence.
func TestHourfixDefaults(t *testing.T) {
	b, l := DefaultBotTuning(), DefaultLootTuning()
	if b.ClearIdleMin != 3 {
		t.Errorf("bot.clearIdleMin = %d, want 3", b.ClearIdleMin)
	}
	if l.PoorMinSell != 20 {
		t.Errorf("loot.poorMinSell = %d, want 20", l.PoorMinSell)
	}
}
