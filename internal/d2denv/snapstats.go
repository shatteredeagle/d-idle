package d2denv

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// snapStatsOn: the dev image counts which caller asks for a full snapshot and how long each takes ("nav.trace
// snapcallers" lines, WALKSMOOTH2): Player(), UI() and Units() each cost a whole snapshot, so a background loop
// that polls them slows every other reader (the channel answers one frame at a time).
var snapStatsOn = os.Getenv("D2BOT_NAV_TRACE") == "1"

var snapStats = struct {
	mu sync.Mutex
	m  map[string]*snapStat
}{m: map[string]*snapStat{}}

type snapStat struct {
	n     int
	total time.Duration
	max   time.Duration
}

func noteSnap(skip int, d time.Duration) {
	var pcs [4]uintptr
	k := runtime.Callers(skip, pcs[:])
	fr := runtime.CallersFrames(pcs[:k])
	var who []string
	for f, more := fr.Next(); len(who) < 3; f, more = fr.Next() {
		name := f.Function
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		who = append(who, fmt.Sprintf("%s:%d", name, f.Line))
		if !more {
			break
		}
	}
	key := strings.Join(who, " < ")
	snapStats.mu.Lock()
	s := snapStats.m[key]
	if s == nil {
		s = &snapStat{}
		snapStats.m[key] = s
	}
	s.n++
	s.total += d
	s.max = max(s.max, d)
	snapStats.mu.Unlock()
}

// snapReport formats and resets the counters.
func snapReport() string {
	snapStats.mu.Lock()
	defer snapStats.mu.Unlock()
	keys := make([]string, 0, len(snapStats.m))
	for k := range snapStats.m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return snapStats.m[keys[i]].n > snapStats.m[keys[j]].n })
	var b strings.Builder
	for _, k := range keys {
		s := snapStats.m[k]
		fmt.Fprintf(&b, "n=%-5d mean=%-6dms max=%-6dms %s\n", s.n, s.total.Milliseconds()/int64(max(s.n, 1)), s.max.Milliseconds(), k)
	}
	snapStats.m = map[string]*snapStat{}
	return b.String()
}
