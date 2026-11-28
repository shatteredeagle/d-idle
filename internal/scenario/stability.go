package scenario

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Stability states of a scenario (design 2026-11-21 section 2.7, owner answers Q1 and Q7). Only a stable scenario may
// gate; a quarantined one keeps running nightly and gates nothing.
const (
	StabilityCandidate   = "candidate"
	StabilityStable      = "stable"
	StabilityQuarantined = "quarantined"
)

// StableRuns is the number of consecutive equal runs (verdict D0 and assertion digest D1, one class and seed, unchanged
// bot BINARY and d2d hash, see sameBot) that makes a candidate stable; StableParallelRuns of them must have run while another slot
// was busy (ledger parallel >= 2), so a scenario that only passes alone never gates (design section 2.7).
const (
	StableRuns         = 5
	StableParallelRuns = 2
)

// StabilityKey is one class and seed of a scenario: the unit the N-run rule counts.
type StabilityKey struct {
	Scenario string `json:"scenario"`
	Class    string `json:"class"`
	Seed     int64  `json:"seed"`
}

func (k StabilityKey) String() string { return fmt.Sprintf("%s/%s/%d", k.Scenario, k.Class, k.Seed) }

// FlakeEvent is a same-binary flip: two consecutive counted runs of one key on the same bot commit and d2d hash with a
// different verdict or assertion digest.
type FlakeEvent struct {
	Key       StabilityKey `json:"key"`
	Prev      string       `json:"prev"` // run id
	Run       string       `json:"run"`
	Detail    string       `json:"detail"`
	BotCommit string       `json:"bot_commit"`
	D2DSHA256 string       `json:"d2d_sha256"`
}

// Release takes a quarantined scenario back to candidate. Only a person releases (Q7), and only with the root cause
// written down; the scenario then needs StableRuns clean runs again, counted from runs started after At.
type Release struct {
	Scenario string `json:"scenario"`
	At       string `json:"at"` // RFC3339 UTC, compared as a string with the ledger's started
	By       string `json:"by"`
	Note     string `json:"note"` // the root cause
}

// Validate refuses a release without a person or a root cause.
func (r Release) Validate() error {
	switch {
	case r.Scenario == "":
		return fmt.Errorf("release: no scenario")
	case r.At == "":
		return fmt.Errorf("release %s: no time", r.Scenario)
	case strings.TrimSpace(r.By) == "":
		return fmt.Errorf("release %s: no person", r.Scenario)
	case strings.TrimSpace(r.Note) == "":
		return fmt.Errorf("release %s: no root cause note", r.Scenario)
	}
	return nil
}

// KeyStability is the derived state of one key.
type KeyStability struct {
	Key    StabilityKey `json:"key"`
	State  string       `json:"state"`
	Streak int          `json:"streak"` // equal runs on the current binaries since the last reset
	// StreakParallel counts the runs of the streak that had another slot busy (parallel >= 2).
	StreakParallel int    `json:"streak_parallel"`
	Runs           int    `json:"runs"`  // counted runs (BLOCKED, cut, dirty and digest-less runs are not counted)
	Since          string `json:"since"` // run id that set the current state ("" for a candidate never stable)
	Last           string `json:"last"`  // last counted run id
	// LastGreen is the last counted GREEN run: the one a T1 state fixture is cut from once the key is stable.
	LastGreen string       `json:"last_green,omitempty"`
	Flakes    []FlakeEvent `json:"flakes,omitempty"`
	// Blocked counts the BLOCKED runs of the key and BlockedNow those since its last counted run. They are reported and
	// never evidence: a BLOCKED run neither extends nor resets a streak. A key with BlockedNow > 0 is not being judged.
	Blocked     int    `json:"blocked,omitempty"`
	BlockedNow  int    `json:"blocked_now,omitempty"`
	LastBlocked string `json:"last_blocked,omitempty"`
}

