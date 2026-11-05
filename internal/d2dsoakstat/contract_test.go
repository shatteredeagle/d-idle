package d2dsoakstat

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

// contractNumbers reads the "## Numbers" table of docs/contracts/<id>.md; only rows whose "enforced in" cell names
// pkg are returned. The contract owns the values (docs/process/contracts-after-openspec.md 3.4).
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

func checkRows(t *testing.T, id string, rows, code map[string]float64) {
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

// contract death-rule: the second copy of the rule (InvLimits) against the same table as cmd/d2soak.
func TestContractDeathRuleNumbers(t *testing.T) {
	l := DefaultInvLimits()
	checkRows(t, "death-rule", contractNumbers(t, "death-rule", "internal/d2dsoakstat"), map[string]float64{
		"death.streakDeaths":    float64(l.StreakDeaths),
		"death.streakWindowMin": float64(l.StreakMinutes),
		"death.hourDeathsMax":   float64(l.StreakAny - 1), // StreakAny is the count that fails: more than max
		"death.hourWindowMin":   float64(l.StreakAnyMinutes),
		"death.ratePerK":        l.DeathsPer1000,
		"death.minKills":        float64(l.DeathsMinKills),
	})
}

// contract soak-gates: Numbers table (Gate 1 windows and per-seed invariants).
func TestContractSoakGatesNumbers(t *testing.T) {
	l := DefaultInvLimits()
	checkRows(t, "soak-gates", contractNumbers(t, "soak-gates", "internal/d2dsoakstat"), map[string]float64{
		"gate1.beltMin":    float64(l.BeltMin),
		"gate1.noKillMin":  float64(l.NoKillMin),
		"inv.wedgeMin":     l.WedgeMin,
		"inv.progressMin":  l.ProgressMin,
		"inv.stallWarnMin": float64(l.StallWarnMin),
		"inv.recoverMin":   l.RecoverMin,
		"inv.earlyFrac":    l.EarlyFrac,
	})
}

// contract fidelity-gate: Numbers table (tolerance, band constants, sample floors in the baselines files).
func TestContractFidelityGateNumbers(t *testing.T) {
	rules := DefaultRules()
	var raw struct {
		MinSeeds        int     `json:"minSeeds"`
		MinHoursPerSeed float64 `json:"minHoursPerSeed"`
		MinTotalHours   float64 `json:"minTotalHours"`
	}
	b, err := os.ReadFile("../../tools/d2dsoak/baselines.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	code := map[string]float64{
		"fidelity.relTol":           rules["xp_per_hour"].Rel,
		"fidelity.deathsPerHourAbs": rules["deaths_per_hour"].Abs,
		"fidelity.deathsPer1000Abs": rules["deaths_per_1000_kills"].Abs,
		"fidelity.minSeeds":         float64(raw.MinSeeds),
		"fidelity.minHoursPerSeed":  raw.MinHoursPerSeed,
		"fidelity.minTotalHours":    raw.MinTotalHours,
		"band.minHours":             BandMinHours,
		"band.hardRuns":             BandHardRuns,
		"band.cvFloor":              BandCVFloor,
	}
	// the four gated metrics share one relative tolerance
	for _, k := range []string{"xp_per_hour", "kills_per_hour", "deaths_per_hour", "deaths_per_1000_kills"} {
		if r := rules[k]; !r.Gate || r.Rel != code["fidelity.relTol"] {
			t.Errorf("rule %s: gate=%v rel=%v, contract says all four gated metrics use %v", k, r.Gate, r.Rel, code["fidelity.relTol"])
		}
	}
	checkRows(t, "fidelity-gate", contractNumbers(t, "fidelity-gate", "internal/d2dsoakstat"), code)
}
