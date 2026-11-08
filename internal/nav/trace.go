package nav

import (
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"d2lod-bot/internal/input"
)

// walkTrace is the click-timing trace of one walk (WALKSMOOTH2, TASK-217). It is off unless Navigator.Trace is set
// (the dev image and D2BOT_NAV_TRACE=1). Output is "nav.trace" lines: one per click, one per stall start, one per
// resume, one summary per walk. Volume is bounded by the click rate (the bot clicks at most every 250 ms).
type walkTrace struct {
	n         *Navigator
	start     time.Time
	startPos  input.Point
	last      input.Point
	pathLen   float64
	clicks    int
	stalls    int
	stillFor  time.Duration
	stillAt   time.Time // zero: moving
	lastClick time.Time
	clickPos  input.Point
	obsMs     int64
	planMs    int64
	obsMax    int64
	polls     int
	fast      int // polls that were position-only (APPROACHFAST)
	started   bool
	callers   string
}

func (t *walkTrace) on() bool { return t != nil && t.n.Trace }

func (t *walkTrace) begin(pos input.Point, now time.Time) {
	if !t.on() || t.started {
		return
	}
	t.started, t.start, t.startPos, t.last = true, now, pos, pos
}

// poll records one observation (cost of the observe call, movement since the last poll) and returns nothing.
func (t *walkTrace) poll(pos input.Point, now time.Time, obs time.Duration, still bool) {
	if !t.on() {
		return
	}
	t.begin(pos, now)
	t.polls++
	ms := obs.Milliseconds()
	t.obsMs += ms
	t.obsMax = max(t.obsMax, ms)
	t.pathLen += dist(pos, t.last)
	t.last = pos
	switch {
	case still && t.stillAt.IsZero():
		t.stillAt = now
		t.stalls++
		sinceClick := int64(-1)
		if !t.lastClick.IsZero() {
			sinceClick = now.Sub(t.lastClick).Milliseconds()
		}
		t.n.log.Info("nav.trace stall", "t", now.UnixMilli(), "pos", pos, "sinceClickMs", sinceClick, "snapMs", t.n.obsSnapMs, "updMs", t.n.obsUpdMs,
			"movedSinceClick", round1(dist(pos, t.clickPos)), "obsMs", ms)
	case !still && !t.stillAt.IsZero():
		d := now.Sub(t.stillAt)
		t.stillFor += d
		t.stillAt = time.Time{}
		t.n.log.Info("nav.trace resume", "t", now.UnixMilli(), "pos", pos, "pauseMs", d.Milliseconds())
	}
}

func (t *walkTrace) click(pos, wp input.Point, now time.Time, reason string, took time.Duration, cur, of int) {
	if !t.on() {
		return
	}
	gap := int64(-1)
	if !t.lastClick.IsZero() {
		gap = now.Sub(t.lastClick).Milliseconds()
	}
	t.clicks++
	t.lastClick, t.clickPos = now, pos
	t.n.log.Info("nav.trace click", "t", now.UnixMilli(), "from", pos, "to", wp, "dist", round1(dist(pos, wp)),
		"reason", reason, "snapMs", t.n.obsSnapMs, "updMs", t.n.obsUpdMs, "clickMs", took.Milliseconds(), "gapMs", gap, "wp", cur, "of", of)
}

func (t *walkTrace) plan(took time.Duration, why string, wps int) {
	if !t.on() {
		return
	}
	t.planMs += took.Milliseconds()
	t.n.log.Info("nav.trace plan", "ms", took.Milliseconds(), "why", why, "wps", wps)
}

func (t *walkTrace) end(pos input.Point, now time.Time, err error) {
	if !t.on() || !t.started {
		return
	}
	if !t.stillAt.IsZero() {
		t.stillFor += now.Sub(t.stillAt)
	}
	dur := now.Sub(t.start)
	if dur < 800*time.Millisecond {
		return // an arrival check; nothing to say
	}
	moved := dist(pos, t.startPos)
	speed := 0.0
	if dur > 0 {
		speed = t.pathLen / dur.Seconds()
	}
	per100 := 0.0
	if t.pathLen > 0 {
		per100 = float64(t.clicks) * 100 / t.pathLen
	}
	t.n.log.Info("nav.trace leg", "durMs", dur.Milliseconds(), "pathSubtiles", round1(t.pathLen), "net", round1(moved),
		"speed", round1(speed), "clicks", t.clicks, "clicksPer100", round1(per100), "stalls", t.stalls,
		"stillMs", t.stillFor.Milliseconds(), "polls", t.polls, "fastPolls", t.fast, "obsAvgMs", avgMs(t.obsMs, t.polls), "obsMaxMs", t.obsMax,
		"planMs", t.planMs, "err", errStr(err), "callers", t.callers)
}

func avgMs(sum int64, n int) int64 {
	if n == 0 {
		return 0
	}
	return sum / int64(n)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// TraceEnv reports whether the click-timing trace is on: D2BOT_NAV_TRACE=1 (cmd/d2bot sets it by default in the dev
// image, never in a stamped release).
func TraceEnv() bool { return os.Getenv("D2BOT_NAV_TRACE") == "1" }

// callerNames returns up to n function names above the skip-th frame, short form, for the trace.
func callerNames(skip, n int) string {
	var pcs [8]uintptr
	k := runtime.Callers(skip, pcs[:])
	fr := runtime.CallersFrames(pcs[:k])
	var who []string
	for f, more := fr.Next(); len(who) < n; f, more = fr.Next() {
		name := f.Function
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		who = append(who, name)
		if !more {
			break
		}
	}
	return strings.Join(who, " < ")
}
