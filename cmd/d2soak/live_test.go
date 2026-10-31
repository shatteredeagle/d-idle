package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type event struct {
	line  int
	code  string
	level string
	raise bool
}

// replay feeds a bot log fixture to the live judge one line at a time (what the sidecar sees as the log grows) and
// returns the alert timeline plus the active alerts after the last line.
func replay(t *testing.T, name string, lim Limits) (tl []event, final []Alert) {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	prev := map[string]string{}
	end := 0
	var acc []byte
	all := strings.SplitAfter(string(b), "\n")
	step := 1 + len(all)/150 // long fixtures are fed in small batches to keep the test fast
	for _, l := range all {
		end++
		acc = append(acc, l...)
		if strings.TrimSpace(l) == "" || (end%step != 0 && end != len(all)-1) {
			continue
		}
		j := judgeLogData(bytes.NewReader(acc), nil, lim)
		cur := map[string]string{}
		final = findingAlerts(j.Findings())
		for _, a := range final {
			cur[a.Code] = a.Level
			if prev[a.Code] != a.Level {
				tl = append(tl, event{end, a.Code, a.Level, true})
			}
		}
		for c, lv := range prev {
			if _, ok := cur[c]; !ok {
				tl = append(tl, event{end, c, lv, false})
			}
		}
		prev = cur
	}
	return tl, final
}

func firstRaise(tl []event, code, level string) int {
	for _, e := range tl {
		if e.raise && e.code == code && e.level == level {
			return e.line
		}
	}
	return -1
}

func firstClear(tl []event, code string) int {
	for _, e := range tl {
		if !e.raise && e.code == code {
			return e.line
		}
	}
	return 1 << 30
}

func cleared(tl []event, code string) bool {
	for _, e := range tl {
		if !e.raise && e.code == code {
			return true
		}
	}
	return false
}

func liveLimits() Limits {
	l := withProfileDefaults(DefaultLimits())
	// a14 gains xp until 00:32, then trickles 25 xp in 25 minutes (idle 00:47-00:57) until a real gain at 01:01
	l.Progress.NoGain = 25 * time.Minute
	l.Progress.IdleInTown = 8 * time.Minute
	return l
}

// V: a14 minute lines (docs/g1-log.md): the waiting stretch raises idle_in_town and no_gain while it lasts; the
// stretch ends before the log does, so idle clears and no_gain (nothing gained since) stays.
func TestLiveIdlePeriodRaisesAndClears(t *testing.T) {
	tl, final := replay(t, "soak_a14_minutes.jsonl", liveLimits())
	idle, noGain := firstRaise(tl, "idle_in_town", "crit"), firstRaise(tl, "no_gain", "crit")
	if idle < 0 || noGain < 0 {
		t.Fatalf("timeline %+v", tl)
	}
	if !cleared(tl, "idle_in_town") {
		t.Errorf("idle_in_town must clear once the bot works again: %+v", tl)
	}
	if firstClear(tl, "no_gain") < firstClear(tl, "idle_in_town") {
		t.Errorf("no_gain must stay at least as long as the idle: %+v", tl)
	}
	if firstRaise(tl, "loop_no_gain", "crit") >= 0 {
		t.Errorf("no loop expected: %+v", tl)
	}
	if len(final) == 0 {
		t.Fatal("no active alert at the end")
	}
}

// A healthy start raises no fail at all.
func TestLiveHealthyStartIsQuiet(t *testing.T) {
	b, _ := os.ReadFile("testdata/soak_a14_minutes.jsonl")
	lines := strings.SplitAfter(string(b), "\n")
	j := judgeLogData(strings.NewReader(strings.Join(lines[:25], "")), nil, liveLimits())
	for _, a := range findingAlerts(j.Findings()) {
		if a.Level == "crit" {
			t.Fatalf("healthy start: %+v", a)
		}
	}
}

// V: the order death alerts appear in matches the oracle of the real attempts (docs/plan/death-data.md): a warning pair
// first, the fail once the third death lands; a stuck recovery follows its streak.
func TestLiveDeathRuleOrder(t *testing.T) {
	lim := liveLimits()
	lim.Progress = ProgressLimits{} // the death fixtures are excerpts, judge only the death rule here
	for _, c := range []struct {
		file         string
		order        [][2]string // code, level: first raise strictly increasing
		neverCrit    bool
		clearedCodes []string
	}{
		{file: "soak_a01_deathrule.jsonl", order: [][2]string{{"death_pair", "warn"}, {"death_streak", "crit"}}},
		{file: "soak_a12_deathrule.jsonl", order: [][2]string{{"death_pair", "warn"}, {"death_streak", "crit"}, {"death_recovery_stuck", "crit"}}},
		{file: "soak_a11_deathrule.jsonl", order: [][2]string{{"death_pair", "warn"}, {"death_streak", "crit"}, {"death_rate", "crit"}}},
		{file: "soak_a08_deathrule.jsonl", neverCrit: true, clearedCodes: []string{"death_recovery_share", "death_rate", "death_repeat_site_no_action"}},
		{file: "soak_a14_deathrule.jsonl"},
	} {
		tl, final := replay(t, c.file, lim)
		at := 0
		for _, o := range c.order {
			n := firstRaise(tl, o[0], o[1])
			if n < 0 || n < at {
				t.Errorf("%s: %s/%s raised at line %d, after line %d expected: %+v", c.file, o[0], o[1], n, at, tl)
			}
			at = n
		}
		for _, e := range tl {
			if c.neverCrit && e.raise && e.level == "crit" {
				t.Errorf("%s: unexpected fail %+v", c.file, e)
			}
		}
		for _, code := range c.clearedCodes {
			if !cleared(tl, code) {
				t.Errorf("%s: %s should have cleared: %+v", c.file, code, tl)
			}
		}
		if c.file == "soak_a14_deathrule.jsonl" && len(tl) != 0 {
			t.Errorf("clean run raised %+v", tl)
		}
		_ = final
	}
}

