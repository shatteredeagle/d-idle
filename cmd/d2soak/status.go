package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AlertRules are the live alarms of a long run (docs/soak.md). Zero disables a rule.
type AlertRules struct {
	StallXP         time.Duration // no XP change for longer (idle/paused time is exempt)
	RestartsPerHour float64       // process restarts (all watched processes) in the trailing hour, alarm above
	FDWindow        time.Duration // trailing window for the fd slope check
	PanicsPerHour   float64       // recovered panics in the trailing hour, alarm above (the bot's own counter, reset by a restart)
	SamplerStale    time.Duration // newest sample older than this: the soak itself is not sampling
	// The next four need the bot log (BotLog: a file, or a directory whose newest *.log is used; "" = off).
	BotLog              string
	TickAge             time.Duration // no minute line for longer: the bot is stalled or dead (0 = off)
	GameRestartsPerHour float64       // game restarts shown by the minute lines in the trailing hour, warn above
	WineOrphans         int           // orphaned winedevice.exe in the newest sample (needs -wine-prefix), warn above
	DeathsFile          string        // deaths.jsonl of the run for the death rule (set by the live writer, not a flag)
	HeapRunawayMB       float64       // crit when the bot's RSS exceeds this many MB (0 = off); normal runs sit under 100 MB
	HeapGrowthMB        float64       // crit when the bot's RSS grew by more than this many MB within HeapGrowthWindow (0 = off)
	HeapGrowthWindow    time.Duration
	XPRiskLevels        float64 // alarm when unsaved XP (memory minus .d2s) exceeds this many levels worth (needs -save-file; 0 = off)
}

func DefaultAlertRules() AlertRules {
	return AlertRules{StallXP: 10 * time.Minute, RestartsPerHour: 8, PanicsPerHour: 3, FDWindow: 3 * time.Hour, SamplerStale: 5 * time.Minute,
		TickAge: 3 * time.Minute, GameRestartsPerHour: 2, WineOrphans: 2, XPRiskLevels: 1,
		HeapRunawayMB: 8192, HeapGrowthMB: 1024, HeapGrowthWindow: 5 * time.Minute}
}

type Alert struct {
	Level string `json:"level"` // crit | warn; a crit alert from the progress or death rules is a "fail" finding of the report
	Code  string `json:"code"`
	Msg   string `json:"msg"`
	// Since is when the alert first appeared in an unbroken stretch (the live writer sets it; absent from EvalAlerts).
	Since time.Time `json:"since,omitempty"`
}

// Alerts is the file the dashboard shows (OUT/alerts.json).
type Alerts struct {
	Updated time.Time `json:"updated"`
	Level   string    `json:"level"` // ok | warn | crit
	Alerts  []Alert   `json:"alerts"`
	Status  string    `json:"status"`            // the one-line status
	Summary string    `json:"summary,omitempty"` // last 60 min of the bot log: clvl, xp/h, deaths, recovery share, top state group
}

// sinceXP is how long the XP counter has stood still at the end of the samples. Idle and paused samples reset the
// clock, as in the no-progress check. A counter that goes down (bot restart) counts as a change.
func sinceXP(ss []Sample) time.Duration {
	if len(ss) == 0 {
		return 0
	}
	ref, v := ss[0].T, ss[0].XP
	for _, s := range ss[1:] {
		if s.Paused || s.State == "idle" || s.State == "paused" || s.XP != v {
			ref, v = s.T, s.XP
		}
	}
	return ss[len(ss)-1].T.Sub(ref)
}

// sinceProgress is the same for the bot's progress counter (XP+kills+drops when the health body has none).
func sinceProgress(ss []Sample) time.Duration {
	sig := func(s Sample) int64 {
		if s.Progress >= 0 {
			return s.Progress
		}
		return s.XP + s.Kills + s.Drops
	}
	if len(ss) == 0 {
		return 0
	}
	ref, v := ss[0].T, sig(ss[0])
	for _, s := range ss[1:] {
		if s.Paused || s.State == "idle" || s.State == "paused" || sig(s) != v {
			ref, v = s.T, sig(s)
		}
	}
	return ss[len(ss)-1].T.Sub(ref)
}

// restartTimes lists when each watched process changed pid or start time.
func restartTimes(ss []Sample) map[string][]time.Time {
	out := map[string][]time.Time{}
	for i := 1; i < len(ss); i++ {
		for n, p := range ss[i].Procs {
			if q, ok := ss[i-1].Procs[n]; ok && (q.PID != p.PID || q.Start != p.Start) {
				out[n] = append(out[n], ss[i].T)
			}
		}
	}
	return out
}

func restartsSince(ss []Sample, since time.Time) (int, map[string]int) {
	total, by := 0, map[string]int{}
	for n, ts := range restartTimes(ss) {
		for _, t := range ts {
			if !t.Before(since) {
				total++
				by[n]++
			}
		}
	}
	return total, by
}

// panicsSince sums the increases of the bot's panic counter over the samples; a drop is a restart (the new count adds).
func panicsSince(ss []Sample) int {
	n := 0
	for i := 1; i < len(ss); i++ {
		if d := ss[i].Panics - ss[i-1].Panics; d > 0 {
			n += d
		} else if d < 0 {
			n += ss[i].Panics
		}
	}
	return n
}

