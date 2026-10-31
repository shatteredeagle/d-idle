package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Limits are the pass/fail thresholds. Zero disables a check, except where noted.
type Limits struct {
	MaxDeathsPerHour  float64       // fail above
	MaxNoProgress     time.Duration // longest window without progress (idle and paused time is exempt)
	MaxUnhealthy      time.Duration // longest run of unreachable / 503 health
	MinXPPerHour      float64       // fail below (0 = off; set once the goal is a leveling one)
	MaxLadderPerHour  float64       // failsafe ladder rungs per hour
	MaxSentinelPerHr  float64       // sentinel actions per hour
	MaxRSSMBPerHour   float64       // slope of any process RSS, MB per hour
	MaxFDsPerHour     float64       // slope of any process or bot fd count
	MaxGoroutinesPerH float64       // slope of the bot's go_goroutines
	MaxThreadsPerHour float64
	MaxZombieSamples  int           // consecutive samples a watched process may stay a zombie
	MinSlopeSpan      time.Duration // slopes need at least this much post-warm-up data (and 10 samples), else a warning
	// WarmUp is dropped from the start of every slope series (and of each segment after a process restart):
	// Game.exe, wineserver and d2input gain fds and threads while they load (docs/leakhunt.md). 0 = judge from the start.
	WarmUp time.Duration
	// AreaStepFDs: an fd increase of at least this many in one sample interval next to an area_change event is a
	// first-visit step (about +70 per new area, VERIFIED LIVE docs/leakhunt.md) and is not counted as growth.
	AreaStepFDs float64
	// Wineserver fds grow in steps (new threads, first-visit areas), not continuously (docs/leakhunt.md WINELEAK,
	// VERIFIED LIVE), so its fd series is judged by a looser slope after a longer span plus an absolute 24 h
	// projection. Before WineMinSpan of post-warm-up data it is "insufficient data", never a failure.
	// Helper and game fds keep MaxFDsPerHour. A zero limit disables that part.
	WineMinSpan       time.Duration
	WineMaxFDsPerHour float64
	// WineMaxGrowth24h backs up the slope limit with an absolute 24 h projection. With the default 40 fds/h the slope
	// limit (960 per 24 h) always fires first, so it only matters when WineMaxFDsPerHour is raised above 83 or off.
	WineMaxGrowth24h float64 // fail when slope * 24 h exceeds this many fds
	// MaxWineOrphans: warn (never fail) when more orphaned winedevice.exe than this stand at the end of a sampled run
	// (docs/leakhunt.md: about 35 MB each, one per wineserver restart). 0 = off.
	MaxWineOrphans int
	// Progress holds the checks on the bot's minute lines (progress.go): no_gain, loop_no_gain, idle_in_town,
	// state_dominant and below_floor_clvl. They run in `d2soak report` when a bot log is known.
	Progress ProgressLimits
	// Death is the death rule (deathrule.go, plan-v2 10.2). A limits.json written before it has no such block and
	// decodes to the zero value, which is off: an old run is judged as it was recorded.
	Death DeathLimits
}

// DefaultLimits are starting points (UNVERIFIED against a real soak: tune after the first 24 h run).
func DefaultLimits() Limits {
	return Limits{
		MaxDeathsPerHour:  6,
		MaxNoProgress:     10 * time.Minute,
		MaxUnhealthy:      3 * time.Minute,
		MaxLadderPerHour:  4,
		MaxSentinelPerHr:  4,
		MaxRSSMBPerHour:   50,
		MaxFDsPerHour:     5,
		MaxGoroutinesPerH: 10,
		MaxThreadsPerHour: 10,
		MaxZombieSamples:  3,
		MinSlopeSpan:      30 * time.Minute,
		WarmUp:            10 * time.Minute,
		AreaStepFDs:       8,
		WineMinSpan:       2 * time.Hour,
		WineMaxFDsPerHour: 40,
		WineMaxGrowth24h:  2000,
		Progress:          DefaultProgressLimits(),
	}
}