// sameBot says whether two ledger lines ran the same bot: by the binary hash when both carry one (lines since fix6),
// else by the commit. The hash is the better key: a merge that does not touch the bot's code builds the same binary
// (tools/provingground/run.sh builds with -trimpath -buildvcs=false), so the streak survives it, where "one commit"
// reset every streak many times a day and nothing could become stable (review 2026-11-28 E6).
func sameBot(a, b LedgerRecord) bool {
	if a.BotSHA256 != "" && b.BotSHA256 != "" {
		return a.BotSHA256 == b.BotSHA256
	}
	return a.BotCommit == b.BotCommit
}

// counted says whether a ledger line is evidence for the N-run rule. A BLOCKED or cut run observed nothing comparable,
// a dirty build has no identifiable binary, and a line without D1 cannot be compared.
func counted(r LedgerRecord) bool {
	return (r.Verdict == Green || r.Verdict == Red) && !r.Cut && !r.BotDirty && r.D1 != "" && r.BotCommit != "" && r.D2DSHA256 != ""
}

// Stability derives the state of every key from the ledger. The state is never stored by hand: it is a pure function
// of the ledger lines (ordered by start time, ties in ledger order) and the releases, so the tool quarantines and a
// person only releases. Rules:
//   - a candidate becomes stable after StableRuns consecutive counted runs with equal verdict and D1 on one bot binary
//     (sameBot) and d2d hash, StableParallelRuns of them at parallel 2 or more; a binary change resets the streak and is not a
//     flake;
//   - a stable key stays stable across binary changes (a new outcome on new binaries is a regression or a fix, which the
//     verdict reports);
//   - any verdict or D1 flip between consecutive counted runs on the same bot binary and d2d hash quarantines the key at
//     once, whatever its state;
//   - a BLOCKED run is counted apart (Blocked, BlockedNow) and changes no state or streak;
//   - a release of the scenario takes its quarantined keys back to candidate; runs before the release no longer count.
func Stability(recs []LedgerRecord, rels []Release) ([]KeyStability, error) {
	for _, r := range rels {
		if err := r.Validate(); err != nil {
			return nil, err
		}
	}
	ord := make([]LedgerRecord, len(recs))
	copy(ord, recs)
	sort.SliceStable(ord, func(i, j int) bool { return ord[i].Started < ord[j].Started })
	rels = append([]Release(nil), rels...)
	sort.SliceStable(rels, func(i, j int) bool { return rels[i].At < rels[j].At })

	st := map[StabilityKey]*KeyStability{}
	prev := map[StabilityKey]LedgerRecord{}
	var order []StabilityKey
	released := func(scn, upTo string, from string) bool { // a release of scn in (from, upTo]
		for _, rl := range rels {
			if rl.Scenario == scn && rl.At > from && rl.At <= upTo {
				return true
			}
		}
		return false
	}
	for _, r := range ord {
		if !counted(r) && r.Verdict != Blocked {
			continue
		}
		k := StabilityKey{r.Scenario, r.Class, r.Seed}
		s, seen := st[k]
		if !seen {
			s = &KeyStability{Key: k, State: StabilityCandidate}
			st[k] = s
			order = append(order, k)
		}
		if r.Verdict == Blocked {
			s.Blocked++
			s.BlockedNow++
			s.LastBlocked = r.Run
			continue
		}
		s.BlockedNow = 0
		p, hasPrev := prev[k]
		if hasPrev && s.State == StabilityQuarantined && released(k.Scenario, r.Started, p.Started) {
			s.State, s.Since, s.Streak, s.StreakParallel, hasPrev = StabilityCandidate, "", 0, 0, false
		}
		s.Runs++
		s.Last = r.Run
		if r.Verdict == Green {
			s.LastGreen = r.Run
		}
		sameBin := hasPrev && sameBot(p, r) && p.D2DSHA256 == r.D2DSHA256
		switch {
		case !sameBin:
			s.Streak, s.StreakParallel = 1, 0
		case p.Verdict != r.Verdict || p.D1 != r.D1:
			s.Flakes = append(s.Flakes, FlakeEvent{Key: k, Prev: p.Run, Run: r.Run, BotCommit: r.BotCommit, D2DSHA256: r.D2DSHA256,
				Detail: fmt.Sprintf("verdict %s -> %s, d1 %s -> %s", p.Verdict, r.Verdict, p.D1, r.D1)})
			s.Streak, s.StreakParallel = 1, 0
			if s.State != StabilityQuarantined {
				s.State, s.Since = StabilityQuarantined, r.Run
			}
		default:
			s.Streak++
		}
		if r.Parallel >= 2 {
			s.StreakParallel++
		}
		if s.State == StabilityCandidate && s.Streak >= StableRuns && s.StreakParallel >= StableParallelRuns {
			s.State, s.Since = StabilityStable, r.Run
		}
		prev[k] = r
	}
	out := make([]KeyStability, 0, len(order))
	for _, k := range order {
		out = append(out, *st[k])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key.String() < out[j].Key.String() })
	return out, nil
}

