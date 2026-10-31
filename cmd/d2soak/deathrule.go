package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

// Death rule of docs/plan/plan-v2.md section 10.2 (approved 2026-10-30), judged offline from the bot log (death,
// state, area_change and combat lines, minute lines) and, when present, deaths.jsonl. Numbers are calibrated on soak
// attempts 1-14 (docs/plan/death-data.md, V by script); the harness fixtures reproduce that table.

// DeathLimits are the thresholds of the rule. The zero value (an old limits.json) is off; DefaultDeathLimits is on.
type DeathLimits struct {
	Enabled bool

	StreakDeaths int           // FAIL-1: this many deaths ...
	StreakWindow time.Duration // ... within this span with no clvl gain
	HourDeaths   int           // FAIL-1: more than this many deaths in 60 min fail regardless of level gain
	WarnDeaths   int           // WARN: this many deaths in 60 min

	RecoveryFail float64       // FAIL-2: death-recovery share of the rolling 60 min above this
	RecoveryWarn float64       // WARN
	MaxBack      time.Duration // FAIL-2: one death to back in the field above this
	MinRun       time.Duration // the share and rate rules need this much run time

	RatePerK     float64 // FAIL-3: deaths per 1000 kills above this ...
	RateWarnPerK float64
	MinKills     int // ... only with at least this many kills in the window

	RepeatWindow time.Duration // repeat site: two deaths in one area within this ...
	RepeatGap    time.Duration // ... or a death within this of the previous one
	BanWait      time.Duration // the action log line must follow a repeat-site death within this
}

func DefaultDeathLimits() DeathLimits {
	return DeathLimits{
		Enabled:      true,
		StreakDeaths: 3, StreakWindow: 30 * time.Minute, HourDeaths: 4, WarnDeaths: 2,
		RecoveryFail: 0.10, RecoveryWarn: 0.05, MaxBack: 5 * time.Minute, MinRun: 30 * time.Minute,
		RatePerK: 15, RateWarnPerK: 5, MinKills: 150,
		RepeatWindow: 30 * time.Minute, RepeatGap: 180 * time.Second, BanWait: 5 * time.Minute,
	}
}

const (
	deathHour    = time.Hour
	repeatBanMsg = "death: repeat-site ban" // the bot's action line (built by another agent; looked for as text)
)

// recoveryStates are the activity texts that mean the character is recovering from a death (10.2). Prefix match after
// digits became N.
var recoveryStates = []string{
	"Recovering after death",
	"Restocking after death",
	"Retrieving the corpse",
	"Chores in town (poverty rebuild after failed retrieval)",
}

// neverRecovery are states that look like recovery but are not death time: the broke-phase gold farm and a carried-over
// area ban (it occurs in runs with 0 deaths, docs/plan/death-data.md section 1).
var neverRecovery = []string{"Recovery: farming", "Waiting"}

