//go:build rigs

// act2live: live Act 2 verification helper (run via tools/live.sh act2live act2live <sub>; docs/act2.md live results).
//
//	state                     player, hostiles, pack
//	units [CLASS]             monsters and objects (optionally one class)
//	presets                   presets of the current level (objects only)
//	go A1 A2 ...              ToArea each level in turn, clearing what blocks the way (Fire Bolt via combat)
//	clear [SECS]              combat.Clear around here
//	kill CLASS                walk to the first live monster of CLASS and kill it
//	obj CLASS                 walk to the nearest object of CLASS and click it
//	pick [NAME...]            loot.PickUp, optionally named items
//	tpto X Y                  teleport toward a point in hops
//	goto X Y                  walk
//	god SECS                  logged god-mode cheat for the rest of this run (the next arguments continue)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/town/hand"
)

func main() {
	src, err := mem.OpenGame()
	die(err)
	defer src.Close()
	rd := input.Reader{Src: src}
	game := input.NewGame(input.NewClient(""), rd)
	home, _ := os.UserHomeDir()
	inst := "Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	if id := os.Getenv(mem.InstanceEnv); id != "" && id != "i0" {
		inst = "Games/d2lod-inst/" + id + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	}
	db, err := data.Open(home+"/"+inst, home+"/d2lod-live/act2live/data")
	die(err)
	lvl := slog.LevelInfo
	if len(os.Args) > 1 && os.Args[1] == "debug" { // first argument "debug": slog debug (live.sh forwards no env)
		lvl = slog.LevelDebug
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
	tune := env.DefaultTuning()
	e := &env.Env{ID: "live", Log: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})), Mem: src, Read: d2.New(src), In: game, Data: db,
		Tune: &tune, Clock: env.RealClock{}, Emit: func(ev goals.Event) { fmt.Printf("EVENT %s %s monster=%q\n", ev.Type, ev.Message, ev.Monster) }}
	e.World = d2map.NewTracker(src)
	nv := nav.New(e)
	nv.TeleportKey = "f2"
	f := combat.New(e, nv)
	rows, err := combat.LoadListRows(home+"/"+inst, db)
	die(err)
	f.SetListRows(rows)
	if os.Getenv("NOHOTKEYS") == "" && len(os.Args) > 1 && os.Args[1] != "state" && os.Args[1] != "units" && os.Args[1] != "presets" {
		fmt.Println("EnsureHotkeys:", f.EnsureHotkeys(context.Background()), f.Hotkeys())
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	args := os.Args[1:]
	if len(args) > 1 && args[0] == "god" {
		secs, _ := strconv.Atoi(args[1])
		gate := cheat.Gate{Allow: true, Env: cheat.DefaultEnv(), Actor: "act2live"}
		off := gate.God(ctx, "ACT2LIVE: survive Act 2 monsters while verifying quest flow on throwaway Navi")
		godOff = off
		defer func() { fmt.Println("god stop:", off()) }()
		// restore max life at once on TERM/INT, even if the current subcommand does not watch ctx
		sigCtx := ctx
		go func() { <-sigCtx.Done(); fmt.Println("god stop (signal):", off()) }()
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(secs)*time.Second)
		defer cancel()
		args = args[2:]
	}
	if len(args) == 0 {
		args = []string{"state"}
	}
	// several subcommands in one run, separated by "--"
	for i, a := range args {
		if a == "--" {
			rest := args[i+1:]
			args = args[:i]
			defer func() { runAll(ctx, e, f, nv, game, rest) }()
			break
		}
	}
	run(ctx, e, f, nv, game, args)
}

func runAll(ctx context.Context, e *env.Env, f *combat.Fighter, nv *nav.Navigator, game *input.Game, args []string) {
	for i, a := range args {
		if a == "--" {
			run(ctx, e, f, nv, game, args[:i])
			runAll(ctx, e, f, nv, game, args[i+1:])
			return
		}
	}
	run(ctx, e, f, nv, game, args)
}

