package d2dsoakstat

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Invariants answer "is the run sound", apart from "is the run good" (the metrics and the fidelity verdict). They are
// counters read from the bot log of one seed (minute lines plus the warn lines of the loops that guard the run) with a
// threshold each, so a hang, a wedge, a firing stall ladder, an unrecovered death or an early end is found by the
// tool and not by someone reading the log. Definitions and thresholds: docs/geninv.md; contract: openspec/specs/soak-gates.
//
// Clock: the minute lines (one per virtual minute on d2d, one per real minute on the original game) are the time axis;
// an event's virtual time is interpolated between the minute lines around its wall timestamp. Pace lines are used only
// for the hang check.

// Invariant codes in Violated (stable: scripts and the spec name them).
const (
	InvHang     = "HANG"
	InvStall    = "STALL"
	InvWedge    = "WEDGE"
	InvProgress = "PROGRESS"
	InvDeaths   = "DEATHS"
	InvRestarts = "RESTARTS"
	InvExit     = "EXIT"
)

// Verdict words of the invariants (not the fidelity verdict: that stays PASS/FAIL/INCONCLUSIVE/NO-DATA).
const (
	InvPass     = "PASS"
	InvViolated = "VIOLATED"
	InvNoData   = "NO-DATA"
)

// InvLimits are the thresholds. Provenance tags (docs/geninv.md): M measured from logs, D derived from a spec, J judged.
type InvLimits struct {
	StallWarnMin     int     // D: bot.stallWarnMin (bot-loop spec): a gap this long makes the ladder fire
	ProgressMin      float64 // D: same number: the longest honest no-progress stretch must stay under the ladder's first window
	WedgeMin         float64 // M: longest span of one "no route"/"no eligible area" storm with no non-town Entered; honest waits stayed at 3 min over 9 logs
	DeathsPer1000    float64 // D: death-rule FAIL-3 (15 per 1000 kills) ...
	DeathsMinKills   int     // D: ... evaluated from 150 kills
	StreakDeaths     int     // D: death-rule FAIL-1: 3 deaths in 30 min without a clvl gain
	StreakMinutes    int
	StreakAny        int // D: more than 4 deaths in 60 min fails whatever the level gain (5 or more)
	StreakAnyMinutes int
	RecoverMin       float64 // J: a death must be followed by progress (kill, area, quest, spend) within this many virtual minutes
	EarlyFrac        float64 // D: soakfix: a run under 95 % of the requested length ended early
}

// DefaultInvLimits is the committed threshold set.
func DefaultInvLimits() InvLimits {
	return InvLimits{StallWarnMin: 30, ProgressMin: 30, WedgeMin: 15, DeathsPer1000: 15, DeathsMinKills: 150,
		StreakDeaths: 3, StreakMinutes: 30, StreakAny: 5, StreakAnyMinutes: 60, RecoverMin: 10, EarlyFrac: 0.95}
}

// ExitInfo is what the caller knows about how the run ended, beyond the log (run.json of d2dsoak run, supervisor.log).
type ExitInfo struct {
	WantHours float64 // requested virtual length (0 = unknown)
	End       string  // "hours reached", "cmd exited", "wall limit", "fake", or "" (unknown)
	CmdErr    string
	// Supervisor log (the fleet supervisor of a hidden soak): launches, and how the last bot ended.
	Launches   int
	SupEnded   bool
	SupCode    int
	SupReason  string
	SupPresent bool
}

