package quest

import "fmt"

// BaalQuest is the quest id whose word means "Baal is dead in this difficulty" (saves quest table index 40,
// docs/progression/quests.md row Eve of Destruction).
//
// U: the game's real "difficulty finished" flag position is not read live. The best evidence is the Eve of
// Destruction word being Done (bit 0 or bit 15, Interpret), the same signal saves.Decode already uses for the
// next difficulty's Unlocked flag (internal/saves/progress.go) and quests.md names as the proxy. Nothing here was
// checked against a character that killed Baal; that is a live check (docs/diffready.md).
const BaalQuest = "eve_of_destruction"

// BaalState is one answer of BaalDead.
type BaalState struct {
	Dead     bool     // the Eve of Destruction word is Done
	Known    bool     // some source knew the word; when false Dead is false and means "unknown", not "alive"
	Evidence Evidence // how sure the answering source is (a proxy never counts as proof for a hop)
	Raw      uint16
}

// BaalDead reads whether Baal is dead in difficulty diff (0 normal, 1 nightmare, 2 hell). Hell has no next
// difficulty, but the question is still well formed. A difficulty outside 0..2 is an error naming it.
func BaalDead(r Reader, diff int) (BaalState, error) {
	if diff < 0 || diff > 2 {
		return BaalState{}, fmt.Errorf("quest: no difficulty %d (want 0 normal, 1 nightmare, 2 hell)", diff)
	}
	in := r.Quest(diff, BaalQuest)
	if in.Evidence == EvNone {
		return BaalState{}, nil
	}
	return BaalState{Dead: in.Status == Done, Known: true, Evidence: in.Evidence, Raw: in.Raw}, nil
}
