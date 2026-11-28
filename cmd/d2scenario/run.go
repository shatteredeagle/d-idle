package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/faultproxy"
	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/scenario"
)

// cmdRun plans, then runs every run in its own lab slot (tools/d2dslot.sh), at most -parallel at once, and aggregates the
// results. It must run ON THE BUILDER (docs/process/proving-ground.md section "Running"); -dry-run prints the commands.
func cmdRun(args []string, w io.Writer) error {
	fs := newFlags("run")
	dir := fs.String("dir", defaultDir, "scenario directory")
	tier := fs.String("tier", "fast", "fast, full or nightly")
	changed := fs.String("changed", "", "changed files (merge gate selection)")
	ids := fs.String("id", "", "ids or id prefixes ending in '.'")
	class := fs.String("class", "", "one class")
	seed := fs.Int64("seed", 0, "one seed")
	out := fs.String("out", "provingground-out", "output directory (results.json, per-run files)")
	parallel := fs.Int("parallel", 2, "lab slots at once (at most 4)")
	dry := fs.Bool("dry-run", false, "print what would run, start nothing")
	slot := fs.String("slot", "tools/d2dslot.sh", "slot wrapper")
	bot := fs.String("bot", os.Getenv("D2BOT_BIN"), "d2bot binary (env D2BOT_BIN)")
	repeat := fs.Int("repeat", 1, "runs per scenario; GREEN needs two thirds of them (3 runs: 2 green), see repeat.go")
	journal := fs.Bool("journal", false, "write the d2dctl command journal of each bot run next to its trace (determinism finder, cmd/d2ddet)")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	seedSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			seedSet = true
		}
	})
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	if *ids != "" && !flagSet(fs, "tier") {
		*tier = scenario.TierNightly // naming scenarios selects them whatever their tier
	}
	sel := selectorFrom(*tier, *changed, *ids, *class, *seed, seedSet)
	picked := scenario.Select(scs, sel)
	runs, err := scenario.Expand(picked, sel)
	if err != nil {
		return err
	}
	by := map[string]scenario.Scenario{}
	var results []scenario.Result
	for _, s := range picked {
		by[s.ID] = s
		if s.Pending() {
			results = append(results, scenario.PendingResult(s))
		}
	}
	if len(runs) == 0 && len(results) == 0 {
		return fmt.Errorf("no scenario matches the selection (tier %s, id %q): nothing to run", *tier, *ids)
	}
	if *parallel > scenario.MaxLabSlots {
		*parallel = scenario.MaxLabSlots
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	self, _ := os.Executable()
	var todo []scenario.Run
	for _, r := range runs {
		s := by[r.ID]
		if why := blockedBefore(s, r); why != "" {
			results = append(results, scenario.Result{Version: scenario.ResultVersion, Run: r, Status: scenario.Blocked, Reason: why, BudgetMin: s.Budget.VirtualMin})
			continue
		}
		todo = append(todo, r)
	}
	cmdFor := func(r scenario.Run, outDir string) []string {
		a := execOneCmd(*slot, *dir, *bot, self, r, by[r.ID].Budget.WallMin, outDir, false)
		if *journal {
			a = append(a, "-journal")
		}
		return a
	}
	if *dry {
		fmt.Fprintf(w, "DRY RUN: %d runs would start, %d slots at once, nothing started\n", len(todo), *parallel)
		for _, r := range todo {
			fmt.Fprintln(w, "  "+strings.Join(cmdFor(r, *out), " "))
		}
		for _, r := range results {
			fmt.Fprintf(w, "  not run: %-30s %s %s\n", r.ID+"/"+r.Class, r.Status, r.Reason)
		}
		return nil
	}
	if *bot == "" || os.Getenv("D2D_BIN") == "" {
		return fmt.Errorf("run needs -bot (or D2BOT_BIN) and D2D_BIN (the lab d2d) in the environment")
	}
	var mu sync.Mutex
	slotLog := filepath.Join(*out, slotLogName)
	logSlot := func(ev, id string) {
		mu.Lock()
		defer mu.Unlock()
		if err := appendRaw(slotLog, scenario.SlotLogLine(ev, id, time.Now().UnixNano())); err != nil {
			fmt.Fprintf(os.Stderr, "d2scenario: slot log: %v\n", err)
		}
	}
	sem := make(chan struct{}, *parallel)
	var wg sync.WaitGroup
	for _, r := range todo {
		wg.Add(1)
		sem <- struct{}{}
		go func(r scenario.Run) {
			defer wg.Done()
			defer func() { <-sem }()
			n := max(1, *repeat)
			var reps []scenario.Result
			for k := 0; k < n; k++ {
				o := *out
				if n > 1 {
					o = filepath.Join(*out, fmt.Sprintf("rep%d", k+1))
					_ = os.MkdirAll(o, 0o755)
				}
				a := cmdFor(r, o)
				c := exec.Command(a[0], a[1:]...)
				c.Stdout, c.Stderr = os.Stderr, os.Stderr
				sid := filepath.Join(o, resultName(r))
				c.Env = append(os.Environ(), slotLogEnv+"="+slotLog, slotIDEnv+"="+sid)
				err := c.Run() // exec-one logs its own start once it holds a slot, so a wait for a slot is not busy time
				logSlot("end", sid)
				res, rerr := readResult(filepath.Join(o, resultName(r)))
				if rerr != nil { // the run wrapper died before writing a result: that is a blocked run, never a pass
					res = scenario.Result{Version: scenario.ResultVersion, Run: r, Status: scenario.Blocked, Reason: fmt.Sprintf("no result written (slot or runner failed: %v)", err), BudgetMin: by[r.ID].Budget.VirtualMin}
				}
				reps = append(reps, res)
			}
			res := reps[0]
			if n > 1 {
				res = aggregateRepeats(reps, repeatNeed(n))
			}
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
		}(r)
	}
	wg.Wait()
	f, err := os.Create(filepath.Join(*out, "results.json"))
	if err != nil {
		return err
	}
	if err := scenario.WriteJSON(f, results); err != nil {
		return err
	}
	f.Close()
	scenario.Report(w, results)
	if !scenario.Gate(results) {
		return exitErr{3, "NOT GREEN"}
	}
	return nil
}

