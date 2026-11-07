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
	"d2lod-bot/internal/resource"
)

// Group E1 (card shortdrills): short live drills of the g2e/g2f fixes, run against the REAL d2bot (not the lab's town
// object): the harness sets the hero up with logged lab verbs BEFORE the run start, drops its control connection (the
// d2d takes one client), and starts `d2bot --backend d2d --attach` on the same slot. After the start the bot acts only
// through its real input channel; "writes after start" comes from the ledger. The verdict is read from the bot's log.
//
//	shortdrill-merc    E1.merc-quote  clvl 12, purse 300: with a read quote no gold farm, the revive happens (BLOCKED when the quote reads 0)
//	                   E1.merc-a  clvl 12, dead merc, purse = real quote + 10: revive BEFORE any potion purchase
//	                   E1.merc-b  clvl 12, purse below the quote: no revive, the potions are bought
//	                   E1.merc-c  clvl 14, purse = quote + 10: waits (needs quote + 50)
//	shortdrill-noportal E1.noportal-a  Tamoe Highland (area 7; area 11 is now in chainHome: g2e F2c walks home from it), 0 potions, 47 percent life, no portal: leaves after about 20 s
//	                   E1.noportal-b  the same with a town portal scroll: the bot portals, does not leave
//
//	shortdrill-tpcount  E1.tp-a  town, 3 town portal scrolls in the pack: the first minute line says tp=3
//	                   E1.tp-b  town, a tome in the pack: the minute line tp equals the tome quantity the inv-tp-dump line counts
//	                   E1.tp-c  town, no scroll, gold 600: a minute line carries tpBuy and tp is then above 0
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

// area reads "dump player" (the same field the bot's reader takes); -1 when unreadable.
func (h *e1h) area(ctx context.Context) int {
	r, err := h.c.Do(ctx, "dump player")
	if err != nil {
		return -1
	}
	line := strings.TrimSpace(r.Rest)
	for _, l := range r.Lines {
		if strings.TrimSpace(l) != "" {
			line = l
			break
		}
	}
	var raw struct {
		Area *int `json:"area"`
		P    struct {
			Area *int `json:"area"`
		} `json:"player"`
	}
	if json.Unmarshal([]byte(line), &raw) != nil {
		return -1
	}
	if raw.Area != nil {
		return *raw.Area
	}
	if raw.P.Area != nil {
		return *raw.P.Area
	}
	return -1
}

// menuText opens Kashya's menu and returns its lines (diagnosis of an unreadable quote), then cancels.
func (h *e1h) menuText(ctx context.Context) string {
	npcs, err := h.c.NPCs(ctx)
	if err != nil {
		return err.Error()
	}
	for _, n := range npcs.Items {
		if strings.EqualFold(n.Name, "Kashya") {
			lines, err := h.c.OpenNPCMenu(ctx, n.Index, 20*time.Second)
			defer func() { _ = h.c.PickMenuLine(ctx, "cancel") }()
			if err != nil {
				return err.Error()
			}
			var t []string
			for _, l := range lines {
				t = append(t, l.Text)
			}
			return strings.Join(t, " | ")
		}
	}
	return "no Kashya"
}

