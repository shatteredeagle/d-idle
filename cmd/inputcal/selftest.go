package main

import (
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/input"
)

func (e *env) moved(p0 input.PlayerState) (bool, input.PlayerState) {
	time.Sleep(700 * time.Millisecond)
	p, _ := e.waitIdle()
	return math.Hypot(p.X-p0.X, p.Y-p0.Y) > 1, p
}

func (e *env) selftest() {
	g := e.g
	g.T = input.DefaultTiming()
	// 1. MoveTo arrival
	p0, _ := e.waitIdle()
	for _, d := range [][2]int{{6, 6}, {-6, 6}, {6, -6}, {-6, -6}, {0, 8}, {8, 0}} {
		w := input.Point{X: int(p0.X) + d[0], Y: int(p0.Y) + d[1]}
		t0 := time.Now()
		err := g.MoveTo(w, input.MoveOpts{Tolerance: 2})
		p, _ := e.waitIdle()
		fmt.Printf("MoveTo %v: err=%v took %v, now (%.1f,%.1f) dist %.1f\n", w, err, time.Since(t0).Round(time.Millisecond), p.X, p.Y, math.Hypot(p.X-float64(w.X), p.Y-float64(w.Y)))
		if err == nil {
			break
		}
	}
	// 2. panel boundary: click straight below the player; walk back with MoveTo to a fixed home each time
	home := input.Point{X: int(p0.X), Y: int(p0.Y)}
	back := func() { g.MoveTo(home, input.MoveOpts{Tolerance: 3, Timeout: 10 * time.Second}); e.waitIdle() }
	for _, y := range []int{500, 520, 530, 540, 550, 560, 580} {
		back()
		p, _ := e.waitIdle()
		g.Click(input.Left, 400, y)
		mv, q := e.moved(p)
		fmt.Printf("panel probe click (400,%d): moved=%v (%.1f,%.1f)->(%.1f,%.1f)\n", y, mv, p.X, p.Y, q.X, q.Y)
	}
	// 3. shift-click on ground: same click without and with shift, from the same spot
	for _, mode := range []string{"plain", "shift-flag", "shift-held"} {
		back()
		p, _ := e.waitIdle()
		switch mode {
		case "plain":
			g.Click(input.Left, 500, 300)
		case "shift-flag":
			g.Click(input.Left, 500, 300, input.ModShift)
		case "shift-held":
			g.Hold("shift")
			g.Click(input.Left, 500, 300)
			g.Release("shift")
		}
		mv, q := e.moved(p)
		fmt.Printf("%s click (500,300): moved=%v (%.1f,%.1f)->(%.1f,%.1f)\n", mode, mv, p.X, p.Y, q.X, q.Y)
	}
	back()
	// 4. UI: inventory key, then CloseAll
	open, _ := e.r.UIOpen()
	fmt.Println("UIOpen at rest:", open)
	g.Key("i")
	time.Sleep(300 * time.Millisecond)
	open, _ = e.r.UIOpen()
	fmt.Print("after key i: UIOpen=", open, " ")
	e.ui()
	fmt.Println("CloseAll:", g.CloseAll())
	open, _ = e.r.UIOpen()
	fmt.Println("after CloseAll UIOpen =", open)
	// 5. Interact with Kashya (opens her menu; closed with Esc)
	for _, n := range e.npcs() {
		if n.Class != 150 {
			continue
		}
		e.gotoNear(n.X, n.Y, 7)
		n = e.refresh(n)
		err := g.Interact(input.Target{Type: 1, ID: n.ID, Pos: input.Point{X: n.X, Y: n.Y}})
		time.Sleep(500 * time.Millisecond)
		open, _ = e.r.UIOpen()
		fmt.Printf("Interact(Kashya id=%d): err=%v UIOpen=%v\n", n.ID, err, open)
		fmt.Println("CloseAll:", g.CloseAll())
		open, _ = e.r.UIOpen()
		fmt.Println("after CloseAll UIOpen =", open)
	}
	// 6. skills
	l, r, err := e.r.Skills()
	fmt.Println("skills left/right:", l, r, err)
	for _, k := range []string{"f1", "f2"} {
		g.SelectSkill(k)
		time.Sleep(300 * time.Millisecond)
		l, r, err = e.r.Skills()
		fmt.Printf("after %s: left=%d right=%d err=%v\n", k, l, r, err)
	}
	open, _ = e.r.UIOpen()
	fmt.Println("final UIOpen:", open)
}

func (e *env) refresh(n unit) unit {
	for _, u := range e.npcs() {
		if u.ID == n.ID && u.Class == n.Class {
			return u
		}
	}
	return n
}
