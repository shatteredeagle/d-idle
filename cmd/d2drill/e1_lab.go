package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dmerc"
	"d2lod-bot/internal/drill"
)

// Group E1 (card shortdrills): short live drills of the g2e/g2f fixes, run against the REAL d2bot (not the lab's town
// object): the harness sets the hero up with logged lab verbs BEFORE the run start, drops its control connection (the
// d2d takes one client), and starts `d2bot --backend d2d --attach` on the same slot. After the start the bot acts only
// through its real input channel; "writes after start" comes from the ledger. The verdict is read from the bot's log.
//
//	shortdrill-merc    E1.merc-a  clvl 12, dead merc, purse = real quote + 10: revive BEFORE any potion purchase
//	                   E1.merc-b  clvl 12, purse below the quote: no revive, the potions are bought
//	                   E1.merc-c  clvl 14, purse = quote + 10: waits (needs quote + 50)
//	shortdrill-noportal E1.noportal-a  Hole Level 1 (area 11), 0 potions, 47 percent life, no portal: leaves after about 20 s
//	                   E1.noportal-b  the same with a town portal scroll: the bot portals, does not leave
//
// Limit (U): the bot is attached, so d2bot cannot restart this d2d; the leave is proven by the bot's own
// "leaving the game" line and its NewGame call, not by a new process.

type e1Cfg struct {
	Bot, Sock, BotAddr, SlotDir, GameDir, Instance string
	Out, Date, Label, Only                         string
	Wall                                           time.Duration // hard wall limit of one bot run
	Led                                            *cheat.Ledger
	Log                                            *slog.Logger
}

// e1h is the harness side: one control client, the ledger and the scenario's cheat list.
type e1h struct {
	c      *d2dctl.Client
	led    *cheat.Ledger
	actor  string
	inst   string
	char   string
	seq    int
	cheats []string
}

func (h *e1h) cheat(ctx context.Context, verb, reason string) (string, error) {
	h.seq++
	id := fmt.Sprintf("%s-%d", time.Now().UTC().Format("20060102T150405"), h.seq)
	e := cheat.Entry{Time: time.Now().UTC(), Actor: h.actor, Instance: h.inst, Character: h.char, Kind: "d2d-debug",
		Params: map[string]any{"id": id, "verb": verb}, Reason: reason}
	if err := h.led.Append(e); err != nil {
		return "", err
	}
	r, err := h.c.Do(ctx, verb)
	out := ""
	if r != nil {
		out = strings.TrimSpace(strings.Join(append(append([]string{}, r.Lines...), r.Rest), "\n"))
	}
	if err == nil && usageReply(out) {
		err = fmt.Errorf("lab binary lacks verb %q", verb)
	}
	e.Time, e.Success, e.After = time.Now().UTC(), err == nil, map[string]any{"reply": out}
	if err != nil {
		e.Error = err.Error()
	}
	_ = h.led.Append(e)
	h.cheats = append(h.cheats, verb)
	return out, err
}

func (h *e1h) step(ctx context.Context, n int) error { _, err := h.c.Step(ctx, n); return err }

func (h *e1h) tick(ctx context.Context) int {
	ci, err := h.c.Step(ctx, 1)
	if err != nil {
		return -1
	}
	return ci.Tick
}

// e1Scenario is one drill. Setup writes (logged cheats) happen before the run start and returns the facts it learned.
type e1Scenario struct {
	ID, Group, Title string
	Setup            func(ctx context.Context, h *e1h) (facts map[string]string, err error)
	Goals            string
	// Done says the evidence is complete (the run may stop); Eval gives the verdict from the bot log.
	Done func(lines []string) bool
	Eval func(lines []string, facts map[string]string) (verdict, reason string)
}