// closeUI sends Escape while a panel, dialog or cursor item is open (never on a clean screen: that opens the game menu).
func (h *e1h) closeUI(ctx context.Context) {
	for i := 0; i < 5; i++ {
		f, err := h.c.UI(ctx)
		if err != nil || !(f.Inventory || f.Stash || f.Cube || f.NPCMenu || f.Vendor || f.SkillTree || f.Waypoint || f.GameMenu) {
			return
		}
		_, _ = h.c.Do(ctx, "key Escape")
		_ = h.step(ctx, 15)
	}
}

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
	tp := func(id, title string, setup func(context.Context, *e1h) (map[string]string, error), done func([]string) bool, eval func([]string, map[string]string) (string, string)) e1Scenario {
		return e1Scenario{ID: id, Group: "shortdrill-tpcount", Title: title, Goals: "farm:Blood Moor", Setup: setup, Done: done, Eval: eval}
	}
	return []e1Scenario{
		tp("E1.tp-a", "town, 3 town portal scrolls in the pack: the minute line shows tp=3", e1TPSetup(3, false, 0),
			func(l []string) bool { return len(e1Minutes(l)) >= 1 }, e1TPEvalA),
		tp("E1.tp-b", "town, a town portal tome in the pack: the minute line tp equals the quantity inv-tp-dump counts", e1TPSetup(0, true, 0),
			func(l []string) bool { return len(e1Minutes(l)) >= 1 }, e1TPEvalB),
		tp("E1.tp-c", "town, no scroll, gold 600: the bot buys scrolls, tpBuy appears and tp is above 0", e1TPSetup(0, false, 600),
			func(l []string) bool { return len(e1Minutes(l)) >= 3 }, e1TPEvalC),
		merc("E1.merc-a", "clvl 12, dead merc, purse = quote + 10, no potions: the revive happens before any potion purchase", 12,
			func(q int) int { return q + 10 }, e1MercEvalA),
		merc("E1.merc-b", "clvl 12, dead merc, purse below the quote: no revive, the potions are bought", 12,
			func(q int) int { return max(q-420, q*3/4) }, e1MercEvalB),
		merc("E1.merc-c", "clvl 14, dead merc, purse = quote + 10 (needs quote + 50): waits, no revive", 14,
			func(q int) int { return q + 10 }, e1MercEvalC),
		merc("E1.merc-quote", "clvl 12, dead merc, purse 300 (far below the formula 1080): a read quote covers it, so no gold farm and the revive happens", 12,
			func(int) int { return 300 }, e1MercQuoteEval),
		np("E1.noportal-a", "area 7, 0 potions, 47 percent life, no scroll or tome: the bot leaves after bot.noPortalWaitSec", false),
		np("E1.noportal-b", "control: the same with a town portal scroll in the pack: the bot portals to town, does not leave", true),
		e1MouthScenario(),
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
	// A cheat-hired merc is level 1 and its revive is free ("Resurrect Aliza: 0", V runs of 2026-11-06 on slots 4 and 5,
	// every hire type 0..4 alike), and no verb sets a merc level. The bot then cannot read a quote (probeQuote keeps only
	// a price above 0) and decides on the level estimate, so the drill sets the purse around that estimate.
	if _, err := h.cheat(ctx, "debug stat 14 5000", "E1 setup: a purse that shows the whole menu while the quote is probed"); err != nil {
		return f, err
	}
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
	rv := d2dmerc.Reviver{C: h.c}
	pctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	q, err := rv.Probe(pctx)
	cancel()
	if err != nil {
		return f, fmt.Errorf("quote probe: %v; menu: %s", err, h.menuText(ctx))
	}
	f["liveQuote"] = strconv.Itoa(q)
	if q <= 0 {
		q = resource.MercReviveCost(clvl)
		f["quoteSource"] = "level estimate " + strconv.Itoa(q) + " (the live quote is 0)"
	} else {
		f["quoteSource"] = "live quote"
	}
	f["quote"] = strconv.Itoa(q)
	g := gold(q)
	f["gold"] = strconv.Itoa(g)
	if _, err := h.cheat(ctx, fmt.Sprintf("debug stat 14 %d", g), fmt.Sprintf("E1 setup: purse %d for the real quote %d", g, q)); err != nil {
		return f, err
	}
	_ = h.step(ctx, 25)
	h.closeUI(ctx)
	if _, err := h.cheat(ctx, "debug goto 1", "E1 setup: the arrival cell of the camp, as after a fresh start (the quote probe walked the hero to Kashya)"); err != nil {
		return f, err
	}
	_ = h.step(ctx, 50)
	st, _, err := h.c.State(ctx)
	if err != nil {
		return f, err
	}
	f["state"] = fmt.Sprintf("area=%d level=%d gold=%d life=%d/%d", h.area(ctx), st.Level, st.Gold, st.Life.Cur, st.Life.Max)
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
		{"debug goto 7", "E1 setup: Tamoe Highland (area 7): not in the walk-home tables (resource.chainHome), no waypoint, so no way to town without a portal"},
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
	a := h.area(ctx)
	f["state"] = fmt.Sprintf("area=%d level=%d gold=%d life=%d/%d", a, st.Level, st.Gold, st.Life.Cur, st.Life.Max)
	if a != 7 {
		return f, fmt.Errorf("not in area 7: %s", f["state"])
	}
	return f, nil
}

