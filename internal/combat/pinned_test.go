package combat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

// stuckMover accepts every walk but the character never leaves its tile (pinned in a crowd).
type stuckMover struct {
	Mover
	moves int
}

func (m *stuckMover) GoTo(ctx context.Context, p input.Point) error { m.moves++; return nil }

// Live (Countess, BOSSLIVE2): kite moves that go nowhere stalled the fight for 10 minutes without a cast. After
// pinMoves no-op evasive moves the fighter must cast instead, and keep casting for pinHold.
func TestPinnedKiteFightsInPlace(t *testing.T) {
	h := newHarness(t)
	h.s.life = 60 // under KiteLifePct: a close melee monster triggers a kite step
	sm := &stuckMover{Mover: h.s}
	h.f.mv = sm
	h.s.addMon(10, clsFallen, 104, 100, 400, 3, 3)
	_, _ = h.f.Clear(context.Background(), ClearOpts{Timeout: 6 * time.Second})
	if sm.moves == 0 || sm.moves > pinMoves {
		t.Fatalf("kite moves = %d, want 1..%d before fighting in place", sm.moves, pinMoves)
	}
	if h.s.casts == 0 {
		t.Fatal("the pinned character never cast")
	}
	if h.f.mayEvade() {
		t.Fatal("evading must stay off for pinHold")
	}
	h.s.advance(pinHold + time.Second)
	if !h.f.mayEvade() {
		t.Fatal("evading must come back after pinHold")
	}
}

// Iron Maiden below the retreat line: the fighter walks away from the pack instead of swinging.
func TestCursedBacksOff(t *testing.T) {
	h := newHarness(t, func(e *env.Tuning) { e.Combat.KiteEnabled = 0 })
	h.s.states = append(h.s.states, d2.DefaultCurseIDs.IronMaiden)
	h.s.life = 40
	h.s.stock(itHP1, 0, 2)
	h.s.healAmount = 0
	h.s.addMon(10, clsFallen, 104, 100, 4000, 0, 0)
	_, _ = h.f.Clear(context.Background(), ClearOpts{Timeout: 3 * time.Second})
	if len(h.s.moves) == 0 {
		t.Fatal("a cursed character under the line must step away")
	}
}