// Invariants is the counter block of one seed.
type Invariants struct {
	Seed    int     `json:"seed,omitempty"`
	Minutes int     `json:"minutes"`    // minute lines (the clock)
	VirtMin float64 `json:"virtualMin"` // run length by that clock
	Engine  string  `json:"engine,omitempty"`
	Acts    []int   `json:"acts,omitempty"`
	ClvlMin int     `json:"clvlMin,omitempty"`
	ClvlMax int     `json:"clvlMax,omitempty"`

	PaceFrozen    int `json:"paceFrozen"`    // HANG: pace intervals with wall time passing and no virtual advance
	CombatHolding int `json:"combatHolding"` // HANG: "combat no progress, holding" (the combat watchdog fired)

	StallWarns int `json:"stallWarns"` // STALL: ladder first rung (fresh game)
	StallExits int `json:"stallExits"` // STALL: ladder second rung (exit status 5)

	WedgeLines    int     `json:"wedgeLines"`    // WEDGE: "no route from level" + "no eligible reachable leveling area" lines
	WedgeMin      float64 `json:"wedgeMin"`      // WEDGE: longest storm span (virtual minutes)
	EnteredGapMin float64 `json:"enteredGapMin"` // longest stretch without a non-town Entered (info)

	NoProgressMin float64 `json:"noProgressMin"` // PROGRESS: longest stretch without kill/xp, non-town area, quest step, gold spent, clvl

	Deaths        int     `json:"deaths"`
	Kills         int     `json:"kills"`
	DeathsPer1000 float64 `json:"deathsPer1000"`
	DeathStreak   bool    `json:"deathStreak"`
	Unrecovered   int     `json:"unrecovered"` // deaths with no progress within RecoverMin virtual minutes (judged only when the log goes on that long)

	FreshGames  int `json:"freshGames"`  // bot-started games (minute `restarts`, summed increments): reported, not gated
	Chickens    int `json:"chickens"`    // Save and Exit requested (chicken / leave game): reported, not gated
	BotRestarts int `json:"botRestarts"` // d2bot starts beyond the first, plus supervisor launches beyond the first
	Panics      int `json:"panics"`      // recovered panics (EXIT: a run with a panic did not end clean)

	Exit     string `json:"exit"` // limit | open | early | crash
	ExitNote string `json:"exitNote,omitempty"`

	Violated []string `json:"violated,omitempty"`
	Verdict  string   `json:"verdict"`
}

// townAreas are the town names of the five acts (an Entered of one is not campaign progress).
var townAreas = map[string]bool{"Rogue Encampment": true, "Lut Gholein": true, "Kurast Docks": true, "Pandemonium Fortress": true, "Harrogath": true}

// ActOfArea maps a level id to its act (1 to 5), 0 when unknown (docs/d2-1.14d-memory.md level table).
func ActOfArea(id int) int {
	switch {
	case id >= 1 && id <= 39:
		return 1
	case id >= 40 && id <= 74:
		return 2
	case id >= 75 && id <= 102:
		return 3
	case id >= 103 && id <= 108:
		return 4
	case id >= 109 && id <= 132:
		return 5
	}
	return 0
}

type evKind int

const (
	evProgress evKind = iota // progress event (kill, quest, ...)
	evEntered                // non-town area entered (also progress)
	evStorm
)

type invEvent struct {
	at   time.Time
	kind evKind
}