func run(ctx context.Context, e *env.Env, f *combat.Fighter, nv *nav.Navigator, game *input.Game, args []string) {
	syncTele(e, nv)
	num := func(i int) int {
		if i < len(args) {
			n, _ := strconv.Atoi(args[i])
			return n
		}
		return 0
	}
	switch args[0] {
	case "state":
		state(e)
	case "units":
		units(e, num(1))
	case "presets":
		_, _ = e.World.Update()
		if lm := e.World.Current(); lm != nil {
			for _, p := range lm.Presets {
				if p.Kind != 5 {
					fmt.Printf("preset kind=%d id=%d at (%d,%d)\n", p.Kind, p.ID, p.Pos.X, p.Pos.Y)
				}
			}
		}
	case "go":
		for _, a := range args[1:] {
			lv, _ := strconv.Atoi(a)
			goArea(ctx, e, f, nv, lv)
		}
		state(e)
	case "clear":
		secs := num(1)
		if secs == 0 {
			secs = 30
		}
		res, err := f.Clear(ctx, combat.ClearOpts{Radius: 60, Timeout: time.Duration(secs) * time.Second})
		fmt.Println("Clear:", res, err)
		state(e)
	case "kill":
		class := num(1)
		s, err := e.Snapshot()
		die(err)
		for _, m := range s.Monsters {
			if m.Class == class && m.Alive() {
				fmt.Printf("target id %d at (%d,%d)\n", m.ID, m.X, m.Y)
				fmt.Println("GoToUnit:", nv.GoToUnitOf(ctx, d2.TypeMonster, m.ID))
				fmt.Println("Kill:", f.Kill(ctx, m.ID))
				break
			}
		}
		state(e)
	case "hunt": // hunt CLASS [obj|mon [HX HY]] (HX HY: exploration hint point): explore this level until a live monster of CLASS shows, then kill it
		class := num(1)
		wantObj := len(args) > 2 && args[2] == "obj"
		found := func() *d2.Unit {
			s, err := e.Snapshot()
			if err != nil || s.Player == nil {
				return nil
			}
			if wantObj {
				for i := range s.Objects {
					if s.Objects[i].Class == class {
						return &s.Objects[i]
					}
				}
				return nil
			}
			for i := range s.Monsters {
				if s.Monsters[i].Class == class && s.Monsters[i].Alive() {
					return &s.Monsters[i]
				}
			}
			return nil
		}
		explored := 0
		for ctx.Err() == nil && found() == nil && explored < 3 {
			syncTele(e, nv)
			cc, cancel := context.WithTimeout(ctx, 20*time.Second)
			var err error
			if hx, hy := num(3), num(4); hx != 0 || hy != 0 {
				err = nv.ExploreToward(cc, input.Point{X: hx, Y: hy}, func() bool { return found() != nil })
			} else {
				err = nv.Explore(cc, func() bool { return found() != nil })
			}
			cancel()
			fmt.Println("Explore:", err)
			if errors.Is(err, nav.ErrExplored) {
				explored++
			}
			if found() == nil {
				res, err := f.Clear(ctx, combat.ClearOpts{Radius: 40, Timeout: 20 * time.Second})
				fmt.Println("  Clear:", res.Kills, err)
			}
		}
		if m := found(); m != nil {
			fmt.Printf("FOUND class %d id %d at (%d,%d) mode %d\n", m.Class, m.ID, m.X, m.Y, m.Mode)
			if !wantObj {
				fmt.Println("Kill:", f.Kill(ctx, m.ID))
			}
		}
		state(e)
	case "use": // use CODE: open the pack and right-click the first item with that code; Esc afterwards
		h := hand.New(e)
		sn, err := h.OpenInventory(ctx)
		die(err)
		lay, err := h.Layout(sn)
		die(err)
		done := false
		for _, it := range sn.Inventory.Container(d2.ContInventory) {
			if baseCode(e, it.Class) == args[1] {
				fmt.Println("RightClick:", h.RightClick(it, lay.Inventory))
				done = true
				break
			}
		}
		fmt.Println("found:", done)
		time.Sleep(1500 * time.Millisecond)
		s2, _ := e.Snapshot()
		fmt.Println("ui after:", s2.UI)
		fmt.Println("CloseAll:", h.CloseAll(ctx))
		state(e)
	case "vis": // levels.txt neighbours (Vis0-7) and waypoint flag of every Act 2 level
		for _, l := range e.Data.Levels {
			if l.ID >= 40 && l.ID <= 75 {
				fmt.Printf("level %d %s act=%d vis=%v\n", l.ID, l.Key, l.Act, l.Vis)
			}
		}
	case "tele": // tele CLASS DX DY N [obj]: teleport in steps of (DX,DY) subtiles until an object (or monster with no "obj") of CLASS is in view
		class, dx, dy, n := num(1), num(2), num(3), num(4)
		for i := 0; i < n && ctx.Err() == nil; i++ {
			_, _ = e.World.Update()
			s, err := e.Snapshot()
			die(err)
			for _, o := range s.Objects {
				if o.Class == class {
					fmt.Printf("SEEN object class %d id %d at (%d,%d) player (%d,%d)\n", o.Class, o.ID, o.X, o.Y, s.Player.X, s.Player.Y)
					return
				}
			}
			g := e.World.Current().Grid
			var tgt *input.Point
			for _, m := range [][2]int{{1, 1}, {1, 0}, {0, 1}, {2, 2}, {1, -1}, {-1, 1}} {
				x, y := s.Player.X+dx*m[0], s.Player.Y+dy*m[1]
				if g.Walkable(x, y) {
					tgt = &input.Point{X: x, Y: y}
					break
				}
			}
			if tgt == nil {
				fmt.Println("blocked at", s.Player.X, s.Player.Y)
				return
			}
			fmt.Println("teleport", *tgt, game.CastAt("f2", *tgt))
			time.Sleep(900 * time.Millisecond)
		}
		state(e)
	case "tp": // tp X,Y X,Y ...: teleport (hotkey f2) to each point in turn, no checks
		for _, a := range args[1:] {
			var x, y int
			fmt.Sscanf(a, "%d,%d", &x, &y)
			fmt.Println("teleport", x, y, game.CastAt("f2", input.Point{X: x, Y: y}))
			time.Sleep(1200 * time.Millisecond)
			s, _ := e.Snapshot()
			fmt.Printf("  now (%d,%d) mana %d\n", s.Player.X, s.Player.Y, s.Player.Mana)
		}
		state(e)
	case "tpto": // tpto X Y: teleport (f2) toward the point in hops over walkable ground; stops within 6
		tpTo(ctx, e, game, input.Point{X: num(1), Y: num(2)})
		state(e)
	case "explore": // explore SECS [HX HY]: nav.Explore(Toward) only, no fighting (teleport hops when a key is set)
		cc, cancel := context.WithTimeout(ctx, time.Duration(num(1))*time.Second)
		if num(2) != 0 || num(3) != 0 {
			fmt.Println("ExploreToward:", nv.ExploreToward(cc, input.Point{X: num(2), Y: num(3)}, nil))
		} else {
			fmt.Println("Explore:", nv.Explore(cc, nil))
		}
		cancel()
		state(e)
	case "summoner": // summoner [ARM...]: Arcane Sanctuary only; teleport blind along each arm (W,E,N,S) until the Summoner shows, then kill him
		// Arm ends from Kolbot's Summoner script (S, from memory, U): the journal preset decides which arm; arrival is the centre (25428,5428).
		arms := map[string]input.Point{"W": {X: 25081, Y: 5446}, "E": {X: 25830, Y: 5447}, "N": {X: 25449, Y: 5081}, "S": {X: 25449, Y: 5812}}
		order := args[1:]
		if len(order) == 0 {
			order = []string{"W", "E", "N", "S"}
		}
		seen := func() *d2.Unit {
			s, err := e.Snapshot()
			if err != nil || s.Player == nil {
				return nil
			}
			for i := range s.Monsters {
				if s.Monsters[i].Class == 250 && s.Monsters[i].Alive() {
					return &s.Monsters[i]
				}
			}
			return nil
		}
		for _, a := range order {
			if seen() != nil || ctx.Err() != nil {
				break
			}
			fmt.Println("arm", a, arms[a])
			s0, _ := e.Snapshot()
			if s0.Player.Area == 74 && math.Hypot(float64(s0.Player.X-25428), float64(s0.Player.Y-5428)) > 60 {
				blindTp(ctx, e, game, input.Point{X: 25428, Y: 5428}, func() bool { return false }) // back to the centre first
			}
			blindTp(ctx, e, game, arms[a], func() bool { return seen() != nil })
		}
		if m := seen(); m != nil {
			fmt.Printf("FOUND Summoner id %d at (%d,%d) mode %d\n", m.ID, m.X, m.Y, m.Mode)
			fmt.Println("Kill:", f.Kill(ctx, m.ID))
		}
		state(e)
	case "blindtp": // blindtp X Y: blind teleport hops toward the point (V/U: see blindTp)
		blindTp(ctx, e, game, input.Point{X: num(1), Y: num(2)}, func() bool { return false })
		state(e)
	case "map": // map [STEP]: coarse walk-grid dump of the current level (known and walkable per STEP x STEP block: # mostly walkable, + some, . known blocked, space unknown), @ = player
		_, _ = e.World.Update()
		lm := e.World.Current()
		if lm == nil {
			break
		}
		st := num(1)
		if st == 0 {
			st = 8
		}
		sp, _ := e.Snapshot()
		b := lm.Grid.Bounds()
		fmt.Printf("level %d bounds %+v player (%d,%d) step %d\n", lm.No, b, sp.Player.X, sp.Player.Y, st)
		for y := b.Y; y < b.Y+b.H; y += st {
			line := make([]byte, 0, b.W/st+1)
			any := false
			for x := b.X; x < b.X+b.W; x += st {
				known, walk := 0, 0
				for dy := 0; dy < st; dy += 2 {
					for dx := 0; dx < st; dx += 2 {
						if lm.Grid.Known(x+dx, y+dy) {
							known++
							if lm.Grid.Walkable(x+dx, y+dy) {
								walk++
							}
						}
					}
				}
				c := byte(' ')
				switch {
				case walk*2 > (st*st)/4:
					c = '#'
				case walk > 0:
					c = '+'
				case known > 0:
					c = '.'
				}
				if sp.Player.X >= x && sp.Player.X < x+st && sp.Player.Y >= y && sp.Player.Y < y+st {
					c = '@'
				}
				if c != ' ' {
					any = true
				}
				line = append(line, c)
			}
			if any {
				fmt.Printf("%5d %s\n", y, line)
			}
		}
	case "hold": // hold SECS: wait (screenshots)
		time.Sleep(time.Duration(num(1)) * time.Second)
	case "exitto": // exitto LEVEL: clear, hop to the known exit, else explore; then ToArea
		fmt.Println("reach:", (&rig{e, f, nv, game, nil}).reach(ctx, num(1)))
		state(e)
	case "findtomb": // findtomb: task/quest FindTomb from the Canyon over the seven tombs (66..72), stays in the tomb with the orifice
		a := a2env{rig: &rig{e, f, nv, game, nil}}
		t, err := quest.FindTomb(ctx, a, quest.AreaCanyon)
		fmt.Println("FindTomb:", t, err)
		state(e)
	case "probefar": // probefar [N]: N steps of the nav far probe in the current level, report the orifice (class 152) after each
		a := a2env{rig: &rig{e, f, nv, game, nil}}
		for i := 0; i < max(num(1), 1); i++ {
			more, err := a.ProbeFar(ctx)
			found := a.HasObject(ctx, 152)
			fmt.Println("probe step", i, more, err, "orifice:", found)
			state(e)
			if err != nil || !more || found {
				break
			}
		}
	case "tombhint": // tombhint: read-only, print ActMisc.dwStaffTombLevel (compare with the tomb holding object 152)
		a := a2env{rig: &rig{e, f, nv, game, nil}}
		fmt.Println(a.StaffTombLevel(ctx))
	case "place": // place CODE CLASS: PlaceItem (click object CLASS with CODE in the pack)
		a := a2env{rig: &rig{e, f, nv, game, nil}}
		fmt.Println("PlaceItem:", a.PlaceItem(ctx, args[1], num(2)))
		time.Sleep(2 * time.Second)
		units(e, 0)
		state(e)
	case "npc": // npc CLASS: Interact (click) the monster/NPC of CLASS from here, then show the ui; Esc closes
		s, err := e.Snapshot()
		die(err)
		for _, m := range s.Monsters {
			if m.Class == num(1) {
				fmt.Println("Interact:", game.Interact(input.Target{Type: d2.TypeMonster, ID: m.ID, Pos: input.Point{X: m.X, Y: m.Y}}))
				break
			}
		}
		time.Sleep(2 * time.Second)
		s2, _ := e.Snapshot()
		fmt.Println("ui:", s2.UI)
	case "npci": // npci CLASS: hand.InteractNPC on the monster/NPC of CLASS, report whether the NPC menu opened, Esc
		h := hand.New(e)
		s, err := e.Snapshot()
		die(err)
		for _, m := range s.Monsters {
			if m.Class == num(1) {
				fmt.Printf("npc id %d at (%d,%d), player (%d,%d)\n", m.ID, m.X, m.Y, s.Player.X, s.Player.Y)
				err := h.InteractNPC(ctx, input.Target{Type: d2.TypeMonster, ID: m.ID, Pos: input.Point{X: m.X, Y: m.Y}})
				time.Sleep(1200 * time.Millisecond)
				s2, _ := e.Snapshot()
				fmt.Println("InteractNPC:", err, "menu open:", s2.UI.Open(d2.UINPCMenu))
				break
			}
		}
		fmt.Println("CloseAll:", h.CloseAll(ctx))
	case "click": // click X Y: left click at client coordinates, then show the ui
		err1 := game.MoveMouse(num(1), num(2))
		time.Sleep(300 * time.Millisecond) // V (ACT2LIVE5): a click right after the move did not register on the waypoint panel rows
		fmt.Println("Click:", err1, game.Click(input.Left, num(1), num(2)))
		time.Sleep(1500 * time.Millisecond)
		s2, _ := e.Snapshot()
		fmt.Println("ui:", s2.UI)
	case "key": // key NAME: press a key (esc, f1 ...)
		fmt.Println("Key:", game.Key(args[1]))
	case "esc":
		fmt.Println("CloseAll:", hand.New(e).CloseAll(ctx))
	case "npcgrid": // npcgrid X Y: click a grid around client X,Y until the ui changes (an NPC dialog opens)
		base, _ := e.Snapshot()
		for dy := -50; dy <= 10; dy += 12 {
			for dx := -24; dx <= 24; dx += 12 {
				x, y := num(1)+dx, num(2)+dy
				_ = game.MoveMouse(x, y)
				time.Sleep(250 * time.Millisecond)
				_ = game.Click(input.Left, x, y)
				time.Sleep(900 * time.Millisecond)
				s2, _ := e.Snapshot()
				if s2.UI != base.UI {
					fmt.Println("ui changed at", x, y, s2.UI)
					return
				}
			}
		}
		fmt.Println("no ui change")
	case "talk": // talk CLASS: grid-click the NPC from its projected screen position (V: the hover check of Interact fails on stairs), then the Talk row
		s, err := e.Snapshot()
		die(err)
		for _, m := range s.Monsters {
			if m.Class != num(1) {
				continue
			}
			dx, dy := m.X-s.Player.X, m.Y-s.Player.Y
			cx, cy := 376+16*(dx-dy), 270+8*(dx+dy)-50 // V: Jerhyn 4 subtiles west of Navi opened his menu at (312,188)
			for _, off := range [][2]int{{0, 0}, {0, -14}, {0, 14}} {
				x, y := cx+off[0], cy+off[1]
				_ = game.MoveMouse(x, y)
				time.Sleep(250 * time.Millisecond)
				_ = game.Click(input.Left, x, y)
				time.Sleep(1200 * time.Millisecond)
				if s2, _ := e.Snapshot(); s2.UI != s.UI {
					fmt.Println("menu opened by click at", x, y)
					fmt.Println("Talk row:", game.MoveMouse(x+24, y-85), game.Click(input.Left, x+24, y-85))
					time.Sleep(1500 * time.Millisecond)
					return
				}
			}
			fmt.Println("no menu; tried around", cx, cy)
			return
		}
		fmt.Println("no such NPC in view")
	case "drop": // drop CODE...: pick each pack item with that code up and drop it on the ground (left half of the screen)
		h := hand.New(e)
		for _, code := range args[1:] {
			sn, err := h.OpenInventory(ctx)
			die(err)
			lay, err := h.Layout(sn)
			die(err)
			for _, it := range sn.Inventory.Container(d2.ContInventory) {
				if baseCode(e, it.Class) == code {
					fmt.Println("Take", code, h.Take(ctx, it, lay.Inventory))
					time.Sleep(300 * time.Millisecond)
					fmt.Println("drop click", game.Click(input.Left, 200, 300))
					time.Sleep(700 * time.Millisecond)
					break
				}
			}
		}
		fmt.Println("CloseAll:", h.CloseAll(ctx))
		state(e)
	case "wp": // wp LEVEL: walk to the waypoint of this area and take it to LEVEL
		fmt.Println("TakeWaypoint:", nv.TakeWaypoint(ctx, num(1)))
		state(e)
	case "wpgo": // wpgo LEVEL ROWY: Lut Gholein only: open the waypoint panel (retrying), click the row icon at x=112, y=ROWY (V: 410 = Canyon), wait for the area
		// V (ACT2LIVE6): nav.TakeWaypoint's 3 s wait after the icon click is shorter than the Canyon load and its text-column
		// fallback then ends in "not reached"; the panel itself also opens only now and then.
		for try := 0; try < 5 && ctx.Err() == nil; try++ {
			if s, err := e.Snapshot(); err == nil && s.Player != nil && s.Player.Area == num(1) {
				break
			}
			_ = nv.GoTo(ctx, input.Point{X: 5100, Y: 5100})
			_ = nv.GoTo(ctx, input.Point{X: 5069, Y: 5090})
			s, err := e.Snapshot()
			die(err)
			var wp *d2.Unit
			for i := range s.Objects {
				if s.Objects[i].Class == 156 {
					wp = &s.Objects[i]
				}
			}
			if wp == nil {
				fmt.Println("waypoint not in view")
				continue
			}
			fmt.Println("Interact:", game.Interact(input.Target{Type: d2.TypeObject, ID: wp.ID, Pos: input.Point{X: wp.X, Y: wp.Y}}))
			time.Sleep(1500 * time.Millisecond)
			if s, err = e.Snapshot(); err != nil || !s.UI.Open(d2.UIWaypoint) {
				fmt.Println("panel not open")
				_ = game.CloseAll()
				continue
			}
			fmt.Println("icon click:", game.Click(input.Left, 112, num(2)))
			for w := 0; w < 20; w++ {
				time.Sleep(500 * time.Millisecond)
				if s, err := e.Snapshot(); err == nil && s.Player != nil && s.Player.Area == num(1) {
					break
				}
			}
		}
		state(e)
	case "takeitem": // takeitem CODE: pack item with that code onto the cursor, panels closed (the cursor keeps it, V: PlaceItem then only needs a click)
		h := hand.New(e)
		sn, err := h.OpenInventory(ctx)
		die(err)
		lay, err := h.Layout(sn)
		die(err)
		for _, it := range sn.Inventory.Container(d2.ContInventory) {
			if baseCode(e, it.Class) == args[1] {
				fmt.Println("Take:", h.Take(ctx, it, lay.Inventory))
				break
			}
		}
		fmt.Println("CloseAll:", game.CloseAll())
		time.Sleep(500 * time.Millisecond)
		state(e)
	case "grab": // grab CLASS_CODE: Interact (hover check, click) with the nearest ground item with that base code; V/U: the item lands on the cursor
		s, err := e.Snapshot()
		die(err)
		for i := range s.Ground {
			it := &s.Ground[i]
			if baseCode(e, it.Class) == args[1] {
				fmt.Println("GoToUnit:", nv.GoToUnitOf(ctx, d2.TypeItem, it.ID))
				fmt.Println("Interact:", game.Interact(input.Target{Type: d2.TypeItem, ID: it.ID, Pos: input.Point{X: it.X, Y: it.Y}}))
				break
			}
		}
		time.Sleep(1500 * time.Millisecond)
		state(e)
	case "objnames": // objnames ID...: objects.txt names
		for _, a := range args[1:] {
			id, _ := strconv.Atoi(a)
			for _, o := range e.Data.Objects {
				if o.ID == id {
					fmt.Printf("object %d %q token=%s act=%d\n", o.ID, o.Name, o.Token, o.Act)
				}
			}
		}
	case "cubeui": // cubeui [CODE]: open the cube from the pack, put one item in at (0,0), hold 8 s for a screenshot, take it back
		code := "gcw"
		if len(args) > 1 {
			code = args[1]
		}
		h := hand.New(e)
		sn, err := h.OpenInventory(ctx)
		die(err)
		lay, err := h.Layout(sn)
		die(err)
		fmt.Printf("cube grid %+v inventory grid %+v\n", lay.Cube, lay.Inventory)
		for _, it := range sn.Inventory.Container(d2.ContInventory) {
			if baseCode(e, it.Class) == "box" {
				fmt.Println("RightClick cube:", h.RightClick(it, lay.Inventory))
				break
			}
		}
		time.Sleep(1200 * time.Millisecond)
		s2, _ := h.Fresh()
		fmt.Printf("ui after cube click: %v cubeOpen=%v\n", s2.UI, s2.UI.Open(d2.UICube))
		for _, it := range s2.Inventory.Container(d2.ContInventory) {
			if baseCode(e, it.Class) == code {
				w, hh := h.Size(it)
				fmt.Println("Take:", h.Take(ctx, it, lay.Inventory))
				fmt.Println("Put in cube:", h.Put(ctx, it.ID, lay.Cube, 0, 0, w, hh, d2.ContCube))
				break
			}
		}
		s3, _ := h.Fresh()
		for _, it := range s3.Inventory.Items {
			if it.Item != nil {
				fmt.Printf("  item %s where=%v at (%d,%d)\n", baseCode(e, it.Class), it.Item.Where, it.X, it.Y)
			}
		}
		fmt.Println("HOLD")
		time.Sleep(14 * time.Second)
		s4, _ := h.Fresh()
		for _, it := range s4.Inventory.Container(d2.ContCube) {
			w, hh := h.Size(it)
			_ = w
			_ = hh
			fmt.Println("Take back:", h.Take(ctx, it, lay.Cube))
			fmt.Println("Put back:", h.Put(ctx, it.ID, lay.Inventory, 6, 3, 1, 1, d2.ContInventory))
		}
		fmt.Println("CloseAll:", h.CloseAll(ctx))
		state(e)
	case "cubemix": // cubemix CODE...: open the cube, move one pack item per code into it, click Transmute, list and take out the results
		h := hand.New(e)
		sn, err := h.OpenInventory(ctx)
		die(err)
		lay, err := h.Layout(sn)
		die(err)
		for _, it := range sn.Inventory.Container(d2.ContInventory) {
			if baseCode(e, it.Class) == "box" {
				fmt.Println("RightClick cube:", h.RightClick(it, lay.Inventory))
				break
			}
		}
		_, ok, _ := h.Wait(ctx, 3*time.Second, func(s *d2.Snapshot) bool { return s.UI.Open(d2.UICube) })
		fmt.Println("cube open:", ok)
		for _, code := range args[1:] {
			s2, _ := h.Fresh()
			for _, it := range s2.Inventory.Container(d2.ContInventory) {
				if baseCode(e, it.Class) == code {
					fmt.Println("Transfer", code, h.Transfer(ctx, it, lay.Inventory, lay.Cube, d2.ContCube))
					break
				}
			}
		}
		s3, _ := h.Fresh()
		for _, it := range s3.Inventory.Container(d2.ContCube) {
			fmt.Printf("in cube: %s at (%d,%d)\n", baseCode(e, it.Class), it.Item.GX, it.Item.GY)
		}
		fmt.Println("Transmute click:", game.Click(input.Left, 240, 328))
		time.Sleep(2500 * time.Millisecond)
		s4, _ := h.Fresh()
		for _, it := range s4.Inventory.Container(d2.ContCube) {
			fmt.Printf("cube after: %s %s at (%d,%d)\n", baseCode(e, it.Class), baseName(e, it.Class), it.Item.GX, it.Item.GY)
		}
		for _, it := range s4.Inventory.Container(d2.ContCube) {
			fmt.Println("Transfer out", baseCode(e, it.Class), h.Transfer(ctx, it, lay.Cube, lay.Inventory, d2.ContInventory))
		}
		fmt.Println("CloseAll:", h.CloseAll(ctx))
		state(e)
	case "invshot": // invshot SECS: open the pack and hold it for a screenshot
		h := hand.New(e)
		_, err := h.OpenInventory(ctx)
		fmt.Println("open:", err, "HOLD")
		time.Sleep(time.Duration(num(1)) * time.Second)
		fmt.Println("CloseAll:", h.CloseAll(ctx))
	case "unequip": // unequip X Y CODE: click the body slot at client X,Y (item goes to the cursor), put it in a free pack cell
		h := hand.New(e)
		sn, err := h.OpenInventory(ctx)
		die(err)
		lay, err := h.Layout(sn)
		die(err)
		fmt.Println("click slot:", game.Click(input.Left, num(1), num(2)))
		time.Sleep(700 * time.Millisecond)
		s2, _ := h.Fresh()
		if s2.Inventory.Cursor == nil {
			fmt.Println("nothing on the cursor")
		} else {
			cu := *s2.Inventory.Cursor
			w, hh := h.Size(cu)
			occ := h.Occupancy(s2.Inventory, d2.ContInventory, lay.Inventory)
			c, r, ok := occ.FindSpot(w, hh)
			fmt.Println("spot:", c, r, ok)
			if ok {
				fmt.Println("Put:", h.Put(ctx, cu.ID, lay.Inventory, c, r, w, hh, d2.ContInventory))
			}
		}
		fmt.Println("CloseAll:", h.CloseAll(ctx))
		state(e)
	case "adj": // adj: exits, adjacent levels and level rectangle of the current level
		_, _ = e.World.Update()
		if lm := e.World.Current(); lm != nil {
			fmt.Printf("level %d rect %+v\n", lm.No, lm.Rect)
			for _, x := range lm.Exits {
				fmt.Printf("exit dest %d at %v\n", x.Dest, x.Pos)
			}
			for _, a := range lm.Adjacent {
				fmt.Printf("adjacent %+v\n", a)
			}
		}
	case "obj":
		class := num(1)
		var s *d2.Snapshot
		var best *d2.Unit
		for try := 0; try < 4 && best == nil && ctx.Err() == nil; try++ {
			var err error
			s, err = e.Snapshot()
			die(err)
			for i := range s.Objects {
				o := &s.Objects[i]
				if o.Class == class && (best == nil || dist(s, o) < dist(s, best)) {
					best = o
				}
			}
			if best == nil { // not streamed yet: hop to its preset (V: Palace Cellar 3 portal 298 is a far preset), clear, look again
				_, _ = e.World.Update()
				if lm := e.World.Current(); lm != nil {
					for _, p := range lm.Presets {
						if p.Kind == d2map.PresetObject && p.ID == class {
							// V: ExploreToward with the preset as hint wandered off (ACT2LIVE4); nav has no exported
							// point-to-point teleport, so this keeps the local hop helper.
							fmt.Println("hop to preset", p.Pos)
							tpTo(ctx, e, game, input.Point{X: p.Pos.X, Y: p.Pos.Y + 6})
							_, _ = f.Clear(ctx, combat.ClearOpts{Radius: 30, Timeout: 15 * time.Second})
							break
						}
					}
				}
			}
		}
		if best == nil {
			die(fmt.Errorf("no object of class %d in view", class))
		}
		fmt.Printf("object id %d at (%d,%d) mode %d\n", best.ID, best.X, best.Y, best.Mode)
		fmt.Println("GoToUnit:", nv.GoToUnitOf(ctx, d2.TypeObject, best.ID))
		fmt.Println("Interact:", game.Interact(input.Target{Type: d2.TypeObject, ID: best.ID, Pos: input.Point{X: best.X, Y: best.Y}}))
		time.Sleep(1500 * time.Millisecond)
		units(e, 0)
	case "clickobj": // clickobj CLASS: plain-click the projected screen position of the nearest object of CLASS (V: needed for the Cellar portal), until the area changes
		s, err := e.Snapshot()
		die(err)
		var best *d2.Unit
		for i := range s.Objects {
			if o := &s.Objects[i]; o.Class == num(1) && (best == nil || dist(s, o) < dist(s, best)) {
				best = o
			}
		}
		if best == nil {
			die(fmt.Errorf("no object of class %d in view", num(1)))
		}
		dx, dy := best.X-s.Player.X, best.Y-s.Player.Y
		cx, cy := 376+16*(dx-dy), 270+8*(dx+dy)
		for _, off := range [][2]int{{0, 0}, {0, -12}, {0, -24}, {0, 12}, {-10, -10}, {10, -10}} {
			x, y := cx+off[0], cy+off[1]
			_ = game.MoveMouse(x, y)
			time.Sleep(250 * time.Millisecond)
			_ = game.Click(input.Left, x, y)
			time.Sleep(2 * time.Second)
			if s2, err := e.Snapshot(); err == nil && s2.Player.Area != s.Player.Area {
				fmt.Println("area changed by click at", x, y, "->", s2.Player.Area)
				break
			}
			if len(args) > 2 && args[2] == "once" { // clickobj CLASS once: one click only (panel objects: later offsets would hit panel rows)
				break
			}
		}
		state(e)
	case "pick":
		l := loot.New(e, nv)
		rep, err := l.PickUp(ctx, loot.PickOpts{Names: args[1:]})
		fmt.Printf("PickUp: %+v err=%v\n", rep, err)
	case "goto":
		fmt.Println("GoTo:", nv.GoTo(ctx, input.Point{X: num(1), Y: num(2)}))
		state(e)
	default:
		die(fmt.Errorf("unknown sub %q", args[0]))
	}
}