type Finding struct {
	Level string `json:"level"` // fail | warn
	Code  string `json:"code"`
	Msg   string `json:"msg"`
	// Active is set by the minute-line and death rules when the condition still holds at the end of the log. The report
	// lists every finding of the whole run; the live alerts keep only the active ones, so they clear when resolved.
	Active bool `json:"-"`
}

type Slope struct {
	Series   string  `json:"series"`
	PerHour  float64 `json:"perHour"`
	Unit     string  `json:"unit"`
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Span     string  `json:"span"`
	Enough   bool    `json:"enough"`
	Limit    float64 `json:"limit,omitempty"`
	Exceeded bool    `json:"exceeded,omitempty"`
	Proj24h  float64 `json:"projected24h,omitempty"` // wineserver fds only: PerHour * 24
}

// Plateau describes one fd series after warm-up: where it stood when the warm-up ended, where it ended, how many
// area_change steps were ignored and the slope of what is left.
type Plateau struct {
	Series  string  `json:"series"`
	AtWarm  float64 `json:"atWarmUp"`
	End     float64 `json:"end"`
	Steps   int     `json:"areaSteps"`
	StepFDs float64 `json:"areaStepFDs"`
	PerHour float64 `json:"perHour"`
	Enough  bool    `json:"enough"`
}

type Report struct {
	BotVersion    string         `json:"botVersion,omitempty"` // d2bot build the run judged (version.txt)
	Pass          bool           `json:"pass"`
	Verdict       string         `json:"verdict"` // PASS, FAIL or NO-DATA (fewer than 2 samples: nothing was judged)
	Start         time.Time      `json:"start"`
	End           time.Time      `json:"end"`
	Span          string         `json:"span"`
	Samples       int            `json:"samples"`
	XP            int64          `json:"xp"`
	XPPerHour     float64        `json:"xpPerHour"`
	XPSource      string         `json:"xpSource,omitempty"` // save | memory | counter: where XP and XPPerHour come from
	XPCounter     int64          `json:"xpCounter"`          // the bot's /stats total (gross: counts XP that was later lost)
	XPCounterPerH float64        `json:"xpCounterPerHour"`
	XPLost        int64          `json:"xpLost,omitempty"` // metric xp_lost: character XP that vanished between samples
	XPLostEvents  int            `json:"xpLostEvents,omitempty"`
	KillsPerHour  float64        `json:"killsPerHour"`
	LevelStart    int            `json:"levelStart,omitempty"`
	LevelEnd      int            `json:"levelEnd,omitempty"`
	Deaths        int64          `json:"deaths"`
	DeathsPerHour float64        `json:"deathsPerHour"`
	LadderSteps   int            `json:"ladderSteps"`
	LadderPerHour float64        `json:"ladderPerHour"`
	Failsafes     int            `json:"failsafeEvents"`
	Sentinel      int            `json:"sentinelActions"`
	SentinelPerHr float64        `json:"sentinelPerHour"`
	NoProgress    string         `json:"longestNoProgress"`
	NoProgressAt  time.Time      `json:"longestNoProgressAt,omitempty"`
	Unhealthy     string         `json:"longestUnhealthy"`
	Restarts      map[string]int `json:"processRestarts,omitempty"`
	WarmUp        string         `json:"warmUp"`
	FDPlateau     []Plateau      `json:"fdPlateau,omitempty"`
	Slopes        []Slope        `json:"slopes,omitempty"`
	WineDevices   string         `json:"wineDevices,omitempty"` // one-line winedevice.exe summary, only when sampled
	DeathRule     *DeathStats    `json:"deathRule,omitempty"`   // death-rule numbers, when the rule ran (needs the bot log)
	Findings      []Finding      `json:"findings"`
	NoData        bool           `json:"noData,omitempty"`
}

// noData marks a report that judged nothing: not a pass, but also not a failure of the bot.
func (r *Report) noData(f string, a ...any) {
	r.NoData, r.Pass = true, false
	r.Findings = append(r.Findings, Finding{Level: "warn", Code: "no-data", Msg: fmt.Sprintf(f, a...)})
}

