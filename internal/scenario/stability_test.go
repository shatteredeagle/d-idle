package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// run builds a counted ledger line; n orders the lines in time.
func run(n int, class, verdict, d1, commit string) LedgerRecord {
	return LedgerRecord{Run: fmt.Sprintf("j%d/heal.town/%s/11", n, class), Scenario: "heal.town", Class: class, Seed: 11,
		Verdict: verdict, D1: d1, BotCommit: commit, D2DSHA256: "d2d1", Started: fmt.Sprintf("2026-11-27T10:%02d:00Z", n), Parallel: 2}
}

// Five equal runs that all ran alone are not enough: the design wants two of them at parallel 2. Dropping the
// parallel rule makes this test fail.
func TestStabilityNeedsTwoParallelRunsMustFail(t *testing.T) {
	var recs []LedgerRecord
	for i := 0; i < StableRuns+2; i++ {
		r := run(i, "Paladin", Green, "aa", "c1")
		r.Parallel = 1
		if i == 0 { // an old line without the field counts as alone
			r.Parallel = 0
		}
		recs = append(recs, r)
	}
	if s := keyState(t, mustStability(t, recs, nil), "Paladin"); s.State != StabilityCandidate || s.StreakParallel != 0 {
		t.Fatalf("all alone: %+v, want candidate", s)
	}
	recs[2].Parallel = 2
	if s := keyState(t, mustStability(t, recs, nil), "Paladin"); s.State != StabilityCandidate || s.StreakParallel != 1 {
		t.Fatalf("one parallel run: %+v, want candidate", s)
	}
	recs[5].Parallel = 3
	s := keyState(t, mustStability(t, recs, nil), "Paladin")
	if s.State != StabilityStable || s.Since != recs[5].Run || s.StreakParallel != 2 {
		t.Fatalf("two parallel runs in the streak: %+v, want stable since the sixth", s)
	}
	// A binary change resets the parallel count with the streak.
	recs = recs[:6]
	recs[3].BotCommit, recs[4].BotCommit, recs[5].BotCommit = "c2", "c2", "c2"
	if s := keyState(t, mustStability(t, recs, nil), "Paladin"); s.State != StabilityCandidate || s.StreakParallel != 1 {
		t.Fatalf("parallel runs across a binary change: %+v, want candidate", s)
	}
}

func keyState(t *testing.T, ks []KeyStability, class string) KeyStability {
	t.Helper()
	for _, k := range ks {
		if k.Key.Class == class {
			return k
		}
	}
	t.Fatalf("no key for %s in %+v", class, ks)
	return KeyStability{}
}

func mustStability(t *testing.T, recs []LedgerRecord, rels []Release) []KeyStability {
	t.Helper()
	ks, err := Stability(recs, rels)
	if err != nil {
		t.Fatal(err)
	}
	return ks
}

func TestStabilityFiveEqualRunsMakeStable(t *testing.T) {
	var recs []LedgerRecord
	for i := 0; i < StableRuns-1; i++ {
		recs = append(recs, run(i, "Paladin", Green, "aa", "c1"))
	}
	if s := keyState(t, mustStability(t, recs, nil), "Paladin"); s.State != StabilityCandidate || s.Streak != StableRuns-1 {
		t.Fatalf("4 runs: %+v, want candidate", s)
	}
	recs = append(recs, run(9, "Paladin", Green, "aa", "c1"))
	s := keyState(t, mustStability(t, recs, nil), "Paladin")
	if s.State != StabilityStable || s.Since != recs[4].Run || s.LastGreen != recs[4].Run {
		t.Fatalf("5 runs: %+v, want stable since the fifth", s)
	}
	if !Gates(s.State, TierEntry{Tier: TierT2}) || Gates(s.State, TierEntry{Tier: TierNone}) {
		t.Fatal("a stable T2 scenario gates, a none-tier one never")
	}
}

