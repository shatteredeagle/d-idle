package launch

import (
	"log/slog"

	denv "d2lod-bot/internal/env"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Reaping of orphaned winedevice.exe (docs/leakhunt.md WINELEAK: every wineserver restart of a prefix leaves a
// ~35 MB winedevice.exe behind that `wineserver -k` does not kill). Go port of tools/winereap.sh with the same
// rules: a process is killed only if its comm is winedevice.exe, it has our uid, its environ WINEPREFIX equals
// the exact prefix of the instance just stopped, and no wineserver* of that prefix runs (checked again before
// every kill). Never another prefix, never ~/.wine. Linux only; bounded in time.

// ReapOpts are the injectable parts; the zero value is the real system.
type ReapOpts struct {
	ProcRoot string              // default /proc
	UID      int                 // 0 = os.Getuid() (which is also 0 for root)
	Kill     func(pid int) error // default SIGTERM, like the script's `kill`
	Kill9    func(pid int) error // default SIGKILL: the last step for what ignored SIGTERM
	// ServerKill is `wineserver -k` of the prefix (CleanPrefix only); GameRunning, when set, vetoes CleanPrefix
	// while a Game.exe of the instance runs (nil: the caller has checked).
	ServerKill  func() error
	GameRunning func() bool
	TermWait    time.Duration // how long SIGTERMed orphans get before SIGKILL; default 3 s
	Log         func(string)
	// ServerWait is how long to wait for the prefix's own wineserver to exit by itself (it lingers a few
	// seconds after its last client) before giving up; default 10 s. Poll is the interval, default 200 ms.
	ServerWait, Poll time.Duration
	Sleep            func(time.Duration)
	Now              func() time.Time
	GOOS             string // default runtime.GOOS
}

func (o ReapOpts) norm() ReapOpts {
	if o.ProcRoot == "" {
		o.ProcRoot = "/proc"
	}
	if o.UID == 0 {
		o.UID = os.Getuid()
	}
	if o.Kill == nil {
		o.Kill = func(pid int) error { return syscall.Kill(pid, syscall.SIGTERM) }
	}
	if o.Kill9 == nil {
		o.Kill9 = func(pid int) error { return syscall.Kill(pid, syscall.SIGKILL) }
	}
	if o.TermWait == 0 {
		o.TermWait = 3 * time.Second
	}
	if o.Log == nil {
		o.Log = func(string) {}
	}
	if o.ServerWait == 0 {
		o.ServerWait = 10 * time.Second
	}
	if o.Poll == 0 {
		o.Poll = 200 * time.Millisecond
	}
	if o.Sleep == nil {
		o.Sleep = time.Sleep
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.GOOS == "" {
		o.GOOS = runtime.GOOS
	}
	return o
}

// ReapWinedevices kills the orphaned winedevice.exe of the given Wine prefix (the .../pfx directory) and returns
// the pids it signalled. It returns nil without touching anything on non-Linux, for an empty prefix, or when a
// wineserver of the prefix is still running after ServerWait.
func ReapWinedevices(prefix string, o ReapOpts) []int {
	o = o.norm()
	prefix = cleanPrefix(prefix)
	if o.GOOS != "linux" || prefix == "" {
		return nil
	}
	deadline := o.Now().Add(o.ServerWait)
	for {
		sp := serverOf(o.ProcRoot, prefix)
		if sp == 0 {
			break
		}
		if !o.Now().Before(deadline) {
			// A lingering wineserver is kept alive by exactly the orphans we are here to reap (G1 attempt 2: it
			// outlived its game by 41 min). When nothing but Wine's own system processes is left in the prefix
			// the server serves no client that matters, so they are the target. Any other client (d2input, a
			// game) means the server is in use: leave everything alone.
			if _, orphans, others := scanPrefix(o, prefix); len(others) == 0 && len(orphans) > 0 {
				o.Log("reap: wineserver " + strconv.Itoa(sp) + " of " + prefix + " is only kept alive by orphans")
				return killOrphans(prefix, o)
			}
			o.Log("reap: wineserver " + strconv.Itoa(sp) + " of " + prefix + " still running: nothing reaped")
			return nil
		}
		o.Sleep(o.Poll)
	}
	ents, err := os.ReadDir(o.ProcRoot)
	if err != nil {
		return nil
	}
	end := o.Now().Add(o.ServerWait + 5*time.Second) // bound for the scan and kills themselves
	var killed []int
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		if o.Now().After(end) {
			o.Log("reap: time budget used up")
			break
		}
		if procComm(o.ProcRoot, pid) != "winedevice.exe" || procUID(o.ProcRoot, pid) != o.UID ||
			procPrefix(o.ProcRoot, pid) != prefix {
			continue
		}
		if sp := serverOf(o.ProcRoot, prefix); sp != 0 {
			o.Log("reap: wineserver " + strconv.Itoa(sp) + " appeared: stopping")
			break
		}
		if o.Kill(pid) == nil {
			killed = append(killed, pid)
		}
	}
	if len(killed) > 0 {
		o.Log("reap: killed orphaned winedevice.exe pids " + joinPIDs(killed) + " of " + prefix)
	}
	return killed
}

// orphanComms are the Wine system processes that survive their clients and keep the wineserver alive.
var orphanComms = map[string]bool{
	"winedevice.exe": true, "explorer.exe": true, "services.exe": true, "rpcss.exe": true,
	"plugplay.exe": true, "svchost.exe": true,
}

// scanPrefix lists the processes of ours whose WINEPREFIX is exactly prefix: wineservers, orphan-set processes and
// every other client (those mean the prefix is in use).
func scanPrefix(o ReapOpts, prefix string) (servers, orphans, others []int) {
	ents, err := os.ReadDir(o.ProcRoot)
	if err != nil {
		return
	}
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 || procUID(o.ProcRoot, pid) != o.UID || procPrefix(o.ProcRoot, pid) != prefix {
			continue
		}
		switch c := procComm(o.ProcRoot, pid); {
		case strings.HasPrefix(c, "wineserver"):
			servers = append(servers, pid)
		case orphanComms[c]:
			orphans = append(orphans, pid)
		case c != "":
			others = append(others, pid)
		}
	}
	return
}

// killOrphans SIGTERMs the orphan-set processes of the prefix, waits TermWait, then SIGKILLs what is left. The
// match (comm, uid, exact prefix) is repeated right before each signal against pid reuse.
func killOrphans(prefix string, o ReapOpts) []int {
	_, orphans, _ := scanPrefix(o, prefix)
	var killed []int
	for _, pid := range orphans {
		if o.Kill(pid) == nil {
			killed = append(killed, pid)
		}
	}
	deadline := o.Now().Add(o.TermWait)
	for {
		_, left, _ := scanPrefix(o, prefix)
		if len(left) == 0 {
			break
		}
		if !o.Now().Before(deadline) {
			for _, pid := range left {
				o.Kill9(pid)
			}
			o.Log("reap: SIGKILL for orphans that ignored SIGTERM: " + joinPIDs(left))
			break
		}
		o.Sleep(o.Poll)
	}
	if len(killed) > 0 {
		o.Log("reap: killed orphaned Wine processes " + joinPIDs(killed) + " of " + prefix)
	}
	return killed
}

// CleanPrefix makes the prefix of an instance whose game is NOT running pristine before a launch: `wineserver -k`,
// wait up to ServerWait for the server to exit, then SIGTERM and SIGKILL of what is left of winedevice, explorer,
// services, rpcss (G1 attempt 2: a new Game.exe launched into a stale wineserver hung). It also ends the instance's
// d2input helper, so ensure the helper afterwards. Returns the killed pids. Refuses (nil) on non-Linux, an empty
// prefix or while GameRunning reports a game.
func CleanPrefix(prefix string, o ReapOpts) []int {
	o = o.norm()
	prefix = cleanPrefix(prefix)
	if o.GOOS != "linux" || prefix == "" {
		return nil
	}
	if o.GameRunning != nil && o.GameRunning() {
		o.Log("clean: a game of this instance is running: prefix left alone")
		return nil
	}
	if o.ServerKill != nil {
		if err := o.ServerKill(); err != nil {
			o.Log("clean: " + err.Error())
		}
	}
	waitServers := func() int {
		deadline := o.Now().Add(o.ServerWait)
		for {
			sp := serverOf(o.ProcRoot, prefix)
			if sp == 0 || !o.Now().Before(deadline) {
				return sp
			}
			o.Sleep(o.Poll)
		}
	}
	waitServers()
	killed := killOrphans(prefix, o)
	if sp := waitServers(); sp != 0 {
		o.Log("clean: wineserver " + strconv.Itoa(sp) + " of " + prefix + " still running")
	}
	return killed
}

func cleanPrefix(p string) string {
	if strings.TrimSpace(p) == "" {
		return ""
	}
	return filepath.Clean(p)
}

func procComm(root string, pid int) string {
	b, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "comm"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// procUID is the real uid from /proc/pid/status, -1 if unreadable.
func procUID(root string, pid int) int {
	b, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "status"))
	if err != nil {
		return -1
	}
	for _, l := range strings.Split(string(b), "\n") {
		if f, ok := strings.CutPrefix(l, "Uid:"); ok {
			if w := strings.Fields(f); len(w) > 0 {
				if n, err := strconv.Atoi(w[0]); err == nil {
					return n
				}
			}
		}
	}
	return -1
}

