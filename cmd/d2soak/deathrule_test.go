package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func deathFixture(t *testing.T, n string, lim DeathLimits) DeathReport {
	t.Helper()
	return deathsFromLog("testdata/soak_"+n+"_deathrule.jsonl", "", lim)
}

func deathCodes(dr DeathReport, level string) []string {
	var out []string
	for _, f := range dr.Findings {
		if f.Level == level {
			out = append(out, f.Code)
		}
	}
	return out
}

func inCodes(cs []string, c string) bool {
	for _, x := range cs {
		if x == c {
			return true
		}
	}
	return false
}

// Real excerpts of the G1 soak logs (docs/plan/death-data.md is the oracle). V: the verdicts and per-death seconds
// below equal that table (it rounds the seconds, the harness truncates to the same second).
func TestDeathRuleRealAttempts(t *testing.T) {
	lim := DefaultDeathLimits()
	for _, c := range []struct {
		att       string
		deaths    int
		back      []int // seconds, oracle table section 4
		fail      []string
		warn      []string
		repeat    int
		openLast  bool
		noFailAny bool
	}{
		{att: "a01", deaths: 4, back: []int{44, 32, 6, 6}, fail: []string{"death_streak"}, repeat: 3},
		{att: "a11", deaths: 5, back: []int{98, 49, 12, 6, 31}, fail: []string{"death_streak", "death_rate"}, repeat: 4},
		{att: "a12", deaths: 3, back: []int{47, 36}, fail: []string{"death_streak", "death_recovery_stuck"}, repeat: 2, openLast: true},
		{att: "a08", deaths: 3, back: []int{98, 6, 133}, warn: []string{"death_pair", "death_recovery_share", "death_rate"}, repeat: 1, noFailAny: true},
		{att: "a14", deaths: 0, noFailAny: true},
	} {
		dr := deathFixture(t, c.att, lim)
		if len(dr.Deaths) != c.deaths {
			t.Fatalf("%s: %d deaths, want %d", c.att, len(dr.Deaths), c.deaths)
		}
		for i, want := range c.back {
			if got := int(dr.Deaths[i].Back / time.Second); got < want-2 || got > want {
				t.Errorf("%s death %d: back in field %d s, oracle %d s", c.att, i+1, got, want)
			}
		}
		if c.openLast && !dr.Deaths[len(dr.Deaths)-1].Open {
			t.Errorf("%s: last death should be open (stuck in town)", c.att)
		}
		fails := deathCodes(dr, "fail")
		for _, f := range c.fail {
			if !inCodes(fails, f) {
				t.Errorf("%s: want fail %s, got %v", c.att, f, dr.Findings)
			}
		}
		if (c.noFailAny || len(c.fail) == 0) && len(fails) > 0 {
			t.Errorf("%s: unexpected FAIL %v", c.att, dr.Findings)
		}
		for _, w := range c.warn {
			if !inCodes(deathCodes(dr, "warn"), w) {
				t.Errorf("%s: want warn %s, got %v", c.att, w, dr.Findings)
			}
		}
		if dr.Summary.RepeatSites != c.repeat {
			t.Errorf("%s: %d repeat-site deaths, want %d", c.att, dr.Summary.RepeatSites, c.repeat)
		}
		if c.repeat > 0 && !inCodes(deathCodes(dr, "warn"), "death_repeat_site_no_action") {
			t.Errorf("%s: missing the no-action warn", c.att)
		}
	}
}

// Attempt 14 (0 deaths, hours of Recovery: farming gold and Waiting: banned) gives no finding at all.
func TestDeathRuleCleanRunNoFinding(t *testing.T) {
	dr := deathFixture(t, "a14", DefaultDeathLimits())
	if len(dr.Findings) != 0 || dr.Summary.PeakShare != 0 {
		t.Fatalf("a14: %v share %.3f", dr.Findings, dr.Summary.PeakShare)
	}
}

func TestDeathRuleOffByDefaultZeroValue(t *testing.T) {
	// an old limits.json has no Death block: nothing is judged
	if dr := deathFixture(t, "a12", DeathLimits{}); dr.Evaluated || len(dr.Findings) != 0 {
		t.Fatalf("zero limits judged something: %v", dr.Findings)
	}
}

func dl(sec int, area int) DeathEvent {
	return DeathEvent{At: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC).Add(time.Duration(sec) * time.Second), Area: area}
}

