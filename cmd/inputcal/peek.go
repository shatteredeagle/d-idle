package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/input"
)

// shot saves a screenshot of the game window on the worker (read-only X capture) to path.
func shot(path string) error {
	env := append(os.Environ(), "DISPLAY=:1")
	if m, _ := filepath.Glob("/run/user/1000/xauth_*"); len(m) > 0 {
		env = append(env, "XAUTHORITY="+m[0])
	}
	c := exec.Command("xdotool", "search", "--name", "Diablo II")
	c.Env = env
	out, err := c.Output()
	if err != nil || len(strings.Fields(string(out))) == 0 {
		return fmt.Errorf("window search: %v", err)
	}
	id := strings.Fields(string(out))[0]
	c = exec.Command("import", "-window", id, path)
	c.Env = env
	return c.Run()
}

// peek walks near the NPC of the given class (never clicks it), parks the pointer on its predicted
// sprite and saves screenshots with the pointer at several offsets, for hover-label inspection.
func (e *env) peek(class int, dys []int) {
	e.g.T = input.Instant()
	var tgt *unit
	for _, u := range e.npcs() {
		if int(u.Class) == class {
			u := u
			tgt = &u
			break
		}
	}
	if tgt == nil {
		fmt.Println("no such npc")
		return
	}
	if err := e.g.MoveTo(input.Point{X: tgt.X, Y: tgt.Y}, input.MoveOpts{Tolerance: 8, Timeout: 20 * time.Second}); err != nil {
		fmt.Println("moveto:", err)
	}
	time.Sleep(500 * time.Millisecond)
	p, _ := e.r.Player()
	// refresh NPC position (it may wander)
	for _, u := range e.npcs() {
		fmt.Printf("  npc class=%d id=%d (%d,%d)\n", u.Class, u.ID, u.X, u.Y)
		if u.ID == tgt.ID && u.Class == tgt.Class {
			cp := u
			tgt = &cp
		}
	}
	sx, sy := input.Calibrated.WorldToScreenF([2]float64{p.X, p.Y}, float64(tgt.X), float64(tgt.Y))
	fmt.Printf("player (%.1f,%.1f) npc (%d,%d) feet screen (%.0f,%.0f)\n", p.X, p.Y, tgt.X, tgt.Y, sx, sy)
	for i, dy := range dys {
		x, y := int(sx), int(sy)+dy
		e.g.MoveMouse(x, y)
		time.Sleep(400 * time.Millisecond)
		fn := fmt.Sprintf("/tmp/peek%d.png", i)
		fmt.Println(fn, "mouse", x, y, shot(fn))
	}
	e.g.MoveMouse(790, 8)
}