// The flip rule must fail the scenario: a verdict or D1 change on the same bot commit and d2d hash quarantines it at
// once, even when it was stable, and a quarantined scenario never gates. Removing the rule breaks every case here.
func TestStabilityFlipQuarantinesMustFail(t *testing.T) {
	for _, tc := range []struct {
		name         string
		last         LedgerRecord
		wantFlakeOut bool
	}{
		{"verdict flip", run(9, "Paladin", Red, "aa", "c1"), true},
		{"d1 flip, both green", run(9, "Paladin", Green, "bb", "c1"), true},
		{"new bot commit, other outcome", run(9, "Paladin", Red, "bb", "c2"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var recs []LedgerRecord
			for i := 0; i < StableRuns; i++ {
				recs = append(recs, run(i, "Paladin", Green, "aa", "c1"))
			}
			recs = append(recs, tc.last)
			s := keyState(t, mustStability(t, recs, nil), "Paladin")
			if got := s.State == StabilityQuarantined; got != tc.wantFlakeOut {
				t.Fatalf("state %s, quarantined want %v", s.State, tc.wantFlakeOut)
			}
			if tc.wantFlakeOut {
				if len(s.Flakes) != 1 || s.Flakes[0].Run != tc.last.Run || s.Since != tc.last.Run {
					t.Fatalf("flake event: %+v", s)
				}
				if Gates(s.State, TierEntry{Tier: TierT2}) {
					t.Fatal("a quarantined scenario gates")
				}
			} else if s.State != StabilityStable || len(s.Flakes) != 0 {
				t.Fatalf("a binary change is not a flake: %+v", s)
			}
		})
	}
}

// A failed blocking run re-run once on the same binaries that then passes is a flake by definition (design 2.7).
func TestStabilityRerunPassQuarantinesCandidate(t *testing.T) {
	recs := []LedgerRecord{run(0, "Sorceress", Red, "aa", "c1"), run(1, "Sorceress", Green, "bb", "c1")}
	if s := keyState(t, mustStability(t, recs, nil), "Sorceress"); s.State != StabilityQuarantined {
		t.Fatalf("%+v", s)
	}
}

func TestStabilityBinaryChangeResetsStreak(t *testing.T) {
	recs := []LedgerRecord{run(0, "Paladin", Green, "aa", "c1"), run(1, "Paladin", Green, "aa", "c1"), run(2, "Paladin", Green, "aa", "c1"),
		run(3, "Paladin", Green, "aa", "c2"), run(4, "Paladin", Green, "aa", "c2")}
	if s := keyState(t, mustStability(t, recs, nil), "Paladin"); s.State != StabilityCandidate || s.Streak != 2 {
		t.Fatalf("%+v, want candidate with streak 2", s)
	}
}

func TestStabilitySkipsUncountedRuns(t *testing.T) {
	blocked := run(1, "Paladin", Blocked, "zz", "c1")
	cut := run(2, "Paladin", Red, "zz", "c1")
	cut.Cut = true
	dirty := run(3, "Paladin", Red, "zz", "c1")
	dirty.BotDirty = true
	nod1 := run(4, "Paladin", Red, "", "c1")
	recs := []LedgerRecord{run(0, "Paladin", Green, "aa", "c1"), blocked, cut, dirty, nod1, run(5, "Paladin", Green, "aa", "c1")}
	s := keyState(t, mustStability(t, recs, nil), "Paladin")
	if s.State != StabilityCandidate || s.Runs != 2 || s.Streak != 2 {
		t.Fatalf("%+v, want 2 counted runs and no flake", s)
	}
}

func TestStabilityOrdersByStart(t *testing.T) {
	var recs []LedgerRecord
	for i := StableRuns - 1; i >= 0; i-- { // ledger order reversed against start time
		recs = append(recs, run(i, "Paladin", Green, "aa", "c1"))
	}
	if s := keyState(t, mustStability(t, recs, nil), "Paladin"); s.State != StabilityStable || s.Last != recs[0].Run {
		t.Fatalf("%+v", s)
	}
}