// levelRange is the first and last level the samples know. A sample without one (the /status call failed,
// the bot was restarting) has Level 0 and must not stand in for the start or end of the run.
func levelRange(ss []Sample) (first, last int) {
	for _, s := range ss {
		if s.Level > 0 {
			if first == 0 {
				first = s.Level
			}
			last = s.Level
		}
	}
	return
}

func hours(d time.Duration) float64 { return d.Hours() }

// Analyze turns samples (oldest first) into a report. Rates use the sample timestamps, so a simulated clock works.
func Analyze(ss []Sample, lim Limits) Report {
	r := analyze(ss, lim)
	switch {
	case r.NoData:
		r.Verdict = "NO-DATA"
	case r.Pass:
		r.Verdict = "PASS"
	default:
		r.Verdict = "FAIL"
	}
	return r
}

func analyze(ss []Sample, lim Limits) Report {
	r := Report{Pass: true}
	fail := func(code, f string, a ...any) {
		r.Findings = append(r.Findings, Finding{Level: "fail", Code: code, Msg: fmt.Sprintf(f, a...)})
		r.Pass = false
	}
	warn := func(code, f string, a ...any) {
		r.Findings = append(r.Findings, Finding{Level: "warn", Code: code, Msg: fmt.Sprintf(f, a...)})
	}

	r.Samples = len(ss)
	if len(ss) < 2 {
		r.noData("need at least 2 samples, have %d", len(ss))
		return r
	}
	first, last := ss[0], ss[len(ss)-1]
	span := last.T.Sub(first.T)
	r.Start, r.End, r.Span = first.T, last.T, span.Round(time.Second).String()
	h := hours(span)
	if h <= 0 {
		r.noData("samples span no time")
		return r
	}
	rate := func(n float64) float64 { return n / h }

	// Counters. Sum positive deltas so a bot restart (counters reset) does not subtract.
	var xp, kills, deaths int64
	evDeaths := 0
	hasStats := false
	for i := 1; i < len(ss); i++ {
		a, b := ss[i-1], ss[i]
		if a.HasStats && b.HasStats {
			hasStats = true
			xp += max(b.XP-a.XP, 0)
			kills += max(b.Kills-a.Kills, 0)
			deaths += max(b.Deaths-a.Deaths, 0)
		}
		evDeaths += b.Events["death"]
		r.Failsafes += b.Events["failsafe"]
		r.Sentinel += b.Sentinel
		if b.Ladder > a.Ladder {
			r.LadderSteps += b.Ladder - a.Ladder
		}
	}
	if !hasStats {
		deaths = int64(evDeaths)
		warn("no-stats", "no /stats totals: deaths come from events, XP and kills are unknown")
	}
	r.XP, r.XPPerHour, r.KillsPerHour = xp, rate(float64(xp)), rate(float64(kills))
	r.XPCounter, r.XPCounterPerH, r.XPSource = xp, r.XPPerHour, "counter"
	if cx := charXP(ss); cx.Source != "" {
		r.XP, r.XPPerHour, r.XPSource = cx.Gain, rate(float64(cx.Gain)), cx.Source
		r.XPLost, r.XPLostEvents = cx.Lost, cx.LostEvents
		if cx.LostEvents > 0 {
			warn("xp_lost", "%s", cx.lostMsg())
		}
	}
	r.LevelStart, r.LevelEnd = levelRange(ss)
	r.Deaths, r.DeathsPerHour = deaths, rate(float64(deaths))
	r.LadderPerHour, r.SentinelPerHr = rate(float64(r.LadderSteps)), rate(float64(r.Sentinel))

	if lim.MaxDeathsPerHour > 0 && r.DeathsPerHour > lim.MaxDeathsPerHour {
		fail("deaths", "%d deaths in %s = %.1f/h (limit %.1f/h)", deaths, r.Span, r.DeathsPerHour, lim.MaxDeathsPerHour)
	}
	if lim.MinXPPerHour > 0 && hasStats && r.XPPerHour < lim.MinXPPerHour {
		fail("xp-rate", "%.0f XP/h is below the floor %.0f XP/h", r.XPPerHour, lim.MinXPPerHour)
	}
	if lim.MaxLadderPerHour > 0 && r.LadderPerHour > lim.MaxLadderPerHour {
		fail("ladder", "%d failsafe ladder rungs = %.1f/h (limit %.1f/h)", r.LadderSteps, r.LadderPerHour, lim.MaxLadderPerHour)
	}
	if lim.MaxSentinelPerHr > 0 && r.SentinelPerHr > lim.MaxSentinelPerHr {
		fail("sentinel", "%d sentinel actions = %.1f/h (limit %.1f/h)", r.Sentinel, r.SentinelPerHr, lim.MaxSentinelPerHr)
	}
	for _, s := range ss {
		if s.Fatal {
			fail("fatal", "bot reported a fatal error at %s (state %q)", s.T.Format(time.RFC3339), s.State)
			break
		}
	}

	// Health: longest run of unhealthy samples.
	uh, uhAt := longest(ss, func(s Sample) (bad, skip bool) { return !s.HealthOK, false })
	r.Unhealthy = uh.Round(time.Second).String()
	if lim.MaxUnhealthy > 0 && uh > lim.MaxUnhealthy {
		fail("unhealthy", "health unreachable or 503 for %s from %s (limit %s)", uh.Round(time.Second), uhAt.Format(time.RFC3339), lim.MaxUnhealthy)
	}

	// No progress. Signal: the bot's progress counter, else XP+kills+drops from /stats.
	signal := func(s Sample) int64 {
		if s.Progress >= 0 {
			return s.Progress
		}
		return s.XP + s.Kills + s.Drops
	}
	haveSignal := false
	for _, s := range ss {
		if s.Progress >= 0 || s.HasStats {
			haveSignal = true
			break
		}
	}
	if !haveSignal {
		warn("no-progress-signal", "neither a progress counter nor /stats available: stall detection is off")
	} else {
		ref, refSig := first.T, signal(first)
		var best time.Duration
		var bestAt time.Time
		for _, s := range ss[1:] {
			exempt := s.Paused || s.State == "idle" || s.State == "paused"
			if exempt || signal(s) != refSig {
				ref, refSig = s.T, signal(s)
				continue
			}
			if w := s.T.Sub(ref); w > best {
				best, bestAt = w, ref
			}
		}
		r.NoProgress, r.NoProgressAt = best.Round(time.Second).String(), bestAt
		if lim.MaxNoProgress > 0 && best > lim.MaxNoProgress {
			fail("no-progress", "no progress for %s starting %s (limit %s)", best.Round(time.Second), bestAt.Format(time.RFC3339), lim.MaxNoProgress)
		}
	}

	// Process restarts and zombies.
	r.Restarts = map[string]int{}
	zrun := map[string]int{}
	for i, s := range ss {
		for n, p := range s.Procs {
			if p.State == "Z" {
				zrun[n]++
				if lim.MaxZombieSamples > 0 && zrun[n] == lim.MaxZombieSamples+1 {
					fail("zombie", "process %s has been a zombie for more than %d samples (gotcha 54)", n, lim.MaxZombieSamples)
				}
			} else {
				zrun[n] = 0
			}
			if i > 0 {
				if q, ok := ss[i-1].Procs[n]; ok && (q.PID != p.PID || q.Start != p.Start) {
					r.Restarts[n]++
				}
			}
		}
	}
	if len(r.Restarts) == 0 {
		r.Restarts = nil
	}

	r.WarmUp = lim.WarmUp.String()
	r.Slopes, r.FDPlateau = slopes(ss, lim)
	for _, sl := range r.Slopes {
		switch {
		case !sl.Enough:
			continue
		case sl.Exceeded:
			if sl.Proj24h != 0 {
				fail("leak", "%s grows %.1f fds/h, %.0f fds in 24 h (%.0f -> %.0f over %s, limits %.1f/h and %.0f per 24 h)", sl.Series, sl.PerHour, sl.Proj24h, sl.Start, sl.End, sl.Span, sl.Limit, lim.WineMaxGrowth24h)
				continue
			}
			fail("leak", "%s grows %.1f %s/h (%.0f -> %.0f over %s, limit %.1f)", sl.Series, sl.PerHour, sl.Unit, sl.Start, sl.End, sl.Span, sl.Limit)
		}
	}
	if w := summarizeWine(ss); w.Samples > 0 {
		r.WineDevices = w.text()
		if lim.MaxWineOrphans > 0 && w.LastOrphans > lim.MaxWineOrphans {
			warn("wine-orphans", "%d orphaned winedevice.exe at the end (limit %d, max %d): ~35 MB each, tools/game.sh reap iN (docs/leakhunt.md)", w.LastOrphans, lim.MaxWineOrphans, w.MaxOrphans)
		}
	}
	if span < lim.MinSlopeSpan+lim.WarmUp {
		warn("short-run", "run is %s, shorter than warm-up %s + %s needed for leak slopes: leaks not judged", r.Span, lim.WarmUp, lim.MinSlopeSpan)
	}
	return r
}