// The live timeline and the report agree: every fail the final report lists was raised live at some point, and what is
// still active after the last line is a subset of the report's findings (the report adds the resolved ones).
func TestLiveTimelineMatchesReport(t *testing.T) {
	lim := liveLimits()
	for _, f := range []string{"soak_a01_deathrule.jsonl", "soak_a08_deathrule.jsonl", "soak_a11_deathrule.jsonl", "soak_a12_deathrule.jsonl", "soak_a14_deathrule.jsonl", "soak_a14_minutes.jsonl"} {
		tl, final := replay(t, f, lim)
		rep := judgeLog("testdata/"+f, "", lim).Findings()
		in := map[string]bool{}
		for _, x := range rep {
			in[x.Code] = true
			if x.Level == "fail" && firstRaise(tl, x.Code, "crit") < 0 {
				t.Errorf("%s: report fail %s was never raised live: %+v", f, x.Code, tl)
			}
		}
		for _, a := range final {
			if !in[a.Code] {
				t.Errorf("%s: live alert %s not in the report", f, a.Code)
			}
		}
	}
}

func TestFindingAlertsMerge(t *testing.T) {
	got := findingAlerts([]Finding{
		{Level: "fail", Code: "idle_in_town", Msg: "old", Active: false},
		{Level: "fail", Code: "idle_in_town", Msg: "new", Active: true},
		{Level: "warn", Code: "death_pair", Msg: "p", Active: true},
		{Level: "warn", Code: "state_dominant", Msg: "s", Active: false},
	})
	if len(got) != 2 || got[0].Code != "death_pair" || got[1].Code != "idle_in_town" || got[1].Msg != "new" || got[1].Level != "crit" {
		t.Fatalf("%+v", got)
	}
}

func TestLiveSummaryLine(t *testing.T) {
	b, _ := os.ReadFile("testdata/soak_a14_minutes.jsonl")
	j := judgeLogData(bytes.NewReader(b), nil, liveLimits())
	s := liveSummary(j)
	for _, want := range []string{"live ", "clvl ", " xp/h", "deaths 0", "recovery ", "top state "} {
		if !strings.Contains(s, want) {
			t.Errorf("summary %q lacks %q", s, want)
		}
	}
	if liveSummary(logJudgement{}) != "" {
		t.Error("no lines, no summary")
	}
}

func TestWithProfileDefaults(t *testing.T) {
	l := withProfileDefaults(Limits{})
	if !l.Death.Enabled || l.Progress.NoGain == 0 {
		t.Fatalf("%+v", l)
	}
	off := DefaultLimits()
	off.Death = DefaultDeathLimits()
	off.Death.Enabled = false
	if withProfileDefaults(off).Death.Enabled {
		t.Fatal("an explicit block (rule off) must stay as given")
	}
}

func liveFixture(t *testing.T) (dir, logPath string, lines []string, now time.Time) {
	t.Helper()
	b, _ := os.ReadFile("testdata/soak_a14_minutes.jsonl")
	lines = strings.SplitAfter(string(b), "\n")
	dir = t.TempDir()
	logPath = filepath.Join(dir, "bot.log")
	now = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	return
}

func liveRules(logPath string) AlertRules {
	return AlertRules{BotLog: logPath} // every other rule off
}

func readAlerts(t *testing.T, dir string) Alerts {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "alerts.json"))
	if err != nil {
		t.Fatal(err)
	}
	var a Alerts
	if err := json.Unmarshal(b, &a); err != nil {
		t.Fatal(err)
	}
	return a
}

