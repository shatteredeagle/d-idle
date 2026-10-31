//go:build rigs

package main

import (
	"testing"

	"d2lod-bot/internal/d2map"
)

func TestExitTarget(t *testing.T) {
	lm := &d2map.LevelMap{
		Exits:    []d2map.Exit{{Dest: 58, Pos: d2map.Point{X: 10, Y: 20}}},
		Adjacent: []d2map.Adjacent{{LevelNo: 46, Contact: d2map.Rect{X: 100, Y: 200, W: 4, H: 160}}},
	}
	if x := exitTarget(lm, 58); x == nil || x.Pos.X != 10 {
		t.Fatalf("exit: %+v", x)
	}
	if x := exitTarget(lm, 46); x == nil || x.Pos.X != 102 || x.Pos.Y != 280 {
		t.Fatalf("adjacent: %+v", x)
	}
	if exitTarget(lm, 99) != nil || exitTarget(nil, 1) != nil {
		t.Fatal("unknown destination must give nil (TeleportToward then returns ErrNoTeleport)")
	}
}
