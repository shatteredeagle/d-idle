package d2dsoakstat

import (
	"math"
	"strings"
	"testing"
)

func TestEmptySeedIsInvalidAndExcluded(t *testing.T) {
	r := Extract("seed-1", nil, Options{Hours: 2})
	if r.Invalid == "" || r.Hours != 0 {
		t.Fatalf("empty run must be invalid with 0 hours: %+v", r)
	}
	b := baselines(t)
	if row := DeathsV2([]Run{r, r, r}, b); row.Status == Pass {
		t.Errorf("three empty runs must not pass: %+v", row)
	}
	v := CompareV2(append(runsWith(8, 2, 0), r), b)
	if v.Overall == Pass || !strings.Contains(v.Text(), "excluded") {
		t.Errorf("an excluded run must stop PASS and be named:\n%s", v.Text())
	}
}

func TestDeathsV2NotFinite(t *testing.T) {
	b := baselines(t)
	for _, h := range []float64{math.NaN(), math.Inf(1), -3} {
		if row := DeathsV2([]Run{{Hours: h}}, b); row.Status == Pass {
			t.Errorf("hours %v gave PASS: %+v", h, row)
		}
	}
	rs := runsWith(8, 2, 0)
	rs[0].Deaths = -5
	if row := DeathsV2(rs, b); row.D2D.Mean < 0 {
		t.Errorf("negative deaths pooled: %+v", row)
	}
}

func TestMinimumSampleIsInconclusive(t *testing.T) {
	b := baselines(t)
	if v := Compare(d2dRuns(10000, 500, 0.1, 0.02, 5), b); v.Overall != Pass {
		t.Fatalf("5 x 2 h should pass:\n%s", v.Text())
	}
	short := d2dRuns(10000, 500, 0.1, 0.02, 5)
	for i := range short {
		short[i].Hours = 0.2
	}
	if v := Compare(short, b); v.Overall != Inconclusive {
		t.Errorf("5 x 0.2 h must be INCONCLUSIVE:\n%s", v.Text())
	}
	if v := Compare(d2dRuns(10000, 500, 0.1, 0.02, 4), b); v.Overall != Inconclusive {
		t.Errorf("4 seeds must be INCONCLUSIVE:\n%s", v.Text())
	}
}

func TestDeathsPerKillFloor(t *testing.T) {
	if row := DeathsPerKillV2([]Run{{Hours: 2, Kills: 1}}, baselines(t)); row.Status != Inconclusive {
		t.Errorf("1 kill: %+v", row)
	}
}

func TestParseDropsBadLines(t *testing.T) {
	good := `{"msg":"minute","time":"2026-01-01T00:00:00Z","area":1,"clvl":5,"kills":1,"deaths":0,"gold":1,"restarts":0,"xp":100}`
	badTime := `{"msg":"minute","time":"nope","area":1,"clvl":5,"kills":1,"deaths":0,"gold":1,"restarts":0,"xp":100}`
	badXP := `{"msg":"minute","time":"2026-01-01T01:00:00Z","area":1,"clvl":5,"kills":1,"deaths":0,"gold":1,"restarts":0,"xp":"x"}`
	ms, n, err := ParseMinutesCount(strings.NewReader(good + "\n" + badTime + "\n" + badXP + "\n"))
	if err != nil || len(ms) != 1 || n != 2 {
		t.Fatalf("ms %d dropped %d err %v", len(ms), n, err)
	}
	r := Extract("x", ms, Options{Hours: 1})
	r.NoteDropped(n)
	if r.Invalid == "" {
		t.Errorf("2 of 3 lines dropped must invalidate: %+v", r)
	}
	r2 := Run{Minutes: 1000}
	r2.NoteDropped(1)
	if r2.Invalid != "" || r2.Dropped != 1 {
		t.Errorf("1 in 1000 is fine: %+v", r2)
	}
}

func TestRepeatSitesCapsJump(t *testing.T) {
	ms := []Minute{{Deaths: 0}, {Deaths: 1_000_000_000, Area: 1}}
	if n := repeatSites(ms, []float64{0, 1}); n > maxJump {
		t.Errorf("repeatSites = %d", n)
	}
}

func TestExcludedSeedsAreNotExposure(t *testing.T) {
	b := baselines(t)
	empty := Extract("seed-11", nil, Options{Hours: 2})
	rs := append(d2dRuns(10000, 500, 0.1, 0.02, 5), empty)
	v := Compare(rs, b)
	if v.Seeds != 5 || len(v.Excluded) != 1 || !strings.Contains(v.Excluded[0], "seed-11") {
		t.Fatalf("seeds %d excluded %v", v.Seeds, v.Excluded)
	}
	if v.Overall == Pass || !strings.Contains(v.Text(), "excluded seeds") {
		t.Errorf("an excluded seed must stop PASS and be listed:\n%s", v.Text())
	}
	// the exposure is the other seeds' hours only: the death row uses 5 x 2 h
	rows := CompareV2(append(runsWith(7, 2, 0), empty), b)
	for _, r := range rows.Rows {
		if r.Metric == "deaths_per_hour" && r.D2D.N != 7 {
			t.Errorf("death exposure counted %d seeds, want 7", r.D2D.N)
		}
	}
	if rows.Seeds != 7 {
		t.Errorf("seed count %d, want 7", rows.Seeds)
	}
}

func TestNoUsableSeedIsNoData(t *testing.T) {
	b := baselines(t)
	empty := Extract("seed-1", nil, Options{Hours: 2})
	for _, v := range []Verdict{Compare([]Run{empty, empty}, b), CompareV2(nil, b)} {
		if v.Overall != NoData || ExitCode(v.Overall) != 3 || !strings.Contains(v.Text(), "OVERALL: NO-DATA (exit 3)") {
			t.Errorf("%s\n%s", v.Overall, v.Text())
		}
	}
}
