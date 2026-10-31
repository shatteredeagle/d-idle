package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T, name string) []Minute {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ms, err := ParseMinuteLines(f)
	if err != nil || len(ms) == 0 {
		t.Fatalf("%s: %d lines, %v", name, len(ms), err)
	}
	return ms
}

func findCodes(pr ProgressReport) map[string]bool {
	c := map[string]bool{}
	for _, f := range pr.Findings {
		c[f.Code] = true
	}
	return c
}

// Real excerpt of G1 attempt 14 (docs/g1-log.md): 18 minutes without any gain from 00:38, ten of them in
// "Waiting: <area> banned" in town.
func TestAttempt14IdleFires(t *testing.T) {
	ms := fixture(t, "soak_a14_minutes.jsonl")
	lim := DefaultProgressLimits()
	lim.NoGain = 25 * time.Minute // the healthy start has a real 20 minute stretch between gains
	c := findCodes(AnalyzeMinutes(ms, lim))
	if !c["idle_in_town"] || !c["no_gain"] {
		t.Fatalf("codes %v", c)
	}
	if c["loop_no_gain"] {
		t.Fatalf("no loop expected: %v", c)
	}
}

func TestAttempt14HealthyStartIsQuiet(t *testing.T) {
	ms := fixture(t, "soak_a14_minutes.jsonl")
	cut := ms[0].At.Add(40 * time.Minute)
	n := 0
	for n < len(ms) && !ms[n].At.After(cut) {
		n++
	}
	lim := DefaultProgressLimits()
	lim.NoGain = 25 * time.Minute // the healthy start has a real 20 minute stretch between gains
	pr := AnalyzeMinutes(ms[:n], lim)
	for _, f := range pr.Findings {
		if f.Level == "fail" {
			t.Fatalf("healthy start failed: %+v (%d lines)", f, n)
		}
	}
}

// Attempt 10: recovery phase at clvl 10, 40 minutes, gold stays around zero and the quest step never moves.
func TestAttempt10BrokeRecovery(t *testing.T) {
	ms := fixture(t, "soak_a10_minutes.jsonl")
	lim := DefaultProgressLimits()
	lim.NoGain = 30 * time.Minute
	// The fixture gains about 3.3k xp in 40 minutes, which the xp term counts as progress; without it the stall shows.
	if findCodes(AnalyzeMinutes(ms, lim))["no_gain"] {
		t.Fatal("xp gain must count as progress by default")
	}
	lim.NoGainMinXP = -1
	pr := AnalyzeMinutes(ms, lim)
	if !findCodes(pr)["no_gain"] {
		t.Fatalf("%+v", pr.Findings)
	}
	if pr.Groups["recovery"] <= 0 || len(pr.Dominant) == 0 || !findCodes(pr)["state_dominant"] {
		t.Fatalf("recovery share: %v %v", pr.Groups, pr.Dominant)
	}
	// m1's 45 minute limit does not fire on a 40 minute run.
	lim.NoGain = 45 * time.Minute
	if findCodes(AnalyzeMinutes(ms, lim))["no_gain"] {
		t.Fatal("45m must not fire")
	}
}

func mk(t0 time.Time, i int, clvl int, gold int, xp uint64, area int, step string) Minute {
	return Minute{At: t0.Add(time.Duration(i) * time.Minute), Clvl: clvl, Gold: gold, XP: xp, Area: area, Step: step}
}

func TestLoopNoGain(t *testing.T) {
	t0 := time.Date(2026, 10, 30, 1, 0, 0, 0, time.UTC)
	var ms []Minute
	for i := 0; i < 50; i += 2 { // two steps alternate: 25 entries of each, no gain
		ms = append(ms, mk(t0, i, 9, 100, 5000, 2, "Going to town ([pack 90% full])"))
		ms = append(ms, mk(t0, i+1, 9, 100, 5000, 1, "Returning to the field"))
	}
	lim := DefaultProgressLimits()
	lim.NoGain = 0
	if !findCodes(AnalyzeMinutes(ms, lim))["loop_no_gain"] {
		t.Fatal("loop not found")
	}
	for i := range ms { // real xp breaks it
		ms[i].XP += uint64(i) * 100
	}
	if findCodes(AnalyzeMinutes(ms, lim))["loop_no_gain"] {
		t.Fatal("xp gain must break the loop")
	}
}

