package d2denv

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2dctl"
)

type beatStub struct {
	ms, frame int
	lock      bool
	err       error
}

func (s *beatStub) Clock(context.Context) (d2dctl.ClockInfo, error) {
	return d2dctl.ClockInfo{Lockstep: s.lock, GameMS: s.ms}, s.err
}
func (s *beatStub) Info(context.Context) (d2dctl.Info, error) {
	return d2dctl.Info{Frame: s.frame}, s.err
}

func TestBeatCountsOnlyAdvance(t *testing.T) {
	s := &beatStub{ms: 1000}
	b := &Beat{C: s}
	a, _ := b.Next()
	s.ms = 1500
	a2, _ := b.Next()
	if a2 <= a {
		t.Fatalf("clock advanced but beat did not: %d %d", a, a2)
	}
	a3, _ := b.Next() // wedged: still answers, clock flat
	if a3 != a2 {
		t.Fatalf("flat clock must not beat: %d %d", a2, a3)
	}
}

func TestBeatLockstepAnyCounter(t *testing.T) {
	s := &beatStub{ms: 1000, frame: 5, lock: true}
	b := &Beat{C: s}
	a, _ := b.Next()
	s.frame = 9 // game_ms frozen between steps, frames still run
	if a2, _ := b.Next(); a2 <= a {
		t.Fatal("frame advance ignored under lockstep")
	}
}

func TestBeatError(t *testing.T) {
	b := &Beat{C: &beatStub{err: errors.New("down")}}
	if _, err := b.Next(); err == nil {
		t.Fatal("want error")
	}
}