// ScanInvariants reads a bot log (slog JSON or text) in one streaming pass and returns the counter block. A log of
// hundreds of MB is fine: only minute lines and a few event times are kept.
func ScanInvariants(r io.Reader, x ExitInfo, lim InvLimits) (Invariants, error) {
	var (
		inv      Invariants
		mins     []Minute
		events   []invEvent
		pacePrev struct {
			ok   bool
			at   time.Time
			virt float64
		}
		exitCode = -1
	)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	botStarts := 0
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "exit ") { // appended by the soak wrapper script after the bot returned
			if n, err := strconv.Atoi(strings.TrimSpace(line[5:])); err == nil {
				exitCode = n
			}
			continue
		}
		storm := strings.Contains(line, "no route from level") || strings.Contains(line, "no eligible reachable leveling area")
		msg := lineMsg(line)
		if !storm && msg == "" {
			continue
		}
		var kv map[string]string
		get := func() map[string]string {
			if kv == nil {
				switch {
				case strings.HasPrefix(line, "{"):
					kv = jsonKV(line)
				case strings.Contains(line, "msg="):
					kv = textKV(line)
				}
				if kv == nil {
					kv = map[string]string{}
				}
			}
			return kv
		}
		at, _ := time.Parse(time.RFC3339Nano, lineTime(line, get))
		if storm {
			inv.WedgeLines++
			events = append(events, invEvent{at, evStorm})
		}
		switch {
		case msg == "minute":
			if m, ok := minuteFromKV(get()); ok {
				mins = append(mins, m)
			}
		case msg == "pace":
			k := get()
			v, e1 := strconv.ParseFloat(k["virtual"], 64)
			if e1 == nil && !at.IsZero() {
				if pacePrev.ok && v <= pacePrev.virt && at.After(pacePrev.at) {
					inv.PaceFrozen++
				}
				pacePrev.ok, pacePrev.at, pacePrev.virt = true, at, v
			}
		case msg == "d2d backend up":
			inv.Engine = "d2d"
		case msg == "d2bot start":
			botStarts++
		case strings.HasPrefix(msg, "Entered "):
			if !townAreas[strings.TrimPrefix(msg, "Entered ")] && !at.IsZero() {
				events = append(events, invEvent{at, evEntered})
			}
		case strings.HasPrefix(msg, "Quest done"):
			if !at.IsZero() {
				events = append(events, invEvent{at, evProgress})
			}
		case strings.HasPrefix(msg, "campaign stalled: no progress"):
			inv.StallWarns++
		case msg == "campaign stalled, exiting for a restart":
			inv.StallExits++
		case msg == "combat no progress, holding":
			inv.CombatHolding++
		case strings.Contains(msg, "Save and Exit requested (chicken"):
			inv.Chickens++
		case strings.HasPrefix(msg, "panic recovered"), strings.HasPrefix(msg, "panic in the panic handler"):
			inv.Panics++
		}
	}
	if err := sc.Err(); err != nil {
		return inv, err
	}
	inv.Minutes = len(mins)
	if botStarts > 1 {
		inv.BotRestarts = botStarts - 1
	}
	if x.Launches > 1 {
		inv.BotRestarts += x.Launches - 1
	}
	if len(mins) < 2 {
		inv.Exit, inv.Verdict = "open", InvNoData
		inv.ExitNote = fmt.Sprintf("%d minute line(s), need 2", len(mins))
		return inv, nil
	}
	times := make([]time.Time, len(mins))
	for i, m := range mins {
		times[i] = m.At
	}
	vt := func(t time.Time) float64 { // virtual minutes since the first minute line
		i := sort.Search(len(times), func(i int) bool { return times[i].After(t) })
		switch {
		case i == 0:
			return 0
		case i == len(times):
			return float64(len(times) - 1)
		}
		span := times[i].Sub(times[i-1]).Seconds()
		if span <= 0 {
			return float64(i - 1)
		}
		return float64(i-1) + t.Sub(times[i-1]).Seconds()/span
	}
	end := float64(len(mins) - 1)
	inv.VirtMin = end
	inv.ClvlMin, inv.ClvlMax = mins[0].Clvl, mins[0].Clvl
	acts := map[int]bool{}
	var progress []float64 // virtual minutes of progress events, sorted below
	var deaths []deathAt
	for i, m := range mins {
		inv.ClvlMin, inv.ClvlMax = min(inv.ClvlMin, m.Clvl), max(inv.ClvlMax, m.Clvl)
		if a := ActOfArea(m.Area); a > 0 {
			acts[a] = true
		}
		if i == 0 {
			continue
		}
		p := mins[i-1]
		if d := m.Kills - p.Kills; d > 0 {
			inv.Kills += d
		}
		if d := m.Deaths - p.Deaths; d > 0 {
			inv.Deaths += d
			for ; d > 0; d-- {
				deaths = append(deaths, deathAt{float64(i), m.Clvl})
			}
		}
		if d := m.Restarts - p.Restarts; d > 0 {
			inv.FreshGames += d
		}
		if m.Kills > p.Kills || m.XP > p.XP || m.Gold < p.Gold || m.Clvl > p.Clvl {
			progress = append(progress, float64(i))
		}
	}
	for a := range acts {
		inv.Acts = append(inv.Acts, a)
	}
	sort.Ints(inv.Acts)
	var entered []float64
	var storms []float64
	for _, e := range events {
		v := vt(e.at)
		switch e.kind {
		case evEntered:
			entered = append(entered, v)
			progress = append(progress, v)
		case evProgress:
			progress = append(progress, v)
		case evStorm:
			storms = append(storms, v)
		}
	}
	sort.Float64s(progress)
	sort.Float64s(entered)
	inv.NoProgressMin = longestGap(progress, 0, end)
	inv.EnteredGapMin = longestGap(entered, 0, end)
	inv.WedgeMin = stormStretch(storms, entered)
	if inv.Kills > 0 {
		inv.DeathsPer1000 = 1000 * float64(inv.Deaths) / float64(inv.Kills)
	}
	inv.DeathStreak = deathStreak(deaths, lim)
	for _, d := range deaths {
		if d.at+lim.RecoverMin > end {
			continue // the log ends before the window closes: not judged
		}
		k := sort.SearchFloat64s(progress, d.at+1e-9)
		if k >= len(progress) || progress[k] > d.at+lim.RecoverMin {
			inv.Unrecovered++
		}
	}
	inv.Exit, inv.ExitNote = exitKind(&inv, x, exitCode, lim)
	violations(&inv, lim)
	return inv, nil
}