func hasPrefixAny(s string, ps []string) bool {
	s = normStep(s)
	for _, p := range ps {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

var townAreas = map[string]bool{"Rogue Encampment": true, "Lut Gholein": true, "Kurast Docks": true,
	"The Pandemonium Fortress": true, "Harrogath": true}

// DeathEvent is one "character died" line.
type DeathEvent struct {
	At       time.Time
	Area     int
	AreaName string
	Killer   string
}

type activity struct {
	At   time.Time
	Text string
}

// DeathTimeline is what the rule needs from a bot log, in file order.
type DeathTimeline struct {
	Start, End time.Time
	Deaths     []DeathEvent
	Acts       []activity  // runner state lines (activity text)
	Field      []time.Time // combat fight lines and area changes to a non-town area
	Bans       []time.Time // repeat-site ban action lines
}

// ParseDeathTimeline reads JSON or slog text lines; unrelated and malformed lines are skipped. The "death" lines that
// replay old corpse records at start are not "character died" lines and are ignored.
func ParseDeathTimeline(r io.Reader) (DeathTimeline, error) {
	var tl DeathTimeline
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		var kv map[string]string
		switch {
		case strings.HasPrefix(line, "{"):
			kv = jsonKV(line)
		case strings.Contains(line, "time=") && strings.Contains(line, "msg="):
			kv = textKV(line)
		}
		if kv == nil {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, kv["time"])
		if err != nil {
			continue
		}
		if tl.Start.IsZero() {
			tl.Start = at
		}
		tl.End = at
		msg := kv["msg"]
		switch {
		case msg == "character died":
			tl.Deaths = append(tl.Deaths, DeathEvent{At: at, Area: atoiOr(kv["area"], 0), Killer: kv["killer"]})
		case msg == "state" && kv["activity"] != "":
			tl.Acts = append(tl.Acts, activity{at, kv["activity"]})
		case msg == "minute":
			tl.Acts = append(tl.Acts, activity{at, kv["step"]})
		case kv["type"] == "area_change":
			if !townAreas[kv["area"]] {
				tl.Field = append(tl.Field, at)
			}
		case strings.HasPrefix(msg, "combat fight area="):
			tl.Field = append(tl.Field, at)
		case strings.Contains(msg, repeatBanMsg):
			tl.Bans = append(tl.Bans, at)
		}
	}
	return tl, sc.Err()
}

// DeathRecord is a deaths.jsonl line (internal/bot DeathRecord, M0 autopsy fields). A follow-up line carries
// followUp:true and the death time.
type DeathRecord struct {
	Time           time.Time `json:"time"`
	Area           int       `json:"area"`
	FollowUp       bool      `json:"followUp"`
	SecondsToField *int      `json:"secondsToField"`
	GoldNextMinute *int      `json:"goldNextMinute"`
	Autopsy        *struct {
		BeltHP          int  `json:"beltHP"`
		PotionsDrunk60s int  `json:"potionsDrunk60s"`
		GoldCarried     int  `json:"goldCarried"`
		Bug             bool `json:"bug"`
	} `json:"autopsy"`
}

// ParseDeathRecords reads deaths.jsonl and merges each follow-up line into its death by time.
func ParseDeathRecords(r io.Reader) ([]DeathRecord, error) {
	var out []DeathRecord
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var d DeathRecord
		if json.Unmarshal([]byte(strings.TrimSpace(sc.Text())), &d) != nil || d.Time.IsZero() {
			continue
		}
		if !d.FollowUp {
			out = append(out, d)
			continue
		}
		for i := range out {
			if out[i].Time.Equal(d.Time) {
				if d.SecondsToField != nil {
					out[i].SecondsToField = d.SecondsToField
				}
				if d.GoldNextMinute != nil {
					out[i].GoldNextMinute = d.GoldNextMinute
				}
			}
		}
	}
	return out, sc.Err()
}

// DeathInfo is one judged death.
type DeathInfo struct {
	At          time.Time
	Area        int
	Killer      string
	Back        time.Duration // death to back in the field
	Open        bool          // never back in the field before the log ended: Back is a lower bound
	FromRecord  bool          // Back comes from deaths.jsonl secondsToField
	Repeat      bool          // repeat site: the bot should have banned the area
	Banned      bool          // the ban action line followed
	Bug         bool          // beltHP >= 1 and no potion in the last 60 s (needs deaths.jsonl)
	GoldCarried int
}

// DeathReport is the outcome: findings plus numbers for the text report and report.json.
type DeathReport struct {
	Findings  []Finding   `json:"-"`
	Deaths    []DeathInfo `json:"-"`
	Summary   DeathStats  `json:"summary"`
	Evaluated bool        `json:"-"`
}

// DeathStats are the report-only and peak numbers.
type DeathStats struct {
	Deaths       int     `json:"deaths"`
	BugDeaths    int     `json:"bugDeaths"`
	RepeatSites  int     `json:"repeatSites"`
	RepeatBanned int     `json:"repeatSitesBanned"`
	PeakShare    float64 `json:"peakRecoveryShare"`
	PeakRate     float64 `json:"peakDeathsPerKKills"`
	MaxBackSecs  int     `json:"maxBackInFieldSeconds"`
	GoldCarried  int     `json:"goldCarriedAtDeaths"`
	GoldEarned   int     `json:"goldEarned"`
	ShareEnd     float64 `json:"-"` // recovery share and deaths per 1000 kills of the 60 min ending at the log end (live summary)
	RateEnd      float64 `json:"-"`
}

