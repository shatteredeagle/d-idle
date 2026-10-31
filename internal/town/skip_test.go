package town_test

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/town"
)

// open opens Akara with the given strategy and returns the visit and the game time it took.
func openWith(t *testing.T, strat int, setup func(r *rig)) (town.Visit, time.Duration, *rig) {
	t.Helper()
	r := newRig(t)
	r.Env.Tune.Town.GreetingSkip = strat
	if setup != nil {
		setup(r)
	}
	start := r.Env.Clock.Now()
	_, err := r.T.OpenForTest(context.Background(), 148)
	d := r.Env.Clock.Now().Sub(start)
	v := r.T.LastVisit()
	if err != nil {
		t.Fatalf("strategy %d: %v (visit %+v)", strat, err, v)
	}
	return v, d, r
}

// The menu comes with the click: no skip action, one click, not first contact.
func TestSkipMenuAtOnce(t *testing.T) {
	for _, st := range []int{town.SkipWait, town.SkipNPC, town.SkipEsc, town.SkipGround} {
		v, d, _ := openWith(t, st, nil)
		if v.Clicks != 1 || v.FirstContact || !v.Opened || d > time.Second {
			t.Errorf("strategy %d: %+v in %v", st, v, d)
		}
	}
}

// A greeting that a click on the NPC's tile ends (the user's "left click skips the talking"), hover dead meanwhile:
// ground ends it with one extra click, nobody walks away, and it is far faster than the 90 s wait.
func TestSkipByClickAnywhere(t *testing.T) {
	setup := func(r *rig) { r.W.Speech, r.W.NoHoverAfterSpeech, r.W.SpeechSkipByClick = 1, 5, true }
	v, d, r := openWith(t, town.SkipGround, setup)
	if !v.FirstContact || v.Clicks != 2 || d > 4*time.Second || r.W.WalkedAway != 0 {
		t.Fatalf("%+v in %v walked %d", v, d, r.W.WalkedAway)
	}
	// the hover-verified strategies still get there (the hand falls back to a plain click on the tile)
	for _, st := range []int{town.SkipNPC, town.SkipEsc} {
		v, d, r := openWith(t, st, setup)
		if !v.Opened || d > 7*time.Second || r.W.WalkedAway != 0 {
			t.Errorf("strategy %d: %+v in %v walked %d", st, v, d, r.W.WalkedAway)
		}
	}
}

// The first click plays the speech, the second does too, the third finds the menu: two skip actions, bounded time.
func TestSkipTwoClicksNeeded(t *testing.T) {
	for _, st := range []int{town.SkipNPC, town.SkipEsc} {
		v, d, _ := openWith(t, st, func(r *rig) { r.W.Speech = 2 })
		if v.Clicks != 3 || !v.FirstContact || d > 6*time.Second {
			t.Errorf("strategy %d: %+v in %v", st, v, d)
		}
	}
}

// Hover stays dead after the speech and no click skips it: the try ends after its bounded actions (about 5 s) and
// the outer retry opens it later; the visit never waits the 90 s out.
func TestSkipIsBounded(t *testing.T) {
	r := newRig(t)
	r.Env.Tune.Town.GreetingSkip = town.SkipGround
	r.W.Speech, r.W.NoHoverAfterSpeech = 3, 100
	start := r.Env.Clock.Now()
	_, _ = r.T.OpenForTest(context.Background(), 148)
	if d := r.Env.Clock.Now().Sub(start); d > 25*time.Second {
		t.Fatalf("took %v", d)
	}
}

// SkipWait keeps the old behaviour and still opens.
func TestSkipWaitStillWorks(t *testing.T) {
	v, _, _ := openWith(t, town.SkipWait, func(r *rig) { r.W.Speech = 1 })
	if !v.Opened || !v.FirstContact {
		t.Fatalf("%+v", v)
	}
}

// The Harem gate guard: wait for his position change, bounded.
func TestWaitNPCMoves(t *testing.T) {
	r := newRig(t)
	g := r.W.AddNPC(250, r.W.Player.X+3, r.W.Player.Y)
	r.W.GuardClass, r.W.GuardMoveAfter = g.Class, 6
	moved, err := r.T.WaitNPCMoves(context.Background(), g.Class, 20*time.Second)
	if err != nil || !moved {
		t.Fatalf("moved %v err %v", moved, err)
	}
	r.W.GuardMoveAfter = 0 // never moves again
	moved, err = r.T.WaitNPCMoves(context.Background(), g.Class, 3*time.Second)
	if err != nil || moved {
		t.Fatalf("moved %v err %v", moved, err)
	}
}

// A cinematic: Esc, then wait for the area to change; a bounded timeout when it never does.
func TestSkipCinematic(t *testing.T) {
	r := newRig(t)
	ok, err := r.T.SkipCinematic(context.Background(), r.W.Player.Area, 3*time.Second)
	if err != nil || ok {
		t.Fatalf("ok %v err %v", ok, err)
	}
	ok, err = r.T.SkipCinematic(context.Background(), r.W.Player.Area+1, 3*time.Second)
	if err != nil || !ok {
		t.Fatalf("ok %v err %v", ok, err)
	}
}
