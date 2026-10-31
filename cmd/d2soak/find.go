package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// finder is the instance-aware process match, spec "find:comm=NAME;cmd=SUBSTR;env=KEY=VAL" (any subset, all
// given terms must hold, ';' separates terms). comm is the exact /proc/PID/comm, cmd a substring of the
// command line, env an exact entry of /proc/PID/environ (readable for our own processes only).
// With several instances running, a plain match: picks the lowest pid, which may be another instance's.
type finder struct{ comm, cmd, env string }

func parseFind(s string) (finder, error) {
	var f finder
	for _, t := range strings.Split(s, ";") {
		k, v, ok := strings.Cut(t, "=")
		if !ok || v == "" {
			return f, fmt.Errorf("bad find term %q", t)
		}
		switch k {
		case "comm":
			f.comm = v
		case "cmd":
			f.cmd = v
		case "env":
			f.env = v
		default:
			return f, fmt.Errorf("unknown find key %q", k)
		}
	}
	return f, nil
}

func (f finder) matches(root string, pid int, comm, cmd string) bool {
	if f.comm != "" && comm != f.comm {
		return false
	}
	if f.cmd != "" && !strings.Contains(cmd, f.cmd) {
		return false
	}
	if f.env != "" {
		b, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "environ"))
		if err != nil {
			return false
		}
		ok := false
		for _, e := range strings.Split(string(b), "\x00") {
			if e == f.env {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// findPID returns the lowest pid matching f (0 = none), skipping skip.
func findPID(root string, f finder, skip int) int {
	ents, _ := os.ReadDir(root)
	var pids []int
	for _, e := range ents {
		if pid, err := strconv.Atoi(e.Name()); err == nil && pid != skip {
			pids = append(pids, pid)
		}
	}
	sort.Ints(pids)
	for _, pid := range pids {
		d := filepath.Join(root, strconv.Itoa(pid))
		comm, _ := os.ReadFile(filepath.Join(d, "comm"))
		cmd, _ := os.ReadFile(filepath.Join(d, "cmdline"))
		if f.matches(root, pid, strings.TrimSpace(string(comm)), strings.ReplaceAll(string(cmd), "\x00", " ")) {
			return pid
		}
	}
	return 0
}

// cmdProcs is the dry run: it resolves the --proc specs once and prints name, pid, comm and command line
// (exit 1 if one is missing), so a soak start can be checked before it begins.
func cmdProcs(args []string) int {
	fs := flag.NewFlagSet("procs", flag.ContinueOnError)
	var procs multi
	fs.Var(&procs, "proc", "name=PID|self|match:SUBSTR|find:... (repeatable)")
	if fs.Parse(args) != nil {
		return 2
	}
	ps, err := parseProcSpecs(procs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return printProcs(os.Stdout, &Collector{Procs: ps}, ps)
}

func printProcs(w io.Writer, c *Collector, ps []ProcSource) int {
	got, miss := c.procs(), 0
	for _, p := range ps {
		s, ok := got[p.Name]
		if !ok {
			fmt.Fprintf(w, "%-10s MISSING  (%s)\n", p.Name, p.Spec)
			miss++
			continue
		}
		root := c.ProcRoot
		if root == "" {
			root = "/proc"
		}
		cmd, _ := os.ReadFile(filepath.Join(root, strconv.Itoa(s.PID), "cmdline"))
		line := strings.ReplaceAll(string(cmd), "\x00", " ")
		if len(line) > 90 {
			line = line[:90]
		}
		fmt.Fprintf(w, "%-10s pid %-8d %s\n", p.Name, s.PID, strings.TrimSpace(line))
	}
	if miss > 0 {
		return 1
	}
	return 0
}
