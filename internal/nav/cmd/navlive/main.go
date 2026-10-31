// navlive: live nav verification (run on the worker via tools/live.sh nav navlive <cmd>).
//
//	navlive state                 position, level, life, nearby units
//	navlive town                  walk to the waypoint, the stash and several NPCs, report distances
//	navlive wpui [secs]           walk to the waypoint, open its panel, hold it open for secs, close it
//	navlive out [secs] [explore]  leave town into the Blood Moor, explore a little, come back (safety guard on)
//	navlive path X Y              plan only: print the waypoints to X,Y
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
)

func die(err error) {
	fmt.Println("FAIL:", err)
	os.Exit(1)
}

func main() {
	god := false
	if len(os.Args) > 1 && os.Args[1] == "--debug" {
		os.Setenv("NAVDEBUG", "1")
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
	if len(os.Args) > 1 && os.Args[1] == "--god" { // logged invincibility on the throwaway character (docs/cheats.md)
		god = true
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
	cmd := "state"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	proc, err := mem.OpenGame()
	if err != nil {
		die(err)
	}
	defer proc.Close()
	rd := d2.New(proc)
	tune := env.DefaultTuning()
	lvl := slog.LevelInfo
	if os.Getenv("NAVDEBUG") != "" {
		lvl = slog.LevelDebug
	}
	e := &env.Env{
		ID: "nav", Log: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})),
		Mem: proc, Read: rd, World: d2map.NewTracker(proc), Tune: &tune, Clock: env.RealClock{},
	}
	if home, err := os.UserHomeDir(); err == nil { // level data for Route, as the bot has it
		dir := home + "/Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II"
		if inst := os.Getenv("D2BOT_INSTANCE"); inst != "" && inst != "i0" {
			dir = home + "/Games/d2lod-inst/" + inst + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
		}
		if db, err := data.Open(dir, "/tmp/d2lod-navlive-cache"); err == nil {
			e.Data = db
		} else {
			fmt.Println("no level data:", err)
		}
	}
	cl := input.NewClient("")
	e.In = input.NewGame(cl, input.Reader{Src: proc})
	n := nav.New(e)
	ctx, cancel := context.WithTimeout(context.Background(), maxRun())
	defer cancel()
	if god {
		stop := cheat.Gate{Allow: true, Env: cheat.DefaultEnv(), Actor: "nav2"}.God(ctx, "nav2: test navigation on the throwaway character, not combat")
		defer func() { _ = stop() }()
	}

	switch cmd {
	case "state":
		state(e)
	case "town":
		town(ctx, e, n)
	case "wpui":
		secs := 6
		if len(os.Args) > 2 {
			secs, _ = strconv.Atoi(os.Args[2])
		}
		wpui(ctx, e, n, secs)
	case "out":
		secs := 40
		if len(os.Args) > 2 {
			secs, _ = strconv.Atoi(os.Args[2])
		}
		out(ctx, e, n, secs)
	case "goto":
		x, _ := strconv.Atoi(os.Args[2])
		y, _ := strconv.Atoi(os.Args[3])
		fmt.Println("GoTo:", n.GoTo(ctx, input.Point{X: x, Y: y}))
		state(e)
	case "home":
		fmt.Println("ToArea(1):", n.ToArea(ctx, 1))
		state(e)
	case "wpactivate":
		t0 := time.Now()
		fmt.Println("ActivateWaypoint:", n.ActivateWaypoint(ctx), time.Since(t0).Round(100*time.Millisecond))
		state(e)
	case "wp":
		lv, _ := strconv.Atoi(os.Args[2])
		t0 := time.Now()
		fmt.Println("TakeWaypoint:", n.TakeWaypoint(ctx, lv), time.Since(t0).Round(100*time.Millisecond))
		state(e)
	case "respawn": // dead: Esc dismisses the overlay and respawns in town
		for i := 0; i < 5; i++ {
			s := snap(e)
			if s.Player.Mode != 0 && s.Player.Mode != 17 && s.Player.Life > 0 {
				break
			}
			fmt.Println("Esc:", e.In.Key("esc"))
			time.Sleep(2 * time.Second)
		}
		state(e)
	case "circuit": // circuit L1 L2 ...: ExitTo each level in turn, no waypoints; reports every error
		bad := 0
		for i, a := range os.Args[2:] {
			lv, _ := strconv.Atoi(a)
			t0 := time.Now()
			err := n.ExitTo(ctx, lv)
			s := snap(e)
			fmt.Printf("HOP %d -> %d: err=%v now=%d known=%d %v\n", i+1, lv, err, s.Player.Area, e.World.Current().Grid.KnownCount(), time.Since(t0).Round(100*time.Millisecond))
			if err != nil || s.Player.Area != lv {
				bad++
			}
			levelDump(e, "  arrival")
			// what the bot does next: sweep -> Explore at once
			end := time.Now().Add(4 * time.Second)
			xerr := n.Explore(ctx, func() bool { return time.Now().After(end) })
			fmt.Printf("  Explore right after arrival: %v (took %v)\n", xerr, time.Since(end.Add(-4*time.Second)).Round(100*time.Millisecond))
			if errors.Is(xerr, nav.ErrExplored) {
				bad++
			}
		}
		fmt.Println("CIRCUIT bad hops:", bad)
	case "chain", "chainwp": // chain L1 L2 ...: ToArea each level in turn (routes, learns), one line per hop; chainwp also touches each waypoint met
		bad := 0
		for i, a := range os.Args[2:] {
			explored := strings.HasPrefix(a, "x") // "x13": Explore the level until its exit to 13 is known, then take it
			lv, _ := strconv.Atoi(strings.TrimPrefix(a, "x"))
			t0 := time.Now()
			from := snap(e).Player.Area
			if explored {
				xerr := n.Explore(ctx, func() bool {
					if _, err := e.World.Update(); err != nil {
						return false
					}
					for _, x := range e.World.Current().Exits {
						if x.Dest == lv {
							return true
						}
					}
					return false
				})
				fmt.Printf("  Explore until exit to %d is known: %v after %v\n", lv, xerr, time.Since(t0).Round(100*time.Millisecond))
			}
			err := n.ToArea(ctx, lv)
			s := snap(e)
			fmt.Printf("HOP %d %d -> %d: err=%v now=%d %v\n", i+1, from, lv, err, s.Player.Area, time.Since(t0).Round(100*time.Millisecond))
			if err != nil || s.Player.Area != lv {
				bad++
			}
			if cmd == "chainwp" && err == nil && wpLevel(lv) {
				t0 := time.Now()
				fmt.Printf("  ActivateWaypoint in %d: %v %v\n", lv, n.ActivateWaypoint(ctx), time.Since(t0).Round(100*time.Millisecond))
			}
		}
		fmt.Println("CHAIN bad hops:", bad)
	case "reentry": // reentry [reset]: visit Blood Moor, Save and Exit, re-enter, then ExitTo(2)/(1) with the SAME tracker
		fmt.Println(reentry(ctx, e, n, proc, len(os.Args) > 2 && os.Args[2] == "reset"))
	case "tiles": // every unit of type TILE the client table shows
		for _, typ := range []int{d2.TypeTile} {
			us, err := rd.Units(typ)
			fmt.Println("tile units:", len(us), err)
			for _, u := range us {
				fmt.Printf("  tile class=%d id=%d at (%d,%d) mode=%d\n", u.Class, u.ID, u.X, u.Y, u.Mode)
			}
		}
		exits(e)
	case "exits":
		exits(e)
	case "explore":
		secs := 60
		if len(os.Args) > 2 {
			secs, _ = strconv.Atoi(os.Args[2])
		}
		explore(ctx, e, n, secs)
	case "exit":
		lv, _ := strconv.Atoi(os.Args[2])
		t0 := time.Now()
		fmt.Println("ExitTo:", n.ExitTo(ctx, lv), time.Since(t0).Round(100*time.Millisecond))
		state(e)
	case "toarea":
		lv, _ := strconv.Atoi(os.Args[2])
		t0 := time.Now()
		fmt.Println("ToArea:", n.ToArea(ctx, lv), time.Since(t0).Round(100*time.Millisecond))
		state(e)
	case "route": // route FROM TO: print the planned route, no walking
		a, _ := strconv.Atoi(os.Args[2])
		b, _ := strconv.Atoi(os.Args[3])
		r, err := n.Route(a, b)
		fmt.Println("Route", a, b, r, err)
		if e.Data != nil {
			for _, l := range e.Data.Levels {
				if l.Act == 0 {
					fmt.Printf("  level %d %s vis=%v\n", l.ID, l.Key, l.Vis)
				}
			}
		}
	case "path":
		x, _ := strconv.Atoi(os.Args[2])
		y, _ := strconv.Atoi(os.Args[3])
		state(e)
		start := time.Now()
		p, err := n.PathTo(input.Point{X: x, Y: y})
		fmt.Println("PathTo", time.Since(start), p, err)
	default:
		die(fmt.Errorf("unknown command %q", cmd))
	}
}