// longest returns the longest run of consecutive samples for which f reports bad, measured from the first bad
// sample to the first following good one (or the last sample). skip samples neither extend nor end a run.
func longest(ss []Sample, f func(Sample) (bad, skip bool)) (time.Duration, time.Time) {
	var best time.Duration
	var bestAt, runAt time.Time
	in := false
	for _, s := range ss {
		bad, skip := f(s)
		if skip {
			continue
		}
		if bad {
			if !in {
				in, runAt = true, s.T
			}
			if d := s.T.Sub(runAt); d > best {
				best, bestAt = d, runAt
			}
		} else if in {
			if d := s.T.Sub(runAt); d > best {
				best, bestAt = d, runAt
			}
			in = false
		}
	}
	return best, bestAt
}

type pt struct {
	h, v float64
	area bool // the sample saw an area_change event
}

// slopes collects (time, value) points per series name; a process restart starts a new segment ("name#2").
// Each segment drops its first WarmUp. fd series additionally ignore first-visit steps at area_change events
// (plateau fit); every other series is fitted as is.
func slopes(ss []Sample, lim Limits) ([]Slope, []Plateau) {
	type def struct {
		unit  string
		limit float64
		fd    bool
		wine  bool // wineserver fd series: plateau judgement, see Limits.WineMinSpan
	}
	pts := map[string][]pt{}
	units := map[string]def{}
	add := func(name string, d def, s Sample, v float64, t0 time.Time) {
		pts[name] = append(pts[name], pt{s.T.Sub(t0).Hours(), v, s.Events["area_change"] > 0})
		units[name] = d
	}
	t0 := ss[0].T
	seg := map[string]int{}
	lastID := map[string][2]uint64{}
	for _, s := range ss {
		for n, p := range s.Procs {
			id := [2]uint64{uint64(p.PID), p.Start}
			if old, ok := lastID[n]; ok && old != id {
				seg[n]++
			}
			lastID[n] = id
			sfx := ""
			if seg[n] > 0 {
				sfx = fmt.Sprintf("#%d", seg[n]+1)
			}
			add("proc:"+n+":rss"+sfx, def{"MB", lim.MaxRSSMBPerHour, false, false}, s, float64(p.RSSKB)/1024, t0)
			if p.FDs >= 0 {
				d := def{"fds", lim.MaxFDsPerHour, true, false}
				if n == "wineserver" {
					d.limit, d.wine = lim.WineMaxFDsPerHour, true
				}
				add("proc:"+n+":fds"+sfx, d, s, float64(p.FDs), t0)
			}
			add("proc:"+n+":threads"+sfx, def{"threads", lim.MaxThreadsPerHour, false, false}, s, float64(p.Threads), t0)
		}
		for g, v := range s.Gauges {
			switch g {
			case "go_goroutines":
				add("bot:goroutines", def{"goroutines", lim.MaxGoroutinesPerH, false, false}, s, v, t0)
			case "process_open_fds":
				add("bot:fds", def{"fds", lim.MaxFDsPerHour, true, false}, s, v, t0)
			case "go_threads":
				add("bot:threads", def{"threads", lim.MaxThreadsPerHour, false, false}, s, v, t0)
			case "process_resident_memory_bytes":
				add("bot:rss", def{"MB", lim.MaxRSSMBPerHour, false, false}, s, v/(1<<20), t0)
			case "go_memstats_heap_inuse_bytes":
				add("bot:heap", def{"MB", lim.MaxRSSMBPerHour, false, false}, s, v/(1<<20), t0)
			case "go_memstats_heap_objects":
				add("bot:heap-objects", def{"objects", 0, false, false}, s, v, t0)
			}
		}
	}
	names := make([]string, 0, len(pts))
	for n := range pts {
		names = append(names, n)
	}
	sort.Strings(names)
	var out []Slope
	var pl []Plateau
	for _, n := range names {
		all := pts[n]
		d := units[n]
		p := all
		if lim.WarmUp > 0 {
			cut := all[0].h + lim.WarmUp.Hours()
			k := sort.Search(len(all), func(i int) bool { return all[i].h >= cut })
			p = all[k:]
		}
		if len(p) < 2 {
			last := all[len(all)-1]
			out = append(out, Slope{Series: n, Unit: d.unit, Start: last.v, End: last.v, Span: "0s", Limit: d.limit})
			continue
		}
		fitted, steps, stepFDs := p, 0, 0.0
		if d.fd {
			fitted, steps, stepFDs = dropAreaSteps(p, lim.AreaStepFDs)
		}
		spanH := p[len(p)-1].h - p[0].h
		span := time.Duration(spanH * float64(time.Hour))
		b := fit(fitted)
		sl := Slope{Series: n, PerHour: b, Unit: d.unit, Start: p[0].v, End: p[len(p)-1].v,
			Span: span.Round(time.Second).String(), Limit: d.limit}
		sl.Enough = len(p) >= 10 && span >= lim.MinSlopeSpan
		if d.wine {
			sl.Enough = sl.Enough && span >= lim.WineMinSpan
			sl.Proj24h = b * 24
		}
		sl.Exceeded = sl.Enough && d.limit > 0 && b > d.limit
		if d.wine && sl.Enough && lim.WineMaxGrowth24h > 0 && sl.Proj24h > lim.WineMaxGrowth24h {
			sl.Exceeded = true
		}
		out = append(out, sl)
		if d.fd {
			pl = append(pl, Plateau{Series: n, AtWarm: p[0].v, End: p[len(p)-1].v, Steps: steps, StepFDs: stepFDs, PerHour: b, Enough: sl.Enough})
		}
	}
	return out, pl
}

