package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/nav"
)

// deathrec records a REAL death as read-only JSONL (one object per 100 ms sample) for docs/combat.md and the
// internal/combat/testdata/death fixtures. It walks to where monsters are, stops defending (no casts, no potions) and
// lets them kill the character, then records the dead state, optionally presses Esc at a chosen time, and records
// the respawn. The tool never writes game memory.
//
//	combatlive deathrec OUTFILE [escAfterSec] [maxFieldSec]
//
// escAfterSec < 0: never press Esc (only watch); 0: press Esc every 400 ms from the first dead sample.

type unitRec struct {
	Type, Class int
	ID          uint32
	Mode        int
	X, Y        int
	Flags       uint32 `json:",omitempty"`
	Owner       uint32 `json:",omitempty"`
}

type sampleRec struct {
	T       float64 // seconds since start
	Phase   string
	Event   string `json:",omitempty"`
	Err     string `json:",omitempty"`
	State   int
	Mode    int
	Life    int
	Max     int
	Mana    int
	Area    int
	X, Y    int
	Gold    int
	Stash   int
	Exp     uint32
	Level   int
	Flags   uint32
	UI      []int `json:",omitempty"` // indexes of non-zero UI flags
	Equip   int
	Inv     int
	Belt    int
	Cursor  bool
	Players []unitRec `json:",omitempty"` // every type 0 unit in the server table
	Near    []unitRec `json:",omitempty"` // living or dead monsters, objects and ground items within 25 subtiles
	Ground  int
	Esc     bool `json:",omitempty"`
}

