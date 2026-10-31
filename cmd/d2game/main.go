// Command d2game starts, stops and inspects game instances on the machine that runs them (the worker).
// It is the Go replacement of the bash in tools/game.sh and prints the same lines.
//
//	d2game status [id]   start <id>   stop <id>   mkprefix <id>   helper <id>
//
// Per-instance flock /tmp/d2lod-game-<id>.lock (i0: /tmp/d2lod-game.lock) is taken for everything but
// status and helper (helper takes /tmp/d2lod-helper-<id>.lock, never the game lock), waiting 60 s; NOLOCK=1 skips it (the caller already holds it, e.g. the sentinel).
// See docs/lifecycle.md and docs/multi.md.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"d2lod-bot/internal/launch"
)

const usage = "usage: game.sh status|start|stop|mkprefix|helper [id]"

// markInheritedCloexec stops fds inherited from the caller (typically the flock of a live run, held by
// `flock ... cmd` without -o) from leaking into the helper/game this process spawns: such a child would hold
// the game lock for its whole life and block every later live run (the old helper.sh bug).
func markInheritedCloexec() {
	ents, _ := os.ReadDir("/proc/self/fd")
	for _, e := range ents {
		if n, err := strconv.Atoi(e.Name()); err == nil && n > 2 {
			syscall.CloseOnExec(n)
		}
	}
}

func main() {
	markInheritedCloexec()
	os.Exit(run(os.Args[1:], launch.NewLife(os.Stdout), os.Getenv("NOLOCK") != "", os.Stderr))
}

func run(args []string, l *launch.Life, nolock bool, errw *os.File) int {
	cmd, id := "status", ""
	if len(args) > 0 {
		cmd = args[0]
	}
	if len(args) > 1 {
		id = args[1]
	}
	if id != "" && !launch.ValidID(id) {
		fmt.Fprintf(errw, "bad instance id %s\n", id)
		return 2
	}
	switch cmd {
	case "status", "start", "stop", "mkprefix", "helper":
	default:
		fmt.Fprintln(errw, usage)
		return 2
	}
	if cmd != "status" && id == "" {
		fmt.Fprintf(errw, "usage: %s %s <id>\n", os.Args[0], cmd)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	// helper only starts a process listening on a port: it is idempotent and must not wait behind a long
	// live run that holds the game lock (that run is exactly what needs its helper back). It takes a
	// short private lock only so two callers do not both spawn one.
	if cmd == "helper" {
		rel, err := launch.LockHelper(id, 30*time.Second)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 1
		}
		defer rel()
	} else if cmd != "status" && !nolock {
		rel, err := launch.LockInstance(id, 60*time.Second)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 1
		}
		defer rel()
	}
	var err error
	switch cmd {
	case "status":
		var ids []string
		if id != "" {
			ids = []string{id}
		}
		err = l.Status(ctx, ids...)
	case "start":
		err = l.Start(ctx, id)
	case "stop":
		err = l.Stop(ctx, id)
	case "mkprefix":
		err = l.MkPrefix(ctx, id)
	case "helper":
		err = l.EnsureHelper(ctx, id)
	}
	var ue launch.UsageError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ue):
		fmt.Fprintln(errw, err)
		return 2
	default:
		fmt.Fprintln(errw, err)
		return 1
	}
}
