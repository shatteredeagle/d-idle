package nav

import (
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// TASK-412: upstream d2d takes a warp only for a click in LvlWarp's Select box; the backend's click point is the push.
func TestTileGatePushesTheClickPoint(t *testing.T) {
	_, n, o := mouthNav(t, denAt(5, 10))
	click := d2map.Point{X: 2046, Y: 2048}
	o.lm.Exits = []d2map.Exit{{Dest: 8, Pos: denExit.Pos, Click: click, HasClick: true}}
	cands := n.gateCands(o, 8)
	if len(cands) != 1 || cands[0].at != mp(denExit.Pos) || cands[0].push != (input.Point{X: 2046, Y: 2048}) {
		t.Fatalf("gate %+v: want at the tile, push on the click point", cands)
	}
	o.lm.Exits = []d2map.Exit{denExit}
	if c := n.gateCands(o, 8); len(c) != 1 || c[0].push != mp(denExit.Pos) {
		t.Fatalf("without a click point the tile is pushed, as before: %+v", c)
	}
}
