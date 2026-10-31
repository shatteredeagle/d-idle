package d2dsoakstat

import (
	"math"
	"os"
	"strings"
	"testing"
)

func load(t *testing.T, name string) []Minute {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ms, err := ParseMinutes(f)
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

func near(t *testing.T, what string, got, want, relTol float64) {
	t.Helper()
	if math.Abs(got-want) > relTol*math.Abs(want) {
		t.Errorf("%s = %.4g, want %.4g (+-%.0f%%)", what, got, want, relTol*100)
	}
}

// Real minute lines of the original-game soaks; the want values are the d2soak report numbers from docs/g1-log.md
// (attempt 18: 9,103 XP/h, 70 potions, level 14; attempt 16: 11,080 XP/h, 0 deaths). The report uses its own 30 s
// samples, so a few percent of difference is expected.
func TestExtractRealLogs(t *testing.T) {
	r18 := Extract("a18", load(t, "a18_minutes.jsonl"), Options{})
	if len(load(t, "a18_minutes.jsonl")) != 124 {
		t.Fatal("fixture changed")
	}
	near(t, "a18 hours", r18.Hours, 2.05, 0.02)
	near(t, "a18 xp/h", r18.Metrics["xp_per_hour"], 9103, 0.05)
	if r18.Deaths != 0 || r18.Metrics["deaths_per_hour"] != 0 {
		t.Errorf("a18 deaths = %d", r18.Deaths)
	}
	if got := int(r18.Metrics["potions_per_hour"] * r18.Hours); got != 70 {
		t.Errorf("a18 potions = %d, want 70", got)
	}
	near(t, "a18 farming share", r18.Metrics["farming_share"], 0.80, 0.03)
	near(t, "a18 town share", r18.Metrics["town_share"], 0.20, 0.10)
	if r18.Metrics["clvl_2h"] != 14 || r18.Metrics["clvl_1h"] != 14 {
		t.Errorf("a18 clvl checkpoints = %v", r18.Metrics)
	}

	r16 := Extract("a16", load(t, "a16_minutes.jsonl"), Options{})
	near(t, "a16 xp/h", r16.Metrics["xp_per_hour"], 11080, 0.05)
	near(t, "a16 kills/h", r16.Metrics["kills_per_hour"], 501, 0.10)
	if r16.Metrics["recovery_share"] < 0.08 || r16.Metrics["recovery_share"] > 0.16 {
		t.Errorf("a16 recovery share = %.3f, the g1-log says about 11 %%", r16.Metrics["recovery_share"])
	}
	if r16.Metrics["clvl_gain"] != 1 {
		t.Errorf("a16 clvl gain = %v", r16.Metrics["clvl_gain"])
	}
	// the old a14 lines have no state field: no share metrics, but rates still work
	r14 := Extract("a14", load(t, "a14_minutes.jsonl"), Options{})
	if _, has := r14.Metrics["farming_share"]; has || r14.Metrics["xp_per_hour"] <= 0 {
		t.Errorf("a14 metrics = %v", r14.Metrics)
	}
}

func TestFixedHoursScalesRatesAndCheckpoints(t *testing.T) {
	ms := load(t, "a18_minutes.jsonl")
	real := Extract("r", ms, Options{})
	fixed := Extract("v", ms, Options{Hours: 4})
	near(t, "xp/h at half speed", fixed.Metrics["xp_per_hour"], real.Metrics["xp_per_hour"]*real.Hours/4, 0.001)
	if _, ok := fixed.Metrics["clvl_2h"]; !ok {
		t.Error("a 4 h run has a 2 h checkpoint")
	}
	if _, ok := Extract("s", ms, Options{Hours: 0.5}).Metrics["clvl_1h"]; ok {
		t.Error("a 0.5 h run must not report a 1 h checkpoint")
	}
}

func TestParseTextAndJSONAndJunk(t *testing.T) {
	in := `junk
time=2026-10-30T03:00:00-04:00 level=INFO msg=minute area=3 clvl=5 xp=1000 kills=10 deaths=0 potions=1 gold=7 restarts=0 rung=0 goal=campaign step="Quest: x y" state=farming rstate=clearing
{"time":"2026-10-30T03:01:00-04:00","msg":"minute","area":3,"clvl":5,"xp":119175,"kills":11,"deaths":1,"potions":0,"potionsUsed":4,"gold":7,"restarts":0}
{"msg":"other"}
time=x msg=minute area=bad
`
	ms, err := ParseMinutes(strings.NewReader(in))
	if err != nil || len(ms) != 2 {
		t.Fatalf("got %d minutes, err %v", len(ms), err)
	}
	if ms[0].Step != "Quest: x y" || ms[0].PotionsUsed != 1 || ms[1].XP != 119175 || ms[1].PotionsUsed != 4 {
		t.Errorf("parsed %+v", ms)
	}
}

func TestCounterResetsAndRepeatSites(t *testing.T) {
	mk := func(kills, deaths, area int) Minute { return Minute{Kills: kills, Deaths: deaths, Area: area} }
	// bot restart: the running counters drop to 0, the positive increments still count; two deaths in area 4 within
	// 30 min (the second is a repeat site), a third in another area 40 minutes later is not
	ms := []Minute{mk(10, 0, 4), mk(20, 1, 4), mk(30, 1, 4), mk(5, 0, 4), mk(15, 1, 4)}
	for i := 0; i < 40; i++ {
		ms = append(ms, mk(15, 1, 9))
	}
	ms = append(ms, mk(16, 2, 9))
	r := Extract("x", ms, Options{Hours: 1})
	if r.Kills != 10+10+10+1 || r.Deaths != 3 {
		t.Errorf("kills %d deaths %d", r.Kills, r.Deaths)
	}
	if r.Metrics["repeat_sites"] != 1 {
		t.Errorf("repeat sites = %v, want 1", r.Metrics["repeat_sites"])
	}
}

func TestAggregateAndTCrit(t *testing.T) {
	runs := []Run{{Metrics: map[string]float64{"xp_per_hour": 100}}, {Metrics: map[string]float64{"xp_per_hour": 120}}, {Metrics: map[string]float64{"xp_per_hour": 110}}}
	s := Aggregate(runs)["xp_per_hour"]
	near(t, "mean", s.Mean, 110, 1e-9)
	near(t, "sd", s.SD, 10, 1e-9)
	near(t, "ci", s.CIHalf95, 4.303*10/math.Sqrt(3), 1e-9)
	if !math.IsInf(Aggregate(runs[:1])["xp_per_hour"].CIHalf95, 1) {
		t.Error("one run has no interval")
	}
	if n := NeededSeeds(2000, 2000); n != 4 {
		t.Errorf("needed seeds = %d", n)
	}
}

func baselines(t *testing.T) Baselines {
	t.Helper()
	mk := func(name string, xp, kills, deaths float64) BaselineRun {
		return BaselineRun{Run: Run{Name: name, Hours: 2, Metrics: map[string]float64{"xp_per_hour": xp, "kills_per_hour": kills, "deaths_per_hour": deaths, "deaths_per_1000_kills": deaths / kills * 1000}}}
	}
	return Baselines{Rules: DefaultRules(), MinSeeds: 5, MinHoursPerSeed: 2, MinTotalHours: 10,
		Runs: []BaselineRun{mk("a", 10000, 500, 0), mk("b", 10000, 500, 0), mk("c", 10000, 500, 0.5)}}
}

func d2dRuns(xp, kills, deaths, spread float64, n int) []Run {
	var rs []Run
	for i := 0; i < n; i++ {
		f := 1 + spread*float64(i%2*2-1) // alternate +-spread
		rs = append(rs, Run{Name: "s", Hours: 2, Metrics: map[string]float64{"xp_per_hour": xp * f, "kills_per_hour": kills * f, "deaths_per_hour": deaths, "deaths_per_1000_kills": deaths / kills * 1000}})
	}
	return rs
}

func TestCompareVerdicts(t *testing.T) {
	b := baselines(t)
	row := func(v Verdict, m string) Row {
		for _, r := range v.Rows {
			if r.Metric == m {
				return r
			}
		}
		t.Fatalf("no row %s", m)
		return Row{}
	}
	// within 20 %, tight spread, 8 seeds: PASS everywhere
	v := Compare(d2dRuns(10800, 540, 0.2, 0.03, 8), b)
	if v.Overall != Pass || row(v, "xp_per_hour").Status != Pass {
		t.Errorf("want PASS:\n%s", v.Text())
	}
	// 30 % under: FAIL with a tight spread
	v = Compare(d2dRuns(7000, 540, 0.2, 0.03, 8), b)
	if v.Overall != Fail || row(v, "xp_per_hour").Status != Fail {
		t.Errorf("want FAIL:\n%s", v.Text())
	}
	// 25 % off but a spread so wide the interval reaches back into the band: INCONCLUSIVE, never FAIL
	v = Compare(d2dRuns(7500, 540, 0.2, 0.40, 4), b)
	if r := row(v, "xp_per_hour"); r.Status != Inconclusive || r.NeedSeed == 0 {
		t.Errorf("want INCONCLUSIVE with a seed estimate: %+v", r)
	}
	// deaths: baseline mean 0.17/h, absolute tolerance 0.5: 0.6/h passes although it is 250 % relative
	v = Compare(d2dRuns(10000, 500, 0.6, 0.02, 8), b)
	if r := row(v, "deaths_per_hour"); r.Status != Pass || r.GapPct < 100 {
		t.Errorf("deaths abs tolerance: %+v", r)
	}
	v = Compare(d2dRuns(10000, 500, 1.5, 0.02, 8), b)
	if row(v, "deaths_per_hour").Status != Fail {
		t.Errorf("1.5 deaths/h must fail:\n%s", v.Text())
	}
}

func TestCompareWarnsOnSmallSamples(t *testing.T) {
	short := d2dRuns(10000, 500, 0, 0.02, 2)
	short[0].Hours = 0.5
	v := Compare(short, baselines(t))
	if v.Overall != Inconclusive {
		t.Errorf("2 seeds must be INCONCLUSIVE, got %s", v.Overall)
	}
	text := v.Text()
	for _, want := range []string{"minimum 5", "under the 2.0 h minimum", "virtual hours in total"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing warning %q in\n%s", want, text)
		}
	}
}

func TestCommittedBaselines(t *testing.T) {
	f, err := os.Open("../../tools/d2dsoak/baselines.json")
	if err != nil {
		t.Skip("baselines.json not there:", err)
	}
	defer f.Close()
	b, err := ReadBaselines(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Runs) < 3 || !b.Rules["xp_per_hour"].Gate {
		t.Errorf("baselines look wrong: %d runs", len(b.Runs))
	}
	for _, r := range b.Runs {
		if r.Source == "" || r.Metrics["xp_per_hour"] <= 0 {
			t.Errorf("baseline run %s: source %q xp/h %v", r.Name, r.Source, r.Metrics["xp_per_hour"])
		}
	}
}
