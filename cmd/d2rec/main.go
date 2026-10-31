// d2rec records and inspects replay fixtures (internal/replay).
//
//	d2rec record [--for 30s] [--every 100ms] [--out FILE] [--char NAME] [--seed N] [--note TEXT]
//	             record the live game (instance from D2BOT_INSTANCE) read-only; run via tools/live.sh
//	d2rec check FILE...   header summary and canary results of fixtures (exit 1 on a failed check)
//	d2rec wrap IN OUT     rewrite a legacy fixture (D2SN snapshot) as an envelope file
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"d2lod-bot/internal/canary"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/replay"
)

const recorderID = "d2rec/1"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "record":
		err = record(os.Args[2:])
	case "check":
		err = check(os.Args[2:])
	case "wrap":
		err = wrap(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2rec:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: d2rec record [--for 30s] [--every 100ms] [--out FILE] | check FILE... | wrap IN OUT")
	os.Exit(2)
}

func record(args []string) error {
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	dur := fs.Duration("for", 30*time.Second, "recording length")
	every := fs.Duration("every", 100*time.Millisecond, "tick interval")
	out := fs.String("out", "", "output file (default runs/rec-<instance>-<time>.d2fx)")
	char := fs.String("char", "", "character name for the header")
	seed := fs.Uint("seed", 0, "map seed for the header, if known")
	note := fs.String("note", "", "free text for the header")
	runner := fs.String("runner", "", "wine/proton version for the header")
	fs.Parse(args)

	p, err := mem.OpenGame() // honors D2BOT_INSTANCE and checks the build
	if err != nil {
		return fmt.Errorf("open game: %w", err)
	}
	defer p.Close()
	inst := os.Getenv(mem.InstanceEnv)
	if err := canary.Gate(p); err != nil {
		return fmt.Errorf("refusing to record: %w", err)
	}
	h := mem.NewHeader(mem.KindStream)
	h.Recorder, h.Runner, h.Instance, h.Char, h.Seed, h.Notes = recorderID, *runner, inst, *char, uint32(*seed), *note
	rec := replay.NewRecorder(p, h)
	rd := d2.New(rec)
	var last *d2.Snapshot
	ctx, cancel := context.WithTimeout(context.Background(), *dur)
	defer cancel()
	rec.Run(ctx, *every, func() {
		if s, err := rd.Snapshot(); err == nil {
			last = s
		}
	})
	st := rec.Stream()
	if last != nil && last.Player != nil {
		st.Header.Area, st.Header.Class = last.Player.Area, fmt.Sprint(last.Player.Class)
		st.Header.Difficulty = int(last.Player.Difficulty)
		if st.Header.Char == "" {
			st.Header.Char = last.Player.Name
		}
	}
	path := *out
	if path == "" {
		os.MkdirAll("runs", 0o755)
		path = fmt.Sprintf("runs/rec-%s-%s.d2fx", inst, time.Now().Format("20060102-150405"))
	}
	if err := st.Save(path); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %d frames, %d input calls, %s\n", path, len(st.Frames), len(st.Calls), st.Duration())
	return nil
}

func check(files []string) error {
	if len(files) == 0 {
		usage()
	}
	bad := 0
	for _, f := range files {
		fx, err := replay.Load(f)
		if err != nil {
			fmt.Printf("%s: %v\n", f, err)
			bad++
			continue
		}
		h := fx.Header
		fmt.Printf("%s: kind=%s schema=%d char=%q instance=%q created=%s\n", f, h.Kind, h.Schema, h.Char, h.Instance, h.CreatedAt)
		if err := replay.CheckBuild(h); err != nil {
			fmt.Println("  FAIL", err)
			bad++
		}
		if src := fx.Mem(); src != nil {
			for _, r := range canary.Check(src) {
				fmt.Println(" ", r)
				if r.Status == canary.Fail {
					bad++
				}
			}
		} else {
			fmt.Printf("  trace with %d records (no memory canary)\n", len(fx.Trace))
		}
	}
	if bad > 0 {
		return fmt.Errorf("%d problems", bad)
	}
	return nil
}

func wrap(args []string) error {
	if len(args) != 2 {
		usage()
	}
	h, s, err := mem.LoadSnapshotHeader(args[0])
	if err != nil {
		return err
	}
	if h.Schema != 0 {
		return fmt.Errorf("%s is already an envelope", args[0])
	}
	h = mem.NewHeader(mem.KindPages)
	h.Recorder = "d2rec/1 wrap of a legacy D2SN file"
	return s.SaveEnvelope(args[1], h)
}
