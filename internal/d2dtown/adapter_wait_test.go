package d2dtown

import (
	"testing"
	"time"

	"d2lod-bot/internal/input"
)

type waitIn struct {
	input.Controller
	waited []time.Duration
}

func (w *waitIn) Wait(d time.Duration)                                 { w.waited = append(w.waited, d) }
func (w *waitIn) MoveMouse(x, y int) error                             { return nil }
func (w *waitIn) Click(b input.Button, x, y int, m ...input.Mod) error { return nil }

func TestHoverFirstWaitsOnTheControllerClock(t *testing.T) {
	w := &waitIn{}
	start := time.Now()
	if err := HoverFirst(w).Click(input.Left, 10, 10); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("slept wall time although the controller has a clock")
	}
	if len(w.waited) != 2 || w.waited[0] != 120*time.Millisecond || w.waited[1] != 350*time.Millisecond {
		t.Fatalf("waited %v", w.waited)
	}
}
