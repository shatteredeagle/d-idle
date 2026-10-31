package fleet

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// Probe reads the host's capacity. OSProbe is real; tests use a fake.
type Probe interface {
	Cores() float64 // usable CPU: min(online cores, cgroup cpu.max quota)
	Load1() float64
	MemAvailMB() int64
	DiskFreeMB(path string) int64
}

// Limits is the admission policy. Measured per instance (V docs/multi.md, 5 idle instances on the worker):
// about 1.05 cores, 265 MB PSS, 125 MB of unique disk. Game.exe busy-loops one core even idle.
type Limits struct {
	CoresPerInstance  float64 // default 1.05
	ReserveCores      float64 // kept free for the app, sentinel, Xvfb (default 1)
	MaxLoadFactor     float64 // refuse when load1 + CoresPerInstance > cores * factor (default 1.25; 0 = default, <0 off)
	MemPerInstanceMB  int64   // default 400 (265 measured, plus headroom for a busy game and Wine)
	ReserveMemMB      int64   // default 1024
	DiskPerInstanceMB int64   // default 1024 (prefix 125 MB + saves + logs, with room)
	MaxSlots          int     // default 9 (launch.ValidID allows i1..i9; i0 is the desktop game)
}

func (l Limits) withDefaults() Limits {
	if l.CoresPerInstance == 0 {
		l.CoresPerInstance = 1.05
	}
	if l.ReserveCores == 0 {
		l.ReserveCores = 1
	}
	if l.MaxLoadFactor == 0 {
		l.MaxLoadFactor = 1.25
	}
	if l.MemPerInstanceMB == 0 {
		l.MemPerInstanceMB = 400
	}
	if l.ReserveMemMB == 0 {
		l.ReserveMemMB = 1024
	}
	if l.DiskPerInstanceMB == 0 {
		l.DiskPerInstanceMB = 1024
	}
	if l.MaxSlots == 0 {
		l.MaxSlots = 9
	}
	return l
}

// ErrAdmission is wrapped by every refusal; errors.As to *AdmissionError for the check that failed.
var ErrAdmission = errors.New("fleet: admission refused")

type AdmissionError struct {
	Check  string // "slots", "cpu", "load", "ram", "disk"
	Need   float64
	Have   float64
	Reason string
}

func (e *AdmissionError) Error() string { return "admission refused (" + e.Check + "): " + e.Reason }
func (e *AdmissionError) Unwrap() error { return ErrAdmission }

// Capacity is a snapshot of the checks; Admit is false with Refusal set when one more slot would not fit.
type Capacity struct {
	Running    int
	MaxSlots   int
	Cores      float64
	Load1      float64
	MemAvailMB int64
	DiskFreeMB int64
	// Headroom is how many more instances fit by the resource checks alone (CPU, RAM, disk).
	Headroom int
	Admit    bool
	Refusal  string `json:",omitempty"`
}

// Check evaluates whether one more instance fits when `running` are already counted (starting or running
// slots; a restarting slot still holds its place).
func (l Limits) Check(p Probe, running int, dir string) (Capacity, *AdmissionError) {
	l = l.withDefaults()
	c := Capacity{Running: running, MaxSlots: l.MaxSlots, Cores: p.Cores(), Load1: p.Load1(), MemAvailMB: p.MemAvailMB(), DiskFreeMB: p.DiskFreeMB(dir)}
	hr := func(have, per float64) int {
		if per <= 0 || have <= 0 {
			return 0
		}
		return int(have / per)
	}
	c.Headroom = min(hr(c.Cores-l.ReserveCores-float64(running)*l.CoresPerInstance, l.CoresPerInstance),
		hr(float64(c.MemAvailMB-l.ReserveMemMB), float64(l.MemPerInstanceMB)),
		hr(float64(c.DiskFreeMB), float64(l.DiskPerInstanceMB)))
	fail := func(e *AdmissionError) (Capacity, *AdmissionError) { c.Refusal = e.Reason; return c, e }
	if running >= l.MaxSlots {
		return fail(&AdmissionError{"slots", float64(running + 1), float64(l.MaxSlots),
			fmt.Sprintf("%d slots is the maximum on this host", l.MaxSlots)})
	}
	if need := l.ReserveCores + float64(running+1)*l.CoresPerInstance; need > c.Cores {
		return fail(&AdmissionError{"cpu", need, c.Cores,
			fmt.Sprintf("%d instances need %.1f cores (about %.2f each plus %.1f reserved), host has %.1f", running+1, need, l.CoresPerInstance, l.ReserveCores, c.Cores)})
	}
	if l.MaxLoadFactor > 0 {
		if lim := c.Cores * l.MaxLoadFactor; c.Load1+l.CoresPerInstance > lim {
			return fail(&AdmissionError{"load", c.Load1 + l.CoresPerInstance, lim,
				fmt.Sprintf("load average %.1f is too high for another instance on %.1f cores (limit %.1f)", c.Load1, c.Cores, lim)})
		}
	}
	if need := l.ReserveMemMB + l.MemPerInstanceMB; c.MemAvailMB < need {
		return fail(&AdmissionError{"ram", float64(need), float64(c.MemAvailMB),
			fmt.Sprintf("%d MB available, need %d MB (%d per instance plus %d reserved)", c.MemAvailMB, need, l.MemPerInstanceMB, l.ReserveMemMB)})
	}
	if c.DiskFreeMB < l.DiskPerInstanceMB {
		return fail(&AdmissionError{"disk", float64(l.DiskPerInstanceMB), float64(c.DiskFreeMB),
			fmt.Sprintf("%d MB free in %s, need %d MB", c.DiskFreeMB, dir, l.DiskPerInstanceMB)})
	}
	c.Admit = true
	return c, nil
}

// OSProbe reads /proc and the cgroup v2 quota. Where a file is unreadable it reports a permissive value
// (cores from runtime.NumCPU, load 0, RAM and disk large), so a missing /proc never blocks a start by itself.
type OSProbe struct{}

func (OSProbe) Cores() float64 {
	n := float64(runtime.NumCPU())
	if b, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		f := strings.Fields(string(b))
		if len(f) == 2 && f[0] != "max" {
			q, e1 := strconv.ParseFloat(f[0], 64)
			p, e2 := strconv.ParseFloat(f[1], 64)
			if e1 == nil && e2 == nil && p > 0 && q/p < n {
				n = q / p
			}
		}
	}
	return n
}

func (OSProbe) Load1() float64 {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}
	f := strings.Fields(string(b))
	if len(f) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(f[0], 64)
	return v
}

func (OSProbe) MemAvailMB() int64 {
	avail := int64(1 << 20)
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, ln := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(ln, "MemAvailable:") {
				f := strings.Fields(ln)
				if len(f) >= 2 {
					kb, _ := strconv.ParseInt(f[1], 10, 64)
					avail = kb / 1024
				}
			}
		}
	}
	// a cgroup memory limit caps what the container can use
	lim, cur := cgInt("/sys/fs/cgroup/memory.max"), cgInt("/sys/fs/cgroup/memory.current")
	if lim > 0 {
		if left := (lim - cur) >> 20; left < avail {
			avail = left
		}
	}
	return avail
}

func cgInt(path string) int64 {
	b, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return -1 // "max"
	}
	return v
}

func (OSProbe) DiskFreeMB(path string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 1 << 20
	}
	return int64(st.Bavail) * int64(st.Bsize) >> 20
}