func resultName(r scenario.Run) string {
	return strings.NewReplacer("/", "_", " ", "_").Replace(r.Key()) + ".json"
}

func readResult(path string) (scenario.Result, error) {
	var r scenario.Result
	return r, readJSON(path, &r)
}

// blockedBefore names a reason when the run cannot start at all.
func blockedBefore(s scenario.Scenario, r scenario.Run) string {
	for _, n := range s.Needs {
		if n == "sandbox-d2d-binary" && os.Getenv("D2D_BIN_SANDBOX") == "" {
			return "needs the shipped sandbox d2d build: set D2D_BIN_SANDBOX"
		}
	}
	if s.Build == "sandbox" && s.Pre.Start == scenario.StartField {
		return "a field start warps the hero with a lab verb and hands the lab d2d to the bot; the shipped build refuses the gated verbs, so its field state cannot be set up (use a town start, or the lab build)"
	}
	su, err := scenario.BuildSetup(s, r.Class, r.Seed)
	if err != nil {
		return err.Error()
	}
	if len(su.Unsupported) > 0 {
		return "setup not buildable: " + strings.Join(su.Unsupported, "; ")
	}
	return ""
}

// cmdExecOne is one run inside a slot (D2D_SOCK, D2D_BOT_ADDR, D2D_SLOT_DIR from tools/d2dslot.sh).
func cmdExecOne(args []string) error {
	fs := newFlags("exec-one")
	dir := fs.String("dir", defaultDir, "scenario directory")
	id := fs.String("id", "", "scenario id")
	class := fs.String("class", "", "class")
	seed := fs.Int64("seed", 0, "seed")
	out := fs.String("out", "", "output directory")
	bot := fs.String("bot", "", "d2bot binary")
	journal := fs.Bool("journal", false, "write the d2dctl journal of the bot run")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	s, err := findScenario(scs, *id)
	if err != nil {
		return err
	}
	r := scenario.Run{ID: s.ID, Class: *class, Seed: *seed}
	start := time.Now()
	logSlotStart(start)
	if *journal {
		os.Setenv(journalEnv, filepath.Join(*out, strings.TrimSuffix(resultName(r), ".json")+".jnl"))
	}
	res := execOne(s, r, *out, *bot)
	if *journal {
		res.Journal = os.Getenv(journalEnv)
	}
	res.Version, res.Run = scenario.ResultVersion, r
	if len(res.Tags) == 0 { // a run blocked before the setup was built still carries the scenario's provenance tags
		if su, err := scenario.BuildSetup(s, *class, *seed); err == nil {
			res.Tags = su.Tags
		}
	}
	if res.Status == "" { // not blocked by the infrastructure: judge the observed state
		if res.State == nil {
			res = withVerdict(res, blocked("no state was observed"))
		} else {
			res.Asserts = scenario.Evaluate(s.Assert, *res.State)
			res.Asserts = append(res.Asserts, loadInvariantPolicy().invariantAsserts(s.ID, res.State.Invariants)...)
			res.Status, res.Reason = scenario.Verdict(res.Asserts)
		}
	}
	res.BudgetMin = s.Budget.VirtualMin
	res.WallSec = int(time.Since(start).Seconds())
	res.Prov = collectProv(*bot)
	res.Prov.Pace = scenario.PaceOf(res.VirtualMin, res.WallSec)
	res.Prov.Parallel = slotParallel()
	if res.State != nil {
		res.Digest = scenario.DigestState(*res.State)
	}
	res.AssertDigest = scenario.AssertDigest(res)
	b, _ := json.MarshalIndent(res, "", " ")
	if err := os.WriteFile(filepath.Join(*out, resultName(r)), b, 0o644); err != nil {
		return err
	}
	shipLedger(res, s.Tier, *out, start)
	return nil
}

func blocked(why string, a ...any) scenario.Result {
	return scenario.Result{Status: scenario.Blocked, Reason: fmt.Sprintf(why, a...)}
}

