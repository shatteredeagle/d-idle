package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"d2lod-bot/internal/input"
)

type sampleRow struct {
	px, py       float64 // player before the click
	vx, vy       int     // viewport before
	dx, dy       int     // screen offset of the click from (400,300)
	tx, ty       int     // Path target read after the click
	fx, fy       float64 // final position
	vx2, vy2     int
	targetSeen   bool
	settleMillis int
}

func (e *env) waitIdle() (input.PlayerState, bool) {
	var last input.PlayerState
	still := 0
	for i := 0; i < 200; i++ {
		p, err := e.r.Player()
		if err != nil {
			return p, false
		}
		if i > 0 && math.Hypot(p.X-last.X, p.Y-last.Y) < 0.01 {
			still++
		} else {
			still = 0
		}
		last = p
		if still >= 8 {
			return p, true
		}
		time.Sleep(40 * time.Millisecond)
	}
	return last, false
}

func (e *env) nearNPC(wx, wy float64, r float64) bool {
	for _, u := range e.npcs() {
		if math.Hypot(float64(u.X)-wx, float64(u.Y)-wy) < r {
			return true
		}
	}
	return false
}

func (e *env) sample(n int) {
	e.g.T = input.Instant()
	home, ok := e.waitIdle()
	if !ok {
		fmt.Println("player not idle / not in game")
		return
	}
	fmt.Printf("home (%.2f,%.2f)\n", home.X, home.Y)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var rows []sampleRow
	for len(rows) < n {
		p, ok := e.waitIdle()
		if !ok {
			break
		}
		vx, vy, _ := e.r.Viewport()
		dx, dy := rng.Intn(481)-240, rng.Intn(261)-130
		if math.Hypot(p.X-home.X, p.Y-home.Y) > 12 {
			// drift home: aim at home's screen position
			sx, sy := input.Calibrated.WorldToScreenF([2]float64{p.X, p.Y}, home.X, home.Y)
			dx, dy = int(sx)-400, int(sy)-300
		}
		dx = max(-380, min(380, dx))
		dy = max(-250, min(200, dy))
		wx, wy := input.Calibrated.ScreenToWorldF([2]float64{p.X, p.Y}, float64(400+dx), float64(300+dy))
		if e.nearNPC(wx, wy, 6) {
			continue
		}
		before := p
		if err := e.g.Click(input.Left, 400+dx, 300+dy); err != nil {
			fmt.Println("click:", err)
			return
		}
		row := sampleRow{px: p.X, py: p.Y, vx: vx, vy: vy, dx: dx, dy: dy}
		t0 := time.Now()
		for time.Since(t0) < 800*time.Millisecond {
			c, _ := e.r.Player()
			if c.TargetX != before.TargetX || c.TargetY != before.TargetY || math.Hypot(c.X-before.X, c.Y-before.Y) > 0.5 {
				row.tx, row.ty, row.targetSeen = c.TargetX, c.TargetY, true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		time.Sleep(60 * time.Millisecond)
		c, _ := e.r.Player()
		if row.targetSeen {
			row.tx, row.ty = c.TargetX, c.TargetY
		}
		f, _ := e.waitIdle()
		row.fx, row.fy = f.X, f.Y
		row.vx2, row.vy2, _ = e.r.Viewport()
		row.settleMillis = int(time.Since(t0) / time.Millisecond)
		rows = append(rows, row)
		fmt.Printf("CSV %.2f %.2f %d %d %d %d\n", row.px, row.py, row.dx, row.dy, row.tx, row.ty)
		fmt.Printf("p=(%.1f,%.1f) vp=(%d,%d) off=(%+4d,%+4d) target=(%d,%d)%v final=(%.1f,%.1f) vp2=(%d,%d)\n",
			row.px, row.py, row.vx, row.vy, dx, dy, row.tx, row.ty, row.targetSeen, row.fx, row.fy, row.vx2, row.vy2)
	}
	// return home
	if p, ok := e.waitIdle(); ok {
		e.g.MoveTo(input.Point{X: int(home.X), Y: int(home.Y)}, input.MoveOpts{Tolerance: 3})
		_ = p
	}
	fit(rows)
}

// fit estimates the screen origin (ox,oy) and scales (kx,ky) of the projection from click -> Path target:
//
//	U = (tx-ty) - (px-py) = (dx-ox)/kx      V = (tx+ty+1) - (px+py) = (dy-oy)/ky
//
// where (dx,dy) is the click offset from (400,300), t the integer target subtile (+0.5 for its centre).
func fit(rows []sampleRow) {
	var use []sampleRow
	for _, r := range rows {
		if r.targetSeen {
			use = append(use, r)
		}
	}
	fmt.Printf("\n%d samples with a target\n", len(use))
	if len(use) < 5 {
		return
	}
	U := func(r sampleRow) float64 { return float64(r.tx-r.ty) - (r.px - r.py) }
	V := func(r sampleRow) float64 { return float64(r.tx+r.ty+1) - (r.px + r.py) }
	for pass := 0; pass < 2; pass++ {
		a, b := line(use, func(r sampleRow) float64 { return float64(r.dx) }, U)
		c, d := line(use, func(r sampleRow) float64 { return float64(r.dy) }, V)
		kx, ky := 1/a, 1/c
		fmt.Printf("pass %d: kx=%.3f ox=%.2f   ky=%.3f oy=%.2f   (model 16, 0, 8, 0; origin relative to (400,300))\n", pass, kx, -b*kx, ky, -d*ky)
		var keep []sampleRow
		var sum, mx float64
		for _, r := range use {
			eu := U(r) - (a*float64(r.dx) + b)
			ev := V(r) - (c*float64(r.dy) + d)
			e := math.Hypot(eu, ev) / 1.4
			sum += e
			mx = math.Max(mx, e)
			if e < 2.5 {
				keep = append(keep, r)
			}
		}
		fmt.Printf("        mean residual %.2f max %.2f subtiles, keep %d/%d\n", sum/float64(len(use)), mx, len(keep), len(use))
		use = keep
	}
	// origin with the scales fixed at 16 and 8
	var ox, oy float64
	for _, r := range use {
		ox += float64(r.dx) - 16*U(r)
		oy += float64(r.dy) - 8*V(r)
	}
	fmt.Printf("fixed scales 16/8: origin offset from (400,300) = (%.2f, %.2f)\n", ox/float64(len(use)), oy/float64(len(use)))
	var mvx, mvy float64
	for _, r := range rows {
		mvx = math.Max(mvx, math.Abs(float64(r.vx)-(16*(r.px-r.py)-400)))
		mvy = math.Max(mvy, math.Abs(float64(r.vy)-(8*(r.px+r.py)-284)))
	}
	fmt.Printf("viewport vs player: max |VX-(16(px-py)-400)| = %.1f, max |VY-(8(px+py)-284)| = %.1f over %d rows\n", mvx, mvy, len(rows))
	// rms of the fixed model (400,300; 16,8) for comparison
	var s2, m2 float64
	for _, r := range use {
		eu := U(r) - float64(r.dx)/16
		ev := V(r) - float64(r.dy)/8
		e := math.Hypot(eu, ev) / 1.4
		s2 += e
		m2 = math.Max(m2, e)
	}
	fmt.Printf("fixed model (16,8 at 400,300): mean %.2f max %.2f subtiles\n", s2/float64(len(use)), m2)
}

// line fits y = a*x + b.
func line(rows []sampleRow, x, y func(sampleRow) float64) (a, b float64) {
	var n, sx, sy, sxx, sxy float64
	for _, r := range rows {
		n++
		sx += x(r)
		sy += y(r)
		sxx += x(r) * x(r)
		sxy += x(r) * y(r)
	}
	a = (n*sxy - sx*sy) / (n*sxx - sx*sx)
	return a, (sy - a*sx) / n
}