func dist(s *d2.Snapshot, u *d2.Unit) int {
	dx, dy := u.X-s.Player.X, u.Y-s.Player.Y
	return dx*dx + dy*dy
}

// goArea walks into level lv; every 25 s without arriving it fights what is near (nav stalls on monsters).
func goArea(ctx context.Context, e *env.Env, f *combat.Fighter, nv *nav.Navigator, lv int) {
	for try := 0; try < 12 && ctx.Err() == nil; try++ {
		s, err := e.Snapshot()
		die(err)
		if s.Player.Area == lv {
			fmt.Println("in area", lv)
			return
		}
		cc, cancel := context.WithTimeout(ctx, 25*time.Second)
		err = nv.ToArea(cc, lv)
		cancel()
		s, _ = e.Snapshot()
		fmt.Printf("ToArea %d try %d: %v (area %d pos %d,%d life %d/%d)\n", lv, try, err, s.Player.Area, s.Player.X, s.Player.Y, s.Player.Life, s.Player.MaxLife)
		if s.Player.Area == lv {
			return
		}
		res, err := f.Clear(ctx, combat.ClearOpts{Radius: 40, Timeout: 25 * time.Second})
		fmt.Println("  Clear:", res.Kills, "kills", err)
	}
}

func state(e *env.Env) {
	s, err := e.Snapshot()
	die(err)
	p := s.Player
	fmt.Printf("player %s lvl %d area %d pos (%d,%d) mode %d life %d/%d mana %d/%d gold %d ui %v\n", p.Name, p.Level, p.Area, p.X, p.Y, p.Mode, p.Life, p.MaxLife, p.Mana, p.MaxMana, p.Gold, s.UI)
	for _, h := range combat.Hostiles(s, combat.HostileCtx{Cat: combat.NewCatalog(e.Data), Diff: p.Difficulty, ClassLvl: p.Level}) {
		fmt.Printf("hostile id %d class %d %q dist %.0f\n", h.Unit.ID, h.Unit.Class, h.Name, h.Dist)
	}
	for _, it := range s.Inventory.Items {
		if it.Item != nil {
			fmt.Printf("item id=%d code=%s %s where=%v cell=(%d,%d) flags=%#x\n", it.ID, baseCode(e, it.Class), baseName(e, it.Class), it.Item.Where, it.Item.GX, it.Item.GY, it.Item.Flags)
		}
	}
	if s.Inventory.Cursor != nil {
		fmt.Println("cursor item class", s.Inventory.Cursor.Class, baseCode(e, s.Inventory.Cursor.Class))
	}
}