// Two classes of one scenario count apart; the scenario word is quarantined when any of them is.
func TestStabilityTwoClasses(t *testing.T) {
	var recs []LedgerRecord
	for i := 0; i < StableRuns; i++ {
		recs = append(recs, run(2*i, "Paladin", Green, "aa", "c1"), run(2*i+1, "Necromancer", Green, "nn", "c1"))
	}
	ks := mustStability(t, recs, nil)
	if got := ScenarioStability(ks)["heal.town"]; got != StabilityStable {
		t.Fatalf("both stable: %s", got)
	}
	recs = append(recs, run(40, "Necromancer", Green, "n2", "c1"))
	ks = mustStability(t, recs, nil)
	if keyState(t, ks, "Paladin").State != StabilityStable || keyState(t, ks, "Necromancer").State != StabilityQuarantined {
		t.Fatalf("%+v", ks)
	}
	if got := ScenarioStability(ks)["heal.town"]; got != StabilityQuarantined {
		t.Fatalf("roll-up %s", got)
	}
}

func TestStabilityReleaseNeedsPersonAndNoteThenFiveRuns(t *testing.T) {
	recs := []LedgerRecord{run(0, "Paladin", Green, "aa", "c1"), run(1, "Paladin", Red, "aa", "c1")}
	for _, bad := range []Release{{Scenario: "heal.town", At: "2026-11-27T10:05:00Z", By: "owner"},
		{Scenario: "heal.town", At: "2026-11-27T10:05:00Z", Note: "cause"}} {
		if _, err := Stability(recs, []Release{bad}); err == nil {
			t.Fatalf("release %+v accepted", bad)
		}
	}
	rel := []Release{{Scenario: "heal.town", At: "2026-11-27T10:05:00Z", By: "owner", Note: "tap on the wall clock, fixed in pgtaps"}}
	for i := 10; i < 10+StableRuns-1; i++ {
		recs = append(recs, run(i, "Paladin", Green, "aa", "c1"))
	}
	if s := keyState(t, mustStability(t, recs, rel), "Paladin"); s.State != StabilityCandidate || s.Streak != StableRuns-1 {
		t.Fatalf("after release and 4 runs: %+v", s)
	}
	recs = append(recs, run(20, "Paladin", Green, "aa", "c1"))
	if s := keyState(t, mustStability(t, recs, rel), "Paladin"); s.State != StabilityStable {
		t.Fatalf("after release and 5 runs: %+v", s)
	}
	if s := keyState(t, mustStability(t, recs, nil), "Paladin"); s.State != StabilityQuarantined {
		t.Fatalf("without a release the tool never lets go: %+v", s)
	}
}

const scenarioDir = "../../tools/provingground/scenarios"

