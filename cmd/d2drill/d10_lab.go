package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/drill"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/paths"
	"d2lod-bot/internal/saves"
)

// d10Cfg is what the fault-recovery drill needs from the command line.
type d10Cfg struct {
	Bot         string        // d2bot binary
	D2D         string        // d2d lab binary
	GameDir     string        // game data (read only)
	Slot        string        // slot number
	Sock        string        // the slot's control socket
	BotAddr     string        // the slot's bot API address
	SlotDir     string        // the slot's private dir
	Name        string        // throwaway character
	Class       int           // d2d class index
	Seed        int64         // map seed
	Instance    string        // D2BOT_INSTANCE of the bot
	KillAfter   time.Duration // wall time from the bot start to the kill
	Observe     time.Duration // wall time observed after the kill
	ReapScript  string        // tools/agentreap.sh
	Agent       string        // registry agent name
	Out         string        // result dir
	Date, Label string
	Log         *slog.Logger
	Tune        *env.Tuning
	DB          *data.DB
}

// runD10 is the injected-fault drill (docs/review-strategy.md, d2d gate ladder, Gate 1): the managed bot loop on this
// slot the way the soak starts it, then kill -9 of the slot's d2d only. Three stages share one character on disk:
// stage A (a d2d of the harness) creates it and reads the P35 gate, stage B is the bot run with the fault, stage C
// loads the saved character in a d2d of the harness and reads the P35 gate again and the saved experience.
func runD10(ctx context.Context, c d10Cfg, repeat int) int {
	dataDir := filepath.Join(c.SlotDir, "d10data")
	ip, err := paths.Layout{Root: dataDir}.Instance(c.Instance)
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2drill d10:", err)
		return 1
	}
	home := filepath.Join(ip.Dir(), "d2d")
	var results []drill.Result
	for run := 1; run <= max(repeat, 1); run++ {
		c.Log.Info("d10 run", "run", run)
		o := d10Once(ctx, c, dataDir, home, run)
		fmt.Printf("run %d: %s %s\n", run, o.Verdict, o.Reason)
		rr := drill.RunResult{Outcome: o}
		if len(results) == 0 {
			results = append(results, drill.Result{ID: "D10", Primitive: "P36", Title: "d2d killed with SIGKILL under the managed bot: the bot restarts it", Cheats: nil})
		}
		res := &results[0]
		res.Runs = append(res.Runs, rr)
		res.Verdict, res.Reason, res.Intended, res.Observed = o.Verdict, o.Reason, o.Intended, o.Observed
		if run == 1 && o.Verdict == drill.Blocked {
			break
		}
	}
	res := &results[0]
	res.Deterministic = true
	for _, r := range res.Runs {
		if r.Verdict != res.Runs[0].Verdict {
			res.Deterministic = false
			res.Verdict = drill.Fail
			res.Reason = "the runs differ: " + res.Reason
		}
	}
	if len(res.Runs) > 1 { // the first run is the evidence; the line shows it and the verdicts of all runs
		res.Observed = "run1: " + res.Runs[0].Observed + " || run2: " + res.Runs[1].Observed
	}
	g := drill.Group{Results: results}
	fmt.Printf("%s\n", res.Line())
	fmt.Printf("\n%s", g.Table())
	if err := g.Write(c.Out, c.Date+"-"+c.Label); err != nil {
		fmt.Fprintln(os.Stderr, "d2drill d10: write results:", err)
		return 1
	}
	return 0
}

func (c d10Cfg) backend(ctx context.Context, home string, spec d2dinput.Spec) (*d2denv.Backend, error) {
	be, err := d2denv.Build(ctx, d2denv.Config{ID: "d10", Log: c.Log, Tune: c.Tune, Data: c.DB, Sock: c.Sock, LinkWait: 60 * time.Second,
		Bin: c.D2D, DataDir: c.GameDir, Home: home, Nice: 10, LogFile: filepath.Join(home, "d2d.out"),
		ExtraArgs: []string{"--toggle", "autoloot=off,autoparty=off", "--lockstep", "--lab-verbs"}, Spec: spec})
	if err != nil {
		return nil, err
	}
	if err := be.Start(ctx, 0); err != nil {
		be.Close()
		return nil, err
	}
	return be, nil
}

// gateOf is the P35 read on a backend: "clean" or what is open.
func gateOf(ctx context.Context, be *d2denv.Backend) string {
	f, err := be.Client.UI(ctx)
	if err != nil {
		return "unreadable: " + err.Error()
	}
	u := drill.UI{Cursor: f.Held}
	for _, p := range []struct {
		on   bool
		name string
	}{{f.Inventory, "inventory"}, {f.Stash, "stash"}, {f.Cube, "cube"}, {f.NPCMenu, "npc-dialog"}, {f.Vendor, "vendor"},
		{f.SkillTree, "skill-tree"}, {f.Waypoint, "waypoint"}, {f.GameMenu, "game-menu"}} {
		if p.on {
			u.Panels = append(u.Panels, p.name)
		}
	}
	return u.String()
}

