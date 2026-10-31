// Command d2dprobe talks to a local d2d control socket (never TCP).
//
//	d2dprobe -sock PATH ping|info|state|npcs|monsters|ground|menu
//	d2dprobe -sock PATH raw 'debug warps'
//	d2dprobe -sock PATH [-json] -script FILE
//
// See docs/d2dctl.md for the script format. Exit status 1 on any failed step or transport error.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("d2dprobe", flag.ContinueOnError)
	fs.SetOutput(errw)
	sock := fs.String("sock", os.Getenv("D2D_SOCK"), "control socket path (default $D2D_SOCK)")
	script := fs.String("script", "", "run a probe script file ('-' = stdin)")
	asJSON := fs.Bool("json", false, "JSON output")
	timeout := fs.Duration("timeout", 10*time.Second, "per-command timeout")
	total := fs.Duration("total", 0, "overall limit (0 = none)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *sock == "" || strings.Contains(*sock, "://") {
		fmt.Fprintln(errw, "need -sock PATH (a local unix socket path)")
		return 2
	}
	ctx := context.Background()
	if *total > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *total)
		defer cancel()
	}
	c := d2dctl.New(*sock, d2dctl.Options{Timeout: *timeout})
	defer c.Close()

	if err := waitFree(ctx, c); err != nil {
		fmt.Fprintln(errw, "error:", err)
		return 1
	}
	if *script != "" {
		return runScript(ctx, c, *script, *asJSON, out, errw)
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(errw, "usage: d2dprobe -sock PATH ping|info|state|npcs|monsters|ground|menu|raw '<verb line>' | -script FILE")
		return 2
	}
	verb, line := rest[0], ""
	switch verb {
	case "ping", "info", "state", "npcs", "monsters", "ground", "menu":
		line = verb
	case "raw":
		if len(rest) < 2 {
			fmt.Fprintln(errw, "raw needs a verb line")
			return 2
		}
		line = strings.Join(rest[1:], " ")
	default:
		fmt.Fprintf(errw, "unknown subcommand %q\n", verb)
		return 2
	}
	t0 := time.Now()
	rep, err := c.Do(ctx, line)
	parsed := parse(verb, rep)
	if *asJSON {
		return emit(out, struct {
			Cmd    string   `json:"cmd"`
			OK     bool     `json:"ok"`
			Err    string   `json:"err,omitempty"`
			Lines  []string `json:"lines,omitempty"`
			Rest   string   `json:"rest,omitempty"`
			Bad    []string `json:"bad,omitempty"`
			Parsed any      `json:"parsed,omitempty"`
			MS     int64    `json:"ms"`
		}{line, err == nil, errStr(err), lines(rep), restOf(rep), badOf(rep), parsed, time.Since(t0).Milliseconds()}, err)
	}
	if rep != nil {
		for _, l := range rep.Lines {
			fmt.Fprintln(out, l)
		}
		if err == nil {
			fmt.Fprintln(out, strings.TrimSpace("ok "+rep.Rest))
		}
	}
	if err != nil {
		fmt.Fprintln(errw, "error:", err)
		return 1
	}
	return 0
}

// waitFree pings until the channel stops answering "busy": the previous probe's connection may still be
// closing on d2d's side (single-client channel), which would fail a quick follow-up run.
func waitFree(ctx context.Context, c *d2dctl.Client) error {
	deadline := time.Now().Add(3 * time.Second)
	for {
		err := c.Ping(ctx)
		if !errors.Is(err, d2dctl.ErrBusy) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func parse(verb string, rep *d2dctl.Reply) any {
	if rep == nil {
		return nil
	}
	switch verb {
	case "npcs":
		return list(rep, d2dctl.ParseNPC)
	case "monsters":
		return list(rep, d2dctl.ParseMonster)
	case "ground":
		return list(rep, d2dctl.ParseGround)
	case "menu":
		return list(rep, d2dctl.ParseMenuLine)
	}
	return nil
}

func list[T any](rep *d2dctl.Reply, p func(string) (T, error)) []T {
	var out []T
	for _, l := range rep.Lines {
		if v, err := p(l); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func lines(r *d2dctl.Reply) []string {
	if r == nil {
		return nil
	}
	return r.Lines
}
func restOf(r *d2dctl.Reply) string {
	if r == nil {
		return ""
	}
	return r.Rest
}
func badOf(r *d2dctl.Reply) []string {
	if r == nil {
		return nil
	}
	return r.Bad
}
func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func emit(out io.Writer, v any, err error) int {
	enc := json.NewEncoder(out)
	enc.SetIndent("", " ")
	enc.Encode(v)
	if err != nil {
		return 1
	}
	return 0
}

func runScript(ctx context.Context, c *d2dctl.Client, path string, asJSON bool, out, errw io.Writer) int {
	var r io.Reader = os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 2
		}
		defer f.Close()
		r = f
	}
	steps, err := d2dctl.RunScript(ctx, c, r)
	if asJSON {
		return emit(out, struct {
			OK    bool          `json:"ok"`
			Err   string        `json:"err,omitempty"`
			Steps []d2dctl.Step `json:"steps"`
		}{err == nil, errStr(err), steps}, err)
	}
	for _, s := range steps {
		mark := "ok  "
		if !s.OK {
			mark = "FAIL"
		}
		fmt.Fprintf(out, "%s %3d %-6s %s (%d ms)\n", mark, s.N, s.Kind, s.Line, s.MS)
		if s.Reply != "" {
			for _, l := range strings.Split(s.Reply, "\n") {
				fmt.Fprintln(out, "       | "+l)
			}
		}
		if s.Err != "" {
			fmt.Fprintln(out, "       ! "+s.Err)
		}
	}
	if err != nil {
		fmt.Fprintln(errw, "script failed:", err)
		return 1
	}
	return 0
}