func e1Scenarios() []e1Scenario {
	merc := func(id, title string, clvl int, gold func(q int) int, eval func([]string, map[string]string) (string, string)) e1Scenario {
		return e1Scenario{ID: id, Group: "shortdrill-merc", Title: title, Goals: "farm:Blood Moor",
			Setup: func(ctx context.Context, h *e1h) (map[string]string, error) { return e1MercSetup(ctx, h, clvl, gold) },
			Done:  e1MercDone, Eval: eval}
	}
	np := func(id, title string, scroll bool) e1Scenario {
		return e1Scenario{ID: id, Group: "shortdrill-noportal", Title: title, Goals: "farm:Blood Moor",
			Setup: func(ctx context.Context, h *e1h) (map[string]string, error) { return e1NoPortalSetup(ctx, h, scroll) },
			Done: func(l []string) bool {
				return e1Find(l, "leaving the game") >= 0 || (scroll && e1Find(l, "town portal") >= 0 && e1Find(l, "heal") >= 0)
			},
			Eval: func(l []string, f map[string]string) (string, string) { return e1NoPortalEval(l, scroll) }}
	}
	return []e1Scenario{
		merc("E1.merc-a", "clvl 12, dead merc, purse = quote + 10, no potions: the revive happens before any potion purchase", 12,
			func(q int) int { return q + 10 }, e1MercEvalA),
		merc("E1.merc-b", "clvl 12, dead merc, purse below the quote: no revive, the potions are bought", 12,
			func(q int) int { return max(q-420, q*3/4) }, e1MercEvalB),
		merc("E1.merc-c", "clvl 14, dead merc, purse = quote + 10 (needs quote + 50): waits, no revive", 14,
			func(q int) int { return q + 10 }, e1MercEvalC),
		np("E1.noportal-a", "area 11, 0 potions, 47 percent life, no scroll or tome: the bot leaves after bot.noPortalWaitSec", false),
		np("E1.noportal-b", "control: the same with a town portal scroll in the pack: the bot portals to town, does not leave", true),
	}
}

// e1Reset puts the hero in town with an empty pack and belt before a scenario.
func e1Reset(ctx context.Context, h *e1h, clvl int) error {
	for _, v := range []struct{ verb, why string }{
		{"debug goto 1", "E1 setup: start in the Rogue Encampment"},
		{"debug clearinv", "E1 setup: empty pack (no potions, no town portal)"},
		{"debug belt clear", "E1 setup: no potion in the belt"},
		{fmt.Sprintf("debug stat 12 %d", clvl), fmt.Sprintf("E1 setup: character level %d", clvl)},
		{"debug stat 15 0", "E1 setup: no stash gold, the purse alone decides"},
	} {
		if _, err := h.cheat(ctx, v.verb, v.why); err != nil {
			return fmt.Errorf("%s: %w", v.verb, err)
		}
	}
	return h.step(ctx, 25)
}

func e1MercSetup(ctx context.Context, h *e1h, clvl int, gold func(int) int) (map[string]string, error) {
	f := map[string]string{"clvl": strconv.Itoa(clvl)}
	if err := e1Reset(ctx, h, clvl); err != nil {
		return f, err
	}
	// the merc is hired at the hero's level (U), killed, and left dead for 140 ticks as D8.a does
	_, _ = h.cheat(ctx, "debug merc hire 0", "E1 setup: a hired mercenary (no gold), to be killed")
	_ = h.step(ctx, 5)
	if _, err := h.cheat(ctx, "debug merc kill", "E1 setup: the mercenary is dead"); err != nil {
		return f, err
	}
	_ = h.step(ctx, 140)
	m, err := h.c.Do(ctx, "dump merc")
	if err != nil {
		return f, err
	}
	f["merc"] = strings.TrimSpace(strings.Join(append(append([]string{}, m.Lines...), m.Rest), " "))
	if !strings.Contains(f["merc"], `"dead":true`) {
		return f, fmt.Errorf("the merc is not dead after the kill: %s", f["merc"])
	}
	// the real quote, read from the NPC menu without paying (what the bot reads too)
	if _, err := h.cheat(ctx, "debug stat 14 5000", "E1 setup: a purse that shows the whole menu while the quote is probed"); err != nil {
		return f, err
	}
	_ = h.step(ctx, 5)
	rv := d2dmerc.Reviver{C: h.c}
	q, err := rv.Probe(ctx)
	if err != nil || q <= 0 {
		return f, fmt.Errorf("quote probe: %d %v", q, err)
	}
	f["quote"] = strconv.Itoa(q)
	g := gold(q)
	f["gold"] = strconv.Itoa(g)
	if _, err := h.cheat(ctx, fmt.Sprintf("debug stat 14 %d", g), fmt.Sprintf("E1 setup: purse %d for the real quote %d", g, q)); err != nil {
		return f, err
	}
	_ = h.step(ctx, 25)
	_, _ = h.c.Do(ctx, "key Escape")
	_ = h.step(ctx, 10)
	st, _, err := h.c.State(ctx)
	if err != nil {
		return f, err
	}
	f["state"] = fmt.Sprintf("area=%v level=%d gold=%d life=%d/%d", st.M["area"], st.Level, st.Gold, st.Life.Cur, st.Life.Max)
	if st.Gold != g || st.Level != clvl {
		return f, fmt.Errorf("setup not as intended: %s", f["state"])
	}
	return f, nil
}