// e1TPSetup puts the hero in the camp with n scrolls (separate pack items: a given scroll is not merged into a tome),
// optionally a tome, and a purse of gold (0 keeps the stat at 0).
func e1TPSetup(n int, tome bool, gold int) func(context.Context, *e1h) (map[string]string, error) {
	return func(ctx context.Context, h *e1h) (map[string]string, error) {
		f := map[string]string{"scrolls": strconv.Itoa(n), "tome": strconv.FormatBool(tome), "gold": strconv.Itoa(gold)}
		if err := e1Reset(ctx, h, 12); err != nil {
			return f, err
		}
		for i := 0; i < n; i++ {
			if _, err := h.cheat(ctx, "debug give tsc pack", "E1 setup: a town portal scroll in the pack"); err != nil {
				return f, err
			}
		}
		if tome {
			if _, err := h.cheat(ctx, "debug give tbk pack", "E1 setup: a town portal tome in the pack"); err != nil {
				return f, err
			}
		}
		if _, err := h.cheat(ctx, fmt.Sprintf("debug stat 14 %d", gold), fmt.Sprintf("E1 setup: purse %d", gold)); err != nil {
			return f, err
		}
		_ = h.step(ctx, 25)
		st, _, err := h.c.State(ctx)
		if err != nil {
			return f, err
		}
		f["state"] = fmt.Sprintf("area=%d level=%d gold=%d", h.area(ctx), st.Level, st.Gold)
		return f, nil
	}
}

var (
	e1TPRe    = regexp.MustCompile(`"tp":(\d+)`)
	e1TPBuyRe = regexp.MustCompile(`"tpBuy":(\d+)`)
	e1CountRe = regexp.MustCompile(`tome class=\d+ where=inventory addr=\S+ counted=(\d+)`)
)

// e1Minutes are the bot's minute lines.
func e1Minutes(lines []string) []string {
	var out []string
	for _, l := range lines {
		if strings.Contains(l, `"msg":"minute"`) && e1TPRe.MatchString(l) {
			out = append(out, l)
		}
	}
	return out
}

