package scenario

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// LedgerVersion is the ledger line format (design 2026-11-21 section 5.1). The ledger is append-only JSON lines: one per
// run, written by the runner itself at the end of exec-one, never from a pulled file and never by an agent by hand.
const LedgerVersion = 1

// Setup tags of a ledger line. EARNED means the hero reached the state by play with zero setup writes (no scenario
// produces it before the earned-save card); SETUP is a declared setup a player could have reached; CHEATED a state play
// could not reach in this run (warped, given, hand-set); IMPOSSIBLE a state the game cannot hold. CHEATED and IMPOSSIBLE
// results are mechanics evidence only (docs/probes/test-characters.md rules).
const (
	SetupEarned     = "EARNED"
	SetupSetup      = "SETUP"
	SetupCheated    = "CHEATED"
	SetupImpossible = "IMPOSSIBLE"
	// SetupUnbuilt is the tag of a BLOCKED run whose setup could not even be built (no tags to reduce), and
	// ProvMissing stands for a binary that was not there to hash. Both are valid on a BLOCKED line only: the
	// infrastructure failures are exactly the runs the ledger must not lose (review 2026-11-28, fix 1).
	SetupUnbuilt = "UNBUILT"
	ProvMissing  = "missing"
)

// LedgerRecord is one run in the results ledger.
type LedgerRecord struct {
	V         int    `json:"v"`
	Run       string `json:"run"` // <job>/<scenario>/<class>/<seed>
	Scenario  string `json:"scenario"`
	Class     string `json:"class"`
	Seed      int64  `json:"seed"`
	Tier      string `json:"tier,omitempty"`
	Stability string `json:"stability"` // candidate until the stability card (nothing is "stable" yet)
	Verdict   string `json:"verdict"`
	Reason    string `json:"reason,omitempty"`
	D1        string `json:"d1,omitempty"` // assertion digest: gates
	D2        string `json:"d2,omitempty"` // whole-state digest: drift only
	Journal   string `json:"journal,omitempty"`
	// FromSave, SaveRun and SavePort say which earned save the hero started from (milestone, the ledger run that exported
	// it, and its portability on this d2d build: same-build, verified); empty for a fresh or verb-built hero.
	FromSave  string   `json:"from_save,omitempty"`
	SaveRun   string   `json:"save_run,omitempty"`
	SavePort  string   `json:"save_portability,omitempty"`
	Setup     string   `json:"setup"`
	SetupTags []string `json:"setup_tags,omitempty"`
	BotCommit string   `json:"bot_commit"`
	BotDirty  bool     `json:"bot_dirty,omitempty"`
	// BotSHA256 is the hash of the bot binary: the stability rule keys on it (two commits that build the same binary are
	// one binary); empty on lines written before fix6, which fall back to the commit.
	BotSHA256  string  `json:"bot_sha256,omitempty"`
	D2DSHA256  string  `json:"d2d_sha256"`
	Host       string  `json:"host"` // a role name, never a machine name
	Lane       string  `json:"lane,omitempty"`
	Load1      float64 `json:"load1,omitempty"`
	Pace       float64 `json:"pace,omitempty"`
	Parallel   int     `json:"parallel,omitempty"` // peak busy slots during the run (Provenance.Parallel)
	VirtualMin float64 `json:"virtual_min"`
	WallS      int     `json:"wall_s"`
	Cut        bool    `json:"cut,omitempty"`
	WritesN    *int    `json:"writes_after_start,omitempty"`
	Event      string  `json:"event"` // nightly, merge, manual
	Branch     string  `json:"branch,omitempty"`
	Card       string  `json:"card,omitempty"`
	Job        string  `json:"job,omitempty"`
	Started    string  `json:"started"`
}

// LedgerCtx is what the result itself does not know: which job and event the run belongs to.
type LedgerCtx struct {
	Job, Event, Branch, Card, Lane, Tier string
	Started                              time.Time
}

