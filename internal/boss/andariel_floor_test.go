package boss

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/env"
)

// The Andariel script's Normal floor is the tuning default, the Opts override replaces it, and 22 reproduces the old hold.
func TestAndarielFloorKnob(t *testing.T) {
	if got := andariel().MinLevel[0]; got != env.DefaultAndarielMinLevel {
		t.Fatalf("script floor %d != knob default %d", got, env.DefaultAndarielMinLevel)
	}
	run := func(level, knob int) error {
		f := newFake()
		f.level = level
		_, err := Run(context.Background(), andariel(), f, f, Opts{AndarielMinLevel: knob})
		return err
	}
	if err := run(env.DefaultAndarielMinLevel-1, 0); !errors.Is(err, ErrTooWeak) {
		t.Errorf("below default floor: %v", err)
	}
	if err := run(21, 22); !errors.Is(err, ErrTooWeak) {
		t.Errorf("knob 22 at clvl 21: %v", err)
	}
	if err := run(env.DefaultAndarielMinLevel, 0); errors.Is(err, ErrTooWeak) {
		t.Errorf("at the default floor: %v", err)
	}
	if err := run(14, 14); errors.Is(err, ErrTooWeak) {
		t.Errorf("knob 14 at clvl 14: %v", err)
	}
}