type deathAt struct {
	at   float64
	clvl int
}

// stormMaxGap is the largest pause between two lines of one storm (virtual minutes). Honest timer waits stay far under it.
const stormMaxGap = 5.0

// stormStretch is the longest virtual-minute span of one storm: consecutive storm lines (no pause over stormMaxGap) with no
// non-town Entered between them. A transient wait ends when the bot enters an area; a wedge never does.
func stormStretch(storms, entered []float64) float64 {
	best := 0.0
	for i := 0; i < len(storms); {
		j := i
		for j+1 < len(storms) && storms[j+1]-storms[j] <= stormMaxGap {
			k := sort.SearchFloat64s(entered, storms[j]+1e-9)
			if k < len(entered) && entered[k] < storms[j+1] {
				break
			}
			j++
		}
		best = max(best, storms[j]-storms[i])
		i = j + 1
	}
	return best
}

// longestGap is the longest distance between consecutive sorted points, with the run start and end as bounds.
func longestGap(pts []float64, start, end float64) float64 {
	prev, best := start, 0.0
	for _, p := range pts {
		best = max(best, p-prev)
		prev = max(prev, p)
	}
	return max(best, end-prev)
}

// deathStreak applies death-rule FAIL-1 at minute resolution: 3 deaths within 30 minutes without a clvl change, or 5 within 60.
func deathStreak(ds []deathAt, lim InvLimits) bool {
	for i := range ds {
		n30, n60 := 0, 0
		lvl := map[int]bool{}
		for j := i; j < len(ds) && ds[j].at-ds[i].at <= float64(lim.StreakAnyMinutes); j++ {
			n60++
			if ds[j].at-ds[i].at <= float64(lim.StreakMinutes) {
				n30++
				lvl[ds[j].clvl] = true
			}
		}
		if n60 >= lim.StreakAny || (n30 >= lim.StreakDeaths && len(lvl) == 1) {
			return true
		}
	}
	return false
}