// procPrefix is the cleaned WINEPREFIX of /proc/pid/environ, "" if unreadable or unset.
func procPrefix(root string, pid int) string {
	b, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "environ"))
	if err != nil {
		return ""
	}
	for _, kv := range strings.Split(string(b), "\x00") {
		if v, ok := strings.CutPrefix(kv, "WINEPREFIX="); ok {
			return cleanPrefix(v)
		}
	}
	return ""
}

// serverOf returns the pid of a wineserver* process of the prefix, 0 if none.
func serverOf(root, prefix string) int {
	ents, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		if strings.HasPrefix(procComm(root, pid), "wineserver") && procPrefix(root, pid) == prefix {
			return pid
		}
	}
	return 0
}

// reapAfterStop is the hook of the stop paths (Game.Shutdown, Life.stopGame): once the managed game is gone, reap
// the leftovers of its prefix. Off with D2BOT_REAP=0 and for i0 or an unset instance (the shared original prefix;
// tools/winereap.sh refuses i0 too). Without a Log it logs through slog.
func reapAfterStop(cfg Config, o ReapOpts) {
	if !denv.ReapEnabled() || cfg.Instance == "" || cfg.Instance == "i0" || cfg.PrefixDir == "" {
		return
	}
	if o.Log == nil {
		o.Log = func(s string) { slog.Info(s) }
	}
	ReapWinedevices(filepath.Join(cfg.PrefixDir, "pfx"), o)
}