type span struct{ a, b time.Time }

// mergeSpans sorts and unions spans.
func mergeSpans(in []span) []span {
	var s []span
	for _, x := range in {
		if x.b.After(x.a) {
			s = append(s, x)
		}
	}
	sort.Slice(s, func(i, j int) bool { return s[i].a.Before(s[j].a) })
	var out []span
	for _, x := range s {
		if n := len(out); n > 0 && !x.a.After(out[n-1].b) {
			if x.b.After(out[n-1].b) {
				out[n-1].b = x.b
			}
			continue
		}
		out = append(out, x)
	}
	return out
}

// subtractSpans removes cut (merged) from in (merged).
func subtractSpans(in, cut []span) []span {
	var out []span
	for _, x := range in {
		cur := x.a
		for _, c := range cut {
			if !c.b.After(cur) || !c.a.Before(x.b) {
				continue
			}
			if c.a.After(cur) {
				out = append(out, span{cur, c.a})
			}
			if c.b.After(cur) {
				cur = c.b
			}
		}
		if cur.Before(x.b) {
			out = append(out, span{cur, x.b})
		}
	}
	return out
}

func overlapSpans(s []span, lo, hi time.Time) time.Duration {
	var d time.Duration
	for _, x := range s {
		a, b := x.a, x.b
		if a.Before(lo) {
			a = lo
		}
		if b.After(hi) {
			b = hi
		}
		if b.After(a) {
			d += b.Sub(a)
		}
	}
	return d
}

// actSpans turns the activity changes into spans of one class each: recovery states and never-counted states.
func actSpans(acts []activity, end time.Time) (rec, never []span) {
	sorted := append([]activity(nil), acts...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].At.Before(sorted[j].At) })
	for i, a := range sorted {
		e := end
		if i+1 < len(sorted) {
			e = sorted[i+1].At
		}
		switch {
		case hasPrefixAny(a.Text, recoveryStates):
			rec = append(rec, span{a.At, e})
		case hasPrefixAny(a.Text, neverRecovery):
			never = append(never, span{a.At, e})
		}
	}
	return mergeSpans(rec), mergeSpans(never)
}

