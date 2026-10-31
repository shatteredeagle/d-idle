package input

import "testing"

func TestRoundTrip(t *testing.T) {
	x := Calibrated
	p := Point{5000, 5100}
	for _, w := range []Point{{5000, 5100}, {5010, 5095}, {4990, 5110}, {5003, 5101}} {
		s := x.WorldToScreen(p, w)
		back := x.ScreenToWorld(p, s)
		if abs(back.X-w.X) > 1 || abs(back.Y-w.Y) > 1 {
			t.Errorf("%v -> %v -> %v", w, s, back)
		}
	}
	if s := x.WorldToScreen(p, p); s != (Screen{400, 284}) {
		t.Errorf("player screen %v", s)
	}
}

func TestAxes(t *testing.T) {
	x := Calibrated
	p := Point{100, 100}
	// +1 in x only: right 16, down 8. +1 in y only: left 16, down 8.
	if s := x.WorldToScreen(p, Point{101, 100}); s != (Screen{416, 292}) {
		t.Error(s)
	}
	if s := x.WorldToScreen(p, Point{100, 101}); s != (Screen{384, 292}) {
		t.Error(s)
	}
}

func TestClamp(t *testing.T) {
	x := Calibrated
	for _, s := range []Screen{{400, 590}, {900, 300}, {-50, -50}, {400, 530}, {790, 519}} {
		c := x.Clamp(s)
		if !x.InView(c) {
			t.Errorf("clamp(%v)=%v not in view", s, c)
		}
	}
	if c := x.Clamp(Screen{500, 400}); c != (Screen{500, 400}) {
		t.Errorf("in-view point moved: %v", c)
	}
	// direction preserved: straight down stays at x=400
	if c := x.Clamp(Screen{400, 590}); c.X != 400 {
		t.Errorf("%v", c)
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func TestViewportMatchesPlayerRelative(t *testing.T) {
	x := Calibrated
	px, py := 6051.5, 4963.5
	vx, vy := int(16*(px-py)-400), int(8*(px+py)-284)
	for _, w := range [][2]float64{{6060.5, 4950.5}, {6040.5, 4970.5}} {
		a, b := x.WorldToScreenVP(vx, vy, w[0], w[1])
		c, d := x.WorldToScreenF([2]float64{px, py}, w[0], w[1])
		if a != c || b != d {
			t.Errorf("%v: vp (%v,%v) rel (%v,%v)", w, a, b, c, d)
		}
	}
}