func units(e *env.Env, class int) {
	s, err := e.Snapshot()
	die(err)
	var out []string
	for _, m := range s.Monsters {
		if class == 0 || m.Class == class {
			out = append(out, fmt.Sprintf("mon class=%d id=%d at (%d,%d) mode=%d", m.Class, m.ID, m.X, m.Y, m.Mode))
		}
	}
	for _, o := range s.Objects {
		if class == 0 || o.Class == class {
			out = append(out, fmt.Sprintf("obj class=%d id=%d at (%d,%d) mode=%d", o.Class, o.ID, o.X, o.Y, o.Mode))
		}
	}
	sort.Strings(out)
	fmt.Println(strings.Join(out, "\n"))
	for _, g := range s.Ground {
		fmt.Printf("ground item id=%d class=%d %s at (%d,%d)\n", g.ID, g.Class, baseCode(e, g.Class), g.X, g.Y)
	}
}

// godOff restores god mode's max life; die calls it because os.Exit skips the deferred stop.
var godOff func() error

func die(err error) {
	if err != nil {
		fmt.Println("error:", err)
		if godOff != nil {
			fmt.Println("god stop (die):", godOff())
		}
		os.Exit(1)
	}
}

func baseCode(e *env.Env, class int) string {
	if class >= 0 && class < len(e.Data.Items) && e.Data.Items[class] != nil {
		return e.Data.Items[class].Code
	}
	return "?"
}

