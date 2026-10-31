// d2ddet compares the d2dctl journals of two same-seed bot runs and reports the first divergence (docs/d2d-determinism.md).
//
//	d2ddet compare A.jnl B.jnl [-context N] [-ignore verb,verb]
//	d2ddet stats A.jnl                       # ticks, commands, checkpoints, verbs by count
//
// A journal is written by the bot when D2DCTL_JOURNAL=path is set (internal/d2dctl/journal.go). The first differing
// record tells which side moved first: a different COMMAND at equal state hashes is the bot (it decided differently on
// the same world), a different REPLY or CHECKPOINT hash under identical commands is the game (the world changed between
// two identical command streams: a wall-time effect inside d2d, or an earlier bot-side effect that only reads differ on).
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type rec struct {
	kind string // ">" command, "C" checkpoint
	tick int
	cmd  string // command line (">") or "" ("C")
	val  string // reply digest / checkpoint hash
	line int    // 1-based line in the file
}

func (r rec) String() string {
	if r.kind == "C" {
		return fmt.Sprintf("line %d tick %d checkpoint %s", r.line, r.tick, r.val)
	}
	return fmt.Sprintf("line %d tick %d %q -> %s", r.line, r.tick, r.cmd, r.val)
}

func load(path string, ignore map[string]bool) ([]rec, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []rec
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n := 0
	for sc.Scan() {
		n++
		l := sc.Text()
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		p := strings.Split(l, "\t")
		if len(p) < 3 {
			return nil, fmt.Errorf("%s:%d: bad record %q", path, n, l)
		}
		t, _ := strconv.Atoi(p[1])
		r := rec{kind: p[0], tick: t, line: n}
		switch p[0] {
		case ">":
			if len(p) < 4 {
				return nil, fmt.Errorf("%s:%d: bad command record %q", path, n, l)
			}
			r.cmd, r.val = p[2], p[3]
			if ignore[verb(r.cmd)] {
				continue
			}
		case "C":
			r.val = p[2]
		default:
			return nil, fmt.Errorf("%s:%d: unknown record kind %q", path, n, p[0])
		}
		out = append(out, r)
	}
	return out, sc.Err()
}

func verb(cmd string) string {
	f := strings.Fields(cmd)
	if len(f) == 0 {
		return ""
	}
	if (f[0] == "debug" || f[0] == "cmd") && len(f) > 1 {
		return f[0] + " " + f[1]
	}
	return f[0]
}

// classify explains the first differing pair.
func classify(a, b rec) string {
	switch {
	case a.kind != b.kind:
		return "RECORD KIND differs (a checkpoint against a command): the runs stepped a different number of ticks before this point, usually a different `step n` batch earlier"
	case a.kind == "C":
		return "STATE HASH differs under identical command streams: the game world diverged with the same inputs (d2d-side wall-time effect, e.g. the neighbour-link worker, or a reply that the bot never read)"
	case a.cmd != b.cmd && verb(a.cmd) == "step" && verb(b.cmd) == "step":
		return "STEP BATCH differs (same place, different tick count): the bot's sleep lengths depend on wall time (a wall-clock deadline, time.Since, or a wall sleep converted to ticks)"
	case a.cmd != b.cmd:
		return "COMMAND differs with identical state so far: BOT-SIDE (the bot chose differently on the same observation: map iteration order, goroutine timing, wall-clock reads, unseeded rand)"
	case strings.HasPrefix(a.val, "err:") != strings.HasPrefix(b.val, "err:"):
		return "one run got an err reply where the other succeeded: the game refused the same command (state differs in something the journal did not hash before this point)"
	default:
		return "REPLY differs for the same command: the observation changed before the bot's next decision (the state diverged earlier than the last checkpoint could show; narrow with a smaller D2DCTL_JOURNAL_EVERY)"
	}
}