// The writer keeps `since` while an alert stays, drops it when resolved, writes the summary into alerts.json and
// status.txt, and runs the alert command once per NEW fail alert.
func TestLiveWriterSinceClearAndHook(t *testing.T) {
	dir, logPath, lines, now := liveFixture(t)
	hookOut := filepath.Join(dir, "hook.txt")
	w := newLiveWriter(dir, `echo "$D2SOAK_ALERT_LEVEL $D2SOAK_ALERT_CODE" >> `+hookOut, t.Logf)
	lim := liveLimits()
	ss := []Sample{{T: now, HealthOK: true, Progress: -1}}
	step := func(n int, at time.Time) Alerts {
		t.Helper()
		if err := os.WriteFile(logPath, []byte(strings.Join(lines[:n], "")), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := w.write(ss, lim, liveRules(logPath), at); err != nil {
			t.Fatal(err)
		}
		w.wait.Wait()
		return readAlerts(t, dir)
	}
	find := func(a Alerts, code string) *Alert {
		for i := range a.Alerts {
			if a.Alerts[i].Code == code {
				return &a.Alerts[i]
			}
		}
		return nil
	}
	a := step(57, now) // before no_gain
	if find(a, "no_gain") != nil || a.Level == "crit" {
		t.Fatalf("early: %+v", a)
	}
	a = step(60, now.Add(2*time.Minute))
	ng := find(a, "no_gain")
	if ng == nil || ng.Level != "crit" || a.Level != "crit" || !ng.Since.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("raised: %+v", a)
	}
	if a.Summary == "" || !strings.Contains(a.Summary, "clvl") {
		t.Errorf("summary %q", a.Summary)
	}
	a = step(61, now.Add(4*time.Minute))
	if ng = find(a, "no_gain"); ng == nil || !ng.Since.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("since must stay: %+v", a)
	}
	if st, _ := os.ReadFile(filepath.Join(dir, "status.txt")); !strings.Contains(string(st), "live ") {
		t.Errorf("status.txt lacks the live summary: %s", st)
	}
	hook, _ := os.ReadFile(hookOut)
	if strings.Count(string(hook), "fail no_gain") != 1 {
		t.Fatalf("hook runs: %q (one per new fail alert)", hook)
	}
	// a restarted writer reads since back from alerts.json and does not raise again
	w2 := newLiveWriter(dir, `echo again >> `+hookOut, t.Logf)
	w2.write(ss, lim, liveRules(logPath), now.Add(6*time.Minute))
	w2.Close()
	if b, _ := os.ReadFile(hookOut); strings.Contains(string(b), "again") {
		t.Errorf("restart re-fired the hook: %q", b)
	}
	// resolved: a shorter log (nothing active) clears it, a new raise gets a new since and fires again
	a = step(10, now.Add(8*time.Minute))
	if find(a, "no_gain") != nil {
		t.Fatalf("not cleared: %+v", a)
	}
	a = step(60, now.Add(10*time.Minute))
	if ng = find(a, "no_gain"); ng == nil || !ng.Since.Equal(now.Add(10*time.Minute)) {
		t.Fatalf("re-raise: %+v", a)
	}
	if hook, _ = os.ReadFile(hookOut); strings.Count(string(hook), "fail no_gain") != 2 {
		t.Errorf("hook runs after re-raise: %q", hook)
	}
}

// No D2SOAK_ALERT_CMD, no command: nothing runs and the writer still works.
func TestLiveWriterNoHookByDefault(t *testing.T) {
	dir, logPath, lines, now := liveFixture(t)
	os.WriteFile(logPath, []byte(strings.Join(lines[:60], "")), 0o644)
	w := newLiveWriter(dir, "", nil)
	if err := w.write([]Sample{{T: now, HealthOK: true, Progress: -1}}, liveLimits(), liveRules(logPath), now); err != nil {
		t.Fatal(err)
	}
	w.Close()
	if a := readAlerts(t, dir); a.Level != "crit" {
		t.Fatalf("%+v", a)
	}
}

// The log grows between refreshes: the cache must not serve an old judgement.
func TestJudgeLogCacheFollowsGrowth(t *testing.T) {
	dir, logPath, lines, _ := liveFixture(t)
	lim := liveLimits()
	os.WriteFile(logPath, []byte(strings.Join(lines[:44], "")), 0o644)
	n1 := len(judgeLog(logPath, "", lim).Minutes)
	os.WriteFile(logPath, []byte(strings.Join(lines[:48], "")), 0o644)
	n2 := len(judgeLog(logPath, "", lim).Minutes)
	if n2 <= n1 {
		t.Fatalf("minutes %d then %d", n1, n2)
	}
	_ = dir
}

// A limits block that is absent (zero) still gets judged in the sidecar: runSlot fills it and runSoak records the
// effective limits in limits.json.
func TestSidecarRecordsEffectiveLimits(t *testing.T) {
	var h1 atomic.Int32
	b := fakeBot(&h1)
	defer b.Close()
	out := t.TempDir()
	cfg := sidecarCfg{Out: out, Every: 20 * time.Millisecond, For: 100 * time.Millisecond, Limits: Limits{MaxNoProgress: time.Hour},
		Rules: DefaultAlertRules(), Now: time.Now, Log: t.Logf}
	runSlot(t.Context(), cfg, slotInfo{ID: "i9", BotAddr: strings.TrimPrefix(b.URL, "http://")}, func() string { return strings.TrimPrefix(b.URL, "http://") })
	runs, _ := filepath.Glob(filepath.Join(out, "i9-*"))
	if len(runs) != 1 {
		t.Fatalf("runs %v", runs)
	}
	l, ok := loadLimits(runs[0])
	if !ok || !l.Death.Enabled || l.Progress.NoGain == 0 {
		t.Fatalf("limits.json must hold the effective limits: %+v", l)
	}
}