// execOne does prep, bot and observation for one run. Every infrastructure failure is BLOCKED, never GREEN or RED: a
// red verdict is only ever a failed assertion on observed state.
func execOne(s scenario.Scenario, r scenario.Run, out, bot string) scenario.Result {
	su, err := scenario.BuildSetup(s, r.Class, r.Seed)
	if err != nil {
		return blocked("%v", err)
	}
	if len(su.Unsupported) > 0 {
		return blocked("setup not buildable: %s", strings.Join(su.Unsupported, "; "))
	}
	cidx, _ := scenario.ClassIndex(r.Class)
	hero := scenario.HeroName(s.ID, r.Class, r.Seed)
	inst := instanceID(r) // the bot wants [a-z0-9-]{1,16}
	work := filepath.Join(os.Getenv("D2D_SLOT_DIR"), "pg-"+inst)
	if os.Getenv("D2D_SLOT_DIR") == "" {
		work = filepath.Join(os.TempDir(), "pg-"+inst)
	}
	_ = os.RemoveAll(work)
	data := filepath.Join(work, "data")
	instDir := filepath.Join(data, "instances", inst)
	home, state := filepath.Join(instDir, "d2d"), filepath.Join(instDir, "state")
	for _, d := range []string{home, state} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return blocked("%v", err)
		}
	}
	sock := os.Getenv("D2D_SOCK")
	if sock == "" {
		sock = filepath.Join(work, "d2d.sock")
	}
	ledger := filepath.Join(instDir, "setup-ledger.jsonl")
	trace := filepath.Join(out, strings.TrimSuffix(resultName(r), ".json")+".bot.log")

	// 1. SETUP on a lab d2d (lab verbs on), every verb ledgered with its reason before it is sent.
	bin, gamedata := os.Getenv("D2D_BIN"), os.Getenv("D2D_DATA")
	prep := &d2dinput.Process{Bin: bin, Data: gamedata, Sock: sock, ConfigDir: filepath.Join(home, "config"), DataDir: filepath.Join(home, "data"),
		Nice: 10, ExtraArgs: withItemVerbs(withPay(withVault([]string{"--lockstep", "--lab-verbs", "--toggle", "autoloot=off,autoparty=off"}, su.VaultVerbs), s.Needs), s.Needs)}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.Budget.WallMin+15)*time.Minute)
	defer cancel()
	if err := prep.Start(ctx, d2dinput.Spec{Name: hero, Class: cidx, Seed: r.Seed}); err != nil {
		return blocked("prep d2d start: %v", err)
	}
	if err := prep.WaitInGame(ctx, 3*time.Minute); err != nil {
		_ = prep.Stop()
		return blocked("prep d2d not in game: %v", err)
	}
	c := prep.Controller().C
	if caps, err := c.Caps(ctx); err != nil {
		_ = prep.Stop()
		return blocked("caps: %v", err)
	} else if miss := missingCaps(s.Needs, caps); len(miss) > 0 {
		_ = prep.Stop()
		return blocked("the lab d2d lacks %s (the scenario needs them)", strings.Join(miss, ", "))
	}
	if err := applySetup(ctx, c, su, ledger); err != nil {
		_ = prep.Stop()
		return blocked("setup: %v", err)
	}
	tabs, err := heroTables(gamedata, data)
	if err != nil {
		_ = prep.Stop()
		return blocked("%v", err)
	}
	if err := spendPoints(ctx, c, s, r.Class, su, tabs, ledger); err != nil {
		_ = prep.Stop()
		return blocked("hero points: %v", err)
	}
	seeded, err := applySeeds(ctx, c, su, ledger, data, gameChain(data, gamedata), hero)
	if err != nil {
		_ = prep.Stop()
		return blocked("seeding: %v", err)
	}
	if err := seedEther(state, su.Ether); err != nil {
		_ = prep.Stop()
		return blocked("ether seed: %v", err)
	}
	if _, err := c.Step(ctx, 50); err != nil {
		_ = prep.Stop()
		return blocked("settle: %v", err)
	}
	if failed := prepSelfCheck(ctx, c, s, r.Class, ledger, tabs); failed != "" { // design 6.2: refuse a hero that is not the declared one
		_ = prep.Stop()
		return blocked("selfcheck: %s (rows in %s)", failed, ledger)
	}
	if err := writeBoardAndTuning(state, hero, su); err != nil {
		_ = prep.Stop()
		return blocked("board record: %v", err)
	}
	if err := writeBotConfig(data, su); err != nil {
		_ = prep.Stop()
		return blocked("bot config: %v", err)
	}
	res := scenario.Result{Tags: su.Tags, Trace: trace}

	if su.Field {
		return runField(ctx, s, r, res, prep, c, bot, data, sock, inst, trace, seeded, su.VaultVerbs)
	}
	if err := prep.SaveExit(ctx); err != nil {
		return blocked("prep save: %v", err)
	}
	before := newestSave(home)

	// 2. The real bot plays on a d2d WITHOUT lab verbs: gated debug verbs are refused, so no gameplay write after the start.
	botBin := bot
	d2dBot := bin
	if s.Build == "sandbox" {
		d2dBot = os.Getenv("D2D_BIN_SANDBOX")
	}
	refused, taps, tapsAt, berr := runBot(ctx, s, r, botBin, d2dBot, gamedata, data, home, sock, inst, hero, trace, false, seeded, su.VaultVerbs)
	res.Taps, res.TapsAt = taps, tapsAt
	if why := botFailure(trace); why != "" {
		return withVerdict(res, blocked("the bot exited with an error, it did not play: %s", why))
	}
	if why := cutReason(trace); why != "" && !needsPay(s.Needs) { // a real-time run (no lockstep clock) is bounded by its wall minutes, not a virtual length
		return withVerdict(res, blocked("%s", why))
	}
	if !after(newestSave(home), before) {
		res.State = &scenario.State{Version: scenario.StateVersion, Source: "none", WritesAfterStart: refused}
		return withVerdict(res, blocked("the bot run left no newer save (bot exit: %v): nothing to observe", berr))
	}
	res.VirtualMin = s.Budget.VirtualMin // the bot stops on the virtual clock (D2DSOAK_HOURS), so a normal exit is the full budget

	// 3. Observe: a read-only d2d loads the saved hero; the facts are read from the game, not from logs.
	obs := &d2dinput.Process{Bin: bin, Data: gamedata, Sock: sock, ConfigDir: filepath.Join(home, "config"), DataDir: filepath.Join(home, "data"),
		Nice: 10, ExtraArgs: []string{"--lockstep", "--toggle", "autoloot=off,autoparty=off"}}
	if err := obs.Start(ctx, d2dinput.Spec{Name: hero, Class: -1}); err != nil {
		return withVerdict(res, blocked("observer start: %v", err))
	}
	defer obs.Stop()
	if err := obs.WaitInGame(ctx, 3*time.Minute); err != nil {
		return withVerdict(res, blocked("observer not in game: %v", err))
	}
	st := observe(ctx, obs.Controller().C, refused, false)
	st.Source = "d2d-observer-reload"
	addTrail(&st, trace)
	addRunFacts(&st, s, r, trace)
	addBoardFacts(&st, s, trace)
	addVaultFacts(&st, s, data, hero, trace, seeded)
	res.State = &st
	return withVerdict(res, scenario.Result{})
}

