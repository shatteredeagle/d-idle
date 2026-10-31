package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sort"
	"time"
)

// exitHeap is the exit code of the heap guard: the supervisor restarts it like any panic storm (supervise.go).
const exitHeap = exitStorm

const (
	heapGuardEvery = 2 * time.Second // a runaway allocates about 2 GB/s (V: soak sample 04:42:55, 27.8 GB in under 30 s)
	heapGuardKeep  = 3               // profile sets kept in the state dir
	softLimitPct   = 125             // GC soft memory limit, percent of the guard limit
)

// heapGuard is the defence in depth against an unbounded allocation anywhere in the bot: past limitMB of heap in use
// it logs, writes a heap profile and a goroutine dump to dir and exits with exitHeap, long before the kernel's OOM
// killer would end the process (which leaves no evidence and takes the tens of GB with it).
type heapGuard struct {
	log     *slog.Logger
	limitMB func() int // knob bot.heapLimitMB, read every check; 0 = off
	dir     string
	every   time.Duration
	heap    func() uint64 // heap in use, bytes
	now     func() time.Time
	sleep   func(ctx context.Context, d time.Duration)
	exit    func(code int)
	soft    func(bytes int64) // debug.SetMemoryLimit
	softMB  int
}

func newHeapGuard(log *slog.Logger, limitMB func() int, dir string) *heapGuard {
	return &heapGuard{log: log, limitMB: limitMB, dir: dir, every: heapGuardEvery, now: time.Now, exit: os.Exit,
		heap: func() uint64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return m.HeapInuse
		},
		sleep: func(ctx context.Context, d time.Duration) {
			select {
			case <-ctx.Done():
			case <-time.After(d):
			}
		},
		soft: func(b int64) { debug.SetMemoryLimit(b) },
	}
}

// check reports whether the guard fired.
func (g *heapGuard) check() bool {
	limit := g.limitMB()
	if limit <= 0 {
		return false
	}
	if _, set := os.LookupEnv("GOMEMLIMIT"); !set && g.softMB != limit && g.soft != nil {
		g.softMB = limit
		g.soft(int64(limit) * softLimitPct / 100 << 20)
	}
	used := g.heap()
	if used>>20 <= uint64(limit) {
		return false
	}
	g.log.Error("watchdog: heap "+fmt.Sprint(used>>20)+" MB > limit: writing heap profile and exiting for a restart",
		"heapMB", used>>20, "limitMB", limit, "dir", g.dir)
	if err := g.writeProfiles(); err != nil {
		g.log.Error("watchdog: heap profile not written", "err", err)
	}
	g.exit(exitHeap)
	return true
}

// writeProfiles writes heap-<time>.pprof and goroutines-<time>.txt and keeps the newest heapGuardKeep of each.
func (g *heapGuard) writeProfiles() error {
	if err := os.MkdirAll(g.dir, 0o755); err != nil {
		return err
	}
	stamp := g.now().Format("20060102-150405")
	var first error
	for _, p := range []struct{ prefix, ext, name string }{{"heap", ".pprof", "heap"}, {"goroutines", ".txt", "goroutine"}} {
		path := filepath.Join(g.dir, p.prefix+"-"+stamp+p.ext)
		f, err := os.Create(path)
		if err != nil {
			first = firstErr(first, err)
			continue
		}
		debugLevel := 0
		if p.name == "goroutine" {
			debugLevel = 2
		}
		err = pprof.Lookup(p.name).WriteTo(f, debugLevel)
		first = firstErr(first, err, f.Close())
		prune(g.dir, p.prefix+"-", p.ext, heapGuardKeep)
	}
	return first
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// prune removes all but the newest keep files named prefix*ext (the time stamp sorts by name).
func prune(dir, prefix, ext string, keep int) {
	names, _ := filepath.Glob(filepath.Join(dir, prefix+"*"+ext))
	sort.Strings(names)
	for len(names) > keep {
		_ = os.Remove(names[0])
		names = names[1:]
	}
}

func (g *heapGuard) run(ctx context.Context) {
	for ctx.Err() == nil {
		if g.check() {
			return
		}
		g.sleep(ctx, g.every)
	}
}
