// Command d2dsnap connects to a d2d control socket, builds one (or several) d2.Snapshot values through
// internal/d2dsrc and prints them as JSON, followed by an "unknown fields" summary on stderr. Read-only.
//
//	d2dsnap -sock ~/d2d-lab/fork/d2d.sock [-tables CACHEDIR] [-origin] [-progress] [-n 3 -every 1s] [-report]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/data"
)

func main() {
	sock := flag.String("sock", os.Getenv("D2D_SOCK"), "d2d control socket (default $D2D_SOCK)")
	tables := flag.String("tables", "", "extracted game table cache dir (data.Load); without it item rows and monster classes are unknown")
	origin := flag.Bool("origin", false, "add the debug level origin to positions (U)")
	progress := flag.Bool("progress", false, "also read quest words and waypoint masks (dump quests)")
	n := flag.Int("n", 1, "snapshots to take")
	every := flag.Duration("every", time.Second, "pause between snapshots")
	reportOnly := flag.Bool("report", false, "print only the field report, not the snapshot")
	corpses := flag.Bool("corpses", false, "fetch debug corpses into extras")
	warps := flag.Bool("warps", false, "fetch debug warps into extras")
	flag.Parse()
	if *sock == "" {
		fmt.Fprintln(os.Stderr, "d2dsnap: need -sock or $D2D_SOCK")
		os.Exit(2)
	}
	opts := d2dsrc.Options{AddOrigin: *origin, Corpses: *corpses, Warps: *warps,
		Logf: func(f string, a ...any) { fmt.Fprintf(os.Stderr, f+"\n", a...) }}
	if *tables != "" {
		db, err := data.Load(*tables)
		if err != nil {
			fmt.Fprintln(os.Stderr, "d2dsnap: tables:", err)
			os.Exit(1)
		}
		opts.Tables = d2dsrc.NewDBTables(db)
	}
	c := d2dctl.New(*sock, d2dctl.Options{Timeout: 8 * time.Second})
	defer c.Close()
	src := d2dsrc.New(c, opts)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	var last *d2dsrc.Result
	for i := 0; i < *n; i++ {
		if i > 0 {
			time.Sleep(*every)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		res, err := src.Read(ctx)
		cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, "d2dsnap:", err)
			os.Exit(1)
		}
		last = res
		out := map[string]any{"report": res.Report}
		if !*reportOnly {
			out["snapshot"], out["extras"] = res.Snap, res.Extras
		}
		if *progress {
			if pr, err := src.Progress(context.Background()); err != nil {
				out["progress_error"] = err.Error()
			} else {
				out["progress"] = pr
			}
		}
		if err := enc.Encode(out); err != nil {
			fmt.Fprintln(os.Stderr, "d2dsnap:", err)
			os.Exit(1)
		}
	}
	summary(last, src)
}

func summary(r *d2dsrc.Result, src *d2dsrc.Source) {
	fmt.Fprintf(os.Stderr, "\nstate=%v sources=%v present=%d derived=%d unknown=%d\n", r.Snap.State, r.Report.Source,
		len(r.Report.Present), len(r.Report.Derived), len(r.Report.Unknown))
	fmt.Fprintln(os.Stderr, "unknown fields (field: reason, times seen):")
	counts := src.UnknownCounts()
	keys := r.Report.UnknownFields()
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(os.Stderr, "  %-28s %s (x%d)\n", k, r.Report.Unknown[k], counts[k])
	}
}