func exitKind(inv *Invariants, x ExitInfo, code int, lim InvLimits) (kind, note string) {
	short := func() bool { return x.WantHours > 0 && inv.VirtMin < lim.EarlyFrac*x.WantHours*60 }
	switch {
	case x.CmdErr != "":
		return "crash", "command failed: " + x.CmdErr
	case x.SupPresent && x.SupEnded && x.SupCode != 0:
		return "crash", fmt.Sprintf("bot exit %d (%s)", x.SupCode, x.SupReason)
	case x.SupPresent && x.SupEnded && !strings.Contains(x.SupReason, "time limit"):
		return "early", "supervisor: " + x.SupReason
	case code > 0:
		return "crash", fmt.Sprintf("exit %d", code)
	case x.End == "cmd exited" || x.End == "wall limit":
		if short() || x.WantHours == 0 {
			return "early", x.End
		}
	}
	if short() {
		return "early", fmt.Sprintf("%.2f of %.2f virtual h", inv.VirtMin/60, x.WantHours)
	}
	if code == 0 || x.End == "hours reached" || x.End == "fake" || x.End == "cmd exited" || x.End == "wall limit" || (x.SupPresent && x.SupEnded) {
		return "limit", ""
	}
	return "open", "no end marker: the run may still be going"
}

// violations fills Violated and Verdict from the counters.
func violations(inv *Invariants, lim InvLimits) {
	add := func(cond bool, code string) {
		if cond {
			inv.Violated = append(inv.Violated, code)
		}
	}
	add(inv.PaceFrozen > 0 || inv.CombatHolding > 0, InvHang)
	add(inv.StallWarns > 0 || inv.StallExits > 0, InvStall)
	add(inv.WedgeMin > lim.WedgeMin, InvWedge)
	add(inv.NoProgressMin > lim.ProgressMin, InvProgress)
	add(inv.DeathStreak || inv.Unrecovered > 0 || (inv.Kills >= lim.DeathsMinKills && inv.DeathsPer1000 > lim.DeathsPer1000), InvDeaths)
	add(inv.BotRestarts > 0, InvRestarts)
	add(inv.Exit == "early" || inv.Exit == "crash" || inv.Panics > 0, InvExit)
	inv.Verdict = InvPass
	if len(inv.Violated) > 0 {
		inv.Verdict = InvViolated
	}
}

// ScanSupervisor reads a supervisor.log (`launch attempt N`, `bot ended (exit code C, REASON)`) into x.
func ScanSupervisor(r io.Reader, x *ExitInfo) error {
	sc := bufio.NewScanner(r)
	x.SupPresent = true
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.Contains(line, "launch attempt"):
			x.Launches++
		case strings.Contains(line, "bot ended (exit code "):
			rest := line[strings.Index(line, "bot ended (exit code ")+len("bot ended (exit code "):]
			num, reason, _ := strings.Cut(rest, ")")
			c, _, _ := strings.Cut(num, ",")
			x.SupCode, _ = strconv.Atoi(strings.TrimSpace(c))
			x.SupReason = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(num[min(len(c), len(num)):], ","), " "))
			if x.SupReason == "" {
				x.SupReason = strings.TrimSpace(reason)
			}
			x.SupEnded = true
		}
	}
	return sc.Err()
}

// lineMsg extracts the msg value of a JSON log line without a full parse; for a text line it returns the msg value
// of `msg=...`; "" when there is none.
func lineMsg(line string) string {
	if i := strings.Index(line, `"msg":"`); i >= 0 {
		s := line[i+7:]
		for j := 0; j < len(s); j++ {
			if s[j] == '\\' {
				j++
			} else if s[j] == '"' {
				return s[:j]
			}
		}
		return ""
	}
	if i := strings.Index(line, "msg="); i >= 0 && !strings.HasPrefix(line, "{") {
		return textKV(line[i:])["msg"]
	}
	return ""
}

func lineTime(line string, get func() map[string]string) string {
	if i := strings.Index(line, `"time":"`); i >= 0 {
		s := line[i+8:]
		if j := strings.IndexByte(s, '"'); j >= 0 {
			return s[:j]
		}
	}
	return get()["time"]
}