// areaChanges returns the last n distinct areas entered, newest last, from the sampled /status area.
func areaChanges(ss []Sample, n int) []string {
	var out []string
	prev := ""
	for _, s := range ss {
		if s.Area == "" || s.Area == prev {
			continue
		}
		if prev != "" {
			out = append(out, s.Area)
		}
		prev = s.Area
	}
	if len(out) > n {
		out = out[len(out)-n:]
	}
	return out
}

func tail(ss []Sample, d time.Duration) []Sample {
	if len(ss) == 0 {
		return nil
	}
	cut := ss[len(ss)-1].T.Add(-d)
	k := sort.Search(len(ss), func(i int) bool { return !ss[i].T.Before(cut) })
	return ss[k:]
}

func mins(d time.Duration) string { return fmt.Sprintf("%.0fm", d.Minutes()) }

// StatusLine is the live one-liner: level, xp/h, deaths, failsafe rungs, restarts/hour, helper fds/threads, the last
// three area changes and minutes since the last progress. now is the wall clock (the sample time of a replay).
func StatusLine(ss []Sample, now time.Time) string {
	if len(ss) == 0 {
		return "soak: no samples yet"
	}
	last := ss[len(ss)-1]
	first := ss[0]
	span := last.T.Sub(first.T)
	_, lv := levelRange(ss)
	var xp, deaths int64
	rungs := 0
	for i := 1; i < len(ss); i++ {
		a, b := ss[i-1], ss[i]
		if a.HasStats && b.HasStats {
			xp += max(b.XP-a.XP, 0)
			deaths += max(b.Deaths-a.Deaths, 0)
		}
		rungs += b.Events["failsafe"]
		if b.Ladder > a.Ladder {
			rungs += b.Ladder - a.Ladder
		}
	}
	xph := "?"
	if span >= time.Minute {
		xph = fmt.Sprintf("%.0f", float64(xp)/span.Hours())
	}
	rst, _ := restartsSince(ss, last.T.Add(-time.Hour))
	rph := float64(rst)
	if span < time.Hour && span > 0 {
		rph = float64(rst) / span.Hours()
	}
	helper := "helper n/a"
	if h, ok := last.Procs["helper"]; ok {
		helper = fmt.Sprintf("helper %d fds/%d thr", h.FDs, h.Threads)
	}
	areas := "none"
	if a := areaChanges(ss, 3); len(a) > 0 {
		areas = strings.Join(a, " > ")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "soak %s: lvl %s, %s xp/h, deaths %d, failsafe rungs %d, restarts %.1f/h, %s, areas [%s], %s since progress",
		span.Round(time.Minute), lvl(lv), xph, deaths, rungs, rph, helper, areas, mins(sinceProgress(ss)))
	if age := now.Sub(last.T); age > 2*time.Minute {
		fmt.Fprintf(&b, ", LAST SAMPLE %s AGO", mins(age))
	}
	return b.String()
}

// EvalAlerts applies the rules to the samples. The fd slope uses the trailing FDWindow with the normal warm-up and
// plateau handling, only for the newest segment of each process (an older segment is a process that is gone).
func EvalAlerts(ss []Sample, lim Limits, rules AlertRules, now time.Time) Alerts {
	out := Alerts{Updated: now, Level: "ok", Alerts: []Alert{}}
	add := func(level, code, f string, a ...any) {
		out.Alerts = append(out.Alerts, Alert{Level: level, Code: code, Msg: fmt.Sprintf(f, a...)})
		if level == "crit" || out.Level == "ok" {
			out.Level = level
		}
	}
	out.Status = StatusLine(ss, now)
	if len(ss) == 0 {
		add("warn", "no-samples", "no samples yet")
		return out
	}
	last := ss[len(ss)-1]
	if rules.SamplerStale > 0 && now.Sub(last.T) > rules.SamplerStale {
		add("crit", "sampler-stale", "newest sample is %s old: d2soak is not sampling", mins(now.Sub(last.T)))
	}
	if !last.HealthOK {
		add("crit", "unhealthy", "bot health is down (HTTP %d)", last.HTTP)
	}
	if rules.StallXP > 0 {
		if d := sinceXP(ss); d > rules.StallXP {
			add("crit", "stall-xp", "no XP for %s (limit %s)", mins(d), mins(rules.StallXP))
		}
	}
	if rules.XPRiskLevels > 0 {
		if risk, worth, ok := xpAtRisk(ss); ok && worth > 0 && float64(risk) > rules.XPRiskLevels*float64(worth) {
			add("warn", "xp_at_risk", "%d XP unsaved (more than %.1f level, %d XP): a hard stop now loses it; the game saves only on Save and Exit", risk, rules.XPRiskLevels, worth)
		}
	}
	if a, ok := heapRunaway(ss, rules); ok {
		add("crit", "heap_runaway", "%s", a)
	}
	if rules.RestartsPerHour > 0 {
		hs := tail(ss, time.Hour)
		n, by := restartsSince(hs, hs[0].T)
		if float64(n) > rules.RestartsPerHour {
			var parts []string
			for k, v := range by {
				parts = append(parts, fmt.Sprintf("%s=%d", k, v))
			}
			sort.Strings(parts)
			add("crit", "restarts", "%d process restarts in the last hour (limit %.0f): %s", n, rules.RestartsPerHour, strings.Join(parts, " "))
		}
	}
	if rules.PanicsPerHour > 0 {
		if n := panicsSince(tail(ss, time.Hour)); float64(n) > rules.PanicsPerHour {
			add("warn", "panics", "%d recovered panics in the last hour (limit %.0f): see the d2bot log", n, rules.PanicsPerHour)
		}
	}
	for _, a := range append(botLogAlerts(rules, now), wineAlert(ss, rules)...) {
		add(a.Level, a.Code, "%s", a.Msg)
	}
	// The progress and death rules of the report, on the same bot log, active findings only.
	fa, j := logFindingAlerts(rules, lim)
	for _, a := range fa {
		add(a.Level, a.Code, "%s", a.Msg)
	}
	out.Summary = liveSummary(j)
	if rules.FDWindow > 0 && lim.MaxFDsPerHour > 0 {
		w := tail(ss, rules.FDWindow)
		if len(w) >= 2 {
			sl, _ := slopes(w, lim)
			newest := map[string]int{}
			for _, sl := range sl {
				base, k := segment(sl.Series)
				newest[base] = max(newest[base], k)
			}
			for _, sl := range sl {
				if base, k := segment(sl.Series); k != newest[base] || sl.Unit != "fds" || !sl.Exceeded {
					continue
				}
				add("warn", "fd-slope", "%s grows %.1f fds/h over %s (limit %.1f)", sl.Series, sl.PerHour, sl.Span, sl.Limit)
			}
		}
	}
	return out
}