// saveXP is the experience stored in the character's save file under the d2d home (-1: no file or unreadable).
func saveXP(home, name string) int64 {
	var best int64 = -1
	_ = filepath.WalkDir(home, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == name+".d2s" {
			if _, exp, e := saves.ExpFile(p); e == nil {
				best = int64(exp)
			}
		}
		return nil
	})
	return best
}

// d2dPIDs lists the pids whose command line carries --devctl <sock> (the slot's d2d, never another slot's).
func d2dPIDs(sock string) []int {
	ents, _ := os.ReadDir("/proc")
	var out []int
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		b, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil {
			continue
		}
		args := strings.Split(string(b), "\x00")
		for i, a := range args {
			if a == "--devctl" && i+1 < len(args) && args[i+1] == sock {
				out = append(out, pid)
			}
		}
	}
	return out
}

func pgidOf(pid int) int {
	g, err := syscall.Getpgid(pid)
	if err != nil {
		return -1
	}
	return g
}

func zombieOrAlive(pid int) bool {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	return err == nil && len(b) > 0
}

func (c d10Cfg) reap(arg string) string {
	out, _ := exec.Command(c.ReapScript, arg).CombinedOutput()
	return strings.TrimSpace(string(out))
}

func d10Once(ctx context.Context, c d10Cfg, dataDir, home string, run int) drill.Outcome {
	f := drill.D10Facts{SavedPre: -1, SavedEnd: -1}
	_ = os.RemoveAll(dataDir)
	if err := os.MkdirAll(home, 0o700); err != nil {
		return drill.Failed("setup: "+err.Error(), "", "")
	}
	// stage A: create and save the character, P35 gate before
	be, err := c.backend(ctx, home, d2dinput.Spec{Name: c.Name, Class: c.Class, Seed: c.Seed})
	if err != nil {
		return drill.Failed("stage A: "+err.Error(), "", "")
	}
	_, _ = be.Client.Step(ctx, 25)
	f.StagePre = gateOf(ctx, be)
	if err := be.Proc.SaveExit(ctx); err != nil {
		c.Log.Warn("d10 stage A save and exit", "err", err)
	}
	be.Close()
	if x := saveXP(home, c.Name); x < 0 {
		return drill.Failed("stage A: the character was not saved (no "+c.Name+".d2s under the d2d home)", "", "")
	}
	for i := 0; i < 50 && len(d2dPIDs(c.Sock)) > 0; i++ {
		time.Sleep(200 * time.Millisecond)
	}

	// stage B: the bot, started the way the soak does (docs/d2d-baselines-early.md section 4), with the fault
	logPath := filepath.Join(c.SlotDir, fmt.Sprintf("d10-bot-%d.log", run))
	lf, err := os.Create(logPath)
	if err != nil {
		return drill.Failed("log: "+err.Error(), "", "")
	}
	defer lf.Close()
	total := c.KillAfter + c.Observe + 2*time.Minute
	var mins []drill.Minute
	var evs []drill.Event
	var cur *botProc
	launch := func(n int) error {
		args := []string{"--backend", "d2d", "--managed", "--d2d-class", "-1", "--char", c.Name, "--seed", strconv.FormatInt(c.Seed, 10),
			"--toggle", "autoloot=off,autoparty=off", "--d2d-args", "--lockstep --lab-verbs", "--d2d-linkwait", "60s", "--gamedir", c.GameDir,
			"--d2d-data", c.GameDir, "--data", dataDir, "--d2d-sock", c.Sock, "--addr", c.BotAddr, "--goals", "campaign",
			"--run-for", fmt.Sprintf("%ds", int(total.Seconds()))}
		if n > 0 {
			args = append(args, "--clean-start")
		}
		cmd := exec.Command(c.Bot, args...)
		cmd.Env = append(os.Environ(), "D2BOT_INSTANCE="+c.Instance, "D2D_RUN=on", "D2D_BIN="+c.D2D, "D2D_DATA="+c.GameDir)
		pr, pw := io.Pipe()
		cmd.Stdout, cmd.Stderr = io.MultiWriter(lf, pw), io.MultiWriter(lf, pw)
		if err := cmd.Start(); err != nil {
			return err
		}
		bp := &botProc{cmd: cmd, done: make(chan struct{})}
		cur = bp
		go func() {
			sc := bufio.NewScanner(pr)
			sc.Buffer(make([]byte, 1<<20), 1<<20)
			for sc.Scan() {
				if m, ok := drill.ParseMinute(sc.Text()); ok {
					minMu.Lock()
					mins = append(mins, m)
					minMu.Unlock()
				} else if e, ok := drill.ParseEvent(sc.Text()); ok {
					minMu.Lock()
					evs = append(evs, e)
					minMu.Unlock()
				}
			}
		}()
		go func() { _ = cmd.Wait(); pw.Close(); close(bp.done) }()
		return nil
	}
	if err := launch(0); err != nil {
		return drill.Failed("start bot: "+err.Error(), "", "")
	}
	t0 := time.Now()
	snap := func() []drill.Minute { minMu.Lock(); defer minMu.Unlock(); return append([]drill.Minute(nil), mins...) }
	// a bot that exits is started again the way the soak's supervisor does (--clean-start); the exits are counted
	relaunch := func() error {
		if cur.exited() {
			f.BotExits++
			c.Log.Warn("d10: the bot process exited", "exits", f.BotExits)
			return launch(f.BotExits)
		}
		return nil
	}

	// wait for the kill moment: KillAfter of wall time, a minute line, and exactly one d2d on the slot
	var victim int
	for victim == 0 && ctx.Err() == nil {
		time.Sleep(2 * time.Second)
		if err := relaunch(); err != nil {
			return drill.Failed("restart bot: "+err.Error(), "", "")
		}
		if time.Since(t0) >= c.KillAfter && len(snap()) > 0 {
			if p := d2dPIDs(c.Sock); len(p) == 1 {
				victim = p[0]
			}
		}
		if victim == 0 && time.Since(t0) > c.KillAfter+5*time.Minute {
			cur.stop()
			return drill.Blocks("the bot never reached a minute line with one d2d on the slot within the time limit")
		}
	}
	if victim == 0 {
		cur.stop()
		return drill.Blocks("interrupted")
	}
	f.KilledPID = victim
	f.SavedPre = saveXP(home, c.Name)
	pg := pgidOf(victim)
	list := c.reap("--list")
	reg := "not in a registered group"
	for _, l := range strings.Split(list, "\n") {
		if fs := strings.Fields(l); len(fs) >= 3 && fs[0] == c.Agent && fs[2] == strconv.Itoa(pg) {
			reg = fmt.Sprintf("pid %d is in the process group %d registered to %s (tools/agentreap.sh --list)", victim, pg, c.Agent)
		}
	}
	f.Registry = reg
	f.KillAt = time.Now()
	if err := syscall.Kill(victim, syscall.SIGKILL); err != nil { // this one pid only
		cur.stop()
		return drill.Failed("kill: "+err.Error(), "", "")
	}
	// observe: a new d2d on the slot, minute lines, bot exits
	for deadline := f.KillAt.Add(c.Observe); time.Now().Before(deadline) && ctx.Err() == nil; {
		time.Sleep(2 * time.Second)
		if err := relaunch(); err != nil {
			break
		}
		if f.NewD2DAt.IsZero() {
			for _, p := range d2dPIDs(c.Sock) {
				if p != victim {
					f.NewD2DAt, f.NewD2DPID = time.Now(), p
				}
			}
		}
	}
	// graceful stop of the bot we started (SIGTERM: walk home, save, stop the game), then the leftovers
	cur.stop()
	f.Minutes = snap()
	minMu.Lock()
	f.Events = append([]drill.Event(nil), evs...)
	minMu.Unlock()
	f.StillThere = zombieOrAlive(victim)
	var left []string
	for _, arg := range []string{"--list", "--orphans"} {
		for _, l := range strings.Split(c.reap(arg), "\n") {
			if strings.Contains(l, fmt.Sprint(victim)) || (pg > 0 && strings.Contains(l, "pgid="+strconv.Itoa(pg)) && strings.Contains(l, "orphan")) {
				left = append(left, strings.TrimSpace(arg+": "+l))
			}
		}
	}
	f.Leftover = strings.Join(left, " / ")
	for i := 0; i < 50 && len(d2dPIDs(c.Sock)) > 0; i++ {
		time.Sleep(200 * time.Millisecond)
	}
	if p := d2dPIDs(c.Sock); len(p) > 0 {
		f.Leftover += fmt.Sprintf(" d2d still on the slot after the bot stopped: %v", p)
	}
	f.SavedEnd = saveXP(home, c.Name)

	// stage C: the saved character in a d2d of the harness, P35 gate after
	be2, err := c.backend(ctx, home, d2dinput.Spec{Name: c.Name, Class: -1, NoSave: true})
	if err != nil {
		f.StagePost = "stage C could not load the saved character: " + err.Error()
	} else {
		_, _ = be2.Client.Step(ctx, 25)
		f.StagePost = gateOf(ctx, be2)
		be2.Close()
	}
	o := drill.EvalD10(f)
	o.Observed += " | bot log " + filepath.Base(logPath)
	return o
}

var minMu = new(muT)

type muT = sync.Mutex

// botProc is one d2bot process of the run.
type botProc struct {
	cmd  *exec.Cmd
	done chan struct{} // closed when the process was waited for
}

func (b *botProc) exited() bool {
	select {
	case <-b.done:
		return true
	default:
		return false
	}
}

// stop ends the bot process this run started: SIGTERM (graceful: walk home, save, stop the game), SIGKILL after 90 s.
func (b *botProc) stop() {
	if b == nil || b.exited() {
		return
	}
	_ = b.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-b.done:
	case <-time.After(90 * time.Second):
		_ = b.cmd.Process.Kill()
		<-b.done
	}
}
