package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/paths"
)

// Checks added by the DOCTOR work package. Each is offline testable: file checks use real temp dirs,
// host effects go through Env (nil Dial/Now/Player mean "cannot tell" and skip or pass quietly).

func layout(o *Options) paths.Layout { return paths.Layout{Root: o.DataDir} }

func instID(n int) string { return "i" + strconv.Itoa(n) }

// checkConfig: every JSON file the tools read must parse. Missing files are fine (defaults apply).
func checkConfig(_ context.Context, e *Env, o *Options) Result {
	l := layout(o)
	files := []string{l.Config(), filepath.Join(o.DataDir, "setup.json")}
	for i := 0; i < o.Slots; i++ {
		in := l.MustInstance(instID(i))
		files = append(files, in.Goals(), in.Runtime(), in.Deaths(), in.Routes())
	}
	var bad, seen []string
	for _, f := range files {
		b, err := e.ReadFile(f)
		if err != nil {
			continue
		}
		seen = append(seen, f)
		if !json.Valid(b) {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return res(Fail, "not valid JSON: "+strings.Join(bad, ", "), "Fix the syntax, or move the file away so the default is used (the tools never overwrite a corrupt file).")
	}
	return res(Pass, fmt.Sprintf("%d config file(s) parse, others absent (defaults)", len(seen)), "")
}

// checkInstancePaths: per planned slot the prefix, save dir and instance dir. Before the first run they
// legitimately do not exist, so that is a warning; a file where a directory belongs fails.
func checkInstancePaths(_ context.Context, e *Env, o *Options) Result {
	l := layout(o)
	var warn []string
	for i := 0; i < o.Slots; i++ {
		in := l.MustInstance(instID(i))
		for _, d := range []struct{ what, p string }{{"prefix", in.Prefix()}, {"save dir", in.SaveDir()}, {"instance dir", in.Dir()}} {
			st, err := e.Stat(d.p)
			switch {
			case err != nil:
				warn = append(warn, fmt.Sprintf("%s %s: %s missing", in.ID, d.what, d.p))
			case st != nil && !st.IsDir():
				return res(Fail, fmt.Sprintf("%s %s %s is not a directory", in.ID, d.what, d.p), "Remove the file; the bot recreates the directory.")
			}
		}
	}
	if len(warn) > 0 {
		return res(Warn, strings.Join(warn, "; "), "Created on the first instance start (d2fleet / the wizard prefix step); only a problem if the instance already ran.")
	}
	return res(Pass, fmt.Sprintf("%d instance(s) laid out under %s", o.Slots, o.DataDir), "")
}

func checkTables(_ context.Context, e *Env, o *Options) Result {
	dir := layout(o).CacheData()
	ents, err := e.ReadDir(dir)
	if err != nil || len(ents) == 0 {
		return res(missing(o), "no extracted data tables in "+dir, "Run the wizard extract step (reads your own d2data.mpq; nothing is copied from elsewhere).")
	}
	return res(Pass, fmt.Sprintf("%d entries in %s", len(ents), dir), "")
}

// checkHelperReach: only meaningful while a game runs, so a closed port is a warning.
func checkHelperReach(_ context.Context, e *Env, o *Options) Result {
	if e.Dial == nil {
		return res(Pass, "not tested", "")
	}
	var down []string
	for i := 0; i < o.Slots; i++ {
		a := "127.0.0.1:" + strconv.Itoa(47012+i)
		if e.Dial(a) != nil {
			down = append(down, a)
		}
	}
	if len(down) > 0 {
		return res(Warn, "no helper answering on "+strings.Join(down, ", "), "Start the game instance (tools/game.sh start <id> or d2fleet); the helper starts with it. Ignore this if no game should be running.")
	}
	return res(Pass, fmt.Sprintf("%d helper(s) answer", o.Slots), "")
}

func checkTokenPerms(_ context.Context, e *Env, o *Options) Result {
	dir := filepath.Dir(layout(o).Token("x"))
	ents, err := e.ReadDir(dir)
	if err != nil {
		return res(Pass, "no token directory "+dir, "")
	}
	var loose []string
	n := 0
	for _, d := range ents {
		if d.IsDir() {
			continue
		}
		st, err := d.Info()
		if err != nil {
			continue
		}
		n++
		if st.Mode().Perm()&0o077 != 0 {
			loose = append(loose, fmt.Sprintf("%s (%04o)", d.Name(), st.Mode().Perm()))
		}
	}
	if len(loose) > 0 {
		return res(Fail, "readable by others: "+strings.Join(loose, ", "), "chmod 600 "+dir+"/*")
	}
	return res(Pass, fmt.Sprintf("%d token file(s), all 0600 or stricter", n), "")
}

// checkLoad: 1-minute load against the core count. Each running game burns about one core (docs/multi.md, V).
func checkLoad(_ context.Context, e *Env, o *Options) Result {
	b, err := e.ReadFile("/proc/loadavg")
	if err != nil {
		return res(Pass, "load not readable here", "")
	}
	f := strings.Fields(string(b))
	l1, err := strconv.ParseFloat(firstOr(f), 64)
	if err != nil || e.NumCPU < 1 {
		return res(Warn, "cannot parse /proc/loadavg", "")
	}
	free := float64(e.NumCPU) - l1
	d := fmt.Sprintf("load %.1f on %d cores (%.1f free), %d slot(s) planned", l1, e.NumCPU, free, o.Slots)
	switch {
	case l1 > 1.5*float64(e.NumCPU) && hostLoadUnreliable(e):
		// A container reads the HOST's load1 (/proc/loadavg is not namespaced), so it says nothing about our share.
		return res(Warn, d+" (host-wide)", "The host is busy and a container sees the whole host's load. Slots are admitted by d2fleet; set D2FLEET_MAX_LOAD_FACTOR (negative = off) if starts are refused.")
	case l1 > 1.5*float64(e.NumCPU):
		return res(Fail, d, "The machine is overloaded; the bot's timing degrades. Stop other jobs first.")
	case free < float64(o.Slots):
		return res(Warn, d, "Fewer free cores than planned slots; expect slow ticks. Check what else runs (top).")
	}
	return res(Pass, d, "")
}

// hostLoadUnreliable: inside a container, or with the fleet load gate switched off on purpose.
func hostLoadUnreliable(e *Env) bool {
	if f, err := strconv.ParseFloat(strings.TrimSpace(e.Getenv("D2FLEET_MAX_LOAD_FACTOR")), 64); err == nil && f < 0 {
		return true
	}
	return exists(e, "/.dockerenv") || exists(e, "/run/.containerenv")
}

func firstOr(f []string) string {
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

// checkClock: a clock far in the past breaks TLS, logs ordering and cache stamps. The data dir mtime in
// the future means the clock went backwards.
func checkClock(_ context.Context, e *Env, o *Options) Result {
	if e.Now == nil {
		return res(Pass, "not tested", "")
	}
	now := e.Now()
	if now.Year() < 2024 {
		return res(Fail, "system clock reads "+now.Format(time.RFC3339), "Set the clock (timedatectl set-ntp true).")
	}
	if st, err := e.Stat(o.DataDir); err == nil && st != nil && st.ModTime().After(now.Add(5*time.Minute)) {
		return res(Warn, "data dir modified in the future ("+st.ModTime().Format(time.RFC3339)+" vs now "+now.Format(time.RFC3339)+")", "The clock went backwards; sync it (timedatectl set-ntp true).")
	}
	return res(Pass, now.Format(time.RFC3339), "")
}

// checkCharacter prints d2.DoctorLine when a game is attached. No game attached is a skip, not a problem.
func checkCharacter(_ context.Context, e *Env, o *Options) Result {
	if e.Player == nil {
		return res(Skip, "no game attached", "")
	}
	p, err := e.Player(o.Instance)
	if err != nil {
		return res(Skip, "no game attached ("+err.Error()+")", "")
	}
	return res(Pass, d2.DoctorLine(p, 0), "")
}
