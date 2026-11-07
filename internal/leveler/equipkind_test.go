package leveler

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

func kindCand(id uint32, class int, slot int) Cand {
	c := cand(id, "kind", 12, true, slot)
	c.Unit.Class = class
	return c
}

// g2g E1: a refused swap is remembered by item kind and slot, so the same Cap (a new unit id in every game) is not
// clicked again in the next game, for any slot or class; the level backoff then lifts it.
func TestEquipFailureMemoryByKindAcrossGames(t *testing.T) {
	for _, tc := range []struct {
		name        string
		class, slot int
		otherClass  int
		otherSlot   int
	}{
		{"cap to head", 20, d2.BodyHead, 21, d2.BodyFeet},
		{"gloves to hands", 31, d2.BodyGloves, 32, d2.BodyFeet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tn := env.DefaultTuning()
			lvl := 15
			l := &Leveler{e: &env.Env{Tune: &tn}, o: Options{Snapshot: func() (*d2.Snapshot, error) {
				return &d2.Snapshot{Player: &d2.Player{Level: lvl}}, nil
			}}}
			l.markBadSwap(Swap{Item: kindCand(38, tc.class, tc.slot), Slot: tc.slot})
			if l.isBadSlot(36, tc.class, tc.slot, lvl) {
				t.Fatal("one failure (equipKindFails 2) must still allow another try with a new id")
			}
			l.markBadSwap(Swap{Item: kindCand(36, tc.class, tc.slot), Slot: tc.slot}) // the next game: new id, same kind
			if !l.isBadSlot(51, tc.class, tc.slot, lvl) || !l.badEverywhere(51, tc.class, []int{tc.slot}, lvl) {
				t.Fatal("a third id of the failed kind must be skipped")
			}
			if l.isBadSlot(51, tc.otherClass, tc.slot, lvl) || l.isBadSlot(51, tc.class, tc.otherSlot, lvl) {
				t.Fatal("another kind or another slot must stay free")
			}
			if l.isBadSlot(51, tc.class, tc.slot, lvl+2) {
				t.Fatal("two levels later the kind is tried again")
			}
		})
	}
}
