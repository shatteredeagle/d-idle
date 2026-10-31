package d2dinput

import (
	"context"
	"os"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/input"
)

// TestRealD2D drives a running d2d: Pos, a short MoveTo, a hover, a click on empty ground, Escape twice
// avoided (it toggles the game menu). Skipped unless D2D_SOCK names its control socket. It moves the
// character a few cells: use a throwaway one in town. Not run by its author (U).
func TestRealD2D(t *testing.T) {
	sock := os.Getenv("D2D_SOCK")
	if sock == "" {
		t.Skip("D2D_SOCK not set")
	}
	c := d2dctl.New(sock, d2dctl.Options{Timeout: 5 * time.Second})
	defer c.Close()
	ctl := New(c)
	ctl.Ctx = context.Background()
	p, err := ctl.Pos()
	if err != nil {
		t.Fatal(err)
	}
	goal := input.Point{X: p.X + 10, Y: p.Y}
	if err := ctl.MoveTo(goal, input.MoveOpts{Tolerance: 4, Timeout: 15 * time.Second}); err != nil {
		t.Logf("MoveTo: %v (the goal may be blocked)", err)
	}
	q, _ := ctl.Pos()
	t.Logf("moved %v -> %v", p, q)
	if err := ctl.MoveMouse(400, 300); err != nil {
		t.Fatal(err)
	}
	if err := ctl.Click(input.Left, 400, 500); err != nil {
		t.Fatal(err)
	}
	if err := ctl.CloseAll(); err != nil {
		t.Fatal(err)
	}
	if err := ctl.Key("i"); err != nil { // opens the inventory (S), press again to close
		t.Fatal(err)
	}
	_ = ctl.Key("i")
}