func baseName(e *env.Env, class int) string {
	if class >= 0 && class < len(e.Data.Items) && e.Data.Items[class] != nil {
		return e.Data.Items[class].Display
	}
	return "?"
}

// tpTo teleports (hotkey f2) toward a point in hops of at most 16 subtiles, landing only on walkable known cells.
func tpTo(ctx context.Context, e *env.Env, game *input.Game, to input.Point) {
	best, bad := math.MaxFloat64, 0
	for i := 0; i < 40 && ctx.Err() == nil; i++ {
		_, _ = e.World.Update()
		s, err := e.Snapshot()
		die(err)
		dx, dy := to.X-s.Player.X, to.Y-s.Player.Y
		d := math.Hypot(float64(dx), float64(dy))
		if d < 6 {
			return
		}
		if d < best-2 {
			best, bad = d, 0
		} else if bad++; bad >= 3 { // no progress: the known grid has no way on, let the caller explore
			return
		}
		g := e.World.Current().Grid
		var tgt *input.Point
	search:
		for _, step := range []float64{16, 12, 8, 5} {
			step = math.Min(d, step)
			for _, off := range [][2]int{{0, 0}, {2, 0}, {0, 2}, {-2, 0}, {0, -2}, {4, 4}, {-4, -4}, {4, -4}, {-4, 4}, {8, 0}, {0, 8}, {-8, 0}, {0, -8}} {
				x := s.Player.X + int(float64(dx)/d*step) + off[0]
				y := s.Player.Y + int(float64(dy)/d*step) + off[1]
				if g.Walkable(x, y) {
					tgt = &input.Point{X: x, Y: y}
					break search
				}
			}
		}
		if tgt == nil {
			fmt.Println("tpto: no walkable hop at", s.Player.X, s.Player.Y)
			return
		}
		fmt.Println("tpto hop", *tgt, game.CastAt("f2", *tgt))
		time.Sleep(900 * time.Millisecond)
		if s2, err := e.Snapshot(); err == nil && s2.Player.X == s.Player.X && s2.Player.Y == s.Player.Y {
			return
		}
	}
}