func deathrec(ctx context.Context, e *env.Env, game *input.Game, nv *nav.Navigator, args []string) {
	out := "death.jsonl"
	if len(args) > 0 {
		out = args[0]
	}
	escAfter := -1.0
	if len(args) > 1 {
		fmt.Sscan(args[1], &escAfter)
	}
	maxField := 180.0
	if len(args) > 2 {
		fmt.Sscan(args[2], &maxField)
	}
	sample, closeRec := newSampler(e, out)
	defer closeRec()
	start := time.Now()
	cat := combat.NewCatalog(e.Data)
	dead := func(s *d2.Snapshot) bool {
		return s != nil && s.Player != nil && (s.Player.Life <= 0 || s.Player.Mode == 0 || s.Player.Mode == 17)
	}

	// field phase: walk to monsters, then stand and take it
	s := sample("field", "start", false)
	if s == nil || s.Player == nil {
		die(fmt.Errorf("not in game"))
	}
	if s.Player.InTown() {
		fmt.Println("leaving town to the Blood Moor")
		die(nv.ExitTo(ctx, 2))
	}
	var deadAt time.Time
	hop := 0
	for time.Since(start).Seconds() < maxField && ctx.Err() == nil {
		s = sample("field", "", false)
		if dead(s) {
			deadAt = time.Now()
			sample("dead", "FIRST DEAD SAMPLE", false)
			break
		}
		if s == nil || s.Player == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		hs := combat.Hostiles(s, combat.HostileCtx{Cat: cat, Diff: s.Player.Difficulty, ClassLvl: s.Player.Level})
		switch {
		case len(hs) > 0 && hs[0].Dist > 6:
			t := hs[0].Unit
			game.MoveTo(input.Point{X: t.X, Y: t.Y}, input.MoveOpts{Tolerance: 4, Timeout: 2 * time.Second})
		case len(hs) == 0 && hop%25 == 0:
			game.MoveTo(input.Point{X: s.Player.X - 9, Y: s.Player.Y - 4*(hop/25%2)}, input.MoveOpts{Tolerance: 2, Timeout: 4 * time.Second})
		}
		hop++
		time.Sleep(100 * time.Millisecond)
	}
	if deadAt.IsZero() {
		fmt.Println("no death within the field budget")
		return
	}
	// dead phase: sample every 100 ms; press Esc as configured
	var respawned time.Time
	nextEsc := deadAt
	for time.Since(deadAt) < 40*time.Second {
		now := time.Now()
		esc := false
		if escAfter >= 0 && now.Sub(deadAt).Seconds() >= escAfter && !now.Before(nextEsc) && respawned.IsZero() {
			game.Key("esc")
			esc = true
			nextEsc = now.Add(400 * time.Millisecond)
		}
		s = sample("dead", "", esc)
		if respawned.IsZero() && s != nil && s.Player != nil && !dead(s) {
			respawned = now
			sample("alive", fmt.Sprintf("RESPAWNED after %.2fs", now.Sub(deadAt).Seconds()), false)
		}
		if !respawned.IsZero() && now.Sub(respawned) > 6*time.Second {
			break
		}
		if escAfter < 0 && now.Sub(deadAt) > 12*time.Second {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("done; samples in", out)
}

func newSampler(e *env.Env, out string) (func(phase, event string, esc bool) *d2.Snapshot, func()) {
	f, err := os.Create(out)
	die(err)
	enc := json.NewEncoder(f)
	start := time.Now()
	var last sampleRec
	sample := func(phase, event string, esc bool) *d2.Snapshot {
		r := sampleRec{T: time.Since(start).Seconds(), Phase: phase, Event: event, Esc: esc}
		s, err := e.Read.Snapshot()
		if err != nil || s == nil {
			r.Err = fmt.Sprint(err)
			enc.Encode(r)
			return nil
		}
		r.State = int(s.State)
		if s.Player != nil {
			p := s.Player
			r.Mode, r.Life, r.Max, r.Mana, r.Area, r.X, r.Y = p.Mode, p.Life, p.MaxLife, p.Mana, p.Area, p.X, p.Y
			r.Gold, r.Stash, r.Exp, r.Level = p.Gold, p.StashGold, p.Exp, p.Level
			if pu, err := e.Read.Units(d2.TypePlayer); err == nil {
				for _, u := range pu {
					r.Players = append(r.Players, unitRec{Type: 0, Class: u.Class, ID: u.ID, Mode: u.Mode, X: u.X, Y: u.Y, Flags: u.Flags})
				}
			}
			near := func(u d2.Unit) bool { return math.Hypot(float64(u.X-p.X), float64(u.Y-p.Y)) <= 25 }
			for _, u := range s.Monsters {
				if near(u) {
					r.Near = append(r.Near, unitRec{Type: 1, Class: u.Class, ID: u.ID, Mode: u.Mode, X: u.X, Y: u.Y, Flags: u.Flags})
				}
			}
			for _, u := range s.Objects {
				if near(u) {
					r.Near = append(r.Near, unitRec{Type: 2, Class: u.Class, ID: u.ID, Mode: u.Mode, X: u.X, Y: u.Y, Flags: u.Flags})
				}
			}
			for _, u := range s.Ground {
				r.Ground++
				if near(u) {
					r.Near = append(r.Near, unitRec{Type: 4, Class: u.Class, ID: u.ID, Mode: u.Mode, X: u.X, Y: u.Y, Flags: u.Flags})
				}
			}
		}
		for i, v := range s.UI {
			if v != 0 {
				r.UI = append(r.UI, i)
			}
		}
		if s.Inventory != nil {
			r.Equip = len(s.Inventory.Container(d2.ContEquipment))
			r.Inv = len(s.Inventory.Container(d2.ContInventory))
			r.Belt = len(s.Inventory.Container(d2.ContBelt))
			r.Cursor = s.Inventory.Cursor != nil
		}
		enc.Encode(r)
		if r.Mode != last.Mode || r.Area != last.Area || r.Life != last.Life && (r.Life == 0 || last.Life == 0) || r.Gold != last.Gold || r.Equip != last.Equip || r.Event != "" {
			fmt.Printf("t=%.1f %s mode %d life %d/%d area %d pos (%d,%d) gold %d equip %d inv %d belt %d ui %v players %d %s\n",
				r.T, phase, r.Mode, r.Life, r.Max, r.Area, r.X, r.Y, r.Gold, r.Equip, r.Inv, r.Belt, r.UI, len(r.Players), event)
		}
		last = r
		return s
	}
	return sample, func() { f.Close() }
}

// corpserec walks back to a corpse and tries to pick it up, recording what changes. Usage:
//
//	combatlive corpse OUTFILE X Y
func corpserec(ctx context.Context, e *env.Env, game *input.Game, nv *nav.Navigator, args []string) {
	var cx, cy int
	fmt.Sscan(args[1], &cx)
	fmt.Sscan(args[2], &cy)
	sample, closeRec := newSampler(e, args[0])
	defer closeRec()
	s := sample("start", "start", false)
	if s != nil && s.Player != nil && s.Player.InTown() {
		fmt.Println("ExitTo(2):", nv.ExitTo(ctx, 2))
	}
	corpse := input.Point{X: cx, Y: cy}
	err := nv.CorpseApproach(ctx, corpse)
	fmt.Println("CorpseApproach:", err)
	if errors.Is(err, nav.ErrDanger) && len(args) > 3 && args[3] == "force" {
		fmt.Println("GoTo (forced):", nv.GoTo(ctx, corpse))
	}
	s = sample("near", "arrived", false)
	pl, _ := e.Read.Units(d2.TypePlayer)
	for _, u := range pl {
		fmt.Printf("player-table unit id %d mode %d at (%d,%d) flags %#x\n", u.ID, u.Mode, u.X, u.Y, u.Flags)
	}
	for _, u := range pl {
		if u.Mode != 17 || s == nil || s.Player == nil || u.ID == s.Player.ID {
			continue
		}
		for try := 0; try < 3; try++ {
			err := game.Interact(input.Target{Type: 0, ID: u.ID, Pos: input.Point{X: u.X, Y: u.Y}})
			fmt.Println("Interact corpse", u.ID, "try", try, "err:", err)
			for i := 0; i < 20; i++ {
				time.Sleep(150 * time.Millisecond)
				sample("pickup", "", false)
			}
			if pl2, _ := e.Read.Units(d2.TypePlayer); len(pl2) < len(pl) {
				break
			}
		}
	}
	for i := 0; i < 20; i++ {
		sample("after", "", false)
		time.Sleep(100 * time.Millisecond)
	}
}

// walkrec walks to a point (GoTo) while recording: combatlive walkto OUTFILE X Y
func walkrec(ctx context.Context, e *env.Env, nv *nav.Navigator, args []string) {
	var x, y int
	fmt.Sscan(args[1], &x)
	fmt.Sscan(args[2], &y)
	sample, closeRec := newSampler(e, args[0])
	defer closeRec()
	sample("start", "start", false)
	fmt.Println("GoTo:", nv.GoTo(ctx, input.Point{X: x, Y: y}))
	for i := 0; i < 10; i++ {
		sample("after", "", false)
		time.Sleep(100 * time.Millisecond)
	}
}

// recoverrec respawns (Esc) if dead, walks to the corpse and then to a gold pile, and goes home, recording all of it:
// combatlive recover OUTFILE CX CY GX GY   (run with --god: monsters stand around the corpse)
func recoverrec(ctx context.Context, e *env.Env, game *input.Game, nv *nav.Navigator, args []string) {
	var cx, cy, gx, gy int
	fmt.Sscan(args[1], &cx)
	fmt.Sscan(args[2], &cy)
	fmt.Sscan(args[3], &gx)
	fmt.Sscan(args[4], &gy)
	sample, closeRec := newSampler(e, args[0])
	defer closeRec()
	s := sample("start", "start", false)
	for i := 0; s != nil && s.Player != nil && (s.Player.Life <= 0 || s.Player.Mode == 0 || s.Player.Mode == 17) && i < 30; i++ {
		game.Key("esc")
		time.Sleep(600 * time.Millisecond)
		s = sample("respawn", "", true)
	}
	if s != nil && s.Player != nil && s.Player.InTown() {
		fmt.Println("ExitTo(2):", nv.ExitTo(ctx, 2))
	}
	fmt.Println("GoTo corpse:", nv.GoTo(ctx, input.Point{X: cx, Y: cy}))
	if units, err := e.Read.Units(d2.TypePlayer); err == nil {
		if s2, _ := e.Snapshot(); s2 != nil && s2.Player != nil {
			if c, ok := combat.FindCorpse(units, s2.Player.ID, s2.Player.Area); ok {
				fmt.Printf("corpse unit %d at %v; GoTo: %v\n", c.ID, c.Pos, nv.GoTo(ctx, c.Pos))
			}
		}
	}
	for i := 0; i < 5; i++ {
		sample("corpse", "", false)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("GoTo gold:", nv.GoTo(ctx, input.Point{X: gx, Y: gy}))
	for i := 0; i < 8; i++ {
		sample("gold", "", false)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("ExitTo(1):", nv.ExitTo(ctx, 1))
	for i := 0; i < 5; i++ {
		sample("home", "", false)
		time.Sleep(100 * time.Millisecond)
	}
}
