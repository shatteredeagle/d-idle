package metrics

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Proc is one process as /proc reports it. Sizes are kB. PSS is -1 when smaps_rollup is unreadable
// (other users' processes, old kernels). Tags: VERIFIED against the /proc(5) layout of Linux 7.0; the Wine
// process names are UNVERIFIED here (see FindPIDs).
type Proc struct {
	PID     int
	Comm    string
	State   string // R S D Z T ...
	RSSKB   int64
	PSSKB   int64
	FDs     int // -1 when /proc/PID/fd is unreadable
	Threads int
	UTicks  uint64 // user and system CPU in clock ticks (usually 100/s), as in /proc/PID/stat
	STicks  uint64
	Start   uint64 // start time in ticks since boot: (PID, Start) identifies a process across PID reuse
}

// CPUTicks is user plus system time.
func (p Proc) CPUTicks() uint64 { return p.UTicks + p.STicks }

var ErrNoProc = errors.New("metrics: no such process")

// ReadProc reads /proc/pid.
func ReadProc(pid int) (Proc, error) { return ReadProcAt("/proc", pid) }

// ReadProcAt reads root/pid, so tests can point at a fake tree.
func ReadProcAt(root string, pid int) (Proc, error) {
	dir := filepath.Join(root, strconv.Itoa(pid))
	statB, err := os.ReadFile(filepath.Join(dir, "stat"))
	if err != nil {
		if os.IsNotExist(err) {
			return Proc{}, ErrNoProc
		}
		return Proc{}, err
	}
	p := Proc{PID: pid, PSSKB: -1, FDs: -1}
	// comm may contain spaces and parentheses: split at the last ')'.
	s := string(statB)
	lo, hi := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
	if lo < 0 || hi < lo {
		return p, errors.New("metrics: bad stat for pid " + strconv.Itoa(pid))
	}
	p.Comm = s[lo+1 : hi]
	f := strings.Fields(s[hi+1:]) // f[0] is field 3 (state)
	if len(f) < 20 {
		return p, errors.New("metrics: short stat for pid " + strconv.Itoa(pid))
	}
	p.State = f[0]
	p.UTicks, _ = strconv.ParseUint(f[11], 10, 64) // field 14
	p.STicks, _ = strconv.ParseUint(f[12], 10, 64) // field 15
	p.Threads, _ = strconv.Atoi(f[17])             // field 20
	p.Start, _ = strconv.ParseUint(f[19], 10, 64)  // field 22

	if b, err := os.ReadFile(filepath.Join(dir, "status")); err == nil {
		p.RSSKB = kbField(string(b), "VmRSS:")
	}
	if b, err := os.ReadFile(filepath.Join(dir, "smaps_rollup")); err == nil {
		if v := kbField(string(b), "Pss:"); v >= 0 {
			p.PSSKB = v
		}
	}
	if ents, err := os.ReadDir(filepath.Join(dir, "fd")); err == nil {
		p.FDs = len(ents)
	}
	return p, nil
}

// kbField returns the number after key in a /proc "Key:   123 kB" file, or 0 when absent (kernel threads and
// zombies have no VmRSS) so callers can sum safely.
func kbField(s, key string) int64 {
	for _, line := range strings.Split(s, "\n") {
		if rest, ok := strings.CutPrefix(line, key); ok {
			f := strings.Fields(rest)
			if len(f) > 0 {
				v, _ := strconv.ParseInt(f[0], 10, 64)
				return v
			}
		}
	}
	return 0
}

// FindPIDs returns the pids under root whose comm or command line satisfy match, ascending. Wine shows
// Game.exe as comm "Game.exe" (15 char limit) and the full path in cmdline; match on cmdline for safety.
func FindPIDs(root string, match func(comm, cmdline string) bool) []int {
	ents, _ := os.ReadDir(root)
	var out []int
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		comm, _ := os.ReadFile(filepath.Join(root, e.Name(), "comm"))
		cmd, _ := os.ReadFile(filepath.Join(root, e.Name(), "cmdline"))
		if match(strings.TrimSpace(string(comm)), strings.ReplaceAll(string(cmd), "\x00", " ")) {
			out = append(out, pid)
		}
	}
	return out
}

// Self reads this process.
func Self() (Proc, error) { return ReadProc(os.Getpid()) }

// RegisterRuntime adds Go runtime and own-process gauges: go_goroutines, go_threads, go_memstats_*,
// process_resident_memory_bytes, process_open_fds, process_cpu_seconds_total (a gauge here; the registry has
// no scrape-time counters).
func (r *Registry) RegisterRuntime() {
	r.AddCollector(func(c *Collect) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		n, _ := runtime.ThreadCreateProfile(nil)
		c.Gauge("go_goroutines", "Number of goroutines.", float64(runtime.NumGoroutine()))
		c.Gauge("go_threads", "Number of OS threads created.", float64(n))
		c.Gauge("go_memstats_heap_inuse_bytes", "Heap bytes in use.", float64(m.HeapInuse))
		c.Gauge("go_memstats_heap_objects", "Allocated heap objects.", float64(m.HeapObjects))
		c.Gauge("go_memstats_sys_bytes", "Bytes obtained from the system.", float64(m.Sys))
		c.Gauge("go_gc_cycles", "Completed GC cycles.", float64(m.NumGC))
		if p, err := Self(); err == nil {
			c.Gauge("process_resident_memory_bytes", "Resident set size.", float64(p.RSSKB)*1024)
			c.Gauge("process_open_fds", "Open file descriptors.", float64(p.FDs))
			c.Gauge("process_cpu_seconds_total", "User and system CPU time.", float64(p.CPUTicks())/clkTck)
		}
	})
}

// clkTck is USER_HZ; 100 on every Linux we run on (UNVERIFIED for exotic kernels, but it is a fixed ABI value).
const clkTck = 100

// RegisterProcs exports per-process gauges d2_proc_{rss_bytes,pss_bytes,open_fds,threads,cpu_seconds,up}
// labelled proc=<name>, for each name whose pid function currently returns a live pid (0: not running).
// Use it for the game, wineserver, helper, Xvfb and bot.
func (r *Registry) RegisterProcs(pids map[string]func() int) {
	r.AddCollector(func(c *Collect) {
		for name, fn := range pids {
			pid := fn()
			p, err := ReadProc(pid)
			if pid <= 0 || err != nil {
				c.Gauge("d2_proc_up", "1 when the process is running.", 0, "proc", name)
				continue
			}
			c.Gauge("d2_proc_up", "1 when the process is running.", map[bool]float64{true: 0, false: 1}[p.State == "Z"], "proc", name)
			c.Gauge("d2_proc_rss_bytes", "Resident set size.", float64(p.RSSKB)*1024, "proc", name)
			if p.PSSKB >= 0 {
				c.Gauge("d2_proc_pss_bytes", "Proportional set size.", float64(p.PSSKB)*1024, "proc", name)
			}
			c.Gauge("d2_proc_open_fds", "Open file descriptors.", float64(p.FDs), "proc", name)
			c.Gauge("d2_proc_threads", "Threads.", float64(p.Threads), "proc", name)
			c.Gauge("d2_proc_cpu_seconds", "User and system CPU time.", float64(p.CPUTicks())/clkTck, "proc", name)
			c.Gauge("d2_proc_zombie", "1 when the process is a zombie (gotcha 54).", map[bool]float64{true: 1}[p.State == "Z"], "proc", name)
		}
	})
}
