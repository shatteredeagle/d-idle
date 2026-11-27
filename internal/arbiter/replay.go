package arbiter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Replay feeds a bot log (JSON lines, slog format) to the shadow offline and reports what the arbiter would have done.
//
// Accepted lines (everything else is skipped):
//   - msg "minute": area, kills, xp, potions, gold, stashGold become facts; progress is kills + xp.
//   - msg "tick": a compact line written by a fixture or by the shadow: "area" (name), "kills", "xp", "chain" (the hook
//     family the chain acted for), "objectiveArea", and "facts" (a map of the Fact* names).
//
// With follow set, the simulated hero obeys the arbiter instead of the log: its area is whatever the arbiter last
// sent it to, facts and progress still come from the log (so a chore that the log never satisfied stays wanted). An
// intent that has run in place for its EstCost with its reason still holding is reported as failed, which backs it off.
// Without follow the arbiter sees the logged hero and only the divergence lines and counters are meaningful.
//
// A tick that carries no "facts" inherits the last facts seen. Time comes from "vtMs" (the bot clock in Unix
// milliseconds, which is the game clock under lockstep) when present, else from the "time" field (RFC 3339).
//
// The bot's own tick lines (knob bot.tickLogSec, docs/design/ticklog-t4.md) carry the full fact set once a second; when a
// log has any tick line its minute lines are skipped, because their area is a number and their counters lag a minute.
type ReplayResult struct {
	Ticks        int
	AreaChanges  int      // changes of the logged area: what the chain did
	FollowMoves  int      // Follow mode: area changes the arbiter itself ordered while the simulated hero obeyed it
	Refused      int      // travel budget refusals
	Divergences  []string // the shadow's divergence lines (rate-limited)
	Trace        []string // one Decision.String per tick, the golden trace
	FamilyCounts string
	Points       []SessionPoint // one per tick: the facts, the chain's family and the arbiter's (ScoreSession)
}

type logLine struct {
	Time          string          `json:"time"`
	Msg           string          `json:"msg"`
	Area          json.RawMessage `json:"area"`
	Kills         int             `json:"kills"`
	XP            int             `json:"xp"`
	Potions       int             `json:"potions"`
	Gold          int             `json:"gold"`
	StashGold     int             `json:"stashGold"`
	Chain         string          `json:"chain"`
	ObjectiveArea string          `json:"objectiveArea"`
	Facts         map[string]int  `json:"facts"`
	VTMs          int64           `json:"vtMs"`
}

func areaName(raw json.RawMessage) string {
	return strings.Trim(string(raw), `"`)
}

func Replay(in io.Reader, cfg Config, sc ShadowConfig, follow bool) (ReplayResult, error) {
	var res ReplayResult
	sh := NewShadow(cfg, sc, StdSources()...)
	sc2 := bufio.NewScanner(in)
	sc2.Buffer(make([]byte, 1<<20), 1<<20)
	var t0 time.Time
	facts := map[string]int{}
	prevArea, simArea := "", ""
	var run *Intent
	var runSince float64
	var lines []logLine
	ticks := false
	for sc2.Scan() {
		var l logLine
		if json.Unmarshal(sc2.Bytes(), &l) != nil || (l.Msg != "minute" && l.Msg != "tick") {
			continue
		}
		ticks = ticks || l.Msg == "tick"
		lines = append(lines, l)
	}
	if err := sc2.Err(); err != nil {
		return res, err
	}
	for _, l := range lines {
		if ticks && l.Msg == "minute" {
			continue
		}
		ts, err := time.Parse(time.RFC3339, l.Time)
		if l.VTMs > 0 {
			ts, err = time.UnixMilli(l.VTMs), nil
		}
		if err != nil {
			return res, fmt.Errorf("bad time %q", l.Time)
		}
		if t0.IsZero() {
			t0 = ts
		}
		if l.Msg == "minute" {
			facts[FactPotions], facts[FactPackGold], facts[FactStashGold] = l.Potions, l.Gold, l.StashGold
		}
		for k, v := range l.Facts {
			facts[k] = v
		}
		area := areaName(l.Area)
		if prevArea != "" && areaName(l.Area) != prevArea {
			res.AreaChanges++
		}
		prevArea = areaName(l.Area)
		if simArea == "" {
			simArea = area
		}
		if follow {
			area = simArea
		}
		cp := make(map[string]int, len(facts))
		for k, v := range facts {
			cp[k] = v
		}
		v := View{Now: ts.Sub(t0).Seconds(), Area: area, Progress: uint64(l.Kills + l.XP), Facts: cp, ObjectiveArea: l.ObjectiveArea}
		line := sh.Tick(v, l.Chain)
		if line != "" {
			res.Divergences = append(res.Divergences, fmt.Sprintf("t=%05.0f %s", v.Now, line))
		}
		d, _ := sh.Last()
		mine := ""
		if d.Chosen != nil {
			mine = Family(d.Chosen.Source)
		}
		res.Points = append(res.Points, SessionPoint{Now: v.Now, Facts: cp, Chain: l.Chain, Arbiter: mine})
		res.Ticks++
		res.Trace = append(res.Trace, d.String())
		if follow && d.Chosen != nil {
			c := *d.Chosen
			switch {
			case c.NeedsArea != Any && c.NeedsArea != simArea:
				simArea = c.NeedsArea
				res.FollowMoves++
				run = nil
			case run == nil || run.Key() != c.Key():
				run, runSince = &c, v.Now
			case v.Now-runSince >= c.EstCost:
				sh.Report(c, v.Now, false)
				run = nil
			}
		}
		for _, r := range d.Rejected {
			if strings.HasPrefix(r.Why, "travel budget") {
				res.Refused++
			}
		}
	}
	res.FamilyCounts = sh.Drain()
	return res, nil
}