// dropAreaSteps removes first-visit fd steps: an increase of at least minStep in the interval ending at a sample
// with an area_change event, or at the sample after it (the descriptors can appear a poll later). Small
// increases at the same samples stay, so a genuine leak is still seen across area changes. The result is the
// cumulative series of the remaining deltas (it starts at p[0]).
func dropAreaSteps(p []pt, minStep float64) ([]pt, int, float64) {
	out := make([]pt, len(p))
	out[0] = p[0]
	steps, total := 0, 0.0
	for i := 1; i < len(p); i++ {
		d := p[i].v - p[i-1].v
		near := p[i].area || p[i-1].area
		if minStep > 0 && near && d >= minStep {
			steps++
			total += d
			d = 0
		}
		out[i] = pt{p[i].h, out[i-1].v + d, p[i].area}
	}
	return out, steps, total
}

// fit is the least-squares slope of v against h.
func fit(p []pt) float64 {
	n := float64(len(p))
	var sx, sy, sxx, sxy float64
	for _, q := range p {
		sx += q.h
		sy += q.v
		sxx += q.h * q.h
		sxy += q.h * q.v
	}
	den := n*sxx - sx*sx
	if den == 0 {
		return 0
	}
	return (n*sxy - sx*sy) / den
}

func lvl(n int) string {
	if n <= 0 {
		return "?"
	}
	return fmt.Sprint(n)
}

