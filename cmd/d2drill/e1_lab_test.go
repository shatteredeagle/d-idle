package main

import (
	"fmt"
	"testing"
)

func TestE1MercVerdicts(t *testing.T) {
	f := map[string]string{"quote": "1920", "gold": "1930"}
	revFirst := []string{"msg=\"merc: revived price=1920 level=12\"", "msg=\"town: restocked mana potions\""}
	buyFirst := []string{"msg=\"town: restocked mana potions\"", "msg=\"merc: revived price=1920 level=12\""}
	if v, _ := e1MercEvalA(revFirst, f); v != "PASS" {
		t.Errorf("revive first: %s", v)
	}
	if v, _ := e1MercEvalA(buyFirst, f); v != "FAIL" {
		t.Errorf("purchase first must fail: %s", v)
	}
	if v, _ := e1MercEvalA(nil, f); v != "FAIL" {
		t.Errorf("no revive must fail: %s", v)
	}
	if v, _ := e1MercEvalB([]string{"merc: revive skipped (gold) price=1920 gold=1500 need=1920", "town: restocked mana potions"}, f); v != "PASS" {
		t.Errorf("poor variant: %s", v)
	}
	if v, _ := e1MercEvalB([]string{"merc: revive skipped (gold)"}, f); v != "FAIL" {
		t.Errorf("poor variant without purchases starves the potions: %s", v)
	}
	if v, _ := e1MercEvalC([]string{"merc: revive skipped (gold) price=1920 gold=1930 need=1970"}, f); v != "PASS" {
		t.Errorf("clvl 14 waits: %s", v)
	}
	if v, _ := e1MercEvalC([]string{"merc: revived price=1920"}, f); v != "FAIL" {
		t.Errorf("clvl 14 must not revive at quote+10: %s", v)
	}
}

func TestE1NoPortal(t *testing.T) {
	l := []string{
		`{"time":"2026-11-06T17:50:00.000-05:00","level":"WARN","msg":"heal trip: town: no town portal available"}`,
		`{"time":"2026-11-06T17:50:09.500-05:00","level":"WARN","msg":"low life, no potion and no town portal: leaving the game","waited":20000000000}`,
	}
	if v, _ := e1NoPortalEval(l, false); v != "PASS" {
		t.Errorf("leave: %s", v)
	}
	if v, _ := e1NoPortalEval(l[:1], false); v != "FAIL" {
		t.Errorf("no leave must fail: %s", v)
	}
	if v, _ := e1NoPortalEval(l[1:], false); v != "FAIL" {
		t.Errorf("a leave without the no-portal line is not the path under test: %s", v)
	}
	town := append([]string{`{"msg":"zone entry","area":1,"clvl":12}`}, l[:1]...)
	if v, _ := e1NoPortalEval(town, true); v != "PASS" {
		t.Errorf("control: %s", v)
	}
	if v, _ := e1NoPortalEval(l[:1], true); v != "FAIL" {
		t.Errorf("control that never reaches town: %s", v)
	}
	res := &e1Result{}
	e1Key(res, l)
	if res.NoPortalToLeaveSec != 9.5 || res.WaitedSec != 20 {
		t.Errorf("timing: %+v", res)
	}
}

func TestE1Scenarios(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range e1Scenarios() {
		if seen[s.ID] || s.Setup == nil || s.Eval == nil || s.Done == nil || s.Goals == "" {
			t.Errorf("scenario %q incomplete or duplicated", s.ID)
		}
		seen[s.ID] = true
	}
	if len(seen) != 8 {
		t.Errorf("want 8 scenarios, got %d", len(seen))
	}
}

func TestE1TPEvals(t *testing.T) {
	min := func(tp, buy int) string {
		s := fmt.Sprintf(`{"msg":"minute","tp":%d`, tp)
		if buy > 0 {
			s += fmt.Sprintf(`,"tpBuy":%d`, buy)
		}
		return s + `}`
	}
	dump := `{"msg":"inv-tp-dump","portal_items":"tome class=518 where=inventory addr=0x3 counted=4 raw70=4"}`
	for _, c := range []struct {
		name  string
		eval  func([]string, map[string]string) (string, string)
		lines []string
		want  string
	}{
		{"a ok", e1TPEvalA, []string{min(3, 0)}, "PASS"},
		{"a zero", e1TPEvalA, []string{min(0, 0)}, "FAIL"},
		{"a none", e1TPEvalA, nil, "FAIL"},
		{"b ok", e1TPEvalB, []string{dump, min(4, 0)}, "PASS"},
		{"b zero", e1TPEvalB, []string{dump, min(0, 0)}, "FAIL"},
		{"c ok", e1TPEvalC, []string{min(0, 0), min(2, 2)}, "PASS"},
		{"c no buy", e1TPEvalC, []string{min(0, 0)}, "FAIL"},
		{"c buy tp0", e1TPEvalC, []string{min(0, 2)}, "FAIL"},
	} {
		if v, why := c.eval(c.lines, nil); v != c.want {
			t.Errorf("%s: %s (%s), want %s", c.name, v, why, c.want)
		}
	}
}