func TestBelowFloor(t *testing.T) {
	t0 := time.Date(2026, 10, 30, 1, 0, 0, 0, time.UTC)
	var ms []Minute
	for i := 0; i <= 250; i += 5 {
		ms = append(ms, mk(t0, i, 6, 0, uint64(i), 1, "Quest: fighting (Level up to 14)"))
	}
	lim := DefaultProgressLimits()
	lim.NoGain = 0
	pr := AnalyzeMinutes(ms, lim)
	if len(pr.Findings) < 2 || pr.Findings[0].Level != "warn" || !findCodes(pr)["below_floor_clvl"] {
		t.Fatalf("%+v", pr.Findings)
	}
	lim.FailBelowFloor = true
	var fails int
	for _, f := range AnalyzeMinutes(ms, lim).Findings {
		if f.Code == "below_floor_clvl" && f.Level == "fail" {
			fails++
		}
	}
	if fails != 2 { // hour 2 (7) and hour 4 (11)
		t.Fatalf("fails %d", fails)
	}
	// A character that starts at 12 still has to be at 12 at hour 4, and the 2 h cell is the table value only.
	for i := range ms {
		ms[i].Clvl = 8
	}
	ms[0].Clvl = 12
	pr = AnalyzeMinutes(ms, lim)
	var msgs []string
	for _, f := range pr.Findings {
		msgs = append(msgs, f.Msg)
	}
	if !strings.Contains(strings.Join(msgs, "\n"), "floor 12 (") {
		t.Fatalf("start floor missing: %v", msgs)
	}
}

func TestStepGroupsAndText(t *testing.T) {
	for step, want := range map[string]string{
		"Quest: fighting (Level up to 14)":           "farming",
		"Quest: fighting (Kill Blood Raven)":         "quest fight",
		"Quest: travelling (Go to the Barracks)":     "quest travel",
		"Recovery: farming gold":                     "recovery",
		"Chores in town ([heal])":                    "town chores",
		"Waiting: Den of Evil banned for 7 more min": "waiting",
		"Retrieving the corpse":                      "dead-recovering",
		"Something new":                              "other",
	} {
		if g, _ := stepGroup(step); g != want {
			t.Errorf("%q -> %q, want %q", step, g, want)
		}
	}
	pr := AnalyzeMinutes(fixture(t, "soak_a14_minutes.jsonl"), DefaultProgressLimits())
	out := pr.Text()
	for _, w := range []string{"clvl at  1 h", "xp/h clvl 10-13", "time in state farming"} {
		if !strings.Contains(out, w) {
			t.Fatalf("%q missing:\n%s", w, out)
		}
	}
	if len(pr.Unmapped) != 0 {
		t.Fatalf("unmapped %v", pr.Unmapped)
	}
}

// G1 attempt 16, 03:15-05:12: clvl 13 -> 14, 23k xp in two hours, gold spent as fast as earned, no quest-step change.
// The M0 rule false-failed this as "no gain for 77m".
func TestAttempt16HealthyLevelIsQuiet(t *testing.T) {
	ms := fixture(t, "soak_a16_minutes.jsonl")
	pr := AnalyzeMinutes(ms, DefaultProgressLimits())
	for _, f := range pr.Findings {
		if f.Level == "fail" || f.Code == "loop_no_gain" || f.Code == "below_floor_clvl" {
			t.Fatalf("healthy long level: %+v", f)
		}
	}
	lim := DefaultProgressLimits()
	lim.NoGainMinXP, lim.NoGain = -1, 45*time.Minute // m1's limit
	if !findCodes(AnalyzeMinutes(ms, lim))["no_gain"] {
		t.Fatal("the old rule (xp ignored) should reproduce the false positive; the fixture lost its point")
	}
}

func TestNoGainXPTrickleStillFails(t *testing.T) {
	t0 := time.Date(2026, 10, 30, 1, 0, 0, 0, time.UTC)
	var ms []Minute
	for i := 0; i <= 100; i++ { // 5 xp a minute: 500 over 100 min, below the threshold within every 90 min
		ms = append(ms, mk(t0, i, 13, 80, 120000+uint64(i)*4, 3, "Quest: fighting (Level up to 14)"))
	}
	if !findCodes(AnalyzeMinutes(ms, DefaultProgressLimits()))["no_gain"] {
		t.Fatal("a trickle must still fail")
	}
	for i := range ms { // 2k/h pace is real progress
		ms[i].XP = 120000 + uint64(i)*40
	}
	if findCodes(AnalyzeMinutes(ms, DefaultProgressLimits()))["no_gain"] {
		t.Fatal("real xp must pass")
	}
}

func TestXPProgressThreshold(t *testing.T) {
	if got := xpProgressThreshold(13, 0); got != 500 {
		t.Fatalf("clvl 13: %d", got)
	}
	if got := xpProgressThreshold(40, 0); got != 19644 { // 1% of 19235252-17270791
		t.Fatalf("clvl 40: %d", got)
	}
	if got := xpProgressThreshold(13, 2000); got != 2000 {
		t.Fatalf("custom: %d", got)
	}
}