// AnalyzeDeaths applies the rule. recs may be nil. ms are the minute lines (clvl, kills, gold).
func AnalyzeDeaths(tl DeathTimeline, ms []Minute, recs []DeathRecord, lim DeathLimits) DeathReport {
	var dr DeathReport
	if !lim.Enabled {
		return dr
	}
	dr.Evaluated = true
	// active: the condition still holds at the end of the log (live alerts keep only those, see Finding.Active). Rules
	// that look at a window count as active while the window ending now still trips them.
	add := func(active bool, level, code, f string, a ...any) {
		dr.Findings = append(dr.Findings, Finding{Level: level, Code: code, Msg: fmt.Sprintf(f, a...), Active: active})
	}
	start, end := tl.Start, tl.End
	if start.IsZero() && len(ms) > 0 {
		start, end = ms[0].At, ms[len(ms)-1].At
	}
	events := append([]DeathEvent(nil), tl.Deaths...)
	for _, r := range recs { // deaths only in deaths.jsonl (no log line within 5 s), inside the judged span
		if r.Time.Before(start) || r.Time.After(end) {
			continue
		}
		dup := false
		for _, e := range events {
			if d := e.At.Sub(r.Time); d < 5*time.Second && d > -5*time.Second {
				dup = true
			}
		}
		if !dup {
			events = append(events, DeathEvent{At: r.Time, Area: r.Area})
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].At.Before(events[j].At) })
	field := append([]time.Time(nil), tl.Field...)
	sort.Slice(field, func(i, j int) bool { return field[i].Before(field[j]) })
	recSpans, never := actSpans(tl.Acts, end)

	// Per death: window = death to the first fight or non-town area change (or the next death, or the log end).
	var windows []span
	for i, e := range events {
		di := DeathInfo{At: e.At, Area: e.Area, Killer: e.Killer}
		limit, open := end, true
		if i+1 < len(events) {
			limit = events[i+1].At
		}
		for _, f := range field {
			if f.After(e.At) {
				if f.Before(limit) {
					limit, open = f, false
				}
				break
			}
		}
		if open && i+1 < len(events) {
			open = false // the next death ended it
		}
		di.Back, di.Open = limit.Sub(e.At), open
		var rec *DeathRecord
		for k := range recs {
			if d := recs[k].Time.Sub(e.At); d < 5*time.Second && d > -5*time.Second {
				rec = &recs[k]
			}
		}
		if rec != nil {
			if rec.SecondsToField != nil {
				di.Back, di.Open, di.FromRecord = time.Duration(*rec.SecondsToField)*time.Second, false, true
				limit = e.At.Add(di.Back)
			}
			if a := rec.Autopsy; a != nil {
				di.Bug = a.Bug || (a.BeltHP >= 1 && a.PotionsDrunk60s == 0)
				di.GoldCarried = a.GoldCarried
			}
		}
		windows = append(windows, span{e.At, limit})
		dr.Deaths = append(dr.Deaths, di)
	}
	// Recovery time = windows without the never-counted states, plus the recovery-state spans themselves.
	recovery := mergeSpans(append(subtractSpans(mergeSpans(windows), never), recSpans...))

	// Repeat site.
	for i := range dr.Deaths {
		d := &dr.Deaths[i]
		for j := 0; j < i; j++ {
			p := dr.Deaths[j]
			if d.At.Sub(p.At) <= lim.RepeatWindow && p.Area == d.Area && d.Area != 0 {
				d.Repeat = true
			}
		}
		if i > 0 && d.At.Sub(dr.Deaths[i-1].At) <= lim.RepeatGap {
			d.Repeat = true
		}
		if d.Repeat {
			for _, b := range tl.Bans {
				if !b.Before(d.At) && b.Sub(d.At) <= lim.BanWait {
					d.Banned = true
				}
			}
		}
	}

	st := &dr.Summary
	st.Deaths = len(dr.Deaths)
	for _, d := range dr.Deaths {
		if d.Bug {
			st.BugDeaths++
		}
		if d.Repeat {
			st.RepeatSites++
			if d.Banned {
				st.RepeatBanned++
			}
		}
		st.GoldCarried += d.GoldCarried
		if s := int(d.Back / time.Second); s > st.MaxBackSecs {
			st.MaxBackSecs = s
		}
	}
	for i := 1; i < len(ms); i++ {
		if g := ms[i].Gold - ms[i-1].Gold; g > 0 && ms[i].At.Sub(ms[i-1].At) <= maxSampleGap && ms[i].Deaths == ms[i-1].Deaths {
			st.GoldEarned += g
		}
	}

	// FAIL-1: streak.
	at := func(t time.Time) string { return t.Format("15:04:05") }
	// nDeaths: some run of n deaths inside span ends within that span of the log end (the rule trips in a window now).
	nDeaths := func(n int, span time.Duration, noGain bool) bool {
		for i := n - 1; n > 1 && i < len(dr.Deaths); i++ {
			first, last := dr.Deaths[i-n+1].At, dr.Deaths[i].At
			if end.Sub(last) <= span && last.Sub(first) <= span && (!noGain || clvlAt(ms, last) <= clvlAt(ms, first.Add(-time.Second))) {
				return true
			}
		}
		return false
	}
	streak := false
	for i := lim.StreakDeaths - 1; lim.StreakDeaths > 1 && i < len(dr.Deaths) && !streak; i++ {
		first, last := dr.Deaths[i-lim.StreakDeaths+1].At, dr.Deaths[i].At
		if last.Sub(first) <= lim.StreakWindow && clvlAt(ms, last) <= clvlAt(ms, first.Add(-time.Second)) {
			streak = true
			add(nDeaths(lim.StreakDeaths, lim.StreakWindow, true), "fail", "death_streak", "%d deaths in %s (%s to %s) with no clvl gain (limit %d in %s)", lim.StreakDeaths, mins(last.Sub(first)), at(first), at(last), lim.StreakDeaths, mins(lim.StreakWindow))
		}
	}
	hour := lim.HourDeaths + 1
	for i := hour - 1; lim.HourDeaths > 0 && i < len(dr.Deaths) && !streak; i++ {
		if first, last := dr.Deaths[i-hour+1].At, dr.Deaths[i].At; last.Sub(first) <= deathHour {
			streak = true
			add(nDeaths(hour, deathHour, false), "fail", "death_streak", "%d deaths in %s (%s to %s), more than %d in 60 min fails whatever the level gain", hour, mins(last.Sub(first)), at(first), at(last), lim.HourDeaths)
		}
	}
	if !streak && lim.WarnDeaths > 1 {
		for i := lim.WarnDeaths - 1; i < len(dr.Deaths); i++ {
			if first, last := dr.Deaths[i-lim.WarnDeaths+1].At, dr.Deaths[i].At; last.Sub(first) <= deathHour {
				add(nDeaths(lim.WarnDeaths, deathHour, false), "warn", "death_pair", "%d deaths in %s (%s to %s)", lim.WarnDeaths, mins(last.Sub(first)), at(first), at(last))
				break
			}
		}
	}

	// FAIL-2: one death stuck, and the rolling recovery share.
	stuck := false
	if lim.MaxBack > 0 {
		for _, d := range dr.Deaths {
			if d.Back > lim.MaxBack {
				stuck = true
				why := "back in the field after"
				if d.Open {
					why = "not back in the field when the log ended after"
				}
				add(d.Open || end.Sub(d.At.Add(d.Back)) <= deathHour, "fail", "death_recovery_stuck", "death at %s: %s %s (limit %s)", at(d.At), why, mins(d.Back), mins(lim.MaxBack))
				break
			}
		}
	}
	// kills so far, monotonic (a drop is a counter reset)
	type kp struct {
		at time.Time
		k  int
	}
	var kills []kp
	acc := 0
	for i, m := range ms {
		if i > 0 && m.Kills > ms[i-1].Kills {
			acc += m.Kills - ms[i-1].Kills
		}
		kills = append(kills, kp{m.At, acc})
	}
	killsAt := func(t time.Time) int {
		k := 0
		for _, p := range kills {
			if p.at.After(t) {
				break
			}
			k = p.k
		}
		return k
	}
	var shareAt, rateAt time.Time
	var shareNow, rateNow float64 // the 60 min window ending at the log end
	if lim.MinRun > 0 {
		for t := start.Add(lim.MinRun); ; t = t.Add(time.Minute) {
			if t.After(end) {
				t = end
			}
			if t.Sub(start) >= lim.MinRun {
				lo := t.Add(-deathHour)
				if lo.Before(start) {
					lo = start
				}
				sh := float64(overlapSpans(recovery, lo, t)) / float64(deathHour)
				shareNow, rateNow = sh, 0
				if sh > st.PeakShare {
					st.PeakShare, shareAt = sh, t
				}
				n := 0
				for _, d := range dr.Deaths {
					if !d.At.Before(lo) && !d.At.After(t) {
						n++
					}
				}
				if k := killsAt(t) - killsAt(lo); k >= lim.MinKills && lim.MinKills > 0 {
					rate := 1000 * float64(n) / float64(k)
					rateNow = rate
					if rate > st.PeakRate {
						st.PeakRate, rateAt = rate, t
					}
				}
			}
			if !t.Before(end) {
				break
			}
		}
	}
	switch {
	case lim.RecoveryFail > 0 && st.PeakShare > lim.RecoveryFail:
		if !stuck {
			add(shareNow > lim.RecoveryFail, "fail", "death_recovery_stuck", "death-recovery time %.1f%% of the 60 min ending %s (limit %.0f%%)", st.PeakShare*100, at(shareAt), lim.RecoveryFail*100)
		}
	case lim.RecoveryWarn > 0 && st.PeakShare > lim.RecoveryWarn:
		add(shareNow > lim.RecoveryWarn, "warn", "death_recovery_share", "death-recovery time %.1f%% of the 60 min ending %s (warn above %.0f%%)", st.PeakShare*100, at(shareAt), lim.RecoveryWarn*100)
	}

	// FAIL-3: rate.
	switch {
	case lim.RatePerK > 0 && st.PeakRate > lim.RatePerK:
		add(rateNow > lim.RatePerK, "fail", "death_rate", "%.1f deaths per 1000 kills in the 60 min ending %s (limit %.0f, needs %d kills in the window)", st.PeakRate, at(rateAt), lim.RatePerK, lim.MinKills)
	case lim.RateWarnPerK > 0 && st.PeakRate > lim.RateWarnPerK:
		add(rateNow > lim.RateWarnPerK, "warn", "death_rate", "%.1f deaths per 1000 kills in the 60 min ending %s (warn above %.0f)", st.PeakRate, at(rateAt), lim.RateWarnPerK)
	}

	// Action expected (not a verdict).
	st.ShareEnd, st.RateEnd = shareNow, rateNow
	if st.RepeatSites > st.RepeatBanned {
		recent := false
		for _, d := range dr.Deaths {
			recent = recent || (d.Repeat && !d.Banned && end.Sub(d.At) <= deathHour)
		}
		add(recent, "warn", "death_repeat_site_no_action", "%d repeat-site death(s), %d followed by a %q log line: the area ban is missing", st.RepeatSites, st.RepeatBanned, repeatBanMsg)
	}
	return dr
}