func snap(e *env.Env) *d2.Snapshot {
	s, err := e.Snapshot()
	if err != nil || s.State != d2.StateInGame {
		die(fmt.Errorf("not in game: %v", err))
	}
	return s
}

func state(e *env.Env) {
	s := snap(e)
	p := s.Player
	st, err := e.World.Update()
	fmt.Printf("pos=(%d,%d) area=%d mode=%d life=%d/%d known=%d err=%v\n", p.X, p.Y, p.Area, p.Mode, p.Life, p.MaxLife, e.World.Current().Grid.KnownCount(), err)
	_ = st
	for _, u := range s.Monsters {
		fmt.Printf("  mon class=%d id=%d at (%d,%d) mode=%d\n", u.Class, u.ID, u.X, u.Y, u.Mode)
	}
	for _, u := range s.Objects {
		fmt.Printf("  obj class=%d id=%d at (%d,%d) mode=%d\n", u.Class, u.ID, u.X, u.Y, u.Mode)
	}
}

func findByClass(s *d2.Snapshot, class int) *d2.Unit {
	for _, list := range [][]d2.Unit{s.Monsters, s.Objects} {
		for i := range list {
			if list[i].Class == class {
				return &list[i]
			}
		}
	}
	return nil
}

// presetPoint is where the loaded level data places an NPC (kind 1) or object (kind 2) of a class. Positions
// differ in every game seed, so nothing absolute may live in this tool.
func presetPoint(e *env.Env, kind, class int) (input.Point, bool) {
	if _, err := e.World.Update(); err != nil || e.World.Current() == nil {
		return input.Point{}, false
	}
	for _, p := range e.World.Current().Presets {
		if p.Kind == kind && p.ID == class {
			return input.Point{X: p.Pos.X, Y: p.Pos.Y}, true
		}
	}
	return input.Point{}, false
}

