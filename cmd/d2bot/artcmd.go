package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"d2lod-bot/internal/art"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/paths"
)

// artMain is `d2bot art extract`: decode every PNG the watch pages can ask for into <data>/cache/art, ahead of
// time (docs/paperdoll-art-build.md). Idempotent (what is cached is skipped), reads only the user's own install.
// Without the archives it says so and exits 1; the pages then keep their drawn cells. d2bot also does this in
// the background at start, so running it is only needed to warm a cache on purpose (after an update or install).
func artMain(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "extract" {
		return errors.New("usage: d2bot art extract [--gamedir DIR] [--data DIR]")
	}
	fs := flag.NewFlagSet("art extract", flag.ContinueOnError)
	fs.SetOutput(out)
	dataDir := fs.String("data", envOr("D2BOT_DATA", ""), "data dir (env D2BOT_DATA; default ~/.local/share/d2bot); the cache goes to <data>/cache/art")
	gameDir := fs.String("gamedir", envOr("D2BOT_GAMEDIR", ""), "Diablo II install dir holding the MPQs (env D2BOT_GAMEDIR)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *dataDir == "" {
		l, err := paths.FromEnv()
		if err != nil {
			return err
		}
		*dataDir = l.Root
	}
	if *gameDir == "" {
		return errors.New("art extract: no install: set --gamedir or D2BOT_GAMEDIR (no art, the pages keep their drawn cells)")
	}
	db, err := data.Open(*gameDir, filepath.Join(*dataDir, "cache", "tables"))
	if err != nil {
		return fmt.Errorf("art extract: game tables: %w (needs the install's MPQs; the pages keep their drawn cells)", err)
	}
	s, err := art.Open(*gameDir, filepath.Join(*dataDir, "cache", "art"))
	if err != nil {
		return fmt.Errorf("art extract: %w (no art, the pages keep their drawn cells)", err)
	}
	defer s.Close()
	step := 0
	p, err := s.Extract(context.Background(), art.Keys(db.InvFiles()), func(done, total int) {
		if step++; step%100 == 0 || done == total {
			fmt.Fprintf(out, "art: %d/%d\n", done, total)
		}
	})
	if err != nil {
		return fmt.Errorf("art extract: stopped at %d/%d: %w", p.Done, p.Total, err)
	}
	fmt.Fprintf(out, "art ready: %d images, %d without art in this install\n", p.Total-p.Missing, p.Missing)
	return nil
}
