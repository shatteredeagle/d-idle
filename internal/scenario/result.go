package scenario

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ResultVersion is the result file format.
const ResultVersion = "didle.scenario-result/1"

// Statuses. RED is a failed assertion on observed state; GREEN all passed; PENDING a skeleton; BLOCKED the run could not
// observe or construct what the scenario needs (a missing verb or build, an unobserved fact): never a pass.
const (
	Green   = "GREEN"
	Red     = "RED"
	Pending = "PENDING"
	Blocked = "BLOCKED"
)

// Result is the outcome of one run.
type Result struct {
	Version string `json:"version"`
	Run
	Status  string         `json:"status"`
	Reason  string         `json:"reason,omitempty"` // why PENDING or BLOCKED, or the first failed assertion
	Asserts []AssertResult `json:"asserts,omitempty"`
	// VirtualMin is how many virtual minutes the bot played; BudgetMin the scenario's budget.
	VirtualMin float64 `json:"virtual_min"`
	BudgetMin  float64 `json:"budget_min"`
	WallSec    int     `json:"wall_sec"`
	// Tags are the provenance tags of the setup (CHEATED ...); a CHEATED run is mechanics evidence only.
	Tags []string `json:"tags,omitempty"`
	// Trace is the path of the bot's decision trace (what it chose and why): output for the human, never asserted on.
	Trace string `json:"trace,omitempty"`
	// Taps are the board taps the driver made ("start scroll", "handin read"), in order (trigger.driver board-taps).
	Taps []string `json:"taps,omitempty"`
	// TapsAt is the bot's virtual second of each tap (same order); two runs of one seed should tap at equal values.
	TapsAt []float64 `json:"taps_at,omitempty"`
	State  *State    `json:"state,omitempty"`
	// Digest is the hash of the whole observed state. It is a DRIFT report: the determinism check shows a difference but
	// does not fail on it (gold 700 against 712 with the same verdict is the same run). AssertDigest gates.
	Digest string `json:"digest,omitempty"`
	// AssertDigest is D1 (assertion digest.go): verdict-level facts only; equal verdict and equal D1 are the gate.
	AssertDigest string `json:"assert_digest,omitempty"`
	// Journal is the path of the d2dctl command journal of the bot run (run -journal), for d2ddet compare.
	Journal string `json:"journal,omitempty"`
	// Prov records the binaries, host role, load and pace of the run (nil in results written before it existed).
	Prov *Provenance `json:"prov,omitempty"`
	// FromSave, SaveRun and SavePort record the earned save the hero started from (pre.from_save), see LedgerRecord.
	FromSave string `json:"from_save,omitempty"`
	SaveRun  string `json:"save_run,omitempty"`
	SavePort string `json:"save_portability,omitempty"`
	// Faults says which fault set ran and what it injected (nil: no fault proxy).
	Faults *FaultReport `json:"faults,omitempty"`
}

// FaultReport is the injections of one run: Counts by kind, Log the path of the time-stamped events (JSON lines).
type FaultReport struct {
	Set    string         `json:"set"`
	Counts map[string]int `json:"counts,omitempty"`
	Log    string         `json:"log,omitempty"`
}

// Verdict derives Status and Reason from the assertion results. A failed assertion makes the run RED even when others
// were unobserved; with no failure, any unobserved assertion makes it BLOCKED.
func Verdict(rs []AssertResult) (status, reason string) {
	var unobserved string
	for _, r := range rs {
		if r.Unobserved {
			if unobserved == "" {
				unobserved = r.Got + " (wanted " + r.Want + ")"
			}
			continue
		}
		if !r.OK {
			return Red, fmt.Sprintf("%s: wanted %s, got %s", r.Kind, r.Want, r.Got)
		}
	}
	if unobserved != "" {
		return Blocked, "unobserved: " + unobserved
	}
	return Green, ""
}

// PendingResult is the result row of a skeleton.
func PendingResult(s Scenario) Result {
	return Result{Version: ResultVersion, Run: Run{ID: s.ID}, Status: Pending, Reason: s.PendingReason, BudgetMin: s.Budget.VirtualMin}
}

// Report writes the one-screen text report: one line per run, then the totals.
func Report(w io.Writer, rs []Result) {
	sorted := append([]Result(nil), rs...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Key() < sorted[j].Key() })
	counts := map[string]int{}
	fmt.Fprintf(w, "%-8s %-34s %-10s %5s  %s\n", "STATUS", "SCENARIO", "CLASS/SEED", "VMIN", "DETAIL")
	for _, r := range sorted {
		counts[r.Status]++
		cs := "-"
		if r.Class != "" {
			cs = fmt.Sprintf("%s/%d", short(r.Class), r.Seed)
		}
		detail := r.Reason
		if len(r.Tags) > 0 && r.Tags[0] == "CHEATED" {
			detail += " [CHEATED setup]"
		}
		fmt.Fprintf(w, "%-8s %-34s %-10s %5.1f  %s\n", r.Status, r.ID, cs, r.VirtualMin, strings.TrimSpace(detail))
	}
	fmt.Fprintf(w, "\nGREEN %d  RED %d  BLOCKED %d  PENDING %d\n", counts[Green], counts[Red], counts[Blocked], counts[Pending])
}

func short(c string) string {
	if len(c) > 4 {
		return c[:4]
	}
	return c
}

// Gate says whether a result set passes the merge gate: no RED and no BLOCKED, and at least one result. PENDING does not
// fail a gate (a skeleton is not yet a promise) but is reported.
func Gate(rs []Result) bool {
	for _, r := range rs {
		if r.Status == Red || r.Status == Blocked {
			return false
		}
	}
	return len(rs) > 0
}

// WriteJSON writes the result set.
func WriteJSON(w io.Writer, rs []Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return enc.Encode(struct {
		Version string   `json:"version"`
		Results []Result `json:"results"`
	}{ResultVersion, rs})
}
