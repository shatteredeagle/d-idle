package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Live judging of the bot log: the minute-line rules (progress.go) and the death rule (deathrule.go) run on every
// alert refresh of a running soak, through the same function `d2soak report` uses (judgeLog), so the live alerts and
// the final report cannot disagree. docs/soak-analysis.md section 12.

// logJudgement is everything the two rule sets found in one bot log.
type logJudgement struct {
	Minutes  []Minute
	Progress ProgressReport
	Death    DeathReport
}

// Findings are the progress findings followed by the death findings, the order of the report.
func (j logJudgement) Findings() []Finding {
	return append(append([]Finding(nil), j.Progress.Findings...), j.Death.Findings...)
}

// judgeLogData parses a bot log (JSON or slog text) from r and judges it. recs may be nil.
func judgeLogData(r io.ReadSeeker, recs []DeathRecord, lim Limits) logJudgement {
	var j logJudgement
	tl, _ := ParseDeathTimeline(r)
	r.Seek(0, io.SeekStart)
	j.Minutes, _ = ParseMinuteLines(r)
	j.Progress = AnalyzeMinutes(j.Minutes, lim.Progress)
	j.Death = AnalyzeDeaths(tl, j.Minutes, recs, lim.Death)
	return j
}

type judgeKey struct {
	path, deaths string
	size         int64
	mtime        time.Time
	dsize        int64
	dtime        time.Time
	lim          Limits
}

var judgeCache = struct {
	sync.Mutex
	m map[string]struct {
		k judgeKey
		j logJudgement
	}
}{m: map[string]struct {
	k judgeKey
	j logJudgement
}{}}

func statOf(p string) (int64, time.Time) {
	if st, err := os.Stat(p); err == nil {
		return st.Size(), st.ModTime()
	}
	return -1, time.Time{}
}

// judgeLog judges the bot log at path (a file, or a directory whose newest *.log is used) and, when present,
// deathsPath. A missing log judges nothing. The result is kept while neither file changes: the sidecar asks every
// ~2 minutes and a 24 h log is big.
func judgeLog(path, deathsPath string, lim Limits) logJudgement {
	if path == "" {
		return logJudgement{}
	}
	path = resolveBotLog(path)
	k := judgeKey{path: path, deaths: deathsPath, lim: lim}
	k.size, k.mtime = statOf(path)
	k.dsize, k.dtime = statOf(deathsPath)
	if k.size < 0 {
		return logJudgement{}
	}
	judgeCache.Lock()
	if c, ok := judgeCache.m[path]; ok && c.k == k {
		judgeCache.Unlock()
		return c.j
	}
	judgeCache.Unlock()
	b, err := os.ReadFile(path)
	if err != nil {
		return logJudgement{}
	}
	var recs []DeathRecord
	if deathsPath != "" {
		if g, err := os.Open(deathsPath); err == nil {
			recs, _ = ParseDeathRecords(g)
			g.Close()
		}
	}
	j := judgeLogData(bytes.NewReader(b), recs, lim)
	judgeCache.Lock()
	judgeCache.m[path] = struct {
		k judgeKey
		j logJudgement
	}{k, j}
	judgeCache.Unlock()
	return j
}

// findingAlerts turns the active findings into alerts: fail is crit, warn stays warn. One alert per code (idle_in_town
// can have several stretches): the newest active one, the crit one when both levels exist.
func findingAlerts(fs []Finding) []Alert {
	by := map[string]Alert{}
	var codes []string
	for _, f := range fs {
		if !f.Active {
			continue
		}
		a := Alert{Level: "warn", Code: f.Code, Msg: f.Msg}
		if f.Level == "fail" {
			a.Level = "crit"
		}
		old, seen := by[f.Code]
		if !seen {
			codes = append(codes, f.Code)
		}
		if !seen || old.Level != "crit" || a.Level == "crit" {
			by[f.Code] = a
		}
	}
	sort.Strings(codes)
	out := make([]Alert, 0, len(codes))
	for _, c := range codes {
		out = append(out, by[c])
	}
	return out
}

// logFindingAlerts is the live alert set from the bot log of the rules.
func logFindingAlerts(rules AlertRules, lim Limits) ([]Alert, logJudgement) {
	if rules.BotLog == "" {
		return nil, logJudgement{}
	}
	j := judgeLog(rules.BotLog, rules.DeathsFile, lim)
	return findingAlerts(j.Findings()), j
}

