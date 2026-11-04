package main

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// contractNumbers reads the "## Numbers" table of a contract (docs/contracts/<id>.md) into key -> value.
// Only rows whose "enforced in" cell names pkg are returned. The table is owned by the contract (user-approved
// gate numbers); this test fails when the code defaults drift from it (docs/process/contracts-after-openspec.md 3.4).
func contractNumbers(t *testing.T, id, pkg string) map[string]float64 {
	t.Helper()
	b, err := os.ReadFile("../../docs/contracts/" + id + ".md")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	i := strings.Index(s, "\n## Numbers")
	if i < 0 {
		t.Skipf("contract %s has no ## Numbers table", id)
	}
	out := map[string]float64{}
	for _, ln := range strings.Split(s[i+1:], "\n")[1:] {
		if strings.HasPrefix(ln, "## ") {
			break
		}
		c := strings.Split(ln, "|")
		if len(c) < 5 || strings.TrimSpace(c[1]) == "key" || strings.Trim(strings.TrimSpace(c[1]), "-: ") == "" {
			continue
		}
		if !strings.Contains(c[4], pkg) {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(c[2]), 64)
		if err != nil {
			t.Fatalf("%s: row %s: value %q is not a number", id, strings.TrimSpace(c[1]), c[2])
		}
		out[strings.TrimSpace(c[1])] = v
	}
	if len(out) == 0 {
		t.Skipf("contract %s has no Numbers row for %s", id, pkg)
	}
	return out
}

func minutes(d time.Duration) float64 { return d.Minutes() }

func checkRows(t *testing.T, id string, rows map[string]float64, code map[string]float64) {
	t.Helper()
	for k, want := range rows {
		got, ok := code[k]
		if !ok {
			t.Errorf("contract %s row %s is not covered by this test: add it to the code map", id, k)
			continue
		}
		if got != want {
			t.Errorf("contract %s %s: contract says %v, code has %v", id, k, want, got)
		}
	}
}

// contract death-rule: Numbers table.
func TestContractDeathRuleNumbers(t *testing.T) {
	d := DefaultDeathLimits()
	checkRows(t, "death-rule", contractNumbers(t, "death-rule", "cmd/d2soak"), map[string]float64{
		"death.streakDeaths":      float64(d.StreakDeaths),
		"death.streakWindowMin":   minutes(d.StreakWindow),
		"death.hourDeathsMax":     float64(d.HourDeaths),
		"death.warnDeaths":        float64(d.WarnDeaths),
		"death.recoveryFailShare": d.RecoveryFail,
		"death.recoveryWarnShare": d.RecoveryWarn,
		"death.maxBackMin":        minutes(d.MaxBack),
		"death.minRunMin":         minutes(d.MinRun),
		"death.ratePerK":          d.RatePerK,
		"death.rateWarnPerK":      d.RateWarnPerK,
		"death.minKills":          float64(d.MinKills),
		"death.repeatWindowMin":   minutes(d.RepeatWindow),
		"death.repeatGapSec":      d.RepeatGap.Seconds(),
	})
}

// contract soak-gates: Numbers table (m24 profile, progress and heap limits).
func TestContractSoakGatesNumbers(t *testing.T) {
	l := DefaultLimits()
	profiles["m24"](&l)
	a := DefaultAlertRules()
	checkRows(t, "soak-gates", contractNumbers(t, "soak-gates", "cmd/d2soak"), map[string]float64{
		"m24.stallMin":          minutes(l.MaxNoProgress),
		"m24.noGainMin":         minutes(l.Progress.NoGain),
		"m24.unhealthyMin":      minutes(l.MaxUnhealthy),
		"m24.warmUpMin":         minutes(l.WarmUp),
		"m24.rssMBPerHour":      l.MaxRSSMBPerHour,
		"m24.fdsPerHour":        l.MaxFDsPerHour,
		"m24.threadsPerHour":    l.MaxThreadsPerHour,
		"m24.goroutinesPerHour": l.MaxGoroutinesPerH,
		"m24.ladderPerHour":     l.MaxLadderPerHour,
		"m24.sentinelPerHour":   l.MaxSentinelPerHr,
		"heap.runawayMB":        a.HeapRunawayMB,
		"heap.growthMB":         a.HeapGrowthMB,
		"heap.growthWindowMin":  minutes(a.HeapGrowthWindow),
	})
	if minutes(a.StallXP) != minutes(l.MaxNoProgress) {
		t.Errorf("alert StallXP %v and m24 MaxNoProgress %v must agree (contract m24.stallMin)", a.StallXP, l.MaxNoProgress)
	}
}