// syncTele sets the navigator's Teleport key for the current area: none in town (V: no Teleport there) and none in
// Duriel's lair (area 73, levels.txt Teleport=2).
func syncTele(e *env.Env, nv *nav.Navigator) {
	nv.TeleportKey = "f2"
	if s, err := e.Snapshot(); err == nil && s.Player != nil && (s.Player.Area == 73 || s.Player.Area == 40) {
		nv.TeleportKey = ""
	}
}

// blindTp casts Teleport (f2) toward a point in hops of about 16 subtiles without consulting the walk grid (the
// Arcane Sanctuary grid only knows rooms already streamed in). When a cast does not move the character it tries
// other directions; it gives up after 6 hops in a row that did not progress. Stops within 8 subtiles or when stop() is true.
func blindTp(ctx context.Context, e *env.Env, game *input.Game, to input.Point, stop func() bool) {
	best, bad := math.MaxFloat64, 0
	for i := 0; i < 80 && ctx.Err() == nil && !stop(); i++ {
		s, err := e.Snapshot()
		if err != nil || s.Player == nil || s.Player.Life <= 0 {
			return
		}
		dx, dy := float64(to.X-s.Player.X), float64(to.Y-s.Player.Y)
		d := math.Hypot(dx, dy)
		if d < 8 {
			return
		}
		if d < best-3 {
			best, bad = d, 0
		} else if bad++; bad >= 6 {
			fmt.Println("blindtp: no progress at", s.Player.X, s.Player.Y)
			return
		}
		base := math.Atan2(dy, dx)
		moved := false
		for _, da := range []float64{0, 0.35, -0.35, 0.7, -0.7, 1.1, -1.1} {
			a := base + da
			step := math.Min(d, 16)
			tgt := input.Point{X: s.Player.X + int(math.Cos(a)*step), Y: s.Player.Y + int(math.Sin(a)*step)}
			if err := game.CastAt("f2", tgt); err != nil {
				fmt.Println("cast:", err)
			}
			time.Sleep(800 * time.Millisecond)
			if s2, err := e.Snapshot(); err == nil && s2.Player != nil && (s2.Player.X != s.Player.X || s2.Player.Y != s.Player.Y) {
				moved = true
				break
			}
		}
		if !moved {
			time.Sleep(1500 * time.Millisecond) // mana
		}
	}
}
