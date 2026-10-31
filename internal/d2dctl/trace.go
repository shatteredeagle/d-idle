package d2dctl

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// VerbStat is the round-trip record of one verb (the first word, "debug x" for debug verbs).
type VerbStat struct {
	Verb  string
	N     uint64
	Total time.Duration
	Max   time.Duration
	Bytes uint64    // reply payload bytes
	Hist  [6]uint64 // round trips under 1, 2, 5, 20, 100 ms, and slower
}

var histEdges = [5]time.Duration{time.Millisecond, 2 * time.Millisecond, 5 * time.Millisecond, 20 * time.Millisecond, 100 * time.Millisecond}

// tracer keeps per-verb wall time and counts. It is always on: one map update per command is noise next to a socket
// round trip.
type tracer struct {
	mu sync.Mutex
	m  map[string]*VerbStat
}

func verbKey(line string) string {
	f := strings.Fields(line)
	switch {
	case len(f) == 0:
		return ""
	case len(f) > 1 && (f[0] == "debug" || f[0] == "cmd" || f[0] == "dump"):
		if f[0] == "dump" {
			return "dump"
		}
		return f[0] + " " + f[1]
	}
	return f[0]
}

func (t *tracer) add(line string, d time.Duration, rep *Reply) {
	k := verbKey(line)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.m == nil {
		t.m = map[string]*VerbStat{}
	}
	s := t.m[k]
	if s == nil {
		s = &VerbStat{Verb: k}
		t.m[k] = s
	}
	s.N++
	s.Total += d
	s.Max = max(s.Max, d)
	if rep != nil {
		for _, l := range rep.Lines {
			s.Bytes += uint64(len(l)) + 1
		}
	}
	i := 0
	for i < len(histEdges) && d >= histEdges[i] {
		i++
	}
	s.Hist[i]++
}

// Trace returns the per-verb stats since the client was made, slowest total first.
func (c *Client) Trace() []VerbStat {
	c.tr.mu.Lock()
	defer c.tr.mu.Unlock()
	out := make([]VerbStat, 0, len(c.tr.m))
	for _, s := range c.tr.m {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	return out
}

// TraceReport formats Trace as one line per verb: count, mean, max, histogram and mean reply size.
func (c *Client) TraceReport() string {
	var b strings.Builder
	for _, s := range c.Trace() {
		fmt.Fprintf(&b, "%-16s n=%-7d mean=%-9s max=%-9s total=%-9s bytes/call=%-6d hist<1,2,5,20,100,more ms=%v\n",
			s.Verb, s.N, (s.Total / time.Duration(max(s.N, 1))).Round(10*time.Microsecond), s.Max.Round(time.Millisecond),
			s.Total.Round(time.Millisecond), s.Bytes/max(s.N, 1), s.Hist)
	}
	return b.String()
}
