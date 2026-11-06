// Command d2drill runs the live primitive drills (card m1-liveproof, docs/primitives.md) on ONE hidden d2d instance
// with a throwaway character: setup through the logged cheat tool only, the primitive through the bot's real code path,
// the end state through the StateReader, a P35 UI gate around every drill.
//
// Run it on the worker host inside a slot, under the process registry:
//
//	D2D_AGENT=drills tools/agentrun.sh drills --ttl 3600 -- tools/d2dslot.sh run -t 3600 -- d2drill -group D1
//
// Output: one "DRILL ..." line per drill on stdout, docs/probes/drills/<date>-<group>.txt and .json.
package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/drill"
	"d2lod-bot/internal/env"
)

func main() { os.Exit(run()) }

func run() int {
	home := os.Getenv("HOME")
	slotDir := os.Getenv("D2D_SLOT_DIR")
	group := flag.String("group", "D1", "drill group (D0 gate self-test, D1 belt, D2 portals and town entry, D3 town, D4 pack, D5 combat, D6 movement, D7 build, D9 lifecycle, DW waypoints, DR repair, D10 fault recovery, H hover audit, P DRILLS11 fix proofs)")
	only := flag.String("only", "", "comma list of drill ids to run (default: the whole group)")
	repeat := flag.Int("repeat", 2, "runs per drill (determinism check)")
	bin := flag.String("bin", envOr("D2D_BIN", ""), "d2d binary (default $D2D_BIN: the Gate 2 lab build, the fork build with the setup verbs debug clearinv/durability/stat)")
	gamedir := flag.String("gamedir", envOr("D2D_DATA", filepath.Join(home, "d2d-lab/data")), "game data directory (read only)")
	hdir := flag.String("home", filepath.Join(slotDir, "drillhome"), "d2d XDG home (own saves and log)")
	cache := flag.String("cache", filepath.Join(home, "d2d-lab/drills/tables"), "extracted game tables cache")
	sock := flag.String("sock", envOr("D2D_SOCK", ""), "control socket (the slot's)")
	seed := flag.Int64("seed", 11, "map seed")
	class := flag.Int("class", 4, "d2d class index of the new character")
	name := flag.String("name", "Drilly", "throwaway character name")
	out := flag.String("out", "docs/probes/drills", "result directory")
	date := flag.String("date", time.Now().UTC().Format("2006-01-02"), "date prefix of the result files")
	ledger := flag.String("ledger", "", "cheat ledger (default: $D2CHEAT_LEDGER or ~/.local/share/d2bot/ledger/cheats.jsonl)")
	belts := flag.String("belts", "lbl,vbl,mbl,tbl,hbl", "belt item codes P2 equips")
	verbose := flag.Bool("v", false, "debug log of the bot code paths on stderr")
	probe := flag.String("probe", "", "semicolon list of verbs to run and print (debug verbs go through the cheat ledger), then exit; for finding out what a verb does")
	label := flag.String("label", "", "result file label after the date (default: the group, lower case)")
	teleMinLeg := flag.Float64("tele-min-leg", -1, "override nav.teleMinLeg (subtiles; default keeps the knob default; 0 hops whenever mana allows)")
	teleReserve := flag.Float64("tele-reserve", -1, "override nav.teleReservePct (percent of max mana kept after a hop; default keeps 40)")
	repairBelow := flag.Int("repair-below", 0, "override the bot's repairBelowPct knob (default 30; `debug durability all 1` leaves 5 percent, so the default reaches the repair path)")
	botBin := flag.String("bot", envOr("D2BOT_BIN", filepath.Join(home, "d2d-lab/drills4/d2bot")), "D10: the d2bot binary the fault drill runs managed")
	killAfter := flag.Duration("d10-kill-after", 4*time.Minute, "D10: wall time from the bot start to the kill -9 of the slot's d2d")
	observe := flag.Duration("d10-observe", 8*time.Minute, "D10: wall time observed after the kill")
	reapScript := flag.String("reap", "tools/agentreap.sh", "D10: the process registry tool")
	limit := flag.Duration("limit", 0, "override the per-run hard time limit")
	capFile := flag.String("cap", "", "cap this saved run log in place (head, tail, verdict and error lines; default max 400 KB) before committing it under docs/probes, then exit")
	t0Profiles := flag.String("t0-profiles", "cautious,bold,inconsistent,holdout:101,holdout:202", "T0: comma list of answer profiles (cautious, bold, inconsistent, holdout:SEED, file)")
	t0Answers := flag.String("t0-answers", "", "T0: scripted answer file for the profile 'file' (JSON, read only; no stdin, no network)")
	t0Minutes := flag.Int("t0-minutes", 10, "T0: virtual minutes of play after the run start")
	t0Dir := flag.String("t0-dir", "", "T0: directory of the persisted answer files (default: in memory)")
	flag.Parse()
	binName = *bin
	if *capFile != "" {
		before, after, err := drill.CapFile(*capFile, drill.DefaultCap)
		if err != nil {
			fmt.Fprintln(os.Stderr, "d2drill:", err)
			return 1
		}
		fmt.Printf("%s: %d -> %d bytes\n", *capFile, before, after)
		return 0
	}
	if *bin == "" {
		fmt.Fprintln(os.Stderr, "d2drill: D2D_BIN is not set: point it (or -bin) at the Gate 2 lab binary, the fork build with the setup verbs debug clearinv/durability/stat; an older lab binary lacks them")
		return 2
	}
	if *sock == "" || slotDir == "" {
		fmt.Fprintln(os.Stderr, "d2drill: run inside tools/d2dslot.sh (needs D2D_SOCK and D2D_SLOT_DIR)")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	lvl := slog.LevelWarn
	if *verbose {
		lvl = slog.LevelDebug
	}
	// the lab keeps every record in memory too (the drills quote the bot's failure lines); stderr only shows lvl and up
	rec := newRecorder(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
	log := slog.New(rec)

	db, err := data.Open(*gamedir, *cache)
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2drill: game tables:", err)
		return 1
	}
	tune := env.DefaultTuning()
	if *teleMinLeg >= 0 { // DRILLS24: the knob override of docs/teletune.md's live recipe
		tune.Nav.TeleMinLeg = *teleMinLeg
	}
	if *teleReserve >= 0 {
		tune.Nav.TeleReservePct = *teleReserve
	}
	if *repairBelow > 0 {
		tune.Town.RepairBelowPct = *repairBelow
	}
	if err := os.MkdirAll(*hdir, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "d2drill:", err)
		return 1
	}
	if strings.EqualFold(*group, "D10") { // fault recovery: the managed bot runs the loop, no harness d2d during the run
		return runD10(ctx, d10Cfg{Bot: *botBin, D2D: *bin, GameDir: *gamedir, Slot: os.Getenv("D2D_SLOT"), Sock: *sock, BotAddr: os.Getenv("D2D_BOT_ADDR"),
			SlotDir: slotDir, Name: *name, Class: *class, Seed: *seed, Instance: "drills4d10", KillAfter: *killAfter, Observe: *observe,
			ReapScript: *reapScript, Agent: envOr("D2D_AGENT", "drills4"), Out: *out, Date: *date, Label: cmp.Or(*label, "d10"), Log: log, Tune: &tune, DB: db}, *repeat)
	}
	be, err := d2denv.Build(ctx, d2denv.Config{ID: "drills", Log: log, Tune: &tune, Data: db, Sock: *sock, LinkWait: 30 * time.Second,
		Bin: *bin, DataDir: *gamedir, Home: *hdir, Nice: 10, LogFile: filepath.Join(*hdir, "d2d.out"),
		ExtraArgs: []string{"--toggle", "autoloot=off,autoparty=off", "--lockstep", "--lab-verbs"},
		Spec:      d2dinput.Spec{Name: *name, Class: *class, Seed: *seed, NoSave: true}})
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2drill: build:", err)
		return 1
	}
	defer be.Close()
	if err := be.Start(ctx, 0); err != nil {
		fmt.Fprintln(os.Stderr, "d2drill: start d2d:", err)
		return 1
	}
	lp := *ledger
	if lp == "" {
		lp = envOr("D2CHEAT_LEDGER", filepath.Join(home, ".local/share/d2bot/ledger/cheats.jsonl")) // not the slot's XDG data dir
	}
	led := cheat.NewLedger(lp)
	if err := led.Ensure(); err != nil {
		fmt.Fprintln(os.Stderr, "d2drill: the cheat ledger must be writable before any cheat:", err)
		return 1
	}
	l := newLab(be, db, led, *name, "slot"+os.Getenv("D2D_SLOT"), log)
	l.rec = rec
	rec.SetTick(l.tick)
	l.tune = &tune
	if *probe != "" {
		return l.runProbe(ctx, *probe)
	}

	var drills []drill.Drill
	switch strings.ToUpper(*group) {
	case "D1":
		drills = drill.D1(l, strings.Split(*belts, ","))
	case "D2":
		drills = drill.D2(l)
	case "D3":
		drills = drill.D3(l)
	case "D4":
		drills = drill.D4(l)
	case "D5":
		drills = drill.D5(l)
	case "D6":
		drills = drill.D6(l)
	case "D7":
		drills = drill.D7(l)
	case "D8":
		drills = drill.D8(l)
	case "D9":
		drills = drill.D9(l)
	case "F": // DRILLS5 field drills: operating field objects, fights with debug spawn (dodge, target choice, retreat, mana drink)
		drills = drill.F(l)
	case "M": // fixmeter live proof (D-meter)
		drills = drill.M(l)
	case "P": // DRILLS11: fixemptyplan and fixportalmelee live proofs, the cut-cast record
		drills = drill.D11(l)
	case "Q": // DRILLS13: fixrecast Frozen Armor proof, M1 mana timing
		drills = drill.D13(l)
	case "R": // DRILLS15 (skill-smoke traces) and DRILLS16 (fixbuff live proof): one group, drill names R-bs* / D15-* tell them apart
		drills = append(drill.D15(l), drill.D16(l)...)
	case "U": // DRILLS26: Frozen Armor recast in a real fight, binary named
		drills = drill.D26(l)
	case "V": // DRILLS28: D28-0 Frozen Armor refresh with no monsters (armorcut step 0)
		drills = drill.D28(l)
	case "W": // DRILLS29: Frozen Armor recast with a pack held in melee at the due time (armorcut retry)
		drills = drill.D29(l)
	case "RT": // retreatdrill: the retreat action executed (drink, town portal, chicken), docs/retreatdrill.md
		drills = drill.RT(l)
	case "DS": // codefix67 P5 stash recheck, docs/livecheckdrills.md
		drills = drill.DStash(l, 10*time.Minute)
	case "TM": // tomefix stat-70 recheck, docs/livecheckdrills.md
		drills = drill.DTome(l, 25*time.Minute)
	case "T": // DRILLS22: live proof of fixtelekey (Teleport capability in Act 2 legs)
		drills = drill.D22(l)
	case "H": // hover blind-spot audit (docs/probes/drills/hover-audit.md)
		drills = drill.H(l)
	case "DW":
		drills = drill.DW(l)
	case "DR": // repair group of DRILLS4: D3.r, D3.r0, D3.r1 and the price table D3.rt (run with -repair-below 100)
		drills = drill.DR(l)
	case "T0": // card t0-slice (TASK-21): ladder.enabled on, scripted answers, wired town points, writes after start 0, docs/probes/t0drill.md
		cfg, err := t0Config(*t0Profiles, *t0Answers, *t0Minutes, *t0Dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "d2drill:", err)
			return 2
		}
		tune.Ladder.Enabled = 1
		drills = drill.T0(l, cfg)
	case "D0": // harness self-test of the P35 gate: the drill leaves the pack open on purpose
		drills = []drill.Drill{{ID: "D0.gate", Primitive: "P35", Title: "selftest: the drill leaves the inventory open, the gate must flag and recover it", Limit: time.Minute,
			Do: func(ctx context.Context, r *drill.Run) drill.Outcome {
				if err := be.Client.Key(ctx, "i"); err != nil {
					return drill.Failed("key i: "+err.Error(), "", "")
				}
				_, _ = be.Client.Step(ctx, 10)
				return drill.Passed("opened the inventory and returned without closing it", "gate reports a dirty UI after the drill", "inventory open")
			}}}
	default:
		fmt.Fprintln(os.Stderr, "d2drill: unknown group", *group)
		return 2
	}
	if *only != "" {
		want := map[string]bool{}
		for _, id := range strings.Split(*only, ",") {
			want[strings.TrimSpace(id)] = true
		}
		var keep []drill.Drill
		for _, d := range drills {
			if want[d.ID] {
				keep = append(keep, d)
			}
		}
		drills = keep
	}
	if *limit > 0 {
		for i := range drills {
			drills[i].Limit = *limit
		}
	}
	r := &drill.Runner{W: l, Repeat: *repeat, Logf: func(f string, a ...any) { fmt.Printf(f+"\n", a...) }}
	g := r.RunGroup(ctx, drills)

	if g := strings.ToUpper(*group); g == "D2" || g == "DW" || g == "Q" || g == "R" || g == "T" { // the portal drills end in the field: put the character back in town (logged cheat)
		if _, err := l.Cheat(ctx, "drills:cleanup", "debug goto 1", "return the throwaway character to the Rogue Encampment after the portal drills"); err != nil {
			fmt.Fprintln(os.Stderr, "d2drill: cleanup goto:", err)
		}
		_ = l.Tick(ctx, 25)
	}
	// The character stays alive and in town.
	if s, err := be.Env.Snapshot(); err == nil && s.Player != nil {
		fmt.Printf("END character %s life %d/%d in town: %v\n", *name, s.Player.Life, s.Player.MaxLife, s.Player.InTown())
	}
	fmt.Printf("\n%s", g.Table())
	fmt.Printf("cheat ledger: %s (actor drills:<drill id>, kind d2d-debug)\n", lp)
	if err := g.Write(*out, *date+"-"+cmp.Or(*label, strings.ToLower(*group))); err != nil {
		fmt.Fprintln(os.Stderr, "d2drill: write results:", err)
		return 1
	}
	return 0
}

// t0Config parses the T0 flags: holdout:SEED profiles are generated from the seed, never from a live person.
func t0Config(profiles, answers string, minutes int, dir string) (drill.T0Config, error) {
	c := drill.T0Config{Answers: answers, Minutes: minutes, Dir: dir}
	for _, p := range strings.Split(profiles, ",") {
		name, seed, ok := strings.Cut(strings.TrimSpace(p), ":")
		if name == "holdout" {
			n, err := strconv.ParseInt(seed, 10, 64)
			if !ok || err != nil {
				return c, fmt.Errorf("t0 profile %q: holdout needs a seed (holdout:101)", p)
			}
			c.Seeds = append(c.Seeds, n)
		} else if ok {
			return c, fmt.Errorf("t0 profile %q: only holdout takes a seed", p)
		}
		c.Profiles = append(c.Profiles, name)
	}
	if c.Answers == "" {
		for _, p := range c.Profiles {
			if p == "file" {
				return c, fmt.Errorf("t0 profile file needs -t0-answers")
			}
		}
	}
	return c, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