func TestDeathRuleStreakAndHour(t *testing.T) {
	lim := DefaultDeathLimits()
	t0 := dl(0, 0).At
	tl := func(ds ...DeathEvent) DeathTimeline {
		t := DeathTimeline{Start: t0, End: t0.Add(3 * time.Hour), Deaths: ds}
		for _, d := range ds {
			t.Field = append(t.Field, d.At.Add(30*time.Second))
		}
		return t
	}
	min := func(m int) int { return m * 60 }
	// 3 in 29 min with a level gain between: no streak
	ms := []Minute{{At: t0, Clvl: 9}, {At: t0.Add(15 * time.Minute), Clvl: 10}, {At: t0.Add(3 * time.Hour), Clvl: 10}}
	if dr := AnalyzeDeaths(tl(dl(60, 1), dl(min(15)+30, 2), dl(min(29), 3)), ms, nil, lim); inCodes(deathCodes(dr, "fail"), "death_streak") {
		t.Errorf("level gain inside the span must clear the 3-in-30 rule: %v", dr.Findings)
	}
	// the same without a gain: fail
	ms[1].Clvl = 9
	if dr := AnalyzeDeaths(tl(dl(60, 1), dl(min(15)+30, 2), dl(min(29), 3)), ms, nil, lim); !inCodes(deathCodes(dr, "fail"), "death_streak") {
		t.Errorf("3 in 29 min, no gain: want fail")
	}
	// 3 in 31 min: no
	if dr := AnalyzeDeaths(tl(dl(0, 1), dl(min(15), 2), dl(min(31), 3)), ms, nil, lim); inCodes(deathCodes(dr, "fail"), "death_streak") {
		t.Errorf("3 in 31 min must pass: %v", dr.Findings)
	}
	// 4 in 60 min with level gains is fine, 5 fail whatever the gain
	gain := []Minute{{At: t0, Clvl: 5}, {At: t0.Add(20 * time.Minute), Clvl: 6}, {At: t0.Add(40 * time.Minute), Clvl: 7}, {At: t0.Add(2 * time.Hour), Clvl: 7}}
	four := tl(dl(0, 1), dl(min(19), 2), dl(min(39), 3), dl(min(58), 4))
	if dr := AnalyzeDeaths(four, gain, nil, lim); inCodes(deathCodes(dr, "fail"), "death_streak") {
		t.Errorf("4 spread deaths must pass: %v", dr.Findings)
	}
	five := tl(dl(0, 1), dl(min(14), 2), dl(min(29), 3), dl(min(44), 4), dl(min(59), 5))
	gain2 := []Minute{{At: t0, Clvl: 5}, {At: t0.Add(10 * time.Minute), Clvl: 6}, {At: t0.Add(20 * time.Minute), Clvl: 7}, {At: t0.Add(40 * time.Minute), Clvl: 8}, {At: t0.Add(55 * time.Minute), Clvl: 9}}
	if dr := AnalyzeDeaths(five, gain2, nil, lim); !inCodes(deathCodes(dr, "fail"), "death_streak") {
		t.Errorf("5 in 60 min must fail regardless of level gain")
	}
	// two deaths in 60 min: warn only
	dr := AnalyzeDeaths(tl(dl(0, 1), dl(min(50), 2)), ms, nil, lim)
	if len(deathCodes(dr, "fail")) != 0 || !inCodes(deathCodes(dr, "warn"), "death_pair") {
		t.Errorf("2 deaths in 60 min: %v", dr.Findings)
	}
}

func TestDeathRuleStuckAndMinimumRun(t *testing.T) {
	lim := DefaultDeathLimits()
	t0 := dl(0, 0).At
	// one death, 6 min in town, fails at any run age
	tl := DeathTimeline{Start: t0, End: t0.Add(10 * time.Minute), Deaths: []DeathEvent{dl(120, 1)},
		Field: []time.Time{t0.Add(8 * time.Minute)}}
	if dr := AnalyzeDeaths(tl, nil, nil, lim); !inCodes(deathCodes(dr, "fail"), "death_recovery_stuck") {
		t.Errorf("6 min death to field must fail: %v", dr.Findings)
	}
	// deaths.jsonl secondsToField wins over the log
	s := 90
	recs := []DeathRecord{{Time: dl(120, 1).At, SecondsToField: &s}}
	if dr := AnalyzeDeaths(tl, nil, recs, lim); len(dr.Findings) != 0 || !dr.Deaths[0].FromRecord {
		t.Errorf("record seconds must replace the log window: %v", dr.Findings)
	}
	// the share rule is silent before 30 min of run time: 4 min of recovery in a 20 min run
	short := DeathTimeline{Start: t0, End: t0.Add(20 * time.Minute), Deaths: []DeathEvent{dl(60, 1)}, Field: []time.Time{t0.Add(5 * time.Minute)}}
	if dr := AnalyzeDeaths(short, nil, nil, lim); len(dr.Findings) != 0 || dr.Summary.PeakShare != 0 {
		t.Errorf("no rate rule before 30 min: %v", dr.Findings)
	}
}