func e1Int(re *regexp.Regexp, l string) int {
	if m := re.FindStringSubmatch(l); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

func e1TPEvalA(lines []string, _ map[string]string) (string, string) {
	m := e1Minutes(lines)
	if len(m) == 0 {
		return "FAIL", "no minute line with a tp field"
	}
	if tp := e1Int(e1TPRe, m[0]); tp != 3 {
		return "FAIL", fmt.Sprintf("first minute line tp=%d, want 3", tp)
	}
	return "PASS", "first minute line tp=3 with 3 scrolls in the pack"
}

func e1TPEvalB(lines []string, _ map[string]string) (string, string) {
	m := e1Minutes(lines)
	if len(m) == 0 {
		return "FAIL", "no minute line with a tp field"
	}
	i := -1
	for j, l := range lines {
		if strings.Contains(l, `"inv-tp-dump"`) && e1CountRe.MatchString(l) {
			i = j
			break
		}
	}
	if i < 0 {
		return "FAIL", "no inv-tp-dump line counting a tome"
	}
	want, tp := e1Int(e1CountRe, lines[i]), e1Int(e1TPRe, m[0])
	if want <= 0 {
		return "FAIL", fmt.Sprintf("the tome counted %d: inconclusive (the given tome is empty), minute tp=%d", want, tp)
	}
	if tp != want {
		return "FAIL", fmt.Sprintf("minute tp=%d, inv-tp-dump counted %d", tp, want)
	}
	return "PASS", fmt.Sprintf("minute tp=%d equals the tome quantity counted by inv-tp-dump", tp)
}

func e1TPEvalC(lines []string, _ map[string]string) (string, string) {
	m := e1Minutes(lines)
	if len(m) == 0 {
		return "FAIL", "no minute line with a tp field"
	}
	buy, hi := -1, 0
	for j, l := range m {
		if buy < 0 && e1Int(e1TPBuyRe, l) > 0 {
			buy = j
		}
		hi = max(hi, e1Int(e1TPRe, l))
	}
	switch {
	case buy < 0:
		return "FAIL", fmt.Sprintf("no minute line with tpBuy over %d minute lines (max tp %d)", len(m), hi)
	case hi == 0:
		return "FAIL", "tpBuy appeared but tp stayed 0"
	}
	return "PASS", fmt.Sprintf("tpBuy>0 on minute line %d, max tp %d", buy+1, hi)
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

// e1MercQuoteEval (g2g M2): once the quote is read the farm bar is the quote, not the level formula. BLOCKED when the
// menu quoted nothing above 0 (a cheat-hired merc is level 1 and says 0), because then only the formula path runs.
func e1MercQuoteEval(lines []string, f map[string]string) (string, string) {
	r, farm := e1Find(lines, "merc: revived"), e1Find(lines, "merc: gold farm on")
	if e1Find(lines, "revive quote read price=") < 0 {
		return "BLOCKED", "the bot read no quote above 0 (live quote " + f["liveQuote"] + "): only the formula path ran, the new rule is not exercised"
	}
	switch {
	case farm >= 0 && (r < 0 || farm < r):
		return "FAIL", "gold farm armed although a quote was read: " + e1LineOr(lines, farm, "")
	case r < 0:
		return "FAIL", "no revive although the quote fits the purse (" + f["gold"] + ")"
	}
	return "PASS", "quote read, no gold farm, revived: " + e1LineOr(lines, r, "")
}

func e1NoPortalEval(lines []string, scroll bool) (string, string) {
	leave := e1Find(lines, "leaving the game")
	if scroll {
		if leave >= 0 {
			return "FAIL", "the control left the game: " + lines[leave]
		}
		if i := e1Town(lines); i >= 0 {
			return "PASS", "no leave; the scroll took the hero to town: " + strings.TrimSpace(lines[i])
		}
		return "FAIL", "no leave, but the hero never reached town either"
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

// e1Town is the first zone entry into area 1 (the Rogue Encampment).
func e1Town(lines []string) int {
	for i, l := range lines {
		if strings.Contains(l, `"zone entry"`) && strings.Contains(l, `"area":1,`) {
			return i
		}
	}
	return -1
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
	WaitedSec          float64
	Key                []string
	LogFile            string
}

var e1TimeRe = regexp.MustCompile(`"time":"([^"]+)"|time=(\S+)`)

func e1Time(line string) (time.Time, bool) {
	m := e1TimeRe.FindStringSubmatch(line)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, m[1]+m[2])
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
		"--addr", c.BotAddr, "--goals", sc.Goals, "--log-level", "info", "--run-for", fmt.Sprintf("%ds", int(c.Wall.Seconds()))}
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
	pat := regexp.MustCompile(`"msg":"minute"|inv-tp-dump|merc|no town portal|leaving the game|low life|heal trip|bought|restocked|town portal|NewGame|new game|Going to town|Resupply|waiting before`)
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
		if m := regexp.MustCompile(`"waited":"?(\d+)`).FindStringSubmatch(lines[b]); m != nil {
			ns, _ := strconv.ParseInt(m[1], 10, 64)
			res.Waited = fmt.Sprintf("bot waited %.0f s (virtual, bot clock, since the first no-portal failure)", float64(ns)/1e9)
			res.WaitedSec = float64(ns) / 1e9
		}
	}
}