func TestTiersCoverEveryScenario(t *testing.T) {
	scs, err := LoadDir(scenarioDir)
	if err != nil {
		t.Fatal(err)
	}
	tf, err := LoadTiers(filepath.Join(scenarioDir, "tiers.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bad := tf.CheckTiers(scs); len(bad) > 0 {
		t.Fatalf("tiers.json out of date (regenerate the missing rows from the catalogue mapping): %v", bad)
	}
	// Catalogue rows of design 2026-11-21 (proving-ground-gate-catalog.md): the seven smoke candidates and the marker.
	for id, want := range map[string]string{"heal.town": TierT2, "town.resupply": TierT2, "points.bank-by-default": TierT2,
		"cain.chain": TierT4, "cain.read": TierT3, "cain.stones.shipped": TierNone, "act1.andariel.slaughter": TierT4} {
		if e := tf.Tiers[id]; e.Tier != want || e.Source != "catalogue" {
			t.Errorf("%s: %+v, want %s from the catalogue", id, e, want)
		}
	}
}

func TestLoadTiersRejects(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"version": `{"version":"x","tiers":{}}`,
		"tier":    `{"version":"didle.pg-tiers/1","tiers":{"a":{"tier":"T9","source":"map"}}}`,
		"source":  `{"version":"didle.pg-tiers/1","tiers":{"a":{"tier":"T2","source":"guess"}}}`,
		"field":   `{"version":"didle.pg-tiers/1","tiers":{"a":{"tier":"T2","source":"map","x":1}}}`,
	} {
		p := filepath.Join(dir, name+".json")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadTiers(p); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	tf := TierFile{Tiers: map[string]TierEntry{"gone": {Tier: TierT3, Source: "map"}}}
	if bad := tf.CheckTiers([]Scenario{{ID: "new"}}); len(bad) != 2 {
		t.Fatalf("%v", bad)
	}
}

// The streak keys on the bot BINARY, not the commit: a new commit that builds the same binary continues the streak (and
// a flip across it is a flake), a new binary resets it. Lines without a hash fall back to the commit. Two classes.
func TestStabilityStreakKeysOnBotBinaryHash(t *testing.T) {
	for _, class := range []string{"Paladin", "Sorceress"} {
		var recs []LedgerRecord
		for i := 0; i < StableRuns; i++ {
			r := run(i, class, Green, "aa", fmt.Sprintf("commit%d", i)) // a merge between every run
			r.BotSHA256 = "bin1"
			recs = append(recs, r)
		}
		if s := keyState(t, mustStability(t, recs, nil), class); s.State != StabilityStable || s.Streak != StableRuns {
			t.Fatalf("%s: same binary over %d commits: %+v, want stable", class, StableRuns, s)
		}
		for i := range recs {
			recs[i].BotSHA256 = ""
		}
		if s := keyState(t, mustStability(t, recs, nil), class); s.State != StabilityCandidate || s.Streak != 1 {
			t.Fatalf("%s: no hash, a commit per run: %+v, want candidate with streak 1", class, s)
		}
		same := run(0, class, Green, "aa", "c1")
		next := run(1, class, Green, "aa", "c1") // the same commit string, another binary (a dirty or re-stamped tree)
		same.BotSHA256, next.BotSHA256 = "bin1", "bin2"
		if s := keyState(t, mustStability(t, []LedgerRecord{same, next}, nil), class); s.Streak != 1 {
			t.Fatalf("%s: a new binary must reset the streak: %+v", class, s)
		}
		flip := run(1, class, Red, "bb", "c2")
		flip.BotSHA256 = "bin1"
		if s := keyState(t, mustStability(t, []LedgerRecord{same, flip}, nil), class); s.State != StabilityQuarantined {
			t.Fatalf("%s: a flip on one binary across two commits is a flake: %+v", class, s)
		}
	}
}

// BLOCKED runs are reported per key and are never evidence: they neither extend nor reset the streak, and a key that
// only ever blocked is listed (candidate, no counted run).
func TestStabilityReportsBlockedWithoutCountingIt(t *testing.T) {
	recs := []LedgerRecord{run(0, "Paladin", Green, "aa", "c1"), run(1, "Paladin", Blocked, "", "c1"), run(2, "Paladin", Green, "aa", "c1"),
		run(3, "Paladin", Blocked, "", "c1"), run(4, "Paladin", Blocked, "", "c1"), run(5, "Sorceress", Blocked, "", "c1")}
	keys := mustStability(t, recs, nil)
	if s := keyState(t, keys, "Paladin"); s.Runs != 2 || s.Streak != 2 || s.Blocked != 3 || s.BlockedNow != 2 || s.LastBlocked != recs[4].Run || len(s.Flakes) != 0 {
		t.Fatalf("%+v", s)
	}
	if s := keyState(t, keys, "Sorceress"); s.State != StabilityCandidate || s.Runs != 0 || s.Blocked != 1 || s.BlockedNow != 1 {
		t.Fatalf("%+v", s)
	}
}
