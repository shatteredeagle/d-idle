package arbiter

import (
	"fmt"
	"strings"
	"testing"
)

// sessionLog writes bot tick lines (knob bot.tickLogSec): a quiet field, two stretches of three ticks under the drink
// line with hostiles near while the chain keeps the quest step, quiet in between. line is the build's drink line
// (survive_below), refill its belt refill line (potions_min).
func sessionLog(line, refill int) string {
	var b strings.Builder
	low := line - 20
	for i := 1; i <= 11; i++ {
		life, hostile := 100, 0
		if i >= 4 && i <= 6 || i >= 9 {
			life, hostile = low, 8
		}
		fmt.Fprintf(&b, `{"time":"2026-11-27T01:00:00Z","msg":"tick","vtMs":%d,"area":"Stony Field","kills":%d,"xp":%d,"chain":"quest","objectiveArea":"Stony Field","facts":{"in_town":0,"life_pct":%d,"hostile":%d,"survive_below":%d,"potions":4,"potions_min":%d,"pack_gold":300,"objective":1}}`+"\n",
			i*1000, i, i*10, life, hostile, line, refill)
	}
	return b.String()
}

func scoreSession(t *testing.T, log string) map[string]Verdict {
	t.Helper()
	f, err := LoadOutcomes(questyOutcomes)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Replay(strings.NewReader(log), DefaultConfig(), ShadowConfig{LogGapSec: 60}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Points) != res.Ticks || res.Ticks != 11 {
		t.Fatalf("points %d ticks %d, want 11", len(res.Points), res.Ticks)
	}
	got := map[string]Verdict{}
	for _, v := range Judge(MergeRules(f).ScoreSession(res.Points), hardOnly) {
		got[v.Family] = v
	}
	return got
}

// Two builds (drink line 50 with refill line 2, drink line 70 with refill line 6): the chain keeps the quest step under
// the drink line twice; each stretch is ONE hard episode (three ticks, counted once), the arbiter picks survive.
func TestSessionOutcomeTwoBuilds(t *testing.T) {
	for _, b := range [][2]int{{50, 2}, {70, 6}} {
		v := scoreSession(t, sessionLog(b[0], b[1]))["survive"]
		if !v.Pass || v.ChainHard != 2 || v.ArbHard != 0 || v.Cases != 6 {
			t.Fatalf("build %v: survive verdict %+v", b, v)
		}
	}
}

// MUST-FAIL: no drink line in the facts (survive_below 0) means no survive source fires; the arbiter keeps the quest step
// like the chain, so there is no survive verdict and the quest verdict does not pass (no fewer hits).
func TestSessionOutcomeMustFailWithoutDrinkLine(t *testing.T) {
	got := scoreSession(t, sessionLog(0, 4))
	if _, ok := got["survive"]; ok {
		t.Fatalf("no survive verdict expected: %+v", got)
	}
	if v := got["quest"]; v.Pass || v.Cases == 0 {
		t.Fatalf("quest verdict must exist and fail: %+v", v)
	}
}

// A hit of the arbiter whose kind the chain never had in the whole session is a new kind, even on a tick where the chain
// hit something else.
func TestScoreSessionNewKindAndEpisodes(t *testing.T) {
	f := OutcomeFile{Rules: []OutcomeRule{
		{Name: "a", Hit: "row-a", Families: []string{"quest"}, When: []string{"x>=1"}},
		{Name: "b", Hit: "row-b", Families: []string{"level"}, When: []string{"x>=1"}},
	}}
	pts := []SessionPoint{
		{Facts: map[string]int{"x": 1}, Chain: "quest", Arbiter: "quest"},
		{Facts: map[string]int{"x": 1}, Chain: "quest", Arbiter: "level"},
		{Facts: map[string]int{"x": 1}, Chain: "quest", Arbiter: "level"},
		{Facts: map[string]int{"x": 0}},
	}
	s := f.ScoreSession(pts)
	if len(s) != 3 || len(s[0].ChainHits) != 1 || len(s[1].ChainHits) != 0 || len(s[1].ArbiterHits) != 1 || len(s[2].ArbiterHits) != 0 {
		t.Fatalf("episodes: %+v", s)
	}
	if s[1].Family != "level" || len(s[1].NewKinds) != 1 || s[1].NewKinds[0] != "row-b" {
		t.Fatalf("new kind: %+v", s[1])
	}
	if v := Judge(s, func(string) bool { return true }); len(v) != 2 || v[0].Pass || v[1].Pass {
		t.Fatalf("verdicts %+v", v)
	}
}
