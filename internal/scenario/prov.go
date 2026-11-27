package scenario

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Provenance says which binaries and which machine state produced a result, so two results can be compared honestly
// (design 2026-11-21 section 2.2 H5/H6: one "pair" once ran two different bot builds). Host is a role name, never a
// machine name.
type Provenance struct {
	BotCommit string  `json:"bot_commit,omitempty"`
	BotDirty  bool    `json:"bot_dirty,omitempty"`
	BotSHA256 string  `json:"bot_sha256,omitempty"`
	D2DSHA256 string  `json:"d2d_sha256,omitempty"`
	Host      string  `json:"host,omitempty"`
	Load1     float64 `json:"load1,omitempty"`
	// Pace is virtual seconds per wall second over the whole run (0 when unknown).
	Pace float64 `json:"pace,omitempty"`
	// Parallel is the peak number of Proving Ground runs of the same runner that were busy at once during this run,
	// itself included (1 alone, 0 unknown: a run started without the runner's slot log).
	Parallel int `json:"parallel,omitempty"`
}

// RunLengthLine is what the bot logs when its virtual clock reached the budget (cmd/d2bot stopAtVirtual).
const RunLengthLine = "virtual run length reached"

// RunLengthReached reports whether the bot log shows the virtual budget was played to the end. A bot stopped by the wall
// limit (--run-for) or killed does not log it, and its state is not the state the scenario asked about: not judged.
func RunLengthReached(botLog []byte) bool { return bytes.Contains(botLog, []byte(RunLengthLine)) }

// ParseBuildInfo reads `go version -m <binary>` output for the VCS stamp: the revision and whether the tree was dirty.
func ParseBuildInfo(out string) (commit string, dirty bool) {
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) < 2 || f[0] != "build" {
			continue
		}
		k, v, ok := strings.Cut(f[1], "=")
		if !ok {
			continue
		}
		switch k {
		case "vcs.revision":
			commit = v
		case "vcs.modified":
			dirty = v == "true"
		}
	}
	return
}

// PaceOf is virtual seconds per wall second; 0 when the wall time is unknown.
func PaceOf(virtualMin float64, wallSec int) float64 {
	if wallSec <= 0 {
		return 0
	}
	return virtualMin * 60 / float64(wallSec)
}

// SlotLogLine is one line of the runner's slot log: a run id started or ended at a wall time (unix nanoseconds). The
// runner appends them around every exec-one; the run reads the log at its end to learn its peak slot load.
func SlotLogLine(ev, id string, t int64) string { return fmt.Sprintf("%s %d %s\n", ev, t, id) }

// PeakParallel is the peak number of runs busy at once during run self, itself included, from the slot log; self is
// normally still busy when it reads the log (no end line yet: busy until now). 0 when self has no start line.
// Unparsable lines are skipped.
func PeakParallel(log []byte, self string) int {
	type span struct{ start, end int64 }
	spans := map[string]*span{}
	for _, l := range strings.Split(string(log), "\n") {
		f := strings.SplitN(l, " ", 3)
		if len(f) != 3 {
			continue
		}
		t, err := strconv.ParseInt(f[1], 10, 64)
		if err != nil {
			continue
		}
		sp := spans[f[2]]
		if sp == nil {
			sp = &span{start: math.MinInt64, end: math.MaxInt64}
			spans[f[2]] = sp
		}
		switch f[0] {
		case "start":
			sp.start = t
		case "end":
			sp.end = t
		}
	}
	me, ok := spans[self]
	if !ok || me.start == math.MinInt64 {
		return 0
	}
	// The busy count only rises at a start, so the peak inside self's span is at self's start or a later start.
	peak := 0
	for _, at := range spans {
		if at.start < me.start || at.start >= me.end || at.start == math.MinInt64 {
			continue
		}
		n := 0
		for _, sp := range spans {
			if sp.start != math.MinInt64 && sp.start <= at.start && at.start < sp.end {
				n++
			}
		}
		peak = max(peak, n)
	}
	return peak
}