func TestDeathRuleNeverCountsFarmingOrBan(t *testing.T) {
	lim := DefaultDeathLimits()
	t0 := dl(0, 0).At
	// a death whose window is spent in "Recovery: farming gold" / "Waiting: banned" counts nothing from those states
	tl := DeathTimeline{Start: t0, End: t0.Add(80 * time.Minute), Deaths: []DeathEvent{dl(0, 1)},
		Acts:  []activity{{t0.Add(time.Second), "Recovery: farming gold"}, {t0.Add(5 * time.Minute), "Waiting: Black Marsh banned for 4 more min"}},
		Field: []time.Time{t0.Add(40 * time.Minute)}}
	lim.MaxBack = 0
	dr := AnalyzeDeaths(tl, nil, nil, lim)
	if dr.Summary.PeakShare > 0.001 {
		t.Errorf("farming/banned states counted as recovery: share %.3f", dr.Summary.PeakShare)
	}
	// and the listed states do count, even without a death line
	tl2 := DeathTimeline{Start: t0, End: t0.Add(60 * time.Minute),
		Acts: []activity{{t0, "Quest: x"}, {t0.Add(10 * time.Minute), "Restocking after death"}, {t0.Add(20 * time.Minute), "Quest: x"}}}
	if dr := AnalyzeDeaths(tl2, nil, nil, lim); len(deathCodes(dr, "fail")) != 1 || dr.Summary.PeakShare < 0.16 {
		t.Errorf("10 min of Restocking after death in 60: %v share %.3f", dr.Findings, dr.Summary.PeakShare)
	}
}

func TestDeathRuleRateNeedsKills(t *testing.T) {
	lim := DefaultDeathLimits()
	t0 := dl(0, 0).At
	mk := func(kills int) DeathReport {
		ms := []Minute{{At: t0, Kills: 0, Clvl: 9}, {At: t0.Add(60 * time.Minute), Kills: kills, Clvl: 9}}
		tl := DeathTimeline{Start: t0, End: t0.Add(60 * time.Minute), Deaths: []DeathEvent{dl(40*60, 1), dl(50*60, 2), dl(55*60, 3)},
			Field: []time.Time{dl(40*60+20, 0).At, dl(50*60+20, 0).At, dl(55*60+20, 0).At}}
		return AnalyzeDeaths(tl, ms, nil, lim)
	}
	if dr := mk(100); inCodes(deathCodes(dr, "fail"), "death_rate") || dr.Summary.PeakRate != 0 {
		t.Errorf("under 150 kills the rate rule is silent: %v", dr.Findings)
	}
	if dr := mk(300); inCodes(deathCodes(dr, "fail"), "death_rate") || !inCodes(deathCodes(dr, "warn"), "death_rate") {
		t.Errorf("3 deaths in 300 kills = 10/1000: warn only: %v", dr.Findings)
	}
	if dr := mk(150); !inCodes(deathCodes(dr, "fail"), "death_rate") {
		t.Errorf("3 deaths in 150 kills = 20 fails? got %v", dr.Findings)
	}
}

func TestDeathRecordsAndTags(t *testing.T) {
	jl := `{"time":"2026-10-30T12:00:00Z","area":7,"autopsy":{"beltHP":2,"potionsDrunk60s":0,"goldCarried":300}}
{"time":"2026-10-30T12:03:00Z","area":7,"autopsy":{"beltHP":0,"potionsDrunk60s":1,"goldCarried":50}}
{"followUp":true,"time":"2026-10-30T12:00:00Z","secondsToField":40,"goldNextMinute":0}
`
	recs, err := ParseDeathRecords(strings.NewReader(jl))
	if err != nil || len(recs) != 2 || recs[0].SecondsToField == nil || *recs[0].SecondsToField != 40 {
		t.Fatalf("recs %+v err %v", recs, err)
	}
	t0 := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	tl := DeathTimeline{Start: t0.Add(-time.Hour), End: t0.Add(time.Hour), Deaths: []DeathEvent{{At: t0, Area: 7}, {At: t0.Add(3 * time.Minute), Area: 7}},
		Bans: []time.Time{t0.Add(3*time.Minute + 2*time.Second)}}
	dr := AnalyzeDeaths(tl, nil, recs, DefaultDeathLimits())
	if dr.Summary.BugDeaths != 1 || !dr.Deaths[0].Bug || dr.Deaths[1].Bug || dr.Summary.GoldCarried != 350 {
		t.Errorf("bug/gold tags: %+v", dr.Summary)
	}
	if dr.Summary.RepeatSites != 1 || dr.Summary.RepeatBanned != 1 || inCodes(deathCodes(dr, "warn"), "death_repeat_site_no_action") {
		t.Errorf("a ban line after the repeat death must silence the no-action warn: %+v %v", dr.Summary, dr.Findings)
	}
}

func TestDeathProfilesAndOldLimits(t *testing.T) {
	for _, p := range []string{"m1", "m24"} {
		l := DefaultLimits()
		profiles[p](&l)
		if !l.Death.Enabled || l.MaxDeathsPerHour != 0 {
			t.Errorf("%s: death rule on and deaths/h gate off expected: %+v %v", p, l.Death, l.MaxDeathsPerHour)
		}
	}
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/limits.json", []byte(`{"MaxDeathsPerHour":1,"Progress":{"NoGain":1}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	l, ok := loadLimits(dir)
	if !ok || l.Death.Enabled {
		t.Fatalf("old limits.json must judge no death rule: %+v", l.Death)
	}
}
