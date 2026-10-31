// Command origcasts is the Game.exe side of the d2d probes (docs/orig-spawns.md, docs/orig-casts.md). It runs on
// The worker through tools/live.sh against instance i0 (Testy) and prints CSV rows in the columns of docs/probes/d2d-*.csv.
//
//	origcasts spawns VISIT AREA...    walk and explore each area in turn from town, census the monster table
//	origcasts casts AREA CBLEVEL CLVL SECONDS   fight with the bot's combat.Fighter, one row per kill
//	origcasts mana SECONDS            mana regeneration rate (frames per wall second, UNVERIFIED method)
//
// Memory is read-only except the logged god cheat (internal/cheat, throwaway character, reason on every call).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
)

func die(err error) {
	fmt.Fprintln(os.Stderr, "FAIL:", err)
	os.Exit(1)
}

type rig struct {
	e    *env.Env
	nav  *nav.Navigator
	game *input.Game
	db   *data.DB
}

func newRig() *rig {
	src, err := mem.OpenGame() // honors D2BOT_INSTANCE
	if err != nil {
		die(err)
	}
	home, _ := os.UserHomeDir()
	dir := home + "/Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	if id := os.Getenv(mem.InstanceEnv); id != "" && id != "i0" {
		dir = home + "/Games/d2lod-inst/" + id + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	}
	db, err := data.Open(dir, home+"/d2lod-live/origprobes/data")
	if err != nil {
		die(err)
	}
	tune := env.DefaultTuning()
	game := input.NewGame(input.NewClient(""), input.Reader{Src: src})
	e := &env.Env{ID: "origprobes", Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
		Mem: src, Read: d2.New(src), In: game, Data: db, Tune: &tune, Clock: env.RealClock{}, World: d2map.NewTracker(src)}
	e.Emit = func(goals.Event) {}
	return &rig{e: e, nav: nav.New(e), game: game, db: db}
}

func main() {
	if len(os.Args) < 2 {
		die(fmt.Errorf("usage: origcasts spawns|casts|mana ..."))
	}
	r := newRig()
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Minute)
	defer cancel()
	switch os.Args[1] {
	case "spawns":
		if len(os.Args) < 4 {
			die(fmt.Errorf("usage: spawns VISIT AREA..."))
		}
		areas := os.Args[3:]
		r.spawns(ctx, os.Args[2], areas)
	case "casts":
		n := func(i int) int { v, _ := strconv.Atoi(os.Args[i]); return v }
		r.casts(ctx, n(2), n(3), n(4), time.Duration(n(5))*time.Second)
	case "mana":
		n, _ := strconv.Atoi(os.Args[2])
		r.manaRegen(ctx, time.Duration(n)*time.Second)
	default:
		die(fmt.Errorf("unknown subcommand %q", os.Args[1]))
	}
}

type navigator interface {
	ToArea(context.Context, int) error
	PathTo(input.Point) ([]input.Point, error)
	GoTo(context.Context, input.Point) error
}

func newNav(r *rig) *nav.Navigator { return nav.New(r.e) }

func installDir() string {
	home, _ := os.UserHomeDir()
	if id := os.Getenv(mem.InstanceEnv); id != "" && id != "i0" {
		return home + "/Games/d2lod-inst/" + id + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	}
	return home + "/Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II"
}

func (r *rig) god(ctx context.Context, reason string) func() error {
	return cheat.Gate{Allow: true, Env: cheat.DefaultEnv(), Actor: "origprobes"}.God(ctx, reason)
}