// instanceID is a stable, valid bot instance id for one run: "pg-" and 12 hex digits of the run key.
func instanceID(r scenario.Run) string {
	h := sha256.Sum256([]byte(r.Key()))
	return "pg-" + hex.EncodeToString(h[:6])
}

// botFailure returns the error text when the bot's own log ends in an "exit" error line (infrastructure: the run did not
// play, so a verdict on the state would be about nothing).
func botFailure(trace string) string {
	b, err := os.ReadFile(trace)
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.Contains(l, `"level":"ERROR","msg":"exit"`) {
			var e struct {
				Err string `json:"err"`
			}
			_ = json.Unmarshal([]byte(l), &e)
			return e.Err
		}
	}
	return ""
}

func after(a, b time.Time) bool { return a.After(b) }

// withVerdict fills Status and Reason: from a blocked infrastructure result when given, else from the assertions.
func withVerdict(res, b scenario.Result) scenario.Result {
	if b.Status != "" {
		res.Status, res.Reason = b.Status, b.Reason
		return res
	}
	return res
}

// newestSave returns the newest .d2s mtime under dir (zero when none).
func newestSave(dir string) time.Time {
	var t time.Time
	_ = filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() && strings.HasSuffix(p, ".d2s") && fi.ModTime().After(t) {
			t = fi.ModTime()
		}
		return nil
	})
	return t
}

type ledgerLine struct {
	Time   string `json:"time"`
	Verb   string `json:"verb"`
	Reason string `json:"reason"`
	Reply  string `json:"reply,omitempty"`
}