func compare(pa, pb string, nctx int, ignore map[string]bool) int {
	a, err := load(pa, ignore)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	b, err := load(pb, ignore)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	n := min(len(a), len(b))
	cps := 0
	lastCP := -1
	for i := 0; i < n; i++ {
		if a[i] == b[i] || a[i].kind == b[i].kind && a[i].tick == b[i].tick && a[i].cmd == b[i].cmd && a[i].val == b[i].val {
			if a[i].kind == "C" {
				cps++
				lastCP = a[i].tick
			}
			continue
		}
		fmt.Printf("DIVERGE at record %d (identical prefix: %d records, %d equal checkpoints, last equal checkpoint tick %d)\n", i, i, cps, lastCP)
		fmt.Printf("  A: %s\n  B: %s\n", a[i], b[i])
		fmt.Printf("  tick A %d, tick B %d\n", a[i].tick, b[i].tick)
		fmt.Printf("  %s\n", classify(a[i], b[i]))
		lo := max(0, i-nctx)
		fmt.Printf("context A (%s):\n", pa)
		for j := lo; j < min(len(a), i+nctx+1); j++ {
			mark(j == i, a[j])
		}
		fmt.Printf("context B (%s):\n", pb)
		for j := lo; j < min(len(b), i+nctx+1); j++ {
			mark(j == i, b[j])
		}
		return 1
	}
	if len(a) != len(b) {
		fmt.Printf("IDENTICAL for %d records (%d checkpoints, last at tick %d); the longer run (%s) has %d more records after that\n",
			n, cps, lastCP, longer(pa, pb, len(a), len(b)), max(len(a), len(b))-n)
		return 0
	}
	fmt.Printf("IDENTICAL: %d records, %d checkpoints, last checkpoint tick %d\n", n, cps, lastCP)
	return 0
}

func longer(pa, pb string, la, lb int) string {
	if la > lb {
		return pa
	}
	return pb
}

func mark(hit bool, r rec) {
	m := "   "
	if hit {
		m = ">> "
	}
	fmt.Println(m + r.String())
}

func stats(path string) int {
	rs, err := load(path, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	byVerb := map[string]int{}
	cps, last, errs := 0, 0, 0
	for _, r := range rs {
		if r.kind == "C" {
			cps++
			continue
		}
		byVerb[verb(r.cmd)]++
		if strings.HasPrefix(r.val, "err:") || strings.HasPrefix(r.val, "fail:") {
			errs++
		}
		last = r.tick
	}
	fmt.Printf("%s: %d records, last tick %d, %d checkpoints, %d err replies\n", path, len(rs), last, cps, errs)
	keys := make([]string, 0, len(byVerb))
	for k := range byVerb {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return byVerb[keys[i]] > byVerb[keys[j]] })
	for _, k := range keys {
		fmt.Printf("  %6d  %s\n", byVerb[k], k)
	}
	return 0
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: d2ddet compare A.jnl B.jnl [-context N] [-ignore verb,verb] | d2ddet stats A.jnl")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "compare":
		fs := flag.NewFlagSet("compare", flag.ExitOnError)
		nctx := fs.Int("context", 6, "records of context around the divergence")
		ign := fs.String("ignore", "", "comma separated verbs to leave out of the comparison (e.g. 'debug portals')")
		var files []string
		args := os.Args[2:]
		for len(args) > 0 && !strings.HasPrefix(args[0], "-") {
			files = append(files, args[0])
			args = args[1:]
		}
		fs.Parse(args)
		files = append(files, fs.Args()...)
		if len(files) != 2 {
			fmt.Fprintln(os.Stderr, "compare needs two journals")
			os.Exit(2)
		}
		ignore := map[string]bool{}
		for _, v := range strings.Split(*ign, ",") {
			if v = strings.TrimSpace(v); v != "" {
				ignore[v] = true
			}
		}
		os.Exit(compare(files[0], files[1], *nctx, ignore))
	case "stats":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "stats needs one journal")
			os.Exit(2)
		}
		os.Exit(stats(os.Args[2]))
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}
