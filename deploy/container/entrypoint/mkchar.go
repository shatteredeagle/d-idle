package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
)

// typer adds the WM_CHAR path to the input.OOG adapter (same as cmd/oog and cmd/d2bot).
type typer struct {
	*input.OOG
	c *input.Client
}

func (t typer) TypeChar(r rune) error { return t.c.TypeChar(r) }

// mkchar launches helper and game as our own children (we are the subreaper, so no capability is needed),
// creates one character through the OOG flow, leaves to the main menu and stops everything. The
// character is saved in the Wine prefix on the volume. Run it under `d2container run --` (needs Xvfb).
//
//	d2container run -- d2container mkchar -name Contai -class sorceress [-data /data]
func mkchar(args []string) int {
	fs := flag.NewFlagSet("mkchar", flag.ExitOnError)
	name := fs.String("name", "Contai", "character name (letters only)")
	class := fs.String("class", "sorceress", "class key of launch.ClassClick")
	data := fs.String("data", os.Getenv("D2BOT_DATA"), "data root (Wine prefix under <data>/proton)")
	fs.Parse(args)
	ctx := context.Background()
	cfg := launch.Defaults()
	cfg.Root = *data
	if err := cfg.Resolve(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	sp := launch.NewSpawner()
	helper := launch.NewHelper(cfg, sp)
	if err := helper.Ensure(ctx, 60*time.Second); err != nil {
		fmt.Fprintln(os.Stderr, "helper:", err)
		return 1
	}
	game := launch.NewGame(cfg, sp)
	t0 := time.Now()
	if _, err := game.Launch(ctx, 90*time.Second); err != nil {
		fmt.Fprintln(os.Stderr, "launch:", err)
		return 1
	}
	pid, _, rd, err := game.Attach()
	if err != nil {
		fmt.Fprintln(os.Stderr, "attach:", err)
		return 1
	}
	fmt.Println("MKCHAR game pid", pid, "attached after", time.Since(t0).Round(time.Millisecond))
	cli := input.NewClient(cfg.HelperAddr)
	flow := &launch.Flow{Mem: rd, In: typer{input.NewOOG(cli), cli}, Log: func(s string) { fmt.Println("  flow:", s) }}
	rc := 0
	if err := flow.Start(ctx, launch.CharSpec{Name: *name, Class: *class, Create: true, ForceNew: true, Difficulty: "normal"}); err != nil {
		fmt.Fprintln(os.Stderr, "create/enter:", err)
		rc = 1
	} else {
		fmt.Println("MKCHAR in game as", *name, "after", time.Since(t0).Round(time.Millisecond))
		time.Sleep(3 * time.Second)
		if err := flow.LeaveGame(ctx, 30*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, "leave:", err)
			rc = 1
		}
	}
	game.Shutdown(5 * time.Second)
	helper.Stop()
	cfg.WineserverKill()
	return rc
}
