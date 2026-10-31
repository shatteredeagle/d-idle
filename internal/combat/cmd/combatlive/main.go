// combatlive: live verification tool for internal/combat (run via tools/live.sh combat combatlive <sub>).
// Subcommands: state | probe | tree SECONDS | bind | fight SECONDS | geo k=v,... (prefix any with geo=... to calibrate).
// Everything that moves the character runs under the shared game lock that live.sh takes.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"time"

	"d2lod-bot/internal/cheat"
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

const install = "Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II"

func main() {
	src, err := mem.OpenGame() // honors D2BOT_INSTANCE (tools/live.sh sets it)
	die(err)
	defer src.Close()
	rd := input.Reader{Src: src}
	cl := input.NewClient("")
	game := input.NewGame(cl, rd)
	home, _ := os.UserHomeDir()
	instDir := install
	if id := os.Getenv(mem.InstanceEnv); id != "" && id != "i0" {
		instDir = "Games/d2lod-inst/" + id + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	}
	db, err := data.Open(home+"/"+instDir, home+"/d2lod-live/combat/data")
	die(err)
	tune := env.DefaultTuning()
	e := &env.Env{ID: "live", Log: slog.New(slog.NewTextHandler(os.Stdout, nil)), Mem: src, Read: d2.New(src), In: game, Data: db,
		Tune: &tune, Clock: env.RealClock{},
		Emit: func(ev goals.Event) {
			fmt.Printf("EVENT %s %s monster=%q boss=%v xp=%d potion=%q\n", ev.Type, ev.Message, ev.Monster, ev.Boss, ev.XP, ev.Potion)
		}}
	tr := d2map.NewTracker(src)
	e.World = tr
	nv := nav.New(e)
	f := combat.New(e, nv)
	if len(os.Args) > 2 && os.Args[1] == "--god" { // logged invincibility cheat for the whole run (docs/cheats.md)
		gate := cheat.Gate{Allow: true, Env: cheat.DefaultEnv(), Actor: "combat"}
		stopGod := gate.God(context.Background(), "test: survive monsters while recording a corpse pickup and mid-level fights (COMBAT2, throwaway Combi)")
		defer func() { fmt.Println("god stop:", stopGod()) }()
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
	sub := "state"
	if len(os.Args) > 1 {
		sub = os.Args[1]
	}
	arg := func(i, def int) int {
		if len(os.Args) > i {
			if n, err := strconv.Atoi(os.Args[i]); err == nil {
				return n
			}
		}
		return def
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	switch sub {
	case "deathrec":
		deathrec(ctx, e, game, nav.New(e), os.Args[2:])
	case "corpse":
		corpserec(ctx, e, game, nav.New(e), os.Args[2:])
	case "walkto":
		walkrec(ctx, e, nav.New(e), os.Args[2:])
	case "recover":
		recoverrec(ctx, e, game, nav.New(e), os.Args[2:])
	case "survive":
		rows, err := combat.LoadListRows(home+"/"+instDir, db)
		die(err)
		f.SetListRows(rows)
		fmt.Println("EnsureHotkeys:", f.EnsureHotkeys(ctx), f.Hotkeys())
		survive(ctx, e, f, nv, src, db, os.Args[2:])
	case "town": // walk back to the town through the exit (nav)
		fmt.Println("ExitTo(1):", nav.New(e).ExitTo(ctx, 1))
	case "respawn": // Fighter.Respawn: wait for mode 17, Esc, verify
		t0 := time.Now()
		fmt.Println("Respawn:", f.Respawn(ctx, 20*time.Second), "in", time.Since(t0).Round(10*time.Millisecond))
		state(e, f)
	case "farm":
		rows, err := combat.LoadListRows(home+"/"+instDir, db)
		die(err)
		f.SetListRows(rows)
		fmt.Println("EnsureHotkeys:", f.EnsureHotkeys(ctx), f.Hotkeys())
		farm(ctx, e, f, nv, db, os.Args[2:])
	case "state":
		state(e, f)
	case "probe":
		probe(ctx, e)
	case "popup": // click the right skill button and hold the popup open for a screenshot
		die(game.CloseAll())
		die(game.Click(input.Left, 660, 578))
		time.Sleep(500 * time.Millisecond)
		die(game.MoveMouse(arg(2, 648), arg(3, 487)))
		time.Sleep(500 * time.Millisecond)
		if len(os.Args) > 4 {
			die(game.Key(os.Args[4]))
			time.Sleep(300 * time.Millisecond)
		}
		fmt.Println("OPEN")
		time.Sleep(8 * time.Second)
		die(game.CloseAll())
		die(game.Key("esc"))
	case "vtest": // select another right skill from the popup, press the key, see whether Fire Bolt comes back
		die(game.CloseAll())
		rs := func() int { s, _ := e.Snapshot(); return s.Player.RightSkill }
		fmt.Println("right before:", rs())
		die(game.Click(input.Left, 660, 578))
		time.Sleep(500 * time.Millisecond)
		die(game.Click(input.Left, arg(3, 648), arg(4, 393)))
		time.Sleep(500 * time.Millisecond)
		fmt.Println("right after popup pick:", rs())
		die(game.Key(os.Args[2]))
		time.Sleep(300 * time.Millisecond)
		fmt.Println("right after", os.Args[2], ":", rs())
	case "bindkey": // bindkey SKILL KEY: bind through the popup from scratch and verify
		rows, err := combat.LoadListRows(home+"/"+instDir, db)
		die(err)
		f.SetListRows(rows)
		fmt.Println("BindSkill:", f.BindSkill(ctx, os.Args[2], os.Args[3]), f.Hotkeys())
	case "home": // back to town through the exit (positions come from the map, never literals)
		fmt.Println("ToArea(1):", nv.ToArea(ctx, 1))
		die(game.CloseAll())
	case "drink": // drink one healing potion through Act and check the belt
		cat := combat.NewCatalog(e.Data)
		before, _ := e.Snapshot()
		err := f.Act(ctx, combat.DrinkLife)
		time.Sleep(600 * time.Millisecond)
		after, _ := e.Snapshot()
		fmt.Println("Act(DrinkLife):", err, "healing before", combat.BeltOf(before.Inventory, cat).Count(combat.PotionHealing), "after", combat.BeltOf(after.Inventory, cat).Count(combat.PotionHealing))
	case "bind":
		rows, err := combat.LoadListRows(home+"/"+instDir, db)
		die(err)
		f.SetListRows(rows)
		err = f.EnsureHotkeys(ctx)
		fmt.Println("EnsureHotkeys:", err, f.Hotkeys())
		probe(ctx, e)
	case "fight":
		rows, err := combat.LoadListRows(home+"/"+instDir, db)
		die(err)
		f.SetListRows(rows)
		fmt.Println("EnsureHotkeys:", f.EnsureHotkeys(ctx), f.Hotkeys())
		fight(ctx, e, f, game, nv, time.Duration(arg(2, 120))*time.Second)
	default:
		fmt.Println("unknown subcommand", sub)
	}
}

func state(e *env.Env, f *combat.Fighter) {
	s, err := e.Snapshot()
	die(err)
	if s.Player == nil {
		fmt.Println("not in game", s.State)
		return
	}
	p := s.Player
	fmt.Printf("player %s lvl %d area %d pos (%d,%d) mode %d life %d/%d mana %d/%d exp %d left %d right %d diff %d\n",
		p.Name, p.Level, p.Area, p.X, p.Y, p.Mode, p.Life, p.MaxLife, p.Mana, p.MaxMana, p.Exp, p.LeftSkill, p.RightSkill, p.Difficulty)
	fmt.Println("skills:", p.Skills)
	fmt.Println("belt:", combat.BeltOf(s.Inventory, combat.NewCatalog(e.Data)))
	fmt.Println("safety:", f.Safety(s))
	for _, h := range combat.Hostiles(s, combat.HostileCtx{Cat: combat.NewCatalog(e.Data), Diff: p.Difficulty, ClassLvl: p.Level}) {
		fmt.Printf("hostile id %d class %d %q dist %.0f life%% %d res %v\n", h.Unit.ID, h.Unit.Class, h.Name, h.Dist, h.LifePct, h.Res)
	}
	fmt.Println("monsters in table:", len(s.Monsters), "missiles:", len(s.Missiles), "ui:", s.UI)
}

func probe(ctx context.Context, e *env.Env) {
	for i := 1; i <= 8; i++ {
		k := fmt.Sprintf("f%d", i)
		die(e.In.Key(k))
		time.Sleep(200 * time.Millisecond)
		s, err := e.Snapshot()
		die(err)
		fmt.Printf("%s -> right skill %d left %d\n", k, s.Player.RightSkill, s.Player.LeftSkill)
	}
}

func die(err error) {
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

// fight steps out of the Rogue Encampment into the Blood Moor, fights what comes with naive movement, and walks
// back into town when life is low or time is up.
func fight(ctx context.Context, e *env.Env, f *combat.Fighter, g *input.Game, mv *nav.Navigator, budget time.Duration) {
	e.Tune.Combat.TownPotionMin = 0 // the throwaway character has two potions; do not portal at once
	s, err := e.Snapshot()
	die(err)
	if s.Player.InTown() {
		fmt.Println("leaving town:", mv.ExitTo(ctx, 2))
	}
	cctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	cat := combat.NewCatalog(e.Data)
	var total combat.ClearResult
	var cerr error
	// look for something to fight: a few short hops west and north-west of the gap, never far from the entrance
	for hop := 0; hop < 8 && cctx.Err() == nil; hop++ {
		cur, err := e.Snapshot()
		die(err)
		hs := combat.Hostiles(cur, combat.HostileCtx{Cat: cat, Diff: cur.Player.Difficulty, ClassLvl: cur.Player.Level})
		fmt.Printf("look %d at (%d,%d): %d hostiles, life %d/%d\n", hop, cur.Player.X, cur.Player.Y, len(hs), cur.Player.Life, cur.Player.MaxLife)
		if len(hs) > 0 && hs[0].Dist < 50 {
			res, err := f.Clear(cctx, combat.ClearOpts{Radius: 60, Timeout: 60 * time.Second})
			fmt.Println("Clear:", res, "err:", err)
			total.Kills += res.Kills
			if err != nil {
				cerr = err
				break
			}
			if res.Kills > 2 || cur.Player.Life < 20 {
				break
			}
			continue
		}
		if cur.Player.Life*100/cur.Player.MaxLife < 50 {
			break
		}
		_ = g.MoveTo(input.Point{X: cur.Player.X - 9, Y: cur.Player.Y - 4*(hop%2)}, input.MoveOpts{Tolerance: 2, Timeout: 5 * time.Second})
	}
	fmt.Println("total kills", total.Kills, "err", cerr)
	s, _ = e.Snapshot()
	if s != nil && s.Player != nil {
		fmt.Printf("after: life %d/%d area %d pos (%d,%d)\n", s.Player.Life, s.Player.MaxLife, s.Player.Area, s.Player.X, s.Player.Y)
		if !s.Player.InTown() && s.Player.Life > 0 {
			fmt.Println("retreating to town:", mv.ExitTo(ctx, 1))
		}
	}
}