func e1NoPortalSetup(ctx context.Context, h *e1h, scroll bool) (map[string]string, error) {
	f := map[string]string{}
	if err := e1Reset(ctx, h, 12); err != nil {
		return f, err
	}
	if scroll {
		if _, err := h.cheat(ctx, "debug give tsc pack", "E1 setup: the control keeps one town portal scroll"); err != nil {
			return f, err
		}
	}
	for _, v := range []struct{ verb, why string }{
		{"debug goto 11", "E1 setup: Hole Level 1 (area 11), the area of g2f seed 11"},
	} {
		if _, err := h.cheat(ctx, v.verb, v.why); err != nil {
			return f, err
		}
	}
	_ = h.step(ctx, 50)
	_, _ = h.cheat(ctx, "debug kill", "E1 setup: no natural monster near, the low life gate runs only without a hostile")
	_ = h.step(ctx, 25)
	st, _, err := h.c.State(ctx)
	if err != nil {
		return f, err
	}
	max := st.Life.Max
	if max <= 0 {
		return f, fmt.Errorf("no max life in state")
	}
	cur := max * 47 / 100
	if _, err := h.cheat(ctx, fmt.Sprintf("debug stat 6 %d", cur*256), fmt.Sprintf("E1 setup: life %d of %d (47 percent)", cur, max)); err != nil {
		return f, err
	}
	_ = h.step(ctx, 25)
	st, _, err = h.c.State(ctx)
	if err != nil {
		return f, err
	}
	f["state"] = fmt.Sprintf("area=%v level=%d gold=%d life=%d/%d", st.M["area"], st.Level, st.Gold, st.Life.Cur, st.Life.Max)
	if a, _ := st.KV.Int("area"); a != 11 {
		return f, fmt.Errorf("not in area 11: %s", f["state"])
	}
	return f, nil
}

// ---- verdicts from the bot log ----

func e1Find(lines []string, sub string) int {
	for i, l := range lines {
		if strings.Contains(l, sub) {
			return i
		}
	}
	return -1
}

// e1Buy matches the bot's purchase lines (potions, scrolls, repair) in the town code.
var e1Buy = regexp.MustCompile(`(?i)bought|restocked|purchase|buy `)

func e1FirstBuy(lines []string) int {
	for i, l := range lines {
		if e1Buy.MatchString(l) && !strings.Contains(l, "merc") && !strings.Contains(l, "no potion") {
			return i
		}
	}
	return -1
}

func e1MercDone(lines []string) bool {
	return (e1Find(lines, "merc: revived") >= 0 && e1FirstBuy(lines) >= 0) || e1Find(lines, "Leaving town") >= 0 ||
		e1Find(lines, "farm:") >= 0 && e1Find(lines, "merc: revive skipped") >= 0 && e1FirstBuy(lines) >= 0
}

func e1MercEvalA(lines []string, f map[string]string) (string, string) {
	r, b := e1Find(lines, "merc: revived"), e1FirstBuy(lines)
	switch {
	case r < 0:
		return "FAIL", "no `merc: revived` line (purse " + f["gold"] + ", quote " + f["quote"] + ")"
	case b >= 0 && b < r:
		return "FAIL", fmt.Sprintf("a purchase (log line %d) came before the revive (line %d)", b, r)
	}
	return "PASS", fmt.Sprintf("revived (log line %d) before the first purchase (%s)", r, e1LineOr(lines, b, "none seen"))
}

func e1MercEvalB(lines []string, f map[string]string) (string, string) {
	if e1Find(lines, "merc: revived") >= 0 {
		return "FAIL", "the merc was revived with a purse below the quote"
	}
	if e1FirstBuy(lines) < 0 {
		return "FAIL", "no potion purchase: the poor purse starved the potions"
	}
	return "PASS", "no revive, purchases happened (" + e1LineOr(lines, e1FirstBuy(lines), "") + ")"
}

func e1MercEvalC(lines []string, f map[string]string) (string, string) {
	if e1Find(lines, "merc: revived") >= 0 {
		return "FAIL", "revived at clvl 14 with quote + 10 (the reserve is 50)"
	}
	if i := e1Find(lines, "revive skipped (gold)"); i >= 0 {
		return "PASS", "skipped for gold: " + lines[i]
	}
	return "FAIL", "no `revive skipped (gold)` line and no revive"
}

func e1NoPortalEval(lines []string, scroll bool) (string, string) {
	leave := e1Find(lines, "leaving the game")
	if scroll {
		if leave >= 0 {
			return "FAIL", "the control left the game: " + lines[leave]
		}
		return "PASS", "no leave; the scroll path was taken"
	}
	first := e1Find(lines, "no town portal available")
	switch {
	case first < 0:
		return "FAIL", "no `no town portal available` line: the scenario never reached the no-portal path"
	case leave < 0:
		return "FAIL", "no leave after the first no-portal line (" + lines[first] + ")"
	}
	return "PASS", "left: " + lines[leave]
}