func town(ctx context.Context, e *env.Env, n *nav.Navigator) {
	targets := []struct {
		name  string
		class int
		kind  int
	}{{"waypoint", 119, 2}, {"stash", 267, 2}, {"Akara", 148, 1}, {"Gheed", 147, 1}, {"Charsi", 154, 1}, {"Kashya", 150, 1}, {"waypoint", 119, 2}}
	for _, t := range targets {
		s := snap(e)
		u := findByClass(s, t.class)
		if u == nil { // units load only near the player: walk to the preset position from the map first
			if pos, ok := presetPoint(e, t.kind, t.class); ok {
				err := n.GoTo(ctx, pos)
				fmt.Printf("%-9s not loaded; GoTo %v first: %v\n", t.name, pos, err)
				s = snap(e)
				u = findByClass(s, t.class)
			}
		}
		if u == nil {
			fmt.Printf("%-9s not in the unit tables (class %d)\n", t.name, t.class)
			continue
		}
		from := input.Point{X: s.Player.X, Y: s.Player.Y}
		start := time.Now()
		err := n.GoToUnitOf(ctx, u.Type, u.ID)
		s2 := snap(e)
		d := dist(input.Point{X: s2.Player.X, Y: s2.Player.Y}, input.Point{X: u.X, Y: u.Y})
		fmt.Printf("%-9s from %v to unit at (%d,%d): now (%d,%d) dist=%.1f in %v err=%v\n", t.name, from, u.X, u.Y, s2.Player.X, s2.Player.Y, d, time.Since(start).Round(100*time.Millisecond), err)
	}
}

func dist(a, b input.Point) float64 { return math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y)) }

