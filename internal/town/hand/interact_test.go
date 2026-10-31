package hand

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

type stubIn struct {
	input.Controller
	interactErr error
	clicks      []input.Point
	openAfter   int // the menu opens at this click number (1-based), 0: never
	ui          *d2.UIState
}

func (s *stubIn) Interact(input.Target) error { return s.interactErr }
func (s *stubIn) ClickWorld(p input.Point) error {
	s.clicks = append(s.clicks, p)
	if s.openAfter > 0 && len(s.clicks) == s.openAfter {
		s.ui[d2.UINPCMenu] = 1
	}
	return nil
}

type stubClock struct{ t time.Time }

func (c *stubClock) Now() time.Time        { return c.t }
func (c *stubClock) Sleep(d time.Duration) { c.t = c.t.Add(d) }

func newStubHand(in *stubIn) *Hand {
	tune := env.DefaultTuning()
	in.ui = &d2.UIState{}
	e := &env.Env{In: in, Clock: &stubClock{t: time.Unix(1, 0)}, Tune: &tune}
	h := New(e)
	h.Snap = func() (*d2.Snapshot, error) {
		return &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{}, Inventory: &d2.Inventory{}, UI: *in.ui}, nil
	}
	return h
}

var tgt = input.Target{ID: 7, Pos: input.Point{X: 100, Y: 200}}

func TestInteractNPCHoverOKNeedsNoFallback(t *testing.T) {
	in := &stubIn{}
	if err := newStubHand(in).InteractNPC(context.Background(), tgt); err != nil || len(in.clicks) != 0 {
		t.Fatalf("err %v clicks %v", err, in.clicks)
	}
}

func TestInteractNPCFallsBackToPlainClicks(t *testing.T) {
	in := &stubIn{interactErr: input.ErrNotHovered, openAfter: 2}
	if err := newStubHand(in).InteractNPC(context.Background(), tgt); err != nil {
		t.Fatal(err)
	}
	if len(in.clicks) != 2 || in.clicks[0] != tgt.Pos {
		t.Fatalf("clicks %v", in.clicks)
	}
}

func TestInteractNPCReportsNotHoveredWhenNothingOpens(t *testing.T) {
	in := &stubIn{interactErr: input.ErrNoHover}
	err := newStubHand(in).InteractNPC(context.Background(), tgt)
	if !errors.Is(err, input.ErrNotHovered) || len(in.clicks) != len(worldProbe) {
		t.Fatalf("err %v clicks %v", err, in.clicks)
	}
}

func TestInteractNPCOtherErrorsPass(t *testing.T) {
	in := &stubIn{interactErr: input.ErrOffscreen}
	if err := newStubHand(in).InteractNPC(context.Background(), tgt); !errors.Is(err, input.ErrOffscreen) || len(in.clicks) != 0 {
		t.Fatalf("err %v", err)
	}
}
