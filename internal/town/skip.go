package town

import (
	"context"
	"errors"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Greeting skip strategies (tuning town.greetingSkip, docs/npc-skip.md).
const (
	SkipWait   = 0 // old behaviour: wait the speech out in greetingSlice slices (greetingWaitS)
	SkipNPC    = 1 // click the NPC again (hover-verified, plain click on its tile as the hand's fallback)
	SkipEsc    = 2 // Esc, then click the NPC again
	SkipGround = 3 // left click on the NPC's own tile, no hover needed
)

// SkipName is the strategy name as logged.
func SkipName(s int) string {
	switch s {
	case SkipNPC:
		return "npc"
	case SkipEsc:
		return "esc"
	case SkipGround:
		return "ground"
	}
	return "wait"
}

// Timing of the skip sequence (SME advice: the menu opens with the click, so 1 s is plenty; U live, to be measured by
// cmd/npcprobe).
const (
	skipFirstWait   = 1 * time.Second // menu wait after the first click before any skip action
	skipAttemptWait = 2 * time.Second // menu wait after each skip action
	skipMaxActions  = 2               // skip actions per try; then the try fails and openNPC starts over (Esc, walk, click)
)

// Visit is what the last NPC open measured (the probe prints it, the log carries the same fields).
type Visit struct {
	Class        int
	Ms           int64
	Clicks       int
	FirstContact bool // the menu was not open within skipFirstWait of the first click
	Strategy     int
	Opened       bool
}

// LastVisit returns the measurement of the most recent openNPC try.
func (t *Town) LastVisit() Visit { return t.lastVisit }

// awaitMenu waits for the NPC menu after the first click of a try and skips the greeting by the configured strategy.
// start is the time of the first click. It returns the snapshot with the menu open.
func (t *Town) awaitMenu(ctx context.Context, classes []int, npc d2.Unit, start time.Time) (*d2.Snapshot, d2.Unit, error) {
	strat := t.tune().GreetingSkip
	v := Visit{Class: npc.Class, Clicks: 1, Strategy: strat}
	finish := func(ok bool) {
		v.Ms, v.Opened = t.e.Clock.Now().Sub(start).Milliseconds(), ok
		t.lastVisit = v
		if ok {
			t.e.Log.Info("npc: time-to-panel", "ms", v.Ms, "first-contact", v.FirstContact, "clicks", v.Clicks, "strategy", SkipName(strat), "class", npc.Class)
			if v.FirstContact {
				t.e.Log.Info("npc: speech skipped by "+SkipName(strat), "class", npc.Class, "clicks", v.Clicks)
			}
		}
	}
	menu := func(s *d2.Snapshot) bool { return s.UI.Open(d2.UINPCMenu) }
	s, ok, err := t.h.Wait(ctx, skipFirstWait, menu)
	if err != nil {
		return nil, npc, err
	}
	if ok {
		finish(true)
		return s, npc, nil
	}
	v.FirstContact = true
	if strat == SkipWait {
		s, err := t.waitOutGreeting(ctx, classes, &npc)
		v.Clicks++
		finish(err == nil)
		return s, npc, err
	}
	for i := 0; i < skipMaxActions; i++ {
		var aerr error
		if npc, aerr = t.skipAction(ctx, classes, npc, strat, i); aerr != nil {
			if !errors.Is(aerr, input.ErrNotHovered) {
				finish(false)
				return nil, npc, aerr
			}
			// Hover is dead while the speech plays; the wait below still sees a menu that opened anyway.
			t.e.Log.Info("npc: skip click not hovered", "class", npc.Class, "strategy", SkipName(strat), "action", i+1)
		}
		v.Clicks++
		s, ok, err = t.h.Wait(ctx, skipAttemptWait, menu)
		if err != nil {
			finish(false)
			return nil, npc, err
		}
		if ok {
			finish(true)
			return s, npc, nil
		}
	}
	finish(false)
	_ = t.e.In.CloseAll() // the outer try starts over from Esc, walk and click
	return nil, npc, errors.New("town: NPC menu did not open after the greeting skip")
}

// skipAction is one skip attempt. Every strategy clicks only on the NPC (its sprite by hover, or its own tile); a
// click on empty ground would walk the character away.
func (t *Town) skipAction(ctx context.Context, classes []int, npc d2.Unit, strat, i int) (d2.Unit, error) {
	if strat == SkipEsc {
		if err := t.e.In.CloseAll(); err != nil {
			return npc, err
		}
	}
	if strat == SkipGround {
		if s, err := t.current(); err == nil {
			if u, ok := findNPC(s, classes); ok && u.ID == npc.ID {
				npc = u
			}
		}
		// The sprite is drawn above the feet (up-left in the iso view): the second try aims one subtile that way.
		off := [][2]int{{0, 0}, {-1, -1}}[i%2]
		return npc, t.e.In.ClickWorld(input.Point{X: npc.X + off[0], Y: npc.Y + off[1]})
	}
	return t.interactFresh(ctx, classes, npc)
}

// waitOutGreeting is the SkipWait behaviour (the pre-NPCSKIP loop): slices with Esc and a re-click until greetingWaitS ran out.
func (t *Town) waitOutGreeting(ctx context.Context, classes []int, npc *d2.Unit) (*d2.Snapshot, error) {
	var s *d2.Snapshot
	err := errors.New("town: NPC menu did not open")
	until := t.e.Clock.Now().Add(t.greetingWait())
	for err != nil && t.e.Clock.Now().Before(until) {
		if s, err = t.waitGreeting(ctx); err == nil {
			break
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		nn, ierr := t.interactFresh(ctx, classes, *npc)
		if ierr == nil {
			*npc = nn
			continue
		}
		if !errors.Is(ierr, input.ErrNotHovered) {
			break
		}
		t.e.Log.Info("npc: greeting still blocks hover, waiting", "class", npc.Class)
	}
	return s, err
}

// WaitNPCMoves waits until the unit of class changes position (the Act 2 palace guard stepping aside from the Harem
// gate: the one NPC wait that is real, not a speech) or timeout passes. It reports whether he moved.
func (t *Town) WaitNPCMoves(ctx context.Context, class int, timeout time.Duration) (bool, error) {
	s, err := t.current()
	if err != nil {
		return false, err
	}
	u0, ok := findNPC(s, []int{class})
	if !ok {
		return false, ErrNoNPC
	}
	_, moved, err := t.h.Wait(ctx, timeout, func(s *d2.Snapshot) bool {
		u, ok := findNPC(s, []int{class})
		return ok && (u.X != u0.X || u.Y != u0.Y)
	})
	return moved, err
}

// SkipCinematic presses Esc (skips an act-transition cinematic) and waits for the area to differ from from, bounded.
func (t *Town) SkipCinematic(ctx context.Context, from int, timeout time.Duration) (bool, error) {
	_ = t.e.In.CloseAll()
	_, ok, err := t.h.Wait(ctx, timeout, func(s *d2.Snapshot) bool { return s.Player != nil && s.Player.Area != from })
	return ok, err
}
