// probes: prebuilt read-only live probes (docs/probes.md). Run through tools/live.sh <agent> probes <sub> [flags].
// Output is one JSON object per line on stdout; nothing here writes game memory. Only `runewords cube` presses
// buttons (Town.Transmute), and it needs an explicit -go.
package main

import (
	"context"
	"d2lod-bot/internal/buildinfo"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/town"
)

const usage = `usage: probes <sub> [flags]
  runewords check            list socketed bases, runes, runeword items and the runewords the pack could make
  runewords cube -go [-recipe hel|custom -inputs a,b,c]   run a cube recipe through Town.Transmute (needs -go)
  monflags   [-sec N -radius N]        monster flag/enchant census to JSON lines
  mercdump   [-sec N]                  d2.FindMerc fields each second plus the UI/NPC dialog flags
  missiledump [-sec N -ms N]           missiles in flight and the player's curse states
  wpdump     [-sec N]                  waypoint masks (acts 3-5), panel flag, nearby waypoint objects`

func main() {
	buildinfo.Exit("probes")
	err := run(os.Args[1:], io.Writer(os.Stdout), realEnv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "probes:", err)
		os.Exit(1)
	}
}

// run parses the sub and flags first (so bad arguments fail without a game), then builds the Env and runs it.
func run(args []string, out io.Writer, build func(io.Writer) (*Env, func(), error)) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	sub, rest := args[0], args[1:]
	var fn func(context.Context, *Env) error
	switch sub {
	case "runewords":
		f, err := parseRunewords(rest)
		if err != nil {
			return err
		}
		fn = f
	case "monflags":
		f, err := parseMonflags(rest)
		if err != nil {
			return err
		}
		fn = f
	case "mercdump":
		f, err := parseMercdump(rest)
		if err != nil {
			return err
		}
		fn = f
	case "missiledump":
		f, err := parseMissiledump(rest)
		if err != nil {
			return err
		}
		fn = f
	case "wpdump":
		f, err := parseWpdump(rest)
		if err != nil {
			return err
		}
		fn = f
	default:
		return fmt.Errorf("unknown sub %q\n%s", sub, usage)
	}
	e, closeFn, err := build(out)
	if err != nil {
		return err
	}
	defer closeFn()
	return fn(context.Background(), e)
}

// newFlags makes a FlagSet that returns errors instead of exiting; -sec is the common run length (hard cap 170 s,
// AGENTS.md: live runs stay under 3 minutes).
func newFlags(name string) (*flag.FlagSet, *int) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	sec := fs.Int("sec", 20, "run length in seconds (1..170)")
	return fs, sec
}

func parseFlags(fs *flag.FlagSet, args []string, sec *int) error {
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%s: %w", fs.Name(), err)
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("%s: unexpected arguments %v", fs.Name(), fs.Args())
	}
	if *sec < 1 || *sec > 170 {
		return fmt.Errorf("%s: -sec %d out of range 1..170", fs.Name(), *sec)
	}
	return nil
}

func realEnv(out io.Writer) (*Env, func(), error) {
	src, err := mem.OpenGame() // honors D2BOT_INSTANCE
	if err != nil {
		return nil, nil, err
	}
	home, _ := os.UserHomeDir()
	db, err := data.Open(home+"/Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II", "/tmp/d2lod-town-cache")
	if err != nil {
		src.Close()
		return nil, nil, err
	}
	rd := d2.New(src)
	game := input.NewGame(input.NewClient(""), input.Reader{Src: src})
	tune := env.DefaultTuning()
	ge := &env.Env{ID: "probes", Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})),
		Mem: src, Read: rd, World: d2map.NewTracker(src), In: game, Data: db, Tune: &tune, Clock: env.RealClock{},
		Emit: func(goals.Event) {}}
	t := town.New(ge, nav.New(ge))
	e := &Env{
		Out: out, Snap: rd.Snapshot, Sleep: time.Sleep, Now: time.Now,
		Progress: rd.ReadProgress,
		Missiles: rd.MissileUnits,
		MissileName: func(c int) string {
			if m := db.MissileByID(c); m != nil {
				return m.Name
			}
			return ""
		},
		StateName: func(id int) string {
			if s := db.StateByID(id); s != nil {
				return s.Name
			}
			return ""
		},
		ItemCode: func(c int) string {
			if c >= 0 && c < len(db.Items) && db.Items[c] != nil {
				return db.Items[c].Code
			}
			return ""
		},
		MonsterName: func(c int) (string, bool) {
			if m := db.MonsterByClassID(c); m != nil {
				return m.Key, m.NPC
			}
			return "", false
		},
		CurseIDs: d2.ResolveCurseIDs(func(n string) (int, bool) {
			if s := db.StateByName(n); s != nil {
				return s.ID, true
			}
			return 0, false
		}),
		Transmute: t.Transmute,
		CloseAll:  game.CloseAll,
	}
	for _, rw := range db.Runewords {
		if rw.Complete {
			e.Runewords = append(e.Runewords, RuneRow{Name: rw.Name, Ladder: rw.Ladder, Runes: rw.Runes})
		}
	}
	return e, func() { src.Close() }, nil
}

func csv(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
