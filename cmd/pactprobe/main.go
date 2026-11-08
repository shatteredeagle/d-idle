// pactprobe: live probe for the native difficulty levers (TASK-170, docs/probes/pactnative.md). Read-only on game
// memory; input only through the d2input helper (no cheat writes).
//
//	pactprobe say TEXT...      open chat with Enter, type TEXT with WM_CHAR, Enter
//	pactprobe mons             one line per monster in the unit table (class, id, mode, life, max life, level, flags)
//	pactprobe fight SECS       like scan, but melee the nearest Fallen (class 19) and log every change of its life and of the
//	                           player life (hit sizes); retreats to town below 40 percent life
//	pactprobe scan SECS        leave town to the Blood Moor, explore SECS and log every monster at first sight
//	                           (retreats to town below 50 percent life), then walk back to town
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

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
	if len(os.Args) < 2 {
		die(fmt.Errorf("usage: pactprobe say|mons|scan"))
	}
	proc, err := mem.OpenGame()
	if err != nil {
		die(err)
	}
	defer proc.Close()
	rd := d2.New(proc)
	tune := env.DefaultTuning()
	e := &env.Env{ID: "pactnative", Mem: proc, Read: rd, World: d2map.NewTracker(proc), Tune: &tune, Clock: env.RealClock{}}
	if home, err := os.UserHomeDir(); err == nil {
		dir := home + "/Games/d2lod-inst/" + os.Getenv("D2BOT_INSTANCE") + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
		if db, err := data.Open(dir, "/tmp/d2lod-pactnative-cache"); err == nil {
			e.Data = db
		}
	}
	cl := input.NewClient("")
	game := input.NewGame(cl, input.Reader{Src: proc})
	e.In = game
	switch os.Args[1] {
	case "say":
		say(cl, strings.Join(os.Args[2:], " "))
	case "mons":
		mons(e, nil)
	case "scan":
		secs, _ := strconv.Atoi(os.Args[2])
		scan(e, nav.New(e), secs)
	case "hc": // hc NAME CLASS: create a HARDCORE character through the normal OOG flow and enter the game
		flow := &launch.Flow{Mem: proc, In: typer{input.NewOOG(cl), cl}, Log: func(s string) { fmt.Println("  flow:", s) }}
		err := flow.Start(context.Background(), launch.CharSpec{Name: os.Args[2], Class: os.Args[3], Create: true, ForceNew: true, Hardcore: true, Difficulty: "normal"})
		fmt.Println("hc start:", err)
	case "fight":
		secs, _ := strconv.Atoi(os.Args[2])
		fight(e, game, nav.New(e), secs)
	default:
		die(fmt.Errorf("unknown command %q", os.Args[1]))
	}
}

func say(cl *input.Client, text string) {
	steps := []string{"key press enter hold=60", "sleep 300"}
	for _, ch := range text {
		c := string(ch)
		if ch == ' ' {
			c = "space"
		}
		steps = append(steps, "char "+c, "sleep 60")
	}
	steps = append(steps, "sleep 200", "key press enter hold=60")
	fmt.Println("say:", text, cl.Seq(steps...))
}

type seenMon struct {
	d2.Unit
	first time.Time
}

func mons(e *env.Env, seen map[uint32]*seenMon) {
	s, err := e.Snapshot()
	if err != nil || s.State != d2.StateInGame {
		die(fmt.Errorf("not in game: %v", err))
	}
	fmt.Printf("player area=%d pos=(%d,%d) life=%d/%d\n", s.Player.Area, s.Player.X, s.Player.Y, s.Player.Life, s.Player.MaxLife)
	for _, u := range s.Monsters {
		if seen != nil {
			if _, ok := seen[u.ID]; !ok && u.Monster != nil {
				seen[u.ID] = &seenMon{Unit: u, first: time.Now()}
			}
			continue
		}
		printMon(u)
	}
}

func printMon(u d2.Unit) {
	m := u.Monster
	if m == nil {
		fmt.Printf("  class=%d id=%d mode=%d (no info)\n", u.Class, u.ID, u.Mode)
		return
	}
	fmt.Printf("  class=%d id=%d mode=%d pos=(%d,%d) life=%d max=%d pct=%d lvl=%d flags=%#x enchants=%v uniq=%d name=%q\n",
		u.Class, u.ID, u.Mode, u.X, u.Y, m.Life, m.MaxLife, m.LifePct, m.Level, m.Flags, m.Enchants, m.UniqueNo, m.Name)
}

