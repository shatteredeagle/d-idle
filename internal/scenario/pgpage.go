package scenario

import (
	"fmt"
	"sort"
	"strings"
)

// PGPageVersion is the format of the board page data (`d2scenario ledger page-json`), read by the ops board's /pg page.
const PGPageVersion = "didle.pg-page/1"

// PGHistory is how many runs a grid cell keeps for the click-through view (design 2026-11-21 section 5.2).
const PGHistory = 30

// PGRun is one run of a cell's history. Flip marks a verdict or D1 change against the previous run of the same seed on
// the same binaries (a flake event, design 2.7); a binary change is not a flip.
type PGRun struct {
	Run       string `json:"run"`
	Seed      int64  `json:"seed"`
	Verdict   string `json:"verdict"`
	Reason    string `json:"reason,omitempty"`
	Setup     string `json:"setup"`
	D1        string `json:"d1,omitempty"`
	BotCommit string `json:"bot_commit"`
	BotDirty  bool   `json:"bot_dirty,omitempty"`
	Host      string `json:"host"`
	Event     string `json:"event"`
	Branch    string `json:"branch,omitempty"`
	Started   string `json:"started"`
	Flip      bool   `json:"flip,omitempty"`
}

// PGCell is one scenario and class of the grid: the latest run, the derived stability (rolled up over its seeds), the
// tier, counts, and the last PGHistory runs newest first. Age is left to the page (the data carries no clock).
type PGCell struct {
	Scenario  string  `json:"scenario"`
	Class     string  `json:"class"`
	Tier      string  `json:"tier,omitempty"`
	Stability string  `json:"stability"`
	Latest    PGRun   `json:"latest"`
	Runs      int     `json:"runs"`
	Flips     int     `json:"flips"`
	History   []PGRun `json:"history"`
}

// PGPage is the whole page data for one branch filter ("" all branches).
type PGPage struct {
	Version string   `json:"version"`
	Branch  string   `json:"branch"`
	Cells   []PGCell `json:"cells"`
}

type pgKey struct{ s, c string }

// BuildPGPage derives the page from the ledger, the releases and the tiers. Stability comes from Stability over the
// whole ledger (it is a property of the scenario, not of a branch view); everything else honours the branch filter.
// Deterministic: sorted, no clock.
func BuildPGPage(recs []LedgerRecord, rels []Release, tiers TierFile, branch string) (PGPage, error) {
	keys, err := Stability(recs, rels)
	if err != nil {
		return PGPage{}, err
	}
	stab := map[pgKey]string{}
	for _, k := range keys {
		pk := pgKey{k.Key.Scenario, k.Key.Class}
		stab[pk] = rollUp(stab[pk], k.State)
	}
	ord := make([]LedgerRecord, 0, len(recs))
	for _, r := range recs {
		if branch == "" || r.Branch == branch {
			ord = append(ord, r)
		}
	}
	sort.SliceStable(ord, func(i, j int) bool { return ord[i].Started < ord[j].Started })
	cells := map[pgKey]*PGCell{}
	prev := map[StabilityKey]LedgerRecord{}
	for _, r := range ord {
		pk := pgKey{r.Scenario, r.Class}
		c := cells[pk]
		if c == nil {
			c = &PGCell{Scenario: r.Scenario, Class: r.Class, Tier: tiers.Tiers[r.Scenario].Tier}
			cells[pk] = c
		}
		run := PGRun{Run: r.Run, Seed: r.Seed, Verdict: r.Verdict, Reason: r.Reason, Setup: r.Setup, D1: r.D1, BotCommit: r.BotCommit,
			BotDirty: r.BotDirty, Host: r.Host, Event: r.Event, Branch: r.Branch, Started: r.Started}
		sk := StabilityKey{r.Scenario, r.Class, r.Seed}
		if p, ok := prev[sk]; ok && counted(p) && counted(r) && p.BotCommit == r.BotCommit && p.D2DSHA256 == r.D2DSHA256 &&
			(p.Verdict != r.Verdict || p.D1 != r.D1) {
			run.Flip = true
			c.Flips++
		}
		if counted(r) {
			prev[sk] = r
		}
		c.Runs++
		c.Latest = run
		c.History = append(c.History, run)
	}
	page := PGPage{Version: PGPageVersion, Branch: branch, Cells: make([]PGCell, 0, len(cells))}
	for pk, c := range cells {
		c.Stability = stab[pk]
		if c.Stability == "" {
			c.Stability = StabilityCandidate
		}
		h := c.History
		if len(h) > PGHistory {
			h = h[len(h)-PGHistory:]
		}
		c.History = make([]PGRun, len(h))
		for i := range h {
			c.History[i] = h[len(h)-1-i]
		}
		page.Cells = append(page.Cells, *c)
	}
	sort.Slice(page.Cells, func(i, j int) bool {
		a, b := page.Cells[i], page.Cells[j]
		if a.Scenario != b.Scenario {
			return a.Scenario < b.Scenario
		}
		return a.Class < b.Class
	})
	return page, nil
}

