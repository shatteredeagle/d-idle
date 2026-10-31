package setup

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/paths"
)

// Status of one check. Skip means a prerequisite check failed, so this one was not run.
type Status string

const (
	Pass Status = "pass"
	Warn Status = "warn"
	Fail Status = "fail"
	Skip Status = "skip"
)

// Result is one check outcome. ID is stable and machine-readable ("kernel.yama"); Fix is a human hint
// with the exact command where one exists. Both are part of the JSON contract (docs/setup.md).
type Result struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Status     Status `json:"status"`
	Detail     string `json:"detail"`
	Fix        string `json:"fix,omitempty"`
	Slow       bool   `json:"slow,omitempty"`
	DurationMS int64  `json:"durationMs"`
}

// Options says what to check against.
type Options struct {
	DataDir    string // required
	GameDir    string // default: base prefix install dir under DataDir
	HelperExe  string // default DataDir/helper/d2input.exe
	Slots      int    // planned instances, default 1
	Instance   string // game instance the character check attaches to ("" = $D2BOT_INSTANCE)
	UIAddr     string // web UI listen address, default 127.0.0.1:8420
	RunnerName string // pinned Proton build, default GE-Proton11-11 (the verified one)
	// GameExpected: the game should already be installed. False (pre-install) turns the missing-game
	// failures into warnings.
	GameExpected bool
	// Quick skips the Slow checks (MPQ read, extraction, probe).
	Quick bool
	// KnownGameExe maps sha256 of a Game.exe to its version label; default KnownGameExe.
	KnownGameExe map[string]string
}

// KnownGameExe: sha256 of Game.exe builds the bot's offsets were verified against.
// V: the worker 1.14.3.71 Game.exe (3,618,792 bytes), file version resource read by launch.FileVersion.
var KnownGameExe = map[string]string{
	"631066c1649c4ea9ffe48bf97e24c00bca1f7a6759c21150f1a79982589adaaf": "1.14.3.71 (1.14d)",
}

// BaseInstance is the prefix id of the shared base prefix the wizard builds (per-instance prefixes are
// hardlink clones of it, docs/multi.md).
const BaseInstance = "base"

// Normalize fills defaults.
func (o *Options) Normalize() {
	if o.Slots < 1 {
		o.Slots = 1
	}
	if o.UIAddr == "" {
		o.UIAddr = "127.0.0.1:8420"
	}
	if o.RunnerName == "" {
		o.RunnerName = "GE-Proton11-11"
	}
	if o.GameDir == "" && o.DataDir != "" {
		o.GameDir = GameDirIn(BasePrefix(o.DataDir))
	}
	if o.HelperExe == "" {
		o.HelperExe = os.Getenv("D2INPUT_EXE") // the container image ships the helper outside /data
	}
	if o.HelperExe == "" && o.DataDir != "" {
		o.HelperExe = filepath.Join(o.DataDir, "helper", "d2input.exe")
	}
	if o.KnownGameExe == nil {
		o.KnownGameExe = KnownGameExe
	}
}

// BasePrefix is the STEAM_COMPAT_DATA_PATH of the base prefix under a data dir.
func BasePrefix(dataDir string) string {
	return paths.Layout{Root: dataDir}.MustInstance(BaseInstance).Prefix()
}

// GameDirIn is the host path of the Diablo II install inside a Proton prefix dir (launch.Config.Resolve).
func GameDirIn(prefixDir string) string {
	return filepath.Join(prefixDir, "pfx", "drive_c", "Program Files (x86)", "Diablo II")
}

// Check is one data-driven rule. Needs lists check IDs that must not have failed.
type Check struct {
	ID    string
	Title string
	Slow  bool
	Needs []string
	Run   func(ctx context.Context, e *Env, o *Options) Result
}

// RunChecks executes the checks in order (Needs must precede their dependents) and returns the results.
func RunChecks(ctx context.Context, e *Env, o Options, checks []Check) []Result {
	o.Normalize()
	var out []Result
	failed := map[string]bool{}
	for _, c := range checks {
		r := Result{ID: c.ID, Title: c.Title, Slow: c.Slow}
		switch {
		case c.Slow && o.Quick:
			r.Status, r.Detail = Skip, "skipped (quick mode)"
		case blocked(c.Needs, failed):
			r.Status, r.Detail = Skip, "skipped: a prerequisite check failed"
		default:
			t0 := time.Now()
			r = c.Run(ctx, e, &o)
			r.ID, r.Title, r.Slow = c.ID, c.Title, c.Slow
			r.DurationMS = time.Since(t0).Milliseconds()
		}
		// A missing game (Warn before install) also blocks the checks that read it.
		if r.Status == Fail || r.Status == Warn && strings.HasPrefix(c.ID, "game.") {
			failed[c.ID] = true
		}
		out = append(out, r)
	}
	return out
}

func blocked(needs []string, failed map[string]bool) bool {
	for _, n := range needs {
		if failed[n] {
			return true
		}
	}
	return false
}

// Worst is the most severe status (Skip counts as Pass).
func Worst(rs []Result) Status {
	w := Pass
	for _, r := range rs {
		if r.Status == Fail {
			return Fail
		}
		if r.Status == Warn {
			w = Warn
		}
	}
	return w
}

// ExitCode: 0 all pass, 1 warnings only, 2 at least one failure. (d2doctor uses 3 for its own errors.)
func ExitCode(rs []Result) int {
	switch Worst(rs) {
	case Fail:
		return 2
	case Warn:
		return 1
	}
	return 0
}

// Report is the JSON document of `d2doctor --json` and GET /checks.
type Report struct {
	Status  Status   `json:"status"`
	Exit    int      `json:"exit"`
	Counts  Counts   `json:"counts"`
	Results []Result `json:"results"`
}

// Counts per status.
type Counts struct {
	Pass int `json:"pass"`
	Warn int `json:"warn"`
	Fail int `json:"fail"`
	Skip int `json:"skip"`
}

// NewReport summarises results.
func NewReport(rs []Result) Report {
	r := Report{Status: Worst(rs), Exit: ExitCode(rs), Results: rs}
	for _, x := range rs {
		switch x.Status {
		case Pass:
			r.Counts.Pass++
		case Warn:
			r.Counts.Warn++
		case Fail:
			r.Counts.Fail++
		default:
			r.Counts.Skip++
		}
	}
	return r
}

// IDs returns the check ids, sorted (docs and tests).
func IDs(cs []Check) []string {
	var ids []string
	for _, c := range cs {
		ids = append(ids, c.ID)
	}
	sort.Strings(ids)
	return ids
}
