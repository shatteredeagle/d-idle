//go:build rigs

package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
)

// melee SECS click|hold [RANGE]: what the combat package cannot do today, done by hand so the numbers exist. It leaves
// town, finds the nearest hostile with combat.Hostiles, walks into RANGE subtiles (default 4) and attacks with the LEFT
// skill ("click": one left click per tick; "hold": the left button held 700 ms on the monster). Reports kills, xp,
// life and mana curve, hits taken. No potions beyond a life potion at 40 percent, retreat to town under 30 percent.
func melee(src mem.Source, g *input.Game, db *data.DB, args []string) {
	secs, mode, reach := 120, "click", 4
	if len(args) > 0 {
		secs, _ = strconv.Atoi(args[0])
	}
	if len(args) > 1 {
		mode = args[1]
	}
	if len(args) > 2 {
		reach, _ = strconv.Atoi(args[2])
	}
	tune := env.DefaultTuning()
	e := &env.Env{ID: "classsmoke", Log: slog.New(slog.NewTextHandler(os.Stderr, nil)), Mem: src, Read: d2.New(src), In: g, Data: db,
		Tune: &tune, Clock: env.RealClock{}, Emit: func(goals.Event) {}, World: d2map.NewTracker(src)}
	nv := nav.New(e)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(secs+30)*time.Second)
	defer cancel()
	cat := combat.NewCatalog(db)
	s, err := e.Snapshot()
	die(err)
	if s.Player.InTown() {
		fmt.Println("ExitTo(2):", nv.ExitTo(ctx, 2))
	}
	start := time.Now()
	s, _ = e.Snapshot()
	xp0 := s.Player.Exp
	seen := map[uint32]string{}
	kills, attacks, walks, potions := 0, 0, 0, 0
	minLife, minMana := s.Player.Life, s.Player.Mana
	lastLog := time.Time{}
	var ownKills []string
	for time.Since(start) < time.Duration(secs)*time.Second && ctx.Err() == nil {
		s, err = e.Snapshot()
		if err != nil || s.Player == nil || s.State != d2.StateInGame {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		p := s.Player
		minLife, minMana = min(minLife, p.Life), min(minMana, p.Mana)
		for _, m := range s.Monsters {
			if m.Item == nil && m.Monster != nil {
				if m.Alive() {
					seen[m.ID] = fmt.Sprint(m.Class)
				} else if _, ok := seen[m.ID]; ok && seen[m.ID] != "dead" {
					kills++
					ownKills = append(ownKills, seen[m.ID])
					seen[m.ID] = "dead"
				}
			}
		}
		if time.Since(lastLog) > 10*time.Second {
			lastLog = time.Now()
			fmt.Printf("t=%3.0fs pos (%d,%d) life %d/%d mana %d/%d kills %d attacks %d walks %d mode %d left %d right %d\n", time.Since(start).Seconds(), p.X, p.Y, p.Life, p.MaxLife, p.Mana, p.MaxMana, kills, attacks, walks, p.Mode, p.LeftSkill, p.RightSkill)
		}
		if p.Life*100/p.MaxLife < 30 {
			fmt.Println("life low, retreating:", nv.ExitTo(ctx, 1))
			break
		}
		if p.Life*100/p.MaxLife < 40 && potions < 4 {
			_ = g.Key("1")
			potions++
			time.Sleep(500 * time.Millisecond)
			continue
		}
		hs := combat.Hostiles(s, combat.HostileCtx{Cat: cat, Diff: p.Difficulty, ClassLvl: p.Level})
		if len(hs) == 0 {
			walks++
			dirs := [][2]int{{-7, 0}, {-5, -5}, {0, -7}, {5, -5}, {7, 0}, {5, 5}, {0, 7}, {-5, 5}}
			d := dirs[(walks/2)%len(dirs)]
			_ = g.MoveTo(input.Point{X: p.X + d[0], Y: p.Y + d[1]}, input.MoveOpts{Tolerance: 2, Timeout: 3 * time.Second})
			continue
		}
		h := hs[0]
		if h.Dist > float64(reach) {
			walks++
			dx, dy := float64(h.Unit.X-p.X), float64(h.Unit.Y-p.Y)
			d := math.Hypot(dx, dy)
			want := math.Max(d-float64(reach-1), 1)
			_ = g.MoveTo(input.Point{X: p.X + int(dx/d*want), Y: p.Y + int(dy/d*want)}, input.MoveOpts{Tolerance: 1, Timeout: 3 * time.Second})
			continue
		}
		x, y := input.Calibrated.WorldToScreenVP(s.Viewport.X, s.Viewport.Y, float64(h.Unit.X)+0.5, float64(h.Unit.Y)+0.5)
		sx, sy := int(math.Round(x)), int(math.Round(y))
		attacks++
		if mode == "hold" {
			_ = g.C.Seq(fmt.Sprintf("move %d %d", sx, sy), "sleep 80", fmt.Sprintf("down l %d %d", sx, sy), "sleep 700", fmt.Sprintf("up l %d %d", sx, sy))
		} else {
			_ = g.Click(input.Left, sx, sy)
			time.Sleep(150 * time.Millisecond)
		}
	}
	s, _ = e.Snapshot()
	if s != nil && s.Player != nil {
		fmt.Printf("RESULT mode=%s secs=%.0f kills=%d attacks=%d walks=%d potions=%d xp+%d life min %d mana min %d (max %d/%d) area %d\n", mode, time.Since(start).Seconds(), kills, attacks, walks, potions, int(s.Player.Exp)-int(xp0), minLife, minMana, s.Player.MaxLife, s.Player.MaxMana, s.Player.Area)
		fmt.Println("killed classes:", ownKills)
		if !s.Player.InTown() && s.Player.Life > 0 {
			fmt.Println("home:", nv.ExitTo(ctx, 1))
		}
	}
}