// liveSummary is the compact one-liner of the last 60 minutes of the bot log: clvl, xp/h, deaths, recovery share and
// the busiest state group. "" without minute lines.
func liveSummary(j logJudgement) string {
	ms := j.Minutes
	if len(ms) == 0 {
		return ""
	}
	last := ms[len(ms)-1]
	lo := last.At.Add(-time.Hour)
	var xp uint64
	var span, top time.Duration
	deaths := 0
	groups := map[string]time.Duration{}
	for i := 0; i+1 < len(ms); i++ {
		a, b := ms[i], ms[i+1]
		if a.At.Before(lo) {
			continue
		}
		d := b.At.Sub(a.At)
		if d <= 0 || d > maxSampleGap {
			continue
		}
		span += d
		g, _ := stepGroup(a.Step)
		groups[g] += d
		if b.XP > a.XP && b.Clvl >= a.Clvl {
			xp += b.XP - a.XP
		}
		if b.Deaths > a.Deaths {
			deaths += b.Deaths - a.Deaths
		}
	}
	rate, topName := "?", "-"
	if span >= 5*time.Minute {
		rate = fmt.Sprintf("%.0f", float64(xp)/span.Hours())
	}
	for _, g := range groupOrder {
		if groups[g] > top {
			top, topName = groups[g], g
		}
	}
	share := "n/a"
	if j.Death.Evaluated {
		share = fmt.Sprintf("%.1f%%", j.Death.Summary.ShareEnd*100)
	}
	topShare := 0.0
	if span > 0 {
		topShare = 100 * float64(top) / float64(span)
	}
	return fmt.Sprintf("live %s: clvl %d, %s xp/h, deaths %d, recovery %s, top state %s %.0f%%",
		mins(span), last.Clvl, rate, deaths, share, topName, topShare)
}

// withProfileDefaults fills a limits block that is entirely zero (a Limits built without the profile, or from an old
// limits.json) with the defaults, so the sidecar always judges the progress and death rules.
func withProfileDefaults(l Limits) Limits {
	if l.Progress == (ProgressLimits{}) {
		l.Progress = DefaultProgressLimits()
	}
	if l.Death == (DeathLimits{}) {
		l.Death = DefaultDeathLimits()
	}
	return l
}

// alertTracker gives every alert the time it first appeared in an unbroken stretch and reports the crit alerts that
// are new (not crit in the previous refresh). State survives a restart through the alerts.json already in the run
// directory.
type alertTracker struct {
	since map[string]time.Time
	level map[string]string
}

func newTracker(dir string) *alertTracker {
	t := &alertTracker{since: map[string]time.Time{}, level: map[string]string{}}
	b, err := os.ReadFile(filepath.Join(dir, "alerts.json"))
	if err != nil {
		return t
	}
	var a Alerts
	if json.Unmarshal(b, &a) == nil {
		for _, x := range a.Alerts {
			if !x.Since.IsZero() {
				t.since[x.Code], t.level[x.Code] = x.Since, x.Level
			}
		}
	}
	return t
}

// apply sets Since on every alert of a, forgets the alerts that are gone, and returns the new crit alerts.
func (t *alertTracker) apply(a *Alerts, now time.Time) (raised []Alert) {
	cur := map[string]bool{}
	for i := range a.Alerts {
		x := &a.Alerts[i]
		cur[x.Code] = true
		if s, ok := t.since[x.Code]; ok {
			x.Since = s
		} else {
			x.Since = now
			t.since[x.Code] = now
		}
		if x.Level == "crit" && t.level[x.Code] != "crit" {
			raised = append(raised, *x)
		}
		t.level[x.Code] = x.Level
	}
	for c := range t.since {
		if !cur[c] {
			delete(t.since, c)
			delete(t.level, c)
		}
	}
	return raised
}

// liveWriter refreshes alerts.json and status.txt of one run and fires the alert command on new crit alerts.
type liveWriter struct {
	dir  string
	tr   *alertTracker
	cmd  string // D2SOAK_ALERT_CMD, "" = none
	log  func(string, ...any)
	wait sync.WaitGroup
}

func newLiveWriter(dir, cmd string, log func(string, ...any)) *liveWriter {
	if log == nil {
		log = func(string, ...any) {}
	}
	return &liveWriter{dir: dir, tr: newTracker(dir), cmd: cmd, log: log}
}

func (w *liveWriter) write(ss []Sample, lim Limits, rules AlertRules, now time.Time) error {
	rules.DeathsFile = filepath.Join(w.dir, "deaths.jsonl")
	a := EvalAlerts(ss, lim, rules, now)
	raised := w.tr.apply(&a, now)
	status := a.Status + "\n"
	if x := ExtraStatus(ss, rules, now); x != "" {
		status += x + "\n"
	}
	if a.Summary != "" {
		status += a.Summary + "\n"
	}
	if err := os.WriteFile(filepath.Join(w.dir, "status.txt"), []byte(status), 0o644); err != nil {
		return err
	}
	err := writeJSONAtomic(filepath.Join(w.dir, "alerts.json"), a)
	for _, x := range raised {
		w.log("d2soak: NEW FAIL alert %s: %s", x.Code, x.Msg)
		w.runHook(x, a.Status)
	}
	return err
}

// Close waits for alert commands still running.
func (w *liveWriter) Close() { w.wait.Wait() }
