package d2dcombat

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

type stubReader struct {
	d2.StateReader
	s *d2.Snapshot
}

func (r stubReader) Snapshot() (*d2.Snapshot, error) { return r.s, nil }

func TestViewportPutsPlayerAtScreenAnchor(t *testing.T) {
	s := &d2.Snapshot{Player: &d2.Player{}}
	s.Player.X, s.Player.Y = 153, 68
	got, _ := viewReader{stubReader{s: s}}.Snapshot()
	x, y := input.Calibrated.WorldToScreenVP(got.Viewport.X, got.Viewport.Y, 153.5, 68.5)
	if x < 399 || x > 401 || y < 283 || y > 285 {
		t.Fatalf("player at (%v,%v), want about (400,284)", x, y)
	}
}
