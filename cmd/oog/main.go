// oog: live OOG explorer/driver. Each argument is one command, run in order:
//
//	state            phase, ESC-menu flag, control list
//	raw              hex of every control struct (layout check)
//	click X Y | key NAME | char C | type TEXT | sleep MS
//	waitphase menu|game|none TIMEOUT_MS
//	host NAME | join NAME IP   LAN TCP/IP (never Battle.net)
//	fresh NAME class   create + verify (name, class, level 1 read from the game) + Save and Exit; see docs/fresh-character.md
//	start NAME [class]   launch.Flow.Start (select; creates when a class is given)   create NAME class   leave   launch.Flow.LeaveGame
package main

import (
	"context"
	"d2lod-bot/internal/buildinfo"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
)

func main() {
	buildinfo.Exit("oog")
	cli := input.NewClient("")
	oog := input.NewOOG(cli)
	var src mem.Source
	if p, err := mem.OpenGame(); err == nil {
		defer p.Close()
		src = p
	} else {
		fmt.Println("no game memory:", err)
	}
	flow := &launch.Flow{Mem: src, In: typer{oog, cli}, Log: func(s string) { fmt.Println("  flow:", s) }}
	for _, a := range os.Args[1:] {
		f := strings.Fields(a)
		if len(f) == 0 {
			continue
		}
		fmt.Printf("> %s\n", a)
		var err error
		switch f[0] {
		case "state":
			err = state(src)
		case "who":
			var pl *d2.Player
			if pl, err = d2.New(src).Player(); err == nil {
				fmt.Printf("  player=%q class=%d level=%d area=%d diff=%d\n", pl.Name, pl.Class, pl.Level, pl.Area, pl.Difficulty)
			}
		case "clickuntil": // clickuntil X Y SCREEN TIMEOUT_MS: click every 1.5 s until the screen is identified
			x, _ := strconv.Atoi(f[1])
			y, _ := strconv.Atoi(f[2])
			ms, _ := strconv.Atoi(f[4])
			t0 := time.Now()
			for {
				cs, _ := launch.ReadControls(src)
				if string(launch.Identify(cs)) == f[3] {
					fmt.Printf("  %s after %v\n", f[3], time.Since(t0).Round(50*time.Millisecond))
					break
				}
				if time.Since(t0) > time.Duration(ms)*time.Millisecond {
					err = fmt.Errorf("timeout")
					break
				}
				fmt.Printf("  click at %v\n", time.Since(t0).Round(50*time.Millisecond))
				_ = oog.Click(x, y)
				time.Sleep(1500 * time.Millisecond)
			}
		case "attach": // attach MS: (re)open game memory, waiting for Game.exe
			ms, _ := strconv.Atoi(f[1])
			t0 := time.Now()
			for {
				p, e := mem.OpenGame()
				if e == nil {
					src = p
					flow.Mem = p
					fmt.Printf("  attached after %v\n", time.Since(t0).Round(50*time.Millisecond))
					break
				}
				if time.Since(t0) > time.Duration(ms)*time.Millisecond {
					err = e
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		case "watch": // watch MS: print every change of (phase, screen, control count) with elapsed time
			ms, _ := strconv.Atoi(f[1])
			t0 := time.Now()
			last := ""
			for time.Since(t0) < time.Duration(ms)*time.Millisecond {
				ph, e := launch.ReadPhase(src)
				cs, _ := launch.ReadControls(src)
				cur := fmt.Sprintf("phase=%v screen=%s controls=%d err=%v", ph, launch.Identify(cs), len(cs), e)
				if cur != last {
					fmt.Printf("  %6v %s\n", time.Since(t0).Round(50*time.Millisecond), cur)
					for _, c := range cs {
						if len(cs) < 8 {
							fmt.Printf("         ctl type=%d x=%d y=%d w=%d h=%d text=%q\n", c.Type, c.X, c.Y, c.W, c.H, c.Text)
						}
					}
					last = cur
				}
				time.Sleep(100 * time.Millisecond)
			}
		case "raw":
			err = raw(src)
		case "party": // party i1:Navi:leader i2:Combi:killer   (then pstatus | pplayers | pleave ID | plost ID | prejoin ID | pdisband)
			err = partyCmd(context.Background(), os.Getenv("D2BOT_INSTANCE"), f[1:])
		case "pstatus", "pplayers", "pdisband":
			err = partyOp(context.Background(), f[0], "")
		case "pleave", "plost", "prejoin":
			err = partyOp(context.Background(), f[0], f[1])
		case "players":
			err = players(src)
		case "click":
			x, _ := strconv.Atoi(f[1])
			y, _ := strconv.Atoi(f[2])
			err = oog.Click(x, y)
		case "key":
			err = oog.Key(f[1])
		case "char":
			err = cli.TypeChar([]rune(f[1])[0])
		case "type":
			for _, r := range f[1] {
				if err = cli.TypeChar(r); err != nil {
					break
				}
				time.Sleep(60 * time.Millisecond)
			}
		case "sleep":
			ms, _ := strconv.Atoi(f[1])
			time.Sleep(time.Duration(ms) * time.Millisecond)
		case "waitphase":
			ms, _ := strconv.Atoi(f[2])
			err = waitPhase(src, f[1], time.Duration(ms)*time.Millisecond)
		case "start":
			ch := launch.CharSpec{Name: f[1], Difficulty: "normal"}
			if len(f) > 2 {
				switch f[2] {
				case "normal", "nightmare", "hell":
					ch.Difficulty = f[2] // "start Name nightmare": pick on the difficulty screen
				default:
					ch.Class, ch.Create = f[2], true
				}
			}
			err = flow.Start(context.Background(), ch)
		case "host": // host NAME: LAN TCP/IP host game with the character
			err = flow.HostGame(context.Background(), launch.CharSpec{Name: f[1], Difficulty: "normal"})
		case "join": // join NAME IP: LAN TCP/IP join
			err = flow.JoinGame(context.Background(), launch.CharSpec{Name: f[1], Difficulty: "normal"}, f[2])
		case "create":
			err = flow.CreateCharacter(context.Background(), f[1], f[2])
		case "fresh":
			if src == nil {
				err = fmt.Errorf("no game")
				break
			}
			err = flow.CreateVerified(context.Background(), f[1], f[2], launch.FreshOptions{Log: func(s string, a ...any) { fmt.Printf("  fresh: "+s+"\n", a...) }, Who: func() (launch.Who, error) {
				pl, e := d2.New(src).Player()
				if e != nil {
					return launch.Who{}, e
				}
				return launch.Who{Name: pl.Name, Class: pl.Class.String(), Level: pl.Level}, nil
			}})
		case "leave":
			err = flow.LeaveGame(context.Background(), 20*time.Second)
		case "fdcycle": // fdcycle N NAME: N times leave + start NAME, one fd line per cycle (agent LEAKHUNT)
			n, _ := strconv.Atoi(f[1])
			err = fdCycle(flow, n, f[2])
		case "recycle": // recycle NAME: scheduled recycle now (Save and Exit, stop, start, login); live check of launch.Recycler
			err = recycleNow(flow, f[1], &src)
		case "fdidle": // fdidle N SECONDS: N samples SECONDS apart without touching the game (control run)
			n, _ := strconv.Atoi(f[1])
			s, _ := strconv.Atoi(f[2])
			fdIdle(n, time.Duration(s)*time.Second)
		default:
			err = fmt.Errorf("unknown command %q", f[0])
		}
		if err != nil {
			fmt.Println("  error:", err)
			os.Exit(1)
		}
	}
}

// typer adds the WM_CHAR path to the input.OOG adapter.
type typer struct {
	*input.OOG
	c *input.Client
}

func (t typer) TypeChar(r rune) error { return t.c.TypeChar(r) }

func state(src mem.Source) error {
	if src == nil {
		return fmt.Errorf("no game")
	}
	ph, err := launch.ReadPhase(src)
	if err != nil {
		return err
	}
	esc, _ := mem.U32(src, mem.Base+0x3A27C0+4*9)
	fmt.Printf("  phase=%s escmenu=%d\n", ph, esc)
	cs, err := launch.ReadControls(src)
	for _, c := range cs {
		fmt.Printf("  ctl type=%d x=%d y=%d w=%d h=%d text=%q\n", c.Type, c.X, c.Y, c.W, c.H, c.Text)
	}
	fmt.Printf("  identify=%s (%d controls)\n", launch.Identify(cs), len(cs))
	return err
}

func raw(src mem.Source) error {
	p, err := mem.U32(src, mem.Base+0x3D55BC)
	for n := 0; p != 0 && err == nil && n < 200; n++ {
		b := make([]byte, 0x4C)
		if err = src.ReadAt(b, p); err != nil {
			break
		}
		fmt.Printf("  %08x % x\n", p, b)
		p = uint32(b[0x3C]) | uint32(b[0x3D])<<8 | uint32(b[0x3E])<<16 | uint32(b[0x3F])<<24
	}
	return err
}

func waitPhase(src mem.Source, want string, d time.Duration) error {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		ph, err := launch.ReadPhase(src)
		if err == nil && (want == "menu" && ph == launch.PhaseMenu || want == "game" && ph == launch.PhaseInGame || want == "none" && ph == launch.PhaseUnknown) {
			fmt.Println("  reached", ph)
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", want)
}
