package viewport

import (
	"math"

	"d2lod-bot/internal/input"
)

// Scene is what the bot sees and plans, in world subtiles. Producers (bot, nav, combat) fill it; the viewport
// never reads game memory itself.
type Scene struct {
	// Player is where the player stands (subtiles, fractional ok). Used with Transform.WorldToScreenF.
	Player [2]float64 `json:"player"`
	// VP, when set, is the game's viewport scroll (VX, VY) and is used instead of Player (exact while the
	// camera lags a walking player, V live in docs/input.md).
	VP     *[2]int      `json:"vp,omitempty"`
	Shapes []WorldShape `json:"shapes"`
}

// Shape kinds.
const (
	KindPoint   = "point"   // one marker
	KindPath    = "path"    // polyline through Pts
	KindCircle  = "circle"  // ground circle of Radius subtiles around Pts[0]; drawn as an ellipse
	KindPolygon = "polygon" // closed outline through Pts (danger zone)
)

// WorldShape is a shape in world subtiles.
type WorldShape struct {
	Kind   string       `json:"kind"`
	Label  string       `json:"label,omitempty"`
	Color  string       `json:"color,omitempty"` // CSS colour, the client picks a default
	Pts    [][2]float64 `json:"pts"`
	Radius float64      `json:"radius,omitempty"` // KindCircle, subtiles
}

// Shape is a shape in screen pixels of the (unscaled) game client area. Circle becomes kind "ellipse" with
// Pts[0] the centre and RX, RY the semi-axes.
type Shape struct {
	Kind    string       `json:"kind"`
	Label   string       `json:"label,omitempty"`
	Color   string       `json:"color,omitempty"`
	Pts     [][2]float64 `json:"pts"`
	RX      float64      `json:"rx,omitempty"`
	RY      float64      `json:"ry,omitempty"`
	Visible bool         `json:"visible"` // at least one point inside the client area
}

// Overlay is the /overlay body. W and H are the client area the coordinates refer to (800x600); a client
// showing a downscaled frame multiplies by frameWidth/W.
type Overlay struct {
	W      int     `json:"w"`
	H      int     `json:"h"`
	Frame  uint64  `json:"frame"` // Seq of the newest frame when the overlay was built
	Shapes []Shape `json:"shapes"`
}

// Project converts a Scene to screen space with t. A circle of radius r subtiles projects to an ellipse with
// semi-axes ScaleX*r*sqrt2 and ScaleY*r*sqrt2 (derived from sx=16(dx-dy), sy=8(dx+dy); the transform itself
// is V live, the ellipse follows from the algebra).
func Project(t input.Transform, sc Scene) Overlay {
	o := Overlay{W: t.ViewW, H: t.ViewH, Shapes: []Shape{}}
	conv := func(p [2]float64) [2]float64 {
		var x, y float64
		if sc.VP != nil {
			x, y = t.WorldToScreenVP(sc.VP[0], sc.VP[1], p[0], p[1])
		} else {
			x, y = t.WorldToScreenF(sc.Player, p[0], p[1])
		}
		return [2]float64{round1(x), round1(y)}
	}
	for _, w := range sc.Shapes {
		s := Shape{Kind: w.Kind, Label: w.Label, Color: w.Color, Pts: make([][2]float64, 0, len(w.Pts))}
		for _, p := range w.Pts {
			c := conv(p)
			s.Pts = append(s.Pts, c)
			if c[0] >= 0 && c[0] < float64(t.ViewW) && c[1] >= 0 && c[1] < float64(t.ViewH) {
				s.Visible = true
			}
		}
		if w.Kind == KindCircle {
			s.Kind = "ellipse"
			s.RX, s.RY = round1(t.ScaleX*w.Radius*math.Sqrt2), round1(t.ScaleY*w.Radius*math.Sqrt2)
		}
		o.Shapes = append(o.Shapes, s)
	}
	return o
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