func wpui(ctx context.Context, e *env.Env, n *nav.Navigator, secs int) {
	s := snap(e)
	u := findByClass(s, 119)
	if u == nil {
		die(fmt.Errorf("no waypoint unit"))
	}
	if err := n.GoToUnitOf(ctx, u.Type, u.ID); err != nil {
		die(err)
	}
	if err := e.In.Interact(input.Target{Type: 2, ID: u.ID, Pos: input.Point{X: u.X, Y: u.Y}}); err != nil {
		die(err)
	}
	for i := 0; i < 20; i++ {
		time.Sleep(150 * time.Millisecond)
		s = snap(e)
		if s.UI.Open(d2.UIWaypoint) {
			fmt.Println("waypoint panel open after", (i+1)*150, "ms; UI flags:", s.UI)
			break
		}
	}
	fmt.Println("holding for screenshot", secs, "s")
	time.Sleep(time.Duration(secs) * time.Second)
	_ = e.In.CloseAll()
	s = snap(e)
	fmt.Println("after CloseAll, any panel open:", s.UI.AnyPanelOpen())
}

// out walks into the Blood Moor and back with a safety guard: life under 60% or any living monster within 35
// subtiles outside town aborts, and the retreat walks back to town.
func out(ctx context.Context, e *env.Env, n *nav.Navigator, secs int) {
	s := snap(e)
	if s.Player.Area != 1 {
		die(fmt.Errorf("start in town, not area %d", s.Player.Area))
	}
	_, _ = e.World.Update()
	var townRect d2map.Rect
	if lm := e.World.Level(1); lm != nil {
		townRect = lm.Rect
	}
	n.Guard = func(sn *d2.Snapshot) error {
		p := sn.Player
		if p.MaxLife > 0 && p.Life*100 < 60*p.MaxLife {
			return fmt.Errorf("life %d/%d", p.Life, p.MaxLife)
		}
		if p.Area != 1 {
			for _, m := range sn.Monsters {
				inTown := townRect.Contains(m.X, m.Y) // town NPCs stay in the unit table across the border
				if m.Alive() && m.Mode != 0 && !inTown {
					dx, dy := float64(m.X-p.X), float64(m.Y-p.Y)
					if dx*dx+dy*dy < 35*35 {
						return fmt.Errorf("monster class %d at (%d,%d), me (%d,%d)", m.Class, m.X, m.Y, p.X, p.Y)
					}
				}
			}
		}
		return nil
	}
	retreat := func(why error) {
		fmt.Println("RETREAT:", why)
		n.Guard = nil
		rctx, c := context.WithTimeout(context.Background(), 60*time.Second)
		defer c()
		fmt.Println("retreat ToArea(1):", n.ToArea(rctx, 1))
	}
	t0 := time.Now()
	err := n.ExitTo(ctx, 2)
	fmt.Println("ExitTo(2):", err, time.Since(t0).Round(100*time.Millisecond))
	if err != nil {
		if s := snap(e); s.Player.Area != 1 {
			retreat(err)
		}
		return
	}
	s = snap(e)
	fmt.Printf("in area %d at (%d,%d), grid known %d cells\n", s.Player.Area, s.Player.X, s.Player.Y, e.World.Current().Grid.KnownCount())
	end := time.Now().Add(time.Duration(secs) * time.Second)
	before := e.World.Current().Grid.KnownCount()
	err = n.Explore(ctx, func() bool { return time.Now().After(end) })
	fmt.Println("Explore:", err)
	if _, uerr := e.World.Update(); uerr == nil {
		fmt.Printf("grid known %d -> %d cells\n", before, e.World.Current().Grid.KnownCount())
	}
	if err != nil {
		retreat(err)
		return
	}
	t0 = time.Now()
	err = n.ExitTo(ctx, 1)
	fmt.Println("ExitTo(1):", err, time.Since(t0).Round(100*time.Millisecond))
	if err != nil {
		retreat(err)
	}
	s = snap(e)
	fmt.Printf("end: area %d at (%d,%d) life %d/%d\n", s.Player.Area, s.Player.X, s.Player.Y, s.Player.Life, s.Player.MaxLife)
}

