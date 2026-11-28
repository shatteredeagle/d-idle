package combat

import (
	"context"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

// TASK-1127: a learned Whirlwind the Barbarian cannot pay for is held every tick; the fight end says so in ONE line.
// At knob 0 nothing is noted. Must fail without the hold notes: the drill log of 2026-11-28 had no line at all.
func TestHoldLineWhirlBarbarian(t *testing.T) {
	for _, mode := range []int{0, 2} {
		h, w := moveWorld(t, d2.Barbarian, "", mode, idWhirl)
		buf := logTo(h)
		packAt(w, 107, 100, 5)
		w.snap.Player.MaxMana, w.snap.Player.Mana = 39, 14 // Battle Orders and Shout paid: under the 24 of the leg
		st := newFightState()
		for i := 0; i < 3; i++ {
			if acted, _ := h.f.whirlTick(w.snap, st); acted {
				t.Fatalf("mode %d: spun without the mana", mode)
			}
		}
		w.snap.Player.Mana = 39
		w.snap.Monsters = w.snap.Monsters[:1] // one Fallen left: no pack
		h.f.whirlTick(w.snap, st)
		h.f.telemSummaries(st, w.snap)
		out := buf.String()
		if mode == 0 {
			if strings.Contains(out, "hold") {
				t.Fatalf("knob 0 logged %q", out)
			}
			continue
		}
		if strings.Count(out, "combat hold") != 1 || !strings.Contains(out, "driver=whirl") || !strings.Contains(out, "ticks=4") ||
			!strings.Contains(out, "mana:3") || !strings.Contains(out, "no pack:1") {
			t.Fatalf("hold line: %q", out)
		}
		buf.Reset()
		h.f.telemSummaries(st, w.snap)
		if strings.Contains(buf.String(), "combat hold") {
			t.Fatalf("the notes outlived their fight: %q", buf.String())
		}
	}
}

// The second class: a Necromancer whose mana went into the attack holds the raise (corpse at the feet, skeletons under
// the want), and the fight end names the group and the reason.
func TestHoldLineMinionNecromancer(t *testing.T) {
	h, w := newNecro(t)
	h.f.SetCatalog(withSummonKeys(necroCat()))
	h.f.e.Tune.Combat.DeliveryUpkeep = 2
	buf := logTo(h)
	w.cost = 9
	w.corpse(20, 101, 100)
	w.snap.Player.Mana = 2
	if acted, err := h.f.minionTick(context.Background(), w.snap); acted || err != nil {
		t.Fatalf("raised without mana: %v %v", acted, err)
	}
	h.f.telemSummaries(newFightState(), w.snap)
	if out := buf.String(); strings.Count(out, "combat hold") != 1 || !strings.Contains(out, "driver=minion") || !strings.Contains(out, " mana:") {
		t.Fatalf("hold line: %q", out)
	}
}
