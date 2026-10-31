package d2dsoakstat

import (
	"math"
	"strings"
	"testing"
)

func runsWith(n int, hours float64, deaths int) []Run {
	var rs []Run
	for i := 0; i < n; i++ {
		d := 0
		if i < deaths {
			d = 1
		}
		rs = append(rs, Run{Name: "s", Hours: hours, Kills: int(500 * hours), Deaths: d, Metrics: map[string]float64{
			"xp_per_hour": 10900, "kills_per_hour": 500, "potions_per_100_kills": 4, "min_life_fraction": 0.4, "damage_per_kill": 10}})
	}
	return rs
}

func TestPoissonBounds(t *testing.T) {
	if p := poissonSF(2, 0.5); math.Abs(p-0.0902) > 0.001 {
		t.Errorf("P(X>=2|0.5)=%v", p)
	}
	if lb := poissonLower(1, 0.05); math.Abs(lb-0.0513) > 0.002 { // exact one-sided 95 % bound for k=1
		t.Errorf("lower(1)=%v", lb)
	}
	if poissonLower(0, 0.05) != 0 {
		t.Error("lower(0)")
	}
}

func TestDeathsV2(t *testing.T) {
	b := baselines(t)
	cases := []struct {
		runs []Run
		want string
	}{
		{runsWith(8, 2, 0), Pass},         // 16 h, none
		{runsWith(8, 2, 2), Pass},         // 16 h, budget 2
		{runsWith(8, 2, 3), Fail},         // over the budget of 2
		{runsWith(3, 2, 0), Pass},         // exactly one 6 h block, none
		{runsWith(2, 2, 0), Inconclusive}, // 4 h: not enough exposure
		{runsWith(3, 2, 2), Fail},         // 2 deaths in 6 h
	}
	for i, c := range cases {
		if r := DeathsV2(c.runs, b); r.Status != c.want {
			t.Errorf("case %d: %s (%s), want %s", i, r.Status, r.Note, c.want)
		}
	}
}

func TestCompareV2(t *testing.T) {
	b := baselines(t)
	v := CompareV2(runsWith(8, 2, 0), b)
	for _, r := range v.Rows {
		if r.Status == Fail {
			t.Errorf("unexpected FAIL:\n%s", v.Text())
		}
	}
	// the baselines carry no life proxies: those rows must keep the verdict from becoming PASS
	if v.Overall == Pass && !strings.Contains(v.Text(), "damage_per_kill") {
		t.Errorf("proxy rows missing:\n%s", v.Text())
	}
	// v1 verdict is untouched by the new info rows
	if Compare(runsWith(8, 2, 0), b).Overall == Fail {
		t.Error("v1 changed")
	}
	// v2 FAIL on deaths even where v1's 0.5/h absolute tolerance passes
	if CompareV2(runsWith(8, 2, 4), b).Overall != Fail {
		t.Error("4 deaths in 16 h must fail v2")
	}
}

func TestProxyExtract(t *testing.T) {
	ms := []Minute{
		{Kills: 0, PotionsUsed: 0, HasHP: true, HPMinPct: 100},
		{Kills: 10, PotionsUsed: 1, DmgTaken: 100, HasHP: true, HPMinPct: 50},
		{Kills: 10, PotionsUsed: 1, DmgTaken: 100, HasHP: true, HPMinPct: 20}, // no kill: ignored
		{Kills: 20, PotionsUsed: 3, DmgTaken: 300, HasHP: true, HPMinPct: 30},
	}
	r := Extract("x", ms, Options{Hours: 1})
	if got := r.Metrics["potions_per_100_kills"]; got != 15 {
		t.Errorf("potions/100 kills %v", got)
	}
	if got := r.Metrics["damage_per_kill"]; got != 15 {
		t.Errorf("damage/kill %v", got)
	}
	if got := r.Metrics["min_life_fraction"]; math.Abs(got-0.4) > 1e-9 {
		t.Errorf("min life %v", got)
	}
	ms[0].HasHP = false
	if _, ok := Extract("x", ms, Options{Hours: 1}).Metrics["damage_per_kill"]; ok {
		t.Error("no HP data must give no proxy")
	}
}