func maxRun() time.Duration {
	if v := os.Getenv("NAVLIVE_MAX"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 14 * time.Minute
}

// exits prints what the map knows about ways out of the current level.
func exits(e *env.Env) {
	s := snap(e)
	if _, err := e.World.Update(); err != nil {
		fmt.Println("update:", err)
	}
	lm := e.World.Current()
	fmt.Printf("level %d pos=(%d,%d) rect=%+v known=%d\n", s.Player.Area, s.Player.X, s.Player.Y, lm.Rect, lm.Grid.KnownCount())
	for _, x := range lm.Exits {
		fmt.Printf("  tile exit -> level %d at %v\n", x.Dest, x.Pos)
	}
	for _, a := range lm.Adjacent {
		fmt.Printf("  adjacent level %d rect=%+v contact=%+v border cells=%d\n", a.LevelNo, a.Rect, a.Contact, len(lm.BorderCells(a.LevelNo)))
	}
	for _, p := range lm.Presets {
		if p.Kind == d2map.PresetTile {
			fmt.Printf("  tile preset id=%d at %v\n", p.ID, p.Pos)
		}
	}
	for _, p := range lm.Waypoints() {
		fmt.Printf("  waypoint preset %d at %v\n", p.ID, p.Pos)
	}
	{
		for _, no := range e.World.LevelNos() {
			o := e.World.Level(no)
			if o == nil || o == lm {
				continue
			}
			fmt.Printf("other level %d rect=%+v known=%d exits=%v\n", no, o.Rect, o.Grid.KnownCount(), o.Exits)
			for _, a := range o.Adjacent {
				fmt.Printf("    adjacent %d rect=%+v contact=%+v border cells=%d\n", a.LevelNo, a.Rect, a.Contact, len(o.BorderCells(a.LevelNo)))
			}
		}
	}
}

func explore(ctx context.Context, e *env.Env, n *nav.Navigator, secs int) {
	end := time.Now().Add(time.Duration(secs) * time.Second)
	err := n.Explore(ctx, func() bool { return time.Now().After(end) })
	fmt.Println("Explore:", err)
	exits(e)
}

type typer struct {
	*input.OOG
	c *input.Client
}

func (t typer) TypeChar(r rune) error { return t.c.TypeChar(r) }

func levelDump(e *env.Env, tag string) {
	if _, err := e.World.Update(); err != nil {
		fmt.Printf("%s: update err %v\n", tag, err)
		return
	}
	lm := e.World.Current()
	s := snap(e)
	fmt.Printf("%s: area=%d lm.No=%d rect=%v known=%d exits=%v adj=%v rooms=%d presets=%d pos=(%d,%d)\n", tag, s.Player.Area, lm.No, lm.Rect,
		lm.Grid.KnownCount(), lm.Exits, lm.Adjacent, len(lm.Rooms), len(lm.Presets), s.Player.X, s.Player.Y)
}

// reentry reproduces the "no exit" after a Save and Exit / re-entry with a tracker that lives across games.
func reentry(ctx context.Context, e *env.Env, n *nav.Navigator, proc mem.Source, reset bool) error {
	levelDump(e, "start")
	if err := n.ExitTo(ctx, 2); err != nil {
		return fmt.Errorf("first ExitTo(2): %w", err)
	}
	levelDump(e, "in blood moor")
	if err := n.ExitTo(ctx, 1); err != nil {
		return fmt.Errorf("first ExitTo(1): %w", err)
	}
	levelDump(e, "back in town")
	cl := input.NewClient("")
	fl := &launch.Flow{Mem: proc, In: typer{input.NewOOG(cl), cl}, Log: func(s string) { fmt.Println("  flow:", s) }}
	if err := fl.LeaveGame(ctx, 30*time.Second); err != nil {
		return fmt.Errorf("leave: %w", err)
	}
	if err := fl.Start(ctx, launch.CharSpec{Name: "Navi", Difficulty: "normal"}); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if reset {
		e.World.Reset()
	}
	time.Sleep(3 * time.Second)
	levelDump(e, "re-entered (reset="+fmt.Sprint(reset)+")")
	if err := n.ExitTo(ctx, 2); err != nil {
		return fmt.Errorf("ExitTo(2) after re-entry: %w", err)
	}
	levelDump(e, "blood moor after re-entry")
	if err := n.ExitTo(ctx, 1); err != nil {
		return fmt.Errorf("ExitTo(1) after re-entry: %w", err)
	}
	levelDump(e, "town after re-entry")
	return nil
}

// wpLevel: levels of act 1 that have a waypoint (levels.txt ids).
func wpLevel(l int) bool {
	switch l {
	case 3, 4, 5, 6, 27, 29, 32, 35:
		return true
	}
	return false
}