// Text renders the report for a terminal or a file.
func (r Report) Text() string {
	var b strings.Builder
	v := r.Verdict
	if v == "" { // a Report built by hand
		v = map[bool]string{true: "PASS", false: "FAIL"}[r.Pass]
	}
	fmt.Fprintf(&b, "d2soak report: %s\n", v)
	if r.BotVersion != "" {
		fmt.Fprintf(&b, "  bot version  %s\n", r.BotVersion)
	}
	if r.NoData {
		for _, f := range r.Findings {
			fmt.Fprintf(&b, "  %s\n", f.Msg)
		}
		return b.String()
	}
	fmt.Fprintf(&b, "  run          %s .. %s (%s, %d samples)\n", r.Start.Format(time.RFC3339), r.End.Format(time.RFC3339), r.Span, r.Samples)
	fmt.Fprintf(&b, "  progress     %.0f XP/h (%d XP, from %s), %.0f kills/h, level %s -> %s\n", r.XPPerHour, r.XP, r.XPSource, r.KillsPerHour, lvl(r.LevelStart), lvl(r.LevelEnd))
	if r.XPSource != "counter" {
		fmt.Fprintf(&b, "  xp counter   %.0f XP/h (%d XP, gross), xp_lost %d XP in %d events\n", r.XPCounterPerH, r.XPCounter, r.XPLost, r.XPLostEvents)
	}
	fmt.Fprintf(&b, "  deaths       %d (%.2f/h)\n", r.Deaths, r.DeathsPerHour)
	fmt.Fprintf(&b, "  interventions ladder rungs %d (%.2f/h), failsafe events %d, sentinel actions %d (%.2f/h)\n", r.LadderSteps, r.LadderPerHour, r.Failsafes, r.Sentinel, r.SentinelPerHr)
	fmt.Fprintf(&b, "  stalls       longest no-progress %s, longest unhealthy %s\n", r.NoProgress, r.Unhealthy)
	if len(r.Restarts) > 0 {
		ks := make([]string, 0, len(r.Restarts))
		for k := range r.Restarts {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		b.WriteString("  restarts    ")
		for _, k := range ks {
			fmt.Fprintf(&b, " %s=%d", k, r.Restarts[k])
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "  warm-up      %s dropped from every slope series (and after each process restart)\n", r.WarmUp)
	if r.WineDevices != "" {
		fmt.Fprintf(&b, "  %s\n", r.WineDevices)
	}
	for _, pl := range r.FDPlateau {
		note := ""
		if !pl.Enough {
			note = " (insufficient data)"
		}
		fmt.Fprintf(&b, "  fd plateau   %-24s %.0f at end of warm-up -> %.0f, %d area_change steps ignored (+%.0f fds), slope %+.2f fds/h%s\n",
			pl.Series, pl.AtWarm, pl.End, pl.Steps, pl.StepFDs, pl.PerHour, note)
	}
	if len(r.Slopes) > 0 {
		b.WriteString("  slopes\n")
		for _, s := range r.Slopes {
			note := ""
			switch {
			case !s.Enough:
				note = "  (insufficient data)"
			case s.Exceeded:
				note = fmt.Sprintf("  EXCEEDS %.1f", s.Limit)
			}
			fmt.Fprintf(&b, "    %-28s %+9.2f %s/h  %.0f -> %.0f over %s%s\n", s.Series, s.PerHour, s.Unit, s.Start, s.End, s.Span, note)
		}
	}
	for _, f := range r.Findings {
		fmt.Fprintf(&b, "  %-4s %-18s %s\n", strings.ToUpper(f.Level), f.Code, f.Msg)
	}
	return b.String()
}