// SetupTag reduces the setup tags of a result to the one ledger word. No tag at all is "": the line is refused rather
// than guessed (a run whose setup nobody declared is not evidence).
func SetupTag(tags []string) string {
	has := map[string]bool{}
	for _, t := range tags {
		has[strings.ToUpper(t)] = true
	}
	switch {
	case has[SetupImpossible]:
		return SetupImpossible
	case has[SetupCheated]:
		return SetupCheated
	case has[SetupEarned]:
		return SetupEarned
	case has["SETUP-ONLY"], has[SetupSetup]:
		return SetupSetup
	}
	return ""
}

// NewLedgerRecord builds the ledger line of a finished result. It fails (the run is not recorded, the caller says so)
// when the result lacks the provenance that makes it comparable.
func NewLedgerRecord(r Result, c LedgerCtx) (LedgerRecord, error) {
	rec := LedgerRecord{V: LedgerVersion, Scenario: r.ID, Class: r.Class, Seed: r.Seed, Tier: c.Tier, Stability: "candidate",
		Verdict: r.Status, Reason: r.Reason, D1: r.AssertDigest, D2: r.Digest, Journal: r.Journal,
		Setup: SetupTag(r.Tags), SetupTags: r.Tags, VirtualMin: r.VirtualMin, WallS: r.WallSec,
		Lane: c.Lane, Event: c.Event, Branch: c.Branch, Card: c.Card, Job: c.Job}
	if c.Event == "" {
		rec.Event = "manual"
	}
	if !c.Started.IsZero() {
		rec.Started = c.Started.UTC().Format(time.RFC3339)
	}
	if p := r.Prov; p != nil {
		rec.BotCommit, rec.BotDirty, rec.D2DSHA256, rec.Host, rec.Load1, rec.Pace = p.BotCommit, p.BotDirty, p.D2DSHA256, p.Host, p.Load1, p.Pace
		rec.Parallel, rec.BotSHA256 = p.Parallel, p.BotSHA256
	}
	if rec.Verdict == Blocked { // a blocked run is a fact whatever was missing: record it with what is known
		if rec.Setup == "" {
			rec.Setup = SetupUnbuilt
		}
		if rec.BotCommit == "" {
			rec.BotCommit = ProvMissing
		}
		if rec.D2DSHA256 == "" {
			rec.D2DSHA256 = ProvMissing
		}
	}
	if r.State != nil && r.State.Source != "none" {
		n := r.State.WritesAfterStart
		rec.WritesN = &n
	}
	rec.FromSave, rec.SaveRun, rec.SavePort = r.FromSave, r.SaveRun, r.SavePort
	rec.Cut = strings.Contains(r.Reason, "cut before its virtual budget")
	rec.Run = fmt.Sprintf("%s/%s", c.Job, r.Key())
	return rec, rec.Validate()
}

// Validate refuses a line without what the ledger exists to keep. A BLOCKED run is recorded too (it is a fact): it may
// say UNBUILT and "missing" where a judged run needs the setup tag and the binaries, but it still names its host role.
func (l LedgerRecord) Validate() error {
	var miss []string
	for _, f := range []struct{ name, v string }{{"scenario", l.Scenario}, {"verdict", l.Verdict}, {"setup tag", l.Setup},
		{"bot_commit", l.BotCommit}, {"d2d_sha256", l.D2DSHA256}, {"host", l.Host}, {"started", l.Started}} {
		if f.v == "" {
			miss = append(miss, f.name)
		}
	}
	if len(miss) > 0 {
		return fmt.Errorf("ledger line refused, missing %s", strings.Join(miss, ", "))
	}
	if l.Verdict != Blocked && (l.Setup == SetupUnbuilt || l.BotCommit == ProvMissing || l.D2DSHA256 == ProvMissing) {
		return fmt.Errorf("ledger line refused: %s with an unbuilt setup or a missing binary (only a BLOCKED run may lack them)", l.Verdict)
	}
	if l.Setup == SetupEarned && l.WritesN != nil && *l.WritesN > 0 {
		return fmt.Errorf("ledger line refused: EARNED with %d writes after the start (a written hero is CHEATED)", *l.WritesN)
	}
	if l.FromSave != "" {
		switch {
		case l.Setup != SetupEarned:
			return fmt.Errorf("ledger line refused: from_save %s with setup %s (an earned-save run is EARNED)", l.FromSave, l.Setup)
		case l.SaveRun == "":
			return fmt.Errorf("ledger line refused: from_save %s without save_run", l.FromSave)
		case l.SavePort != PortSame && l.SavePort != PortVerified:
			return fmt.Errorf("ledger line refused: from_save %s with portability %q", l.FromSave, l.SavePort)
		}
	}
	if l.Host == "unknown" {
		return fmt.Errorf("ledger line refused: host role unknown (set D2D_HOST_ROLE)")
	}
	return nil
}