func scan(e *env.Env, n *nav.Navigator, secs int) {
	s, err := e.Snapshot()
	if err != nil || s.State != d2.StateInGame || s.Player.Area != 1 {
		die(fmt.Errorf("start in town: %v", err))
	}
	_, _ = e.World.Update()
	var townRect d2map.Rect
	if lm := e.World.Level(1); lm != nil {
		townRect = lm.Rect
	}
	seen := map[uint32]*seenMon{}
	n.Guard = func(sn *d2.Snapshot) error {
		for _, u := range sn.Monsters {
			if u.Monster != nil && !townRect.Contains(u.X, u.Y) {
				if _, ok := seen[u.ID]; !ok {
					seen[u.ID] = &seenMon{Unit: u, first: time.Now()}
				}
			}
		}
		p := sn.Player
		if p.MaxLife > 0 && p.Life*100 < 50*p.MaxLife {
			return fmt.Errorf("life %d/%d", p.Life, p.MaxLife)
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(secs+120)*time.Second)
	defer cancel()
	fmt.Println("ExitTo(2):", n.ExitTo(ctx, 2))
	end := time.Now().Add(time.Duration(secs) * time.Second)
	fmt.Println("Explore:", n.Explore(ctx, func() bool { return time.Now().After(end) }))
	n.Guard = nil
	rctx, c := context.WithTimeout(context.Background(), 90*time.Second)
	defer c()
	fmt.Println("ToArea(1):", n.ToArea(rctx, 1))
	list := make([]*seenMon, 0, len(seen))
	for _, v := range seen {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Class < list[j].Class })
	fmt.Println("monsters at first sight:", len(list))
	for _, v := range list {
		printMon(v.Unit)
	}
}

const fallenClass = 19

// fight explores the Blood Moor until a Fallen is near, melees it with the left-click Attack, and logs every
// change of its client life (128 scale, so a drop is a fraction of max life) and of the player life.
func fight(e *env.Env, g *input.Game, n *nav.Navigator, secs int) {
	s, err := e.Snapshot()
	if err != nil || s.State != d2.StateInGame || s.Player.Area != 1 {
		die(fmt.Errorf("start in town: %v", err))
	}
	_, _ = e.World.Update()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(secs+150)*time.Second)
	defer cancel()
	fmt.Println("ExitTo(2):", n.ExitTo(ctx, 2))
	end := time.Now().Add(time.Duration(secs) * time.Second)
	lastLife := map[uint32]int{}
	lastPl := -1
	t0 := time.Now()
	nearest := func(sn *d2.Snapshot) *d2.Unit {
		var best *d2.Unit
		bd := 1e18
		for i := range sn.Monsters {
			u := &sn.Monsters[i]
			if u.Class != fallenClass || !u.Alive() || u.Monster == nil {
				continue
			}
			dx, dy := float64(u.X-sn.Player.X), float64(u.Y-sn.Player.Y)
			if d := dx*dx + dy*dy; d < bd {
				best, bd = u, d
			}
		}
		if bd > 400 {
			return nil
		}
		return best
	}
	// A fast poller (about 40 ms) so that single hits are seen; the main loop only drives the fight.
	stopPoll := make(chan struct{})
	pollDone := make(chan struct{})
	rd := d2.New(e.Mem)
	go func() {
		defer close(pollDone)
		for {
			select {
			case <-stopPoll:
				return
			case <-time.After(40 * time.Millisecond):
			}
			el := time.Since(t0).Round(10 * time.Millisecond)
			if pl, err := rd.Player(); err == nil {
				if lastPl >= 0 && pl.Life != lastPl {
					fmt.Printf("%v PLAYER life %d -> %d (%+d)\n", el, lastPl, pl.Life, pl.Life-lastPl)
				}
				lastPl = pl.Life
			}
			us, err := rd.Units(d2.TypeMonster)
			if err != nil {
				continue
			}
			for _, u := range us {
				if u.Class != fallenClass || u.Monster == nil {
					continue
				}
				if old, ok := lastLife[u.ID]; ok && old != u.Monster.Life {
					fmt.Printf("%v FALLEN id=%d life %d -> %d mode=%d\n", el, u.ID, old, u.Monster.Life, u.Mode)
				}
				lastLife[u.ID] = u.Monster.Life
			}
		}
	}()
	defer func() { close(stopPoll); <-pollDone }()
	for time.Now().Before(end) {
		sn, err := e.Snapshot()
		if err != nil {
			continue
		}
		if sn.Player.MaxLife > 0 && sn.Player.Life*100 < 40*sn.Player.MaxLife {
			fmt.Println("RETREAT life", sn.Player.Life, "/", sn.Player.MaxLife)
			break
		}
		tgt := nearest(sn)
		if tgt == nil {
			stop := func() bool {
				sn2, err := e.Snapshot()
				return err == nil && (nearest(sn2) != nil || time.Now().After(end))
			}
			if err := n.Explore(ctx, stop); err != nil {
				fmt.Println("Explore:", err)
			}
			continue
		}
		dx, dy := float64(tgt.X-sn.Player.X), float64(tgt.Y-sn.Player.Y)
		if dx*dx+dy*dy > 9 {
			_ = g.MoveTo(input.Point{X: tgt.X, Y: tgt.Y}, input.MoveOpts{Tolerance: 2, Timeout: 4 * time.Second})
		}
		if err := g.Interact(input.Target{Type: 1, ID: tgt.ID, Pos: input.Point{X: tgt.X, Y: tgt.Y}}); err != nil {
			fmt.Println("interact:", err)
		}
		time.Sleep(250 * time.Millisecond)
	}
	rctx, c := context.WithTimeout(context.Background(), 90*time.Second)
	defer c()
	fmt.Println("ToArea(1):", n.ToArea(rctx, 1))
}

// typer adds the WM_CHAR path to the input.OOG adapter (as cmd/oog does).
type typer struct {
	*input.OOG
	c *input.Client
}

func (t typer) TypeChar(r rune) error { return t.c.TypeChar(r) }
