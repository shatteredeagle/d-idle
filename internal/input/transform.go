package input

import "math"

// The isometric projection of the 1.14d client, fitted in docs/input.md.
//
//	sx = PlayerX + 16*((wx-wy) - (px-py))
//	sy = PlayerY +  8*((wx+wy) - (px+py))
//
// (wx,wy) and (px,py) are subtile coordinates; the player is drawn at (PlayerX,PlayerY).
// Parameters are filled in from measurement, see Calibrated.

// Point is a position in subtiles (game world coordinates).
type Point struct{ X, Y int }

// Screen is a client-area pixel.
type Screen struct{ X, Y int }

// Transform converts between world subtiles and client pixels relative to the player.
type Transform struct {
	PlayerX, PlayerY float64 // where the player's feet are drawn
	ScaleX, ScaleY   float64 // pixels per subtile along the iso axes (16, 8)
	ViewW, ViewH     int     // client area (800x600)
	// Click-to-move is accepted inside [MarginX, ViewW-MarginX) x [MarginTop, PanelTop).
	MarginX, MarginTop, PanelTop int
}

// Calibrated holds the measured values (VERIFIED LIVE, see docs/input.md).
var Calibrated = Transform{
	PlayerX: 400, PlayerY: 284, ScaleX: 16, ScaleY: 8,
	ViewW: 800, ViewH: 600, MarginX: 4, MarginTop: 4, PanelTop: 520,
}

// WorldToScreen returns the pixel at which the world point w appears when the player stands at p
// (p may be fractional). The result is not clamped; see InView / Clamp.
func (t Transform) WorldToScreenF(p [2]float64, wx, wy float64) (float64, float64) {
	du, dv := (wx-wy)-(p[0]-p[1]), (wx+wy)-(p[0]+p[1])
	return t.PlayerX + t.ScaleX*du, t.PlayerY + t.ScaleY*dv
}

// WorldToScreenVP uses the game's viewport scroll directly (sx = 16(wx-wy) - VX, sy = 8(wx+wy) - VY),
// which stays exact while the camera lags behind a walking player. VERIFIED LIVE: with a standing player
// VX = 16(px-py)-400 and VY = 8(px+py)-284 to the pixel (px,py incl. the +0.5 rest offset).
func (t Transform) WorldToScreenVP(vx, vy int, wx, wy float64) (float64, float64) {
	return t.ScaleX*(wx-wy) - float64(vx), t.ScaleY*(wx+wy) - float64(vy)
}

func (t Transform) WorldToScreen(player, w Point) Screen {
	x, y := t.WorldToScreenF([2]float64{float64(player.X), float64(player.Y)}, float64(w.X), float64(w.Y))
	return Screen{int(math.Round(x)), int(math.Round(y))}
}

// ScreenToWorld inverts WorldToScreen for a player at p.
func (t Transform) ScreenToWorldF(p [2]float64, sx, sy float64) (float64, float64) {
	u := (sx-t.PlayerX)/t.ScaleX + (p[0] - p[1])
	v := (sy-t.PlayerY)/t.ScaleY + (p[0] + p[1])
	return (u + v) / 2, (v - u) / 2
}

func (t Transform) ScreenToWorld(player Point, s Screen) Point {
	x, y := t.ScreenToWorldF([2]float64{float64(player.X), float64(player.Y)}, float64(s.X), float64(s.Y))
	return Point{int(math.Round(x)), int(math.Round(y))}
}

// InView reports whether a click at s would reach the world (not the panel, not outside the window).
func (t Transform) InView(s Screen) bool {
	return s.X >= t.MarginX && s.X < t.ViewW-t.MarginX && s.Y >= t.MarginTop && s.Y < t.PanelTop
}

// Clamp pulls s into the clickable world rectangle, keeping its direction from the player,
// so a far target becomes a click toward it instead of a click on the panel.
func (t Transform) Clamp(s Screen) Screen {
	if t.InView(s) {
		return s
	}
	dx, dy := float64(s.X)-t.PlayerX, float64(s.Y)-t.PlayerY
	k := 1.0
	lim := func(d, c float64, lo, hi int) {
		if d == 0 {
			return
		}
		var bound float64
		if d > 0 {
			bound = float64(hi) - c
		} else {
			bound = float64(lo) - c
		}
		k = math.Min(k, bound/d)
	}
	lim(dx, t.PlayerX, t.MarginX, t.ViewW-t.MarginX-1)
	lim(dy, t.PlayerY, t.MarginTop, t.PanelTop-1)
	return Screen{int(math.Round(t.PlayerX + dx*k)), int(math.Round(t.PlayerY + dy*k))}
}