// LedgerLine is the JSON line (no trailing newline).
func (l LedgerRecord) LedgerLine() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(l)
}

// ReadLedger reads JSON lines; a bad line is an error naming its number (the ledger is never silently shortened).
func ReadLedger(r io.Reader) ([]LedgerRecord, error) {
	var out []LedgerRecord
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var l LedgerRecord
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			return nil, fmt.Errorf("ledger line %d: %v", n, err)
		}
		out = append(out, l)
	}
	return out, sc.Err()
}

// StatusDoc renders the generated status page: per scenario and class the latest verdict of the runs on the chosen
// branch (all branches when branch is ""), with run ids, the setup tag, the digests, the host role and the count of runs.
// Output is deterministic (sorted, no clock) so regenerating from an unchanged ledger changes nothing.
func StatusDoc(recs []LedgerRecord, branch string) string {
	type key struct{ s, c string }
	last := map[key]LedgerRecord{}
	count := map[key]int{}
	flips := map[key]int{}
	for _, r := range recs {
		if branch != "" && r.Branch != branch {
			continue
		}
		k := key{r.Scenario, r.Class}
		if prev, ok := last[k]; ok && (prev.Verdict != r.Verdict || prev.D1 != r.D1) && prev.BotCommit == r.BotCommit && prev.D2DSHA256 == r.D2DSHA256 {
			flips[k]++ // same binaries, different outcome: a flake event (design 2.7)
		}
		if prev, ok := last[k]; !ok || r.Started >= prev.Started {
			last[k] = r
		}
		count[k]++
	}
	keys := make([]key, 0, len(last))
	for k := range last {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].s != keys[j].s {
			return keys[i].s < keys[j].s
		}
		return keys[i].c < keys[j].c
	})
	var b strings.Builder
	b.WriteString("# Proving Ground status (generated)\n\n> status: GENERATED from the results ledger by `d2scenario ledger status-doc`; never edit by hand. The ledger lives on the board host; probe docs cite run ids and no longer carry verdict tables.\n\n")
	if branch == "" {
		b.WriteString("Branch filter: all branches.\n\n")
	} else {
		fmt.Fprintf(&b, "Branch filter: `%s`.\n\n", branch)
	}
	b.WriteString("CHEATED and IMPOSSIBLE rows are mechanics evidence only. `flips` counts same-binary verdict or assertion-digest changes (flake events).\n\n")
	b.WriteString("| scenario | class | verdict | setup | runs | flips | D1 | bot | host | latest run |\n|---|---|---|---|---|---|---|---|---|---|\n")
	for _, k := range keys {
		r := last[k]
		bot := r.BotCommit
		if r.BotDirty {
			bot += "+dirty"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %d | %d | %s | %s | %s | %s |\n", k.s, k.c, r.Verdict, r.Setup, count[k], flips[k], r.D1, bot, r.Host, r.Run)
	}
	if len(keys) == 0 {
		b.WriteString("| (no runs recorded) | | | | | | | | | |\n")
	}
	return b.String()
}