// botRSSMB is the bot's resident memory in a sample in MB: its own gauge, else the /proc reading; ok is false when
// neither is there (the bot is down).
func botRSSMB(s Sample) (float64, bool) {
	if v, ok := s.Gauges["process_resident_memory_bytes"]; ok && v > 0 {
		return v / (1 << 20), true
	}
	if p, ok := s.Procs["bot"]; ok && p.RSSKB > 0 {
		return float64(p.RSSKB) / 1024, true
	}
	return 0, false
}

// heapRunaway flags a runaway allocation in the bot (G1 attempt 16: 66 MB to 32 GB between two 30 s samples, then the
// OOM killer): the newest RSS above HeapRunawayMB, or up by more than HeapGrowthMB within HeapGrowthWindow. A restart
// lowers the RSS again, so the alert clears by itself.
func heapRunaway(ss []Sample, rules AlertRules) (string, bool) {
	now, ok := botRSSMB(ss[len(ss)-1])
	if !ok {
		return "", false
	}
	if rules.HeapRunawayMB > 0 && now > rules.HeapRunawayMB {
		return fmt.Sprintf("bot RSS %.0f MB is above %.0f MB: a runaway allocation; the heap guard (bot.heapLimitMB) or the OOM killer ends it", now, rules.HeapRunawayMB), true
	}
	if rules.HeapGrowthMB > 0 && rules.HeapGrowthWindow > 0 {
		low := now
		for _, s := range tail(ss, rules.HeapGrowthWindow) {
			if v, ok := botRSSMB(s); ok && v < low {
				low = v
			}
		}
		if now-low > rules.HeapGrowthMB {
			return fmt.Sprintf("bot RSS grew %.0f MB in %s (%.0f to %.0f MB, limit %.0f MB): a runaway allocation", now-low, mins(rules.HeapGrowthWindow), low, now, rules.HeapGrowthMB), true
		}
	}
	return "", false
}

// segment splits "proc:game:fds#2" into the series and its segment number (1 without a suffix).
func segment(series string) (string, int) {
	base, n, ok := strings.Cut(series, "#")
	if !ok {
		return base, 1
	}
	k := 1
	fmt.Sscan(n, &k)
	return base, k
}

// writeJSONAtomic replaces path in one rename so a dashboard never reads half a file.
func writeJSONAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// writeLive refreshes OUT/alerts.json and OUT/status.txt once (no alert command; a running soak uses a liveWriter).
func writeLive(dir string, ss []Sample, lim Limits, rules AlertRules, now time.Time) error {
	return newLiveWriter(dir, "", nil).write(ss, lim, rules, now)
}

// loadLimits reads OUT/limits.json (written by run) so status and alerts judge with the limits of the run.
func loadLimits(dir string) (Limits, bool) {
	b, err := os.ReadFile(filepath.Join(dir, "limits.json"))
	if err != nil {
		return Limits{}, false
	}
	var l Limits
	if json.Unmarshal(b, &l) != nil {
		return Limits{}, false
	}
	return l, true
}

// loadBotLog reads the bot log path the run recorded in OUT/botlog.txt ("" = none).
func loadBotLog(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "botlog.txt"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// versionFile records which d2bot build a run judged (the soak is only meaningful against a known commit).
const versionFile = "version.txt"

func loadVersion(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, versionFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