func e1LineOr(lines []string, i int, def string) string {
	if i < 0 || i >= len(lines) {
		return def
	}
	return strings.TrimSpace(lines[i])
}

// ---- the runner ----

// e1Result is the record of one scenario.
type e1Result struct {
	ID, Group, Title   string
	Verdict, Reason    string
	Facts              map[string]string
	Cheats             []string
	Run                string
	WritesAfterStart   int
	VirtualSec         float64
	WallSec            float64
	FirstNoPortalWall  string `json:",omitempty"`
	LeaveWall          string `json:",omitempty"`
	NoPortalToLeaveSec float64
	Waited             string `json:",omitempty"`
	Key                []string
	LogFile            string
}

var e1TimeRe = regexp.MustCompile(`time=(\S+)`)

func e1Time(line string) (time.Time, bool) {
	m := e1TimeRe.FindStringSubmatch(line)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, m[1])
	return t, err == nil
}

func runE1(ctx context.Context, c e1Cfg, be *d2denv.Backend) int {
	_ = be.Client.Close() // the d2d takes one control client; the bot is the next one
	want := map[string]bool{}
	for _, id := range strings.Split(c.Only, ",") {
		if id = strings.TrimSpace(id); id != "" {
			want[id] = true
		}
	}
	var out []e1Result
	for _, sc := range e1Scenarios() {
		if len(want) > 0 && !want[sc.ID] && !want[sc.Group] {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		r := e1One(ctx, c, sc)
		fmt.Printf("DRILL %s %s %s | %s\n", r.ID, r.Verdict, r.Reason, r.Title)
		out = append(out, r)
	}
	if h, err := e1Client(ctx, c); err == nil { // leave the hero in town
		_, _ = h.cheat(ctx, "debug goto 1", "E1 cleanup: back to the Rogue Encampment")
		_ = h.step(ctx, 25)
		h.c.Close()
	}
	b, _ := json.MarshalIndent(out, "", " ")
	_ = os.MkdirAll(c.Out, 0o755)
	base := filepath.Join(c.Out, c.Date+"-"+c.Label)
	if err := os.WriteFile(base+".json", b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "d2drill e1:", err)
		return 1
	}
	var sb strings.Builder
	for _, r := range out {
		fmt.Fprintf(&sb, "%s %s\n  %s\n  facts %v\n  cheats before the start: %s\n  run %s, writes after start: %d, virtual %.0f s, wall %.0f s\n", r.ID, r.Verdict, r.Reason, r.Facts, strings.Join(r.Cheats, "; "), r.Run, r.WritesAfterStart, r.VirtualSec, r.WallSec)
		if r.NoPortalToLeaveSec > 0 {
			fmt.Fprintf(&sb, "  first no-portal line %s, leave line %s: %.0f s wall; %s\n", r.FirstNoPortalWall, r.LeaveWall, r.NoPortalToLeaveSec, r.Waited)
		}
		for _, k := range r.Key {
			fmt.Fprintf(&sb, "    | %s\n", k)
		}
	}
	_ = os.WriteFile(base+".txt", []byte(sb.String()), 0o644)
	for _, r := range out {
		if r.Verdict != "PASS" {
			return 3
		}
	}
	return 0
}

func e1Client(ctx context.Context, c e1Cfg) (*e1h, error) {
	cl, err := d2dctl.Dial(ctx, c.Sock, d2dctl.Options{})
	if err != nil {
		return nil, err
	}
	d2dctl.Pause = func(ctx context.Context, d time.Duration) error { // lockstep: wait in ticks, wall time does nothing
		if err := ctx.Err(); err != nil {
			return err
		}
		_, err := cl.Step(ctx, max(int(d/(40*time.Millisecond)), 1))
		return err
	}
	return &e1h{c: cl, led: c.Led, inst: "slot" + os.Getenv("D2D_SLOT"), char: "Drilly"}, nil
}