// ScenarioStability rolls the keys up to one word per scenario: quarantined when any key is, stable when every key that
// ran is, else candidate. A scenario with no counted and no BLOCKED run is absent (its caller reads that as candidate).
func ScenarioStability(keys []KeyStability) map[string]string {
	out := map[string]string{}
	for _, k := range keys {
		cur, ok := out[k.Key.Scenario]
		switch {
		case k.State == StabilityQuarantined || cur == StabilityQuarantined:
			out[k.Key.Scenario] = StabilityQuarantined
		case !ok:
			out[k.Key.Scenario] = k.State
		case k.State != StabilityStable:
			out[k.Key.Scenario] = StabilityCandidate
		}
	}
	return out
}

// Proving Ground tiers of the 2026-11-21 design (section 3.1). TierNone marks a scenario kept as a marker that never
// gates (BLOCKED by design).
const (
	TierT2   = "T2"
	TierT3   = "T3"
	TierT4   = "T4"
	TierNone = "none"
)

// TiersVersion is the format of tools/provingground/scenarios/tiers.json.
const TiersVersion = "didle.pg-tiers/1"

// TierEntry is the proposed tier of one scenario. Source is "catalogue" (a row of the design catalogue) or "map" (a
// scenario added after it: fast and full map to T3, nightly to T4, the conservative reading of design section 3.2).
type TierEntry struct {
	Tier   string `json:"tier"`
	Source string `json:"source"`
	Note   string `json:"note,omitempty"`
}

// TierFile is tiers.json. It sits next to the scenario files and is skipped by LoadDir.
type TierFile struct {
	Version string               `json:"version"`
	Tiers   map[string]TierEntry `json:"tiers"`
}

// LoadTiers reads tiers.json; unknown fields, an unknown tier or source are errors.
func LoadTiers(path string) (TierFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return TierFile{}, err
	}
	var f TierFile
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return TierFile{}, fmt.Errorf("%s: %w", path, err)
	}
	if f.Version != TiersVersion {
		return TierFile{}, fmt.Errorf("%s: version %q, want %q", path, f.Version, TiersVersion)
	}
	for id, e := range f.Tiers {
		switch e.Tier {
		case TierT2, TierT3, TierT4, TierNone:
		default:
			return TierFile{}, fmt.Errorf("%s: %s: tier %q", path, id, e.Tier)
		}
		if e.Source != "catalogue" && e.Source != "map" {
			return TierFile{}, fmt.Errorf("%s: %s: source %q", path, id, e.Source)
		}
	}
	return f, nil
}

// CheckTiers reports scenario ids without a tier and tier rows naming no scenario (both are errors: every scenario has
// exactly one tier). Sorted, nil when clean.
func (f TierFile) CheckTiers(scs []Scenario) []string {
	var bad []string
	ids := map[string]bool{}
	for _, s := range scs {
		ids[s.ID] = true
		if _, ok := f.Tiers[s.ID]; !ok {
			bad = append(bad, "no tier: "+s.ID)
		}
	}
	for id := range f.Tiers {
		if !ids[id] {
			bad = append(bad, "tier for unknown scenario: "+id)
		}
	}
	sort.Strings(bad)
	return bad
}

// Gates says whether a scenario may block an event at all: it must be stable and have a tier that runs (not none).
// Which event a tier blocks is the scheduler's business (design 3.2).
func Gates(stability string, t TierEntry) bool {
	return stability == StabilityStable && t.Tier != TierNone && t.Tier != ""
}