// Text prints the death-rule numbers ("" when the rule was off or there was nothing to say).
func (dr DeathReport) Text() string {
	if !dr.Evaluated {
		return ""
	}
	st := dr.Summary
	var b strings.Builder
	fmt.Fprintf(&b, "  death rule   %d deaths, peak recovery share %.1f%%, peak %.1f deaths/1000 kills, longest death to field %ds, bug deaths %d (report only), repeat sites %d (action expected, %d banned)\n",
		st.Deaths, st.PeakShare*100, st.PeakRate, st.MaxBackSecs, st.BugDeaths, st.RepeatSites, st.RepeatBanned)
	if st.GoldCarried > 0 {
		share := "n/a"
		if st.GoldEarned > 0 {
			share = fmt.Sprintf("%.0f%%", 100*float64(st.GoldCarried)/float64(st.GoldEarned))
		}
		fmt.Fprintf(&b, "  gold lost    %d carried at deaths of %d earned = %s (report only)\n", st.GoldCarried, st.GoldEarned, share)
	}
	for _, d := range dr.Deaths {
		tags := ""
		if d.Open {
			tags += " OPEN"
		}
		if d.Repeat {
			tags += " repeat-site"
			if d.Banned {
				tags += "(banned)"
			}
		}
		if d.Bug {
			tags += " BUG"
		}
		fmt.Fprintf(&b, "    %s area %d %s back in field %ds%s\n", d.At.Format("15:04:05"), d.Area, d.Killer, int(d.Back/time.Second), tags)
	}
	return b.String()
}

// deathsFromLog reads the bot log (and deaths.jsonl, when given and present) and judges the death rule.
func deathsFromLog(logPath, deathsPath string, lim DeathLimits) DeathReport {
	if !lim.Enabled || logPath == "" {
		return DeathReport{}
	}
	f, err := os.Open(resolveBotLog(logPath))
	if err != nil {
		return DeathReport{}
	}
	defer f.Close()
	tl, _ := ParseDeathTimeline(f)
	f.Seek(0, io.SeekStart)
	ms, _ := ParseMinuteLines(f)
	var recs []DeathRecord
	if deathsPath != "" {
		if g, err := os.Open(deathsPath); err == nil {
			recs, _ = ParseDeathRecords(g)
			g.Close()
		}
	}
	return AnalyzeDeaths(tl, ms, recs, lim)
}
