package d2dcombat

import "d2lod-bot/internal/d2"

// viewReader fills Snapshot.Viewport, which d2dsrc leaves zero (the field is Game.exe's camera variable). Combat decides
// "on screen" and the shot from it, so without it every target looks off screen and the character only walks. The camera
// of a standing player sits at the fixed pixel of Game.exe (V there: VX = 16(px-py)-400, VY = 8(px+py)-284, input/transform.go);
// that d2d draws its player at the same pixel is U (state shows cam= in cells; not compared yet).
type viewReader struct{ d2.StateReader }

func (v viewReader) Snapshot() (*d2.Snapshot, error) {
	s, err := v.StateReader.Snapshot()
	if err != nil || s == nil || s.Player == nil {
		return s, err
	}
	px, py := float64(s.Player.X)+0.5, float64(s.Player.Y)+0.5
	s.Viewport = d2.Point{X: int(16*(px-py)) - 400, Y: int(8*(px+py)) - 284}
	return s, nil
}
