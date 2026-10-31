package launch

import (
	"context"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// A child subreaper inherits every orphaned descendant, and nothing waits for the ones we did not start ourselves
// (Wine's detached services, a game whose parent launcher exited): they stay zombies until we exit. PID 1 would
// reap them, but as subreaper we are their parent. A blanket wait4(-1) would steal the exit status of our own
// exec.Cmd children, so only zombie children that were already zombies on the previous scan are reaped by pid:
// a Cmd that is being waited for has collected its child long before the next scan.

// ZombieReaper reaps adopted zombie children. The zero value is the real system.
type ZombieReaper struct {
	Zombies func() []int        // pids of our zombie children; default scans /proc
	Wait    func(pid int) error // default wait4(pid, WNOHANG)
	Log     func(pid int)       // optional
	seen    map[int]bool
}

// Scan reaps the zombies seen on the previous scan and remembers the new ones. It returns the reaped pids.
func (z *ZombieReaper) Scan() []int {
	if z.Zombies == nil {
		z.Zombies = ownZombies
	}
	if z.Wait == nil {
		z.Wait = func(pid int) error {
			var ws syscall.WaitStatus
			_, err := syscall.Wait4(pid, &ws, syscall.WNOHANG, nil)
			return err
		}
	}
	now := map[int]bool{}
	var done []int
	for _, pid := range z.Zombies() {
		if z.seen[pid] {
			if z.Wait(pid) == nil {
				done = append(done, pid)
				if z.Log != nil {
					z.Log(pid)
				}
				continue
			}
		}
		now[pid] = true
	}
	z.seen = now
	return done
}

// Run scans every interval (default 5 s) until ctx ends.
func (z *ZombieReaper) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			z.Scan()
		}
	}
}

func ownZombies() []int {
	self := os.Getpid()
	ents, _ := os.ReadDir("/proc")
	var out []int
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		b, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		if st, ppid, ok := parseStat(string(b)); ok && st == 'Z' && ppid == self {
			out = append(out, pid)
		}
	}
	return out
}

// parseStat returns the state and ppid of a /proc/PID/stat line (comm may contain spaces and parens).
func parseStat(s string) (state byte, ppid int, ok bool) {
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return 0, 0, false
	}
	f := strings.Fields(s[i+1:])
	if len(f) < 2 || len(f[0]) != 1 {
		return 0, 0, false
	}
	p, err := strconv.Atoi(f[1])
	if err != nil {
		return 0, 0, false
	}
	return f[0][0], p, true
}