func e1One(ctx context.Context, c e1Cfg, sc e1Scenario) e1Result {
	res := e1Result{ID: sc.ID, Group: sc.Group, Title: sc.Title}
	fail := func(why string) e1Result { res.Verdict, res.Reason = "BLOCKED", why; return res }
	h, err := e1Client(ctx, c)
	if err != nil {
		return fail("harness client: " + err.Error())
	}
	h.actor = "drills:" + sc.ID
	facts, err := sc.Setup(ctx, h)
	res.Facts, res.Cheats = facts, h.cheats
	if err != nil {
		h.c.Close()
		return fail("setup: " + err.Error())
	}
	run := fmt.Sprintf("%s-%s", sc.ID, time.Now().UTC().Format("20060102-150405"))
	res.Run = run
	if err := c.Led.MarkRun(run, h.actor, h.inst, "E1: setup written, the bot plays through its own code paths from here", time.Now()); err != nil {
		h.c.Close()
		return fail("run start: " + err.Error())
	}
	t0 := h.tick(ctx)
	h.c.Close() // free the socket for the bot
	logPath := filepath.Join(c.Out, c.Date+"-"+c.Label+"-"+sc.ID+".log.txt")
	_ = os.MkdirAll(c.Out, 0o755)
	lf, err := os.Create(logPath)
	if err != nil {
		return fail("log: " + err.Error())
	}
	defer lf.Close()
	res.LogFile = filepath.Base(logPath)
	args := []string{"--backend", "d2d", "--attach", "--d2d-attach", "--d2d-sock", c.Sock, "--gamedir", c.GameDir, "--data", filepath.Join(c.SlotDir, "e1data"),
		"--addr", c.BotAddr, "--goals", sc.Goals, "--log-level", "debug", "--run-for", fmt.Sprintf("%ds", int(c.Wall.Seconds()))}
	cmd := exec.Command(c.Bot, args...)
	cmd.Env = append(os.Environ(), "D2BOT_INSTANCE="+c.Instance, "D2BOT_RUN="+run)
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = io.MultiWriter(lf, pw), io.MultiWriter(lf, pw)
	if err := cmd.Start(); err != nil {
		return fail("start bot: " + err.Error())
	}
	bp := &botProc{cmd: cmd, done: make(chan struct{})}
	var mu sync.Mutex
	var lines []string
	go func() {
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			mu.Lock()
			lines = append(lines, sc.Text())
			mu.Unlock()
		}
	}()
	go func() { _ = cmd.Wait(); pw.Close(); close(bp.done) }()
	start := time.Now()
	snap := func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), lines...) }
	settle := time.Time{}
	for ctx.Err() == nil && !bp.exited() && time.Since(start) < c.Wall {
		time.Sleep(2 * time.Second)
		if settle.IsZero() && sc.Done(snap()) {
			settle = time.Now().Add(6 * time.Second) // let the lines after the evidence land
		}
		if !settle.IsZero() && time.Now().After(settle) {
			break
		}
	}
	res.WallSec = time.Since(start).Seconds()
	bp.stop()
	all := snap()
	h2, err := e1Client(ctx, c)
	if err == nil {
		if t1 := h2.tick(ctx); t0 >= 0 && t1 >= 0 {
			res.VirtualSec = float64(t1-t0) * 0.04
		}
		h2.c.Close()
	}
	n, _, werr := c.Led.WritesAfterStart(run)
	if werr != nil {
		res.Reason = "ledger: " + werr.Error()
	}
	res.WritesAfterStart = n
	res.Verdict, res.Reason = sc.Eval(all, facts)
	if n != 0 {
		res.Verdict, res.Reason = "CHEATED", fmt.Sprintf("writes after start: %d (%s)", n, res.Reason)
	}
	e1Key(&res, all)
	_, _, _ = drill.CapFile(logPath, drill.DefaultCap)
	return res
}

// e1Key picks the lines worth quoting and the no-portal timing.
func e1Key(res *e1Result, lines []string) {
	pat := regexp.MustCompile(`merc|no town portal|leaving the game|low life|heal trip|bought|restocked|town portal|NewGame|new game|Going to town|Resupply|waiting before`)
	for _, l := range lines {
		if pat.MatchString(l) && len(res.Key) < 40 {
			res.Key = append(res.Key, strings.TrimSpace(l))
		}
	}
	a, b := e1Find(lines, "no town portal available"), e1Find(lines, "leaving the game")
	if a >= 0 && b >= 0 {
		ta, oka := e1Time(lines[a])
		tb, okb := e1Time(lines[b])
		if oka && okb {
			res.FirstNoPortalWall, res.LeaveWall = ta.Format("15:04:05.000"), tb.Format("15:04:05.000")
			res.NoPortalToLeaveSec = tb.Sub(ta).Seconds()
		}
		if m := regexp.MustCompile(`waited=(\S+)`).FindStringSubmatch(lines[b]); m != nil {
			res.Waited = "bot waited=" + m[1] + " (virtual, bot clock)"
		}
	}
}
