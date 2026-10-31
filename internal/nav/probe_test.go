package nav

import (
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

func TestFarTargetsRooms(t *testing.T) {
	lm := &d2map.LevelMap{Rooms: []d2map.Room2{
		{Rect: d2map.Rect{X: 0, Y: 0, W: 40, H: 40}},
		{Rect: d2map.Rect{X: 200, Y: 0, W: 40, H: 40}},
		{Rect: d2map.Rect{X: 400, Y: 0, W: 40, H: 40}},
	}}
	got := FarTargets(lm, input.Point{X: 20, Y: 20}, []input.Point{{X: 20, Y: 20}})
	if len(got) != 2 || got[0].X != 420 || got[1].X != 220 {
		t.Fatalf("targets %v", got)
	}
}

func TestFarTargetsFallsBackToRect(t *testing.T) {
	lm := &d2map.LevelMap{Rect: d2map.Rect{X: 100, Y: 100, W: 1000, H: 1000}}
	got := FarTargets(lm, input.Point{X: 600, Y: 600}, nil)
	if len(got) != 8 {
		t.Fatalf("want 8 probe points, got %v", got)
	}
	got = FarTargets(lm, input.Point{X: 600, Y: 600}, []input.Point{got[0]})
	if len(got) != 7 {
		t.Fatalf("visited point not dropped: %v", got)
	}
}
