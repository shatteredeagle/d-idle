package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/savesafe"
)

// Save safety (docs/ops/save-backups.md): a startup repair of the character file, a rolling backup while the bot runs
// and a disk-space guard. Always on for a managed d2d; nothing here writes game state.

// d2dSaveFolder is where a managed d2d keeps the character files under its home (the path classForStart reads).
func d2dSaveFolder(home string) string { return filepath.Join(home, "config", "d2d", "save") }

const (
	defaultSaveFolder = "/data/instances/main/d2d/config/d2d/save"
	defaultStateDir   = "/data/instances/main/state"
	saveGuardPoll     = 5 * time.Second
)

// saveNoticeState is the state folder the standing notices are read from (set once by the wiring).
var saveNoticeState atomic.Pointer[string]

// saveNotices are the standing save and disk lines for `ops status --why` and the viewer.
func saveNotices() []string {
	p := saveNoticeState.Load()
	if p == nil || *p == "" {
		return nil
	}
	return savesafe.Notices(*p, savesafe.FreeBytes, time.Now())
}

// repairSaveAtStart runs before d2d is started: a missing, empty or corrupt character file comes back from the newest
// valid backup. It must run before classForStart, which reads an empty file as "no save" and lets d2d create a new hero.
func repairSaveAtStart(home, state, name string, log *slog.Logger) {
	if home == "" || state == "" || name == "" {
		return
	}
	saveNoticeState.Store(&state)
	rep, ok, err := savesafe.Repair(d2dSaveFolder(home), state, name, savesafe.FreeBytes, time.Now())
	switch {
	case err != nil:
		log.Error("save repair: NOT repaired", "char", name, "err", err)
	case ok:
		log.Error(rep.String(), "char", name, "broken", rep.Broken, "note", "progress since that backup is lost")
	}
}

// startSaveGuard runs the rolling backup until ctx ends. The clock is the wall clock: a backup every ten real minutes
// whatever the game clock does.
func startSaveGuard(ctx context.Context, home, state, name string, snap func() *d2.Snapshot, log *slog.Logger) {
	if home == "" || state == "" || name == "" {
		return
	}
	saveNoticeState.Store(&state)
	dir := d2dSaveFolder(home)
	g := &savesafe.Guard{State: state, SaveDir: func() string { return dir }, Name: func() string { return name }, Now: time.Now,
		Free: savesafe.FreeBytes, Logf: func(msg string, kv ...any) { log.Warn(msg, kv...) }}
	go goSafe(ctx, log, "saveguard", func() {
		t := time.NewTicker(saveGuardPoll)
		defer t.Stop()
		for {
			s := snap()
			g.Observe(s != nil && s.Player != nil && s.Player.InTown())
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	})
}

// saveDiskGuard is bot.Deps.DiskLow: true while the state folder or the save folder has under 200 MB free. Each change
// is logged once.
func saveDiskGuard(home, state string, log *slog.Logger) func() bool {
	dir := d2dSaveFolder(home)
	var was atomic.Bool
	return func() bool {
		low := savesafe.Low(savesafe.FreeBytes, state, dir)
		if was.Swap(low) != low {
			if low {
				log.Error("disk guard: under 200 MB free, planned save points are off until space is freed", "state", state)
			} else {
				log.Warn("disk guard: space is back, planned save points resume")
			}
		}
		return low
	}
}

// opsSave is `d2bot ops save check`: the validity of the live character file and the list of its backups. It reads
// files only, needs no socket and no running bot. Exit 0: every live file valid, 1: one is not, 2: usage.
func opsSave(args []string, out io.Writer) (int, error) {
	if len(args) == 0 || args[0] != "check" {
		return 2, errors.New(opsUsage)
	}
	fs := flag.NewFlagSet("ops save check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	char := fs.String("char", "", "character name (default: every character with a live file or a backup)")
	save := fs.String("save", defaultSaveFolder, "the d2d save folder")
	state := fs.String("state", defaultStateDir, "the instance state folder")
	if err := fs.Parse(args[1:]); err != nil || fs.NArg() != 0 {
		return 2, fmt.Errorf("%v\n%s", err, opsUsage)
	}
	names := []string{*char}
	if *char == "" {
		names = saveNames(*save, filepath.Join(*state, savesafe.Dir))
	}
	now := time.Now()
	code := 0
	for _, n := range names {
		r, err := savesafe.Check(*save, *state, n, savesafe.FreeBytes, now)
		if err != nil {
			return 1, err
		}
		verdict := "valid"
		if r.LiveErr != nil {
			verdict, code = "INVALID: "+r.LiveErr.Error(), 1
		}
		fmt.Fprintf(out, "live\t%s\t%d bytes\t%s\n", r.Live, r.LiveLen, verdict)
		for _, b := range r.Backups {
			v := "valid"
			if b.Err != nil {
				v = "INVALID: " + b.Err.Error()
			}
			fmt.Fprintf(out, "backup\t%s\t%d bytes\t%s\t%s\n", b.Path, b.Size, time.Unix(b.Unix, 0).UTC().Format(time.RFC3339), v)
		}
	}
	for _, n := range savesafe.Notices(*state, savesafe.FreeBytes, now) {
		fmt.Fprintf(out, "notice\t%s\n", n)
	}
	fmt.Fprintf(out, "save check: %d character(s), nothing was changed\n", len(names))
	return code, nil
}

// saveNames lists the characters with a live .d2s or a backup.
func saveNames(saveDir, backupDir string) []string {
	seen := map[string]bool{}
	for _, d := range []string{saveDir, backupDir} {
		ents, _ := os.ReadDir(d)
		for _, e := range ents {
			if n := e.Name(); strings.HasSuffix(n, ".d2s") && !strings.HasPrefix(n, ".") {
				seen[n[:strings.Index(n, ".")]] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