// rollUp folds one seed's state into a cell's: quarantined wins, stable only when every seed is stable.
func rollUp(cur, next string) string {
	switch {
	case cur == StabilityQuarantined || next == StabilityQuarantined:
		return StabilityQuarantined
	case cur == "":
		return next
	case cur == StabilityStable && next == StabilityStable:
		return StabilityStable
	}
	return StabilityCandidate
}

// ReleaseView is the release view of the page: only EARNED and SETUP runs on one bot commit (a prefix match, so a short
// hash works), the latest per scenario, class and seed, sorted. CHEATED and IMPOSSIBLE runs never speak for a release.
func ReleaseView(recs []LedgerRecord, commit string) []LedgerRecord {
	if commit == "" {
		return nil
	}
	last := map[StabilityKey]LedgerRecord{}
	for _, r := range recs {
		if r.BotDirty || r.BotCommit == "" || !strings.HasPrefix(r.BotCommit, commit) && !strings.HasPrefix(commit, r.BotCommit) {
			continue
		}
		if r.Setup != SetupEarned && r.Setup != SetupSetup {
			continue
		}
		k := StabilityKey{r.Scenario, r.Class, r.Seed}
		if p, ok := last[k]; !ok || r.Started >= p.Started {
			last[k] = r
		}
	}
	out := make([]LedgerRecord, 0, len(last))
	for _, r := range last {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		return StabilityKey{out[i].Scenario, out[i].Class, out[i].Seed}.String() < StabilityKey{out[j].Scenario, out[j].Class, out[j].Seed}.String()
	})
	return out
}

// PGStatusDoc renders docs/process/proving-ground-status.md from the page data: one row per scenario and class with
// tier, stability, latest verdict and run id. Deterministic, so regenerating from an unchanged ledger changes nothing.
func PGStatusDoc(p PGPage) string {
	var b strings.Builder
	b.WriteString("# Proving Ground status (generated)\n\n")
	b.WriteString("> status: GENERATED by `tools/provingground/status_doc.sh` from the results ledger (board host) and the stability rules; never edit by hand. Probe docs keep findings and cite run ids; the verdicts live here and on the board page /pg.\n\n")
	if p.Branch == "" {
		b.WriteString("Branch filter: all branches.\n\n")
	} else {
		fmt.Fprintf(&b, "Branch filter: `%s`.\n\n", p.Branch)
	}
	b.WriteString("Only a `stable` scenario with a tier other than `none` may gate. CHEATED and IMPOSSIBLE rows are mechanics evidence only. `flips` counts same-binary verdict or assertion-digest changes of one seed (flake events).\n\n")
	b.WriteString("| scenario | class | tier | stability | verdict | setup | runs | flips | D1 | bot | host | latest run |\n|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, c := range p.Cells {
		r := c.Latest
		bot := r.BotCommit
		if r.BotDirty {
			bot += "+dirty"
		}
		tier := c.Tier
		if tier == "" {
			tier = "?"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %d | %d | %s | %s | %s | %s |\n", c.Scenario, c.Class, tier, c.Stability,
			r.Verdict, r.Setup, c.Runs, c.Flips, r.D1, bot, r.Host, r.Run)
	}
	if len(p.Cells) == 0 {
		b.WriteString("| (no runs recorded) | | | | | | | | | | | |\n")
	}
	return b.String()
}
