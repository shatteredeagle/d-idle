package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Exit/crash log. The helper is a GUI-subsystem exe, so stdout/stderr go nowhere and a Go panic or a
// "fatal error" used to vanish without a trace. Everything now goes to a file in the prefix drive_c
// (host path: <prefix>/pfx/drive_c/d2input-<port>.log), including the runtime's crash output.
var (
	logMu   sync.Mutex
	logFile *os.File
	started = time.Now()

	nConns, nCmds, nErrs, nPanics atomic.Int64
)

func logPath(addr string) string {
	port := addr[strings.LastIndex(addr, ":")+1:]
	return filepath.Join(logDir(), "d2input-"+port+".log")
}

// initLog opens the log (append: exits of earlier incarnations stay visible) and routes fatal runtime errors to it.
func initLog(addr string) {
	f, err := os.OpenFile(logPath(addr), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	if st, err := f.Stat(); err == nil && st.Size() > 1<<20 {
		f.Truncate(0) // keep it small; a restart loop must not fill the disk
	}
	logFile = f
	debug.SetCrashOutput(f, debug.CrashOptions{})
	logf("start pid=%d addr=%s go=%s args=%v gomaxprocs=%d", os.Getpid(), addr, runtime.Version(), os.Args[1:], runtime.GOMAXPROCS(0))
}

func logf(f string, a ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile == nil {
		return
	}
	fmt.Fprintf(logFile, "%s %s\n", time.Now().Format("2006-01-02 15:04:05.000"), fmt.Sprintf(f, a...))
	logFile.Sync()
}

// exitf logs why the process ends, then exits. Every exit path of the helper goes through here.
func exitf(code int, f string, a ...any) {
	logf("EXIT code=%d up=%s: %s", code, time.Since(started).Round(time.Second), fmt.Sprintf(f, a...))
	os.Exit(code)
}

// watchSignals logs any signal or console control event (Wine delivers CTRL_CLOSE/SHUTDOWN, SIGTERM from
// "kill", wineserver shutdown) and exits like the default would. SIGHUP-style events must not be ignored
// silently: knowing who sent them is the point.
func watchSignals() {
	ch := make(chan os.Signal, 4)
	signal.Notify(ch)
	go func() {
		for s := range ch {
			logf("signal %v", s)
			if s == os.Interrupt || s.String() == "terminated" || s.String() == "hangup" {
				exitf(1, "signal %v", s)
			}
		}
	}()
}

// heartbeat logs resource use every minute, so growth (handles, goroutines, heap) shows before a death.
func heartbeat(s sys) {
	var last int64
	for range time.Tick(time.Minute) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		c := nCmds.Load()
		logf("hb up=%s conns=%d goroutines=%d threads=%d heap=%dK sys=%dK handles=%d cmds=%d (+%d) errs=%d panics=%d windows=%d",
			time.Since(started).Round(time.Second), nConns.Load(), runtime.NumGoroutine(), threadsCreated(), m.HeapAlloc>>10, m.Sys>>10,
			procHandles(), c, c-last, nErrs.Load(), nPanics.Load(), len(s.list()))
		last = c
	}
}

// safeHandle runs one command line; a panic becomes an "err internal" reply and a log entry
// instead of killing the process (an unrecovered panic in any goroutine ends the whole helper).
func (v *server) safeHandle(line string) (reply string) {
	defer func() {
		if r := recover(); r != nil {
			nPanics.Add(1)
			logf("PANIC handling %q: %v\n%s", line, r, debug.Stack())
			reply = "err internal " + fmt.Sprint(r)
		}
	}()
	return v.handle(line)
}

// threadsCreated is how many OS threads the Go runtime has started so far (it never ends idle ones), the
// number that tracks the host fd count under Wine. Logged in the heartbeat as threads=.
func threadsCreated() int {
	if p := pprof.Lookup("threadcreate"); p != nil {
		return p.Count()
	}
	return -1
}
