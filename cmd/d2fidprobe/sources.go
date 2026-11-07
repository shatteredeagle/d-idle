package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/fidprobe"
	"d2lod-bot/internal/mem"
)

// Both live sources are read-only: memory reads through internal/mem, and the d2d dump and read-only
// verbs through d2dsrc (the same code d2dsnap runs). Nothing here sends input or writes.

type gameExe struct{ instance string }

func (g gameExe) Name() string { return "gameexe" }

func (g gameExe) Read(context.Context) (fidprobe.Input, error) {
	p, err := mem.OpenInstance(g.instance)
	if err != nil {
		return fidprobe.Input{}, err
	}
	defer p.Close()
	r := d2.New(p)
	snap, err := r.Snapshot()
	if err != nil {
		return fidprobe.Input{}, err
	}
	in := fidprobe.Input{Snap: snap}
	if snap.Player == nil {
		return in, nil
	}
	su, err := r.FindServerPlayer(snap.Player)
	if err == nil {
		in.Progress, err = r.ReadProgress(su)
	}
	if err != nil {
		in.ProgressWhy = "quest words: " + err.Error()
	}
	return in, nil
}

type d2dSide struct {
	sock, tables string
}

func (d d2dSide) Name() string { return "d2d" }

func (d d2dSide) Read(ctx context.Context) (fidprobe.Input, error) {
	opts := d2dsrc.Options{}
	if d.tables != "" {
		db, err := data.Load(d.tables)
		if err != nil {
			return fidprobe.Input{}, err
		}
		opts.Tables = d2dsrc.NewDBTables(db)
	}
	c := d2dctl.New(d.sock, d2dctl.Options{Timeout: 8 * time.Second})
	defer c.Close()
	src := d2dsrc.New(c, opts)
	res, err := src.Read(ctx)
	if err != nil {
		return fidprobe.Input{}, err
	}
	in := fidprobe.Input{Snap: res.Snap, Unreadable: res.Report.Unknown}
	if in.Progress, err = src.Progress(ctx); err != nil {
		in.ProgressWhy = "quest words: " + err.Error()
	}
	return in, nil
}

// jsonSide replays a recorded snapshot document (d2dsnap output or a plain d2.Snapshot).
type jsonSide struct{ name, path string }

func (j jsonSide) Name() string { return j.name }

func (j jsonSide) Read(context.Context) (fidprobe.Input, error) {
	b, err := os.ReadFile(j.path)
	if err != nil {
		return fidprobe.Input{}, err
	}
	return fidprobe.ParseInput(b)
}

func newSource(spec, role string, f *flags) (fidprobe.Source, error) {
	switch {
	case spec == "gameexe":
		return gameExe{f.instance}, nil
	case spec == "d2d":
		if f.sock == "" {
			return nil, fmt.Errorf("d2fidprobe: d2d side needs -sock or $D2D_SOCK")
		}
		return d2dSide{f.sock, f.tables}, nil
	case strings.HasPrefix(spec, "json:"):
		return jsonSide{role, spec[5:]}, nil
	}
	return nil, fmt.Errorf("d2fidprobe: source %q: want gameexe, d2d or json:FILE", spec)
}