func applySetup(ctx context.Context, c *d2dctl.Client, su scenario.Setup, ledger string) error {
	lf, err := os.OpenFile(ledger, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer lf.Close()
	enc := json.NewEncoder(lf)
	for _, v := range su.Verbs {
		l := ledgerLine{Time: time.Now().Format(time.RFC3339), Verb: v.Line, Reason: v.Reason}
		rep, err := c.Do(ctx, v.Line)
		if rep != nil {
			l.Reply = strings.Join(append([]string{rep.Rest}, rep.Lines...), " | ")
		}
		_ = enc.Encode(l)
		if err != nil {
			return fmt.Errorf("%s: %w", v.Line, err)
		}
	}
	if su.LifePct > 0 {
		return hurtTo(ctx, c, su.LifePct, enc)
	}
	return nil
}

// hurtTo brings the hero to pct percent of maximum life with `debug hit` (the game's own hurt path, V docs/forkcheats.md),
// a ledgered SETUP verb whose damage is computed from the live life.
func hurtTo(ctx context.Context, c *d2dctl.Client, pct int, enc *json.Encoder) error {
	sv, _, err := c.State(ctx)
	if err != nil || sv.Life.Max <= 0 {
		return fmt.Errorf("life setup: no life read (%v)", err)
	}
	dmg := sv.Life.Cur - sv.Life.Max*pct/100
	if dmg <= 0 {
		return nil
	}
	line := fmt.Sprintf("debug hit %d", dmg)
	l := ledgerLine{Time: time.Now().Format(time.RFC3339), Verb: line, Reason: fmt.Sprintf("SETUP: life to %d%% (%d of %d)", pct, sv.Life.Max*pct/100, sv.Life.Max)}
	rep, err := c.Do(ctx, line)
	if rep != nil {
		l.Reply = rep.Rest
	}
	_ = enc.Encode(l)
	return err
}

// writeBoardAndTuning writes the player's accepted chunk with the real plan store and the knob overrides.
func writeBoardAndTuning(state, hero string, su scenario.Setup) error {
	su.Tuning = envKnobs(su.Tuning, os.Getenv("D2SCEN_KNOBS"))
	if len(su.Tuning) > 0 {
		b, _ := json.Marshal(map[string]any{"version": 1, "data": map[string]any{"overrides": su.Tuning}})
		if err := os.WriteFile(filepath.Join(state, "tuning.json"), b, 0o644); err != nil {
			return err
		}
	}
	if su.Board == nil {
		return nil
	}
	b, err := questboard.Load()
	if err != nil {
		return err
	}
	dir := filepath.Join(state, "quests") // cmd/d2bot questStoreDir
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	st, err := questboard.Open(b, dir, hero)
	if err != nil {
		return err
	}
	if err := st.Accept(su.Board.Quest, su.Board.Chunk, 0, "train"); err != nil {
		return err
	}
	// The store refuses a second concurrent chunk; a plan file that holds two is the state under test.
	for _, q := range su.BoardAlso {
		if err := st.AcceptIf(q.Quest, q.Chunk, 0, "train", func(questboard.Record) bool { return true }); err != nil {
			return err
		}
	}
	return nil
}

// journalEnv is the variable cmd/d2bot (internal/d2denv) reads to write the d2dctl command journal; exec-one sets it for
// the bot it starts (runBot inherits the environment) when run -journal asks for it.
const journalEnv = "D2DCTL_JOURNAL"

// tapEveryVirtual is the virtual-second grid of the board tapper (was: 20 wall s, then every 5 wall s).
const tapEveryVirtual = 30

// runBot starts the real bot like tools/survival/abrun.sh does and returns how many gated writes the d2d refused (the
// bot tried a write after the start; the CRUTCH RULE line) and the bot's exit error.
func runBot(ctx context.Context, s scenario.Scenario, r scenario.Run, bot, d2dBin, gamedata, data, home, sock, inst, hero, trace string, attach bool, seeded seedOut, vaultVerbs bool) (int, []string, []float64, error) {
	f, err := os.Create(trace)
	if err != nil {
		return -1, nil, nil, err
	}
	defer f.Close()
	hours := strconv.FormatFloat(s.Budget.VirtualMin/60, 'f', -1, 64)
	args := []string{"--backend", "d2d", "--char", hero, "--seed", strconv.FormatInt(r.Seed, 10),
		"--toggle", "autoloot=off,autoparty=off", "--d2d-linkwait", "60s", "--gamedir", gamedata, "--data", data,
		"--d2d-sock", sock, "--addr", os.Getenv("D2D_BOT_ADDR"), "--goals", s.Trigger.Goals,
		"--run-for", strconv.Itoa(s.Budget.WallMin) + "m"}
	if attach {
		args = append(args, "--attach", "--d2d-attach")
	} else {
		args = append(args, "--managed", "--d2d-class", "-1", "--d2d-args", strings.Join(botD2dArgs(s.Needs, vaultVerbs), " "))
	}
	var tap *tapper
	var scriptedTaps string
	var apiRun, sampler func(context.Context)
	if s.Trigger.Driver == scenario.DriverBoardTaps || len(s.Trigger.API) > 0 || scenario.NeedsBoardSamples(s) || scenario.NeedsVault(s) {
		tokFile := filepath.Join(data, "wtoken")
		tok, err := newWatchToken(tokFile)
		if err != nil {
			return -1, nil, nil, err
		}
		slot, _ := strconv.Atoi(os.Getenv("D2D_SLOT"))
		vp := 9300 + slot*10 // the L3 harness's port plan; the answer route is the next port
		view, ans := fmt.Sprintf("127.0.0.1:%d", vp), fmt.Sprintf("127.0.0.1:%d", vp+1)
		args = append(args, "--ladder", "--watch", "--watch-addr", view, "--watch-answer-addr", ans, "--watch-token-file", tokFile)
		tlog, _ := os.Create(strings.TrimSuffix(trace, ".bot.log") + ".taps.log")
		if tlog != nil {
			defer tlog.Close()
		} else {
			tlog = os.Stderr
		}
		if s.Trigger.Driver == scenario.DriverBoardTaps {
			if os.Getenv("D2SCEN_SCRIPTED_TAPS") == "1" {
				// card pgtaps: the bot taps by itself on its stepping goroutine (lockstep d2d only); the HTTP tapper stays the default
				scriptedTaps = strings.TrimSuffix(trace, ".bot.log") + ".tapscript.json"
				b, _ := json.Marshal(scenario.TapScript{Quest: s.TappedQuest(), Chunks: s.Trigger.TapChunks, EveryS: tapEveryVirtual})
				if err := os.WriteFile(scriptedTaps, b, 0o644); err != nil {
					return -1, nil, nil, err
				}
				args = append(args, "--scripted-taps", scriptedTaps)
			} else {
				tap = newTapper("http://"+view, "http://"+ans, tok, s.TappedQuest(), s.Trigger.TapChunks, tlog)
			}
		}
		sp, rp := boardFiles(trace)
		if len(s.Trigger.API) > 0 {
			apiRun = func(c context.Context) {
				replies := runAPICalls(c, "http://"+ans, "http://"+view, tok, s.Trigger.API, seeded.VaultIDs, tlog)
				if b, err := json.Marshal(replies); err == nil {
					_ = os.WriteFile(rp, b, 0o644)
				}
			}
		}
		if f, err := os.Create(sp); err == nil {
			defer f.Close()
			sampler = func(c context.Context) { sampleBoard(c, "http://"+view, tok, 15*time.Second, f) }
		}
		if scenario.NeedsVault(s) {
			inner := sampler
			sampler = func(c context.Context) {
				go sampleVault(c, "http://"+view, tok, 10*time.Second, vaultSampleFile(trace))
				if inner != nil {
					inner(c)
					return
				}
				<-c.Done()
			}
		}
	}
	tctx, stopTap := context.WithCancel(ctx)
	defer stopTap()
	done := make(chan struct{})
	sdone := make(chan struct{})
	go func() {
		defer close(sdone)
		if sampler != nil {
			sampler(tctx)
		}
	}()
	go func() {
		defer close(done)
		if apiRun != nil {
			apiRun(tctx)
		}
		if tap != nil {
			// The watch server comes up with the bot; the board is evaluated on a grid of the bot's virtual clock.
			tap.RunVirtual(tctx, tapEveryVirtual, 250*time.Millisecond, 90*time.Second)
		}
	}()
	runLeg := func(c context.Context, hrs, release string) error {
		cmd := exec.CommandContext(c, "nice", append([]string{"-n", "10", bot}, args...)...)
		cmd.Env = append(os.Environ(), "D2BOT_INSTANCE="+inst, "D2DSOAK_HOURS="+hrs, "D2D_BIN="+d2dBin, "D2D_DATA="+gamedata)
		if release != "" {
			cmd.Env = append(cmd.Env, "D2BOT_RELEASE_ID="+release)
		}
		cmd.Stdout, cmd.Stderr = f, f
		// a stop is the bot's own save and quit (SIGTERM), as a restart of the service would do; only a hang is killed
		cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
		cmd.WaitDelay = 90 * time.Second
		return cmd.Run()
	}
	if at := s.Trigger.RestartAtS; at > 0 && !attach {
		// trigger.restart_at_s: the bot process is stopped once and started again on the same hero and state directory
		// (the plan file lives there), with the rest of the virtual budget. The game restarts with it (managed mode).
		leg, stopLeg := context.WithTimeout(ctx, time.Duration(at)*time.Second)
		rel1, rel2 := "", ""
		if s.Trigger.RestartNewRelease {
			rel1, rel2 = "scenario-release-1", "scenario-release-2"
		}
		err = runLeg(leg, hours, rel1)
		stopLeg()
		if ctx.Err() == nil {
			b, _ := os.ReadFile(trace)
			rest := restHours(s.Budget.VirtualMin, lastVirtualMin(b))
			_, _ = fmt.Fprintf(f, "\nd2scenario: bot restarted after %d s (virtual %.1f min done)\n", at, lastVirtualMin(b))
			err = runLeg(ctx, rest, rel2)
		}
	} else {
		err = runLeg(ctx, hours, "")
	}
	stopTap()
	<-done
	<-sdone
	b, _ := os.ReadFile(trace)
	var taps []string
	var tapsAt []float64
	if tap != nil {
		taps, tapsAt = tap.Taps, tap.TapsAt
	}
	if scriptedTaps != "" {
		taps, tapsAt = readScriptedTaps(scriptedTaps + ".taps.json")
	}
	return strings.Count(string(b), "err lab verbs off"), taps, tapsAt, err
}

// writeBotConfig writes the bot's config file (<data>/config.json, the default of d2bot --config, S cmd/d2bot/main.go) with
// the leveler build the scenario selects (trigger.build). No build: no file, the bot keeps its defaults.
func writeBotConfig(data string, su scenario.Setup) error {
	if su.Build == "" {
		return nil
	}
	b, _ := json.Marshal(map[string]any{"levelerBuild": su.Build})
	return os.WriteFile(filepath.Join(data, "config.json"), b, 0o644)
}

// addRunFacts adds what the assertions on skill points and bot log lines need: the hero's class and the log lines that
// match a log_line assertion (S: the bot's own log, with a place in the run by a linear map of its wall clock).
func addRunFacts(st *scenario.State, s scenario.Scenario, r scenario.Run, trace string) {
	st.Class = r.Class
	b, err := os.ReadFile(trace)
	if err != nil {
		return
	}
	if h := invariantCounts(b); len(h) > 0 {
		st.Invariants = h
	}
	if lines, ok := scenario.CollectLog(s.Assert, b, s.Budget.VirtualMin); ok {
		st.Has = append(st.Has, "log")
		st.LogLines = lines
	}
}

// addTrail sets the area-change count from the bot's log (S: the bot's own `zone entry` lines).
func addTrail(st *scenario.State, trace string) {
	b, err := os.ReadFile(trace)
	if err != nil {
		return
	}
	if trail, ok := scenario.CountZoneEntries(b); ok {
		n := scenario.AreaChangeCount(trail)
		st.AreaChanges, st.AreaTrail = &n, trail
	}
}

// runField is the field start: the lab d2d (verbs on) was warped by the setup and is now handed to the bot with
// --attach; afterwards the same d2d is read. The write count is unknown (verbs stay on), so no_writes is UNOBSERVED.
func runField(ctx context.Context, s scenario.Scenario, r scenario.Run, res scenario.Result, prep *d2dinput.Process, c *d2dctl.Client, bot, data, sock, inst, trace string, seeded seedOut, vaultVerbs bool) scenario.Result {
	hero := scenario.HeroName(s.ID, r.Class, r.Seed)
	_ = c.Close() // the d2d takes one control client: the bot gets it, then we take it back
	// The lab d2d has its verbs on, so nothing refuses a bot write: the bot's command journal says what it sent (card bsm10).
	jnl := os.Getenv(journalEnv)
	if jnl == "" {
		jnl = strings.TrimSuffix(trace, ".bot.log") + ".jnl"
		os.Setenv(journalEnv, jnl) // exec-one runs one scenario per process; runBot inherits it
	}
	botSock := sock
	var fp *faultproxy.Proxy
	faultSet := faultsFor(s)
	if faultSet != "" { // lab soak only: the bot talks to the fault proxy, the observer below to the game directly
		var err error
		if fp, err = startFaults(faultSet, sock, filepath.Join(filepath.Dir(sock), "fault-"+inst+".sock")); err != nil {
			_ = prep.Stop()
			return withVerdict(res, blocked("fault proxy: %v", err))
		}
		botSock = fp.Listen
	}
	_, taps, tapsAt, berr := runBot(ctx, s, r, bot, os.Getenv("D2D_BIN"), os.Getenv("D2D_DATA"), data, "", botSock, inst, hero, trace, true, seeded, vaultVerbs)
	if fp != nil {
		res.Faults = faultReport(faultSet, fp, strings.TrimSuffix(trace, ".bot.log")+".faults.jsonl")
	}
	res.Taps, res.TapsAt = taps, tapsAt
	if why := botFailure(trace); why != "" {
		_ = prep.Stop()
		return withVerdict(res, blocked("the bot exited with an error, it did not play: %s", why))
	}
	if why := cutReason(trace); why != "" {
		_ = prep.Stop()
		return withVerdict(res, blocked("%s", why))
	}
	cc := d2dctl.New(sock, d2dctl.Options{})
	writes := -1
	if n, ok := journalWrites(jnl); ok {
		writes = n
	}
	st := observe(ctx, cc, writes, true)
	st.Source = "d2d-live-attach"
	addTrail(&st, trace)
	addRunFacts(&st, s, r, trace)
	addBoardFacts(&st, s, trace)
	addVaultFacts(&st, s, data, hero, trace, seeded)
	res.State = &st
	res.VirtualMin = s.Budget.VirtualMin
	_ = prep.Stop()
	if berr != nil && len(st.Items) == 0 && st.Area == nil {
		return withVerdict(res, blocked("bot run failed and nothing could be observed: %v", berr))
	}
	return res
}

// observe reads the facts of a loaded hero. Item places: loc 1 body, loc 2 belt, panel "stash" the stash, the rest the
// pack (U: the "panel" attribute value for the stash was not seen live).
func observe(ctx context.Context, c *d2dctl.Client, writes int, live bool) scenario.State {
	st := scenario.State{Version: scenario.StateVersion, WritesAfterStart: writes}
	if sv, _, err := c.State(ctx); err == nil {
		l, g := sv.Level, sv.Gold
		st.Level, st.Gold = &l, &g
		if sv.Life.Max > 0 {
			p := 100 * sv.Life.Cur / sv.Life.Max
			st.LifePct = &p
		}
		if v, ok := sv.Int("rskill"); ok {
			st.RightSkill = &v
		}
		if v, ok := sv.Int("lskill"); ok {
			st.LeftSkill = &v
		}
		if m, ok := sv.M["merc"]; ok {
			st.Merc = &m
		}
	}
	if li, err := c.Level(ctx); err == nil {
		a := li.ID
		st.Area = &a
	}
	if _, stash, err := c.Gold(ctx); err == nil {
		st.Stash = &stash
	}
	if items, _, err := c.Items(ctx); err == nil {
		st.Has = append(st.Has, "items")
		for _, it := range items {
			where := "pack"
			switch {
			case it.Loc() == 1:
				where = "body"
			case it.Loc() == 2:
				where = "belt"
			case strings.Contains(it.Attrs["panel"], "stash"):
				where = "stash"
			}
			st.Items = append(st.Items, scenario.StateItem{Code: it.Code, Where: where})
		}
	}
	if ps, err := c.Portals(ctx); err == nil {
		st.Has = append(st.Has, "portals")
		for _, p := range ps.Items {
			st.Portals = append(st.Portals, scenario.StatePortal{Which: p.Which, Level: p.Level})
		}
	}
	if gs, err := c.Ground(ctx); err == nil && live { // the floor of the level the hero stands in: only a live observation means it
		st.Has = append(st.Has, "ground")
		for _, g := range gs.Items {
			if !g.IsGold() {
				st.Items = append(st.Items, scenario.StateItem{Code: g.Code, Where: "ground"})
			}
		}
	}
	if rep, err := c.Do(ctx, "dump player,stats,skills"); err == nil {
		st.Stats = pointStats(rep.Lines, rep.Rest)
		st.SkillLevels = skillLevels(rep.Lines, rep.Rest)
	}
	if rep, err := c.Do(ctx, "debug quest list"); err == nil {
		st.Quests = map[int]uint16{}
		for _, l := range rep.Lines {
			f := strings.Fields(l)
			if len(f) < 2 {
				continue
			}
			rec, e1 := strconv.Atoi(f[0])
			w, e2 := strconv.ParseUint(strings.TrimPrefix(f[1], "0x"), 16, 16)
			if e1 == nil && e2 == nil {
				st.Quests[rec] = uint16(w)
			}
		}
	}
	if rep, err := c.Do(ctx, "debug quest words"); err == nil { // records 16 to 47 (devp23 labverbs; see questWords)
		if w := questWords(rep.Lines, rep.Rest); w != nil {
			if st.Quests == nil {
				st.Quests = map[int]uint16{}
			}
			mergeQuestWords(st.Quests, w)
		}
	}
	if m, err := c.Minions(ctx); err == nil {
		n := 0
		for _, x := range m.Minions {
			if x.Here {
				n++
			}
		}
		st.Minions = &n
	}
	if l, err := c.CombatLog(ctx, 300); err == nil && l.Enabled {
		if ss, err := c.SkillState(ctx); err == nil {
			names := map[int]string{}
			for _, r := range ss.Skills {
				names[r.ID] = r.Name
			}
			st.HeroCasts = map[string]int{}
			for _, e := range l.Events {
				if e.Kind == "cast" && e.ByHero {
					st.HeroCasts[names[e.Skill]]++
				}
			}
		}
	}
	if rep, err := c.Do(ctx, "debug objects"); err == nil {
		st.Has = append(st.Has, "objects")
		for _, l := range rep.Lines { // columns: index x y kind class locked state (V docs/probes/search-for-cain-l0)
			f := strings.Fields(l)
			if len(f) >= 7 {
				if cl, e := strconv.Atoi(f[4]); e == nil {
					st.Objects = append(st.Objects, scenario.StateObject{Class: cl, State: f[len(f)-1]})
				}
			}
		}
	}
	return st
}

// pointStats reads the hero's points out of `dump player,stats,skills` (V dump_head_late.json: player.stat_pts and
// skill_pts, stats.base str/dex/vit/ene, skills.base_levels). nil when the dump is not there (stock d2d) or unreadable.
func pointStats(lines []string, rest string) map[string]int {
	d, ok := readPointDump(lines, rest)
	if !ok {
		return nil
	}
	out := map[string]int{}
	for _, k := range []string{"str", "dex", "vit", "ene"} {
		if v, ok := d.Stats.Base[k]; ok {
			out[k] = v
		}
	}
	if d.Player.StatPts != nil {
		out["statpts"] = *d.Player.StatPts
	}
	if d.Player.SkillPts != nil {
		out["skillpts"] = *d.Player.SkillPts
	}
	if d.Skills.Levels != nil {
		sum := 0
		for _, n := range d.Skills.Levels {
			sum += n
		}
		out["skillsspent"] = sum
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// skillLevels is the dump's skills.base_levels (class order), nil when the dump has none.
func skillLevels(lines []string, rest string) []int {
	if d, ok := readPointDump(lines, rest); ok {
		return d.Skills.Levels
	}
	return nil
}

type pointDump struct {
	Player struct {
		StatPts  *int `json:"stat_pts"`
		SkillPts *int `json:"skill_pts"`
	} `json:"player"`
	Stats struct {
		Base map[string]int `json:"base"`
	} `json:"stats"`
	Skills struct {
		Levels []int `json:"base_levels"`
	} `json:"skills"`
}

// readPointDump decodes the first non-empty line of a `dump player,stats,skills` reply.
func readPointDump(lines []string, rest string) (pointDump, bool) {
	line := strings.TrimSpace(rest)
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			line = l
			break
		}
	}
	var d pointDump
	return d, json.Unmarshal([]byte(line), &d) == nil
}

// withVault adds --vault-verbs (the fork's item create, item full and item vault) to d2d's arguments when asked.
func withVault(args []string, on bool) []string {
	if on {
		return append(args, "--vault-verbs")
	}
	return args
}

// needsPay is true for a scenario that needs a stash write of the fork: the gold reveal's paygold or the instant sale's creditgold
// (both behind --vault-pay, both refused under --lockstep).
// The direct equip verbs (item.equipdirect, ADR 0029 amended 2026-11-25) are the same kind of player-ordered write and are refused
// under --lockstep too, so they run unmeasured with --vault-verbs (which gates them); --vault-pay is harmless there.
func needsPay(needs []string) bool {
	return slices.Contains(needs, "vault.paygold") || slices.Contains(needs, "vault.creditgold") || slices.Contains(needs, "item.equipdirect")
}

// withPay adds --vault-pay when the scenario needs the capability vault.paygold, so the SETUP d2d lists it in `caps` (it pays nothing: the
// fork refuses the verb under --lockstep).
func withPay(args, needs []string) []string {
	if needsPay(needs) {
		return append(args, "--vault-pay")
	}
	return args
}

// botD2dArgs are the d2d arguments of the bot phase. A scenario that needs `vault.paygold` is an UNMEASURED run (the stash payment is
// a player-paid write, ADR 0029, which the fork refuses under --lockstep): d2d then runs in real time with --vault-pay, and the
// scenario's provenance names the write. Every other scenario keeps --lockstep.
func botD2dArgs(needs []string, vaultVerbs bool) []string {
	if needsPay(needs) {
		return append(withVault(nil, true), "--vault-pay")
	}
	return withItemVerbs(withRead(withVault([]string{"--lockstep"}, vaultVerbs), needs), needs)
}

// withItemVerbs adds --item-verbs (the fork's `item equip` and the other player item verbs) when the scenario lists the
// marker need "flag.item-verbs" (not a capability name: missingCaps ignores it). The sandbox runs with the flag.
func withItemVerbs(args, needs []string) []string {
	if slices.Contains(needs, "flag.item-verbs") {
		return append(args, "--item-verbs")
	}
	return args
}

// withRead adds the fork's combat read verbs (`debug minions`, skillstate, combatlog: reads, never writes, ADR 0022) when the
// scenario needs a read.* capability. --lab-verbs implies it already, so a field start needs nothing.
func withRead(args, needs []string) []string {
	for _, n := range needs {
		if strings.HasPrefix(n, "read.") {
			return append(args, "--read-combat")
		}
	}
	return args
}

// missingCaps lists the d2d capabilities in needs (the fork's names: item.*, ground.*, vault.*) that caps does not list.
// Other needs (a build, an input module) are not d2d capabilities and are checked elsewhere.
func missingCaps(needs, caps []string) []string {
	var out []string
	for _, n := range needs {
		if (strings.HasPrefix(n, "item.") || strings.HasPrefix(n, "ground.") || strings.HasPrefix(n, "vault.") || strings.HasPrefix(n, "read.")) && !d2dctl.HasCap(caps, n) {
			out = append(out, n)
		}
	}
	return out
}

// envKnobs layers "k=v,k=v" (env D2SCEN_KNOBS, numbers only) over the scenario's tuning, so one scenario runs with and
// without a knob (A/B probes) without a second scenario file. Anything that is not a number is ignored.
func envKnobs(base map[string]any, spec string) map[string]any {
	for _, kv := range strings.Split(spec, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(kv), "=")
		f, err := strconv.ParseFloat(v, 64)
		if !ok || err != nil || k == "" {
			continue
		}
		if base == nil {
			base = map[string]any{}
		}
		base[k] = f
	}
	return base
}

// readScriptedTaps reads the taps the bot made itself (d2bot --scripted-taps writes <script>.taps.json); none when the file is missing.
func readScriptedTaps(path string) (taps []string, at []float64) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var recs []scenario.TapRec
	if json.Unmarshal(b, &recs) != nil {
		return nil, nil
	}
	for _, r := range recs {
		taps, at = append(taps, r.Tap), append(at, r.At)
	}
	return taps, at
}
