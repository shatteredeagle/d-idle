package d2

import "d2lod-bot/internal/mem"

// Point is a pixel or subtile pair.
type Point struct{ X, Y int }

// Mouse returns the game's own last mouse position in client pixels. VERIFIED LIVE (tracks d2input moves).
func (r *Reader) Mouse() (Point, error) {
	x, err := r.i32(mem.Base + rvaMouseX)
	if err != nil {
		return Point{}, err
	}
	y, err := r.i32(mem.Base + rvaMouseY)
	return Point{int(x), int(y)}, err
}

// ScreenSize returns the game resolution. VERIFIED LIVE: 800x600.
func (r *Reader) ScreenSize() (Point, error) {
	x, err := r.i32(mem.Base + rvaScreenSizeX)
	if err != nil {
		return Point{}, err
	}
	y, err := r.i32(mem.Base + rvaScreenSizeY)
	return Point{int(x), int(y)}, err
}

// Viewport returns the camera offset (ViewportX/Y). With absX=(wx-wy)*16, absY=(wx+wy)*8 the player's
// screen position is abs-viewport: VERIFIED LIVE to give (400, 276) for a standing player.
func (r *Reader) Viewport() (Point, error) {
	x, err := r.i32(mem.Base + rvaViewportX)
	if err != nil {
		return Point{}, err
	}
	y, err := r.i32(mem.Base + rvaViewportY)
	return Point{int(x), int(y)}, err
}

// WorldToScreen converts subtile coordinates to client pixels using a viewport.
func WorldToScreen(wx, wy int, vp Point) Point {
	return Point{(wx-wy)*16 - vp.X, (wx+wy)*8 - vp.Y}
}
