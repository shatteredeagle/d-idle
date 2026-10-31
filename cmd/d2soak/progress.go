package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/stepgroup"
	"d2lod-bot/internal/telemetry"
)

// Progress measurement from the bot's minute lines (docs/plan/plan-v2.md M0, docs/soak-analysis.md section 10). The
// judgement uses only what the character really has (clvl, gold, quest step, xp), never the bot's own walking
// progress counter, which ticks while the bot idles.

// maxSampleGap is the longest time one minute line stands for. A longer hole in the log is the bot being down (other
// codes judge that); it resets the no-gain clock and is not counted as time in a state.
const maxSampleGap = 5 * time.Minute

// minLoopXP is the xp a loop stretch must gain to count as a gain: kills that only trickle xp (a few points per minute
// from xpRisk noise) do not break a loop.
const minLoopXP = 200

// paceFloor is the clvl floor per elapsed hour (docs/sme/pace-tables.md 5.2, cells are U-derived: re-derive them from
// the first fresh-character run).
type paceCell struct {
	Hours, Clvl int
}

var paceFloor = []paceCell{{2, 7}, {4, 11}, {8, 16}, {12, 20}, {24, 26}}

var groupOrder = stepgroup.Order

func normStep(s string) string { return stepgroup.Norm(s) }

func stepGroup(step string) (group string, known bool) { return stepgroup.Of(step) }

// questName is the quest step inside "Quest: <kind> (<name>)", "" for any other step and for the level-up filler,
// which is farming and not a quest step.
func questName(step string) string {
	if !strings.HasPrefix(step, "Quest:") {
		return ""
	}
	i, j := strings.Index(step, "("), strings.LastIndex(step, ")")
	if i < 0 || j <= i {
		return ""
	}
	n := normStep(step[i+1 : j])
	if strings.HasPrefix(n, "Level up") {
		return ""
	}
	return n
}

// DefaultNoGainMinXP is the xp floor of the no_gain xp term. A character levelling at clvl 13 spends over an hour per
// level (G1 attempt 16: 11k XP/h, 28k XP span), so xp is progress; the minute lines carry no xpNext, the span comes
// from the experience table.
const DefaultNoGainMinXP = 500

// xpProgressThreshold is the xp gain since the last progress point that counts as progress at this clvl.
func xpProgressThreshold(clvl, minXP int) uint64 {
	if minXP == 0 {
		minXP = DefaultNoGainMinXP
	}
	thr := uint64(minXP)
	lo, ok1 := telemetry.DefaultExpForLevel(clvl)
	hi, ok2 := telemetry.DefaultExpForLevel(clvl + 1)
	if ok1 && ok2 && hi > lo {
		thr = max(thr, uint64(hi-lo)/100)
	}
	return thr
}

// ProgressLimits are the thresholds of the minute-line checks (a zero disables the check).
type ProgressLimits struct {
	NoGain         time.Duration // no clvl gain, quest-step change, gold above its mark or xp gain for this long: fail
	NoGainMinXP    int           // xp that counts as progress for no_gain: max(this, 1% of the level's xp span); 0 = default, < 0 = xp does not count
	LoopRepeats    int           // the same step in the same area entered this many times without a gain: fail
	IdleInTown     time.Duration // one unbroken waiting state this long: fail
	StateShare     float64       // a state group above this share of an hour: warn
	FailBelowFloor bool          // below_floor_clvl fails instead of warning
}

func DefaultProgressLimits() ProgressLimits {
	return ProgressLimits{NoGain: 90 * time.Minute, NoGainMinXP: DefaultNoGainMinXP, LoopRepeats: 20, IdleInTown: 10 * time.Minute, StateShare: 0.4}
}

// ProgressReport is what AnalyzeMinutes found: findings plus the tables the text report prints.
type ProgressReport struct {
	Findings   []Finding
	Checkpoint []Checkpoint
	Bands      []Band
	Groups     map[string]time.Duration
	Total      time.Duration
	Unmapped   []string
	Dominant   []string // "group hour N: P%" lines behind the state_dominant warnings
}

type Checkpoint struct {
	Hour, Clvl, Floor int
	Judged            bool // a table hour (2, 4, 8, 12, 24) the run reached
}

type Band struct {
	Lo, Hi  int
	XP      uint64
	Time    time.Duration
	PerHour float64
}

// clvlBand: 1-5, then four levels each (6-9, 10-13, ...), 26 and up.
func clvlBand(c int) (lo, hi int) {
	switch {
	case c < 6:
		return 1, 5
	case c >= 26:
		return 26, 99
	}
	lo = 6 + (c-6)/4*4
	return lo, lo + 3
}

// AnalyzeMinutes judges a run from its minute lines (file order).
func AnalyzeMinutes(ms []Minute, lim ProgressLimits) ProgressReport {
	pr := ProgressReport{Groups: map[string]time.Duration{}}
	if len(ms) < 2 {
		return pr
	}
	add := func(active bool, level, code, f string, a ...any) {
		pr.Findings = append(pr.Findings, Finding{Level: level, Code: code, Msg: fmt.Sprintf(f, a...), Active: active})
	}
	t0 := ms[0].At
	at := func(m Minute) string { return m.At.Format("15:04") }

	// Time in state, XP per clvl band.
	unmapped := map[string]bool{}
	hourGroup := map[int]map[string]time.Duration{}
	hourTotal := map[int]time.Duration{}
	bands := map[int]*Band{}
	for i := 0; i+1 < len(ms); i++ {
		a, b := ms[i], ms[i+1]
		d := b.At.Sub(a.At)
		if d <= 0 || d > maxSampleGap {
			continue
		}
		g, known := stepGroup(a.Step)
		if !known {
			unmapped[normStep(a.Step)] = true
		}
		pr.Groups[g] += d
		pr.Total += d
		h := int(a.At.Sub(t0) / time.Hour)
		if hourGroup[h] == nil {
			hourGroup[h] = map[string]time.Duration{}
		}
		hourGroup[h][g] += d
		hourTotal[h] += d
		lo, hi := clvlBand(a.Clvl)
		bd := bands[lo]
		if bd == nil {
			bd = &Band{Lo: lo, Hi: hi}
			bands[lo] = bd
		}
		bd.Time += d
		if b.XP > a.XP && b.Clvl >= a.Clvl {
			bd.XP += b.XP - a.XP
		}
	}
	for s := range unmapped {
		pr.Unmapped = append(pr.Unmapped, s)
	}
	sort.Strings(pr.Unmapped)
	for _, bd := range bands {
		if bd.Time > 0 {
			bd.PerHour = float64(bd.XP) / bd.Time.Hours()
		}
		pr.Bands = append(pr.Bands, *bd)
	}
	sort.Slice(pr.Bands, func(i, j int) bool { return pr.Bands[i].Lo < pr.Bands[j].Lo })
	if lim.StateShare > 0 {
		var hs []int
		for h := range hourTotal {
			hs = append(hs, h)
		}
		sort.Ints(hs)
		lastJudged, domNow := -1, false // the newest hour with enough data stands for "now"
		for _, h := range hs {
			if hourTotal[h] >= 30*time.Minute {
				lastJudged = h
			}
		}
		for _, h := range hs {
			if hourTotal[h] < 30*time.Minute { // a short tail says little
				continue
			}
			for _, g := range groupOrder {
				if s := float64(hourGroup[h][g]) / float64(hourTotal[h]); s > lim.StateShare {
					line := fmt.Sprintf("%s hour %d: %.0f%%", g, h+1, s*100)
					pr.Dominant = append(pr.Dominant, line)
					domNow = domNow || h == lastJudged
				}
			}
		}
		if len(pr.Dominant) > 0 {
			add(domNow, "warn", "state_dominant", "state group above %.0f%% of an hour: %s (farming is expected to dominate while leveling; the others are time not spent progressing)", lim.StateShare*100, strings.Join(pr.Dominant, "; "))
		}
	}

	// no_gain and loop_no_gain share one progress definition; loops also accept real xp.
	var (
		lastProg   = t0
		clvl       = ms[0].Clvl
		goldMark   = ms[0].Gold
		xpMark     = ms[0].XP
		quest      = questName(ms[0].Step)
		worst      time.Duration
		worstStart time.Time
		loopXP     = ms[0].XP
		loopCount  = map[string]int{}
		loopFired  = false
		loopIdx    = -1 // the newest loop_no_gain finding, active when its loop is still unbroken at the end
		prevKey    string
	)
	for i, m := range ms {
		gap := i > 0 && m.At.Sub(ms[i-1].At) > 2*maxSampleGap
		if gap {
			lastProg, clvl, goldMark, xpMark, quest = m.At, m.Clvl, m.Gold, m.XP, questName(m.Step)
			loopCount, loopXP, loopFired, prevKey = map[string]int{}, m.XP, false, ""
			continue
		}
		prog := false
		if m.Clvl > clvl {
			prog = true
		}
		clvl = m.Clvl
		if q := questName(m.Step); q != "" && q != quest {
			prog = true
			quest = q
		}
		if m.Gold > goldMark {
			prog = true
			goldMark = m.Gold
		}
		if m.XP < xpMark { // a death took xp: only a net regain above the new low counts
			xpMark = m.XP
		} else if lim.NoGainMinXP >= 0 && m.XP-xpMark >= xpProgressThreshold(m.Clvl, lim.NoGainMinXP) {
			prog = true
		}
		if prog {
			xpMark = m.XP
			if d := m.At.Sub(lastProg); d > worst {
				worst, worstStart = d, lastProg
			}
			lastProg = m.At
		}
		if prog || m.XP >= loopXP+minLoopXP { // a loop is broken by any real gain, xp included
			loopCount, loopXP, loopFired, prevKey = map[string]int{}, m.XP, false, ""
		}
		if key := fmt.Sprintf("%d|%s", m.Area, normStep(m.Step)); key != prevKey {
			prevKey = key
			loopCount[key]++
			if lim.LoopRepeats > 0 && loopCount[key] >= lim.LoopRepeats && !loopFired {
				loopFired = true
				loopIdx = len(pr.Findings)
				add(false, "fail", "loop_no_gain", "%q in area %d entered %d times (by %s) with no clvl, quest-step, gold or xp (+%d) gain", normStep(m.Step), m.Area, loopCount[key], at(m), minLoopXP)
			}
		}
	}
	if loopFired && loopIdx >= 0 {
		pr.Findings[loopIdx].Active = true
	}
	cur := ms[len(ms)-1].At.Sub(lastProg)
	if cur > worst {
		worst, worstStart = cur, lastProg
	}
	if lim.NoGain > 0 && worst >= lim.NoGain {
		add(cur >= lim.NoGain, "fail", "no_gain", "no clvl gain, quest-step change, gold above its mark or xp gain (>= %d) for %s from %s (limit %s); planner dump needed", xpProgressThreshold(ms[len(ms)-1].Clvl, lim.NoGainMinXP), mins(worst), worstStart.Format("15:04"), mins(lim.NoGain))
	}

	// idle_in_town: one unbroken waiting state.
	if lim.IdleInTown > 0 {
		for i := 0; i < len(ms); i++ {
			if g, _ := stepGroup(ms[i].Step); g != "waiting" {
				continue
			}
			j := i
			for j+1 < len(ms) && func() bool { g, _ := stepGroup(ms[j+1].Step); return g == "waiting" }() {
				j++
			}
			end := ms[j].At
			if j+1 < len(ms) {
				end = ms[j+1].At
			}
			if d := end.Sub(ms[i].At); d >= lim.IdleInTown {
				add(j == len(ms)-1, "fail", "idle_in_town", "waiting for %s from %s (%q ... in area %d; limit %s)", mins(d), at(ms[i]), normStep(ms[i].Step), ms[i].Area, mins(lim.IdleInTown))
			}
			i = j
		}
	}

	// Checkpoints against the pace floor.
	start := ms[0].Clvl
	floorIdx := map[int]int{}
	for h := 1; ; h++ {
		cut := t0.Add(time.Duration(h) * time.Hour)
		if ms[len(ms)-1].At.Before(cut) {
			break
		}
		c := Checkpoint{Hour: h, Clvl: clvlAt(ms, cut)}
		for _, p := range paceFloor {
			if p.Hours == h {
				c.Judged, c.Floor = true, p.Clvl
				if h > 2 {
					c.Floor = max(c.Floor, start)
				}
			}
		}
		pr.Checkpoint = append(pr.Checkpoint, c)
		if c.Judged && c.Clvl < c.Floor {
			level := "warn"
			if lim.FailBelowFloor {
				level = "fail"
			}
			floorIdx[h] = len(pr.Findings)
			add(false, level, "below_floor_clvl", "clvl %d at hour %d, floor %d (docs/sme/pace-tables.md; the floor is max(start clvl %d, table) after hour 2)", c.Clvl, h, c.Floor, start)
		}
	}
	// Active while the newest judged checkpoint is still missed: catching up to its floor clears it.
	for k := len(pr.Checkpoint) - 1; k >= 0; k-- {
		if c := pr.Checkpoint[k]; c.Judged {
			if i, ok := floorIdx[c.Hour]; ok && ms[len(ms)-1].Clvl < c.Floor {
				pr.Findings[i].Active = true
			}
			break
		}
	}
	return pr
}

func clvlAt(ms []Minute, t time.Time) int {
	c := ms[0].Clvl
	for _, m := range ms {
		if m.At.After(t) {
			break
		}
		c = m.Clvl
	}
	return c
}

// Text prints the checkpoint, band and time-in-state tables.
func (pr ProgressReport) Text() string {
	if pr.Total == 0 && len(pr.Checkpoint) == 0 {
		return ""
	}
	var b strings.Builder
	for _, c := range pr.Checkpoint {
		note := ""
		if c.Judged {
			note = fmt.Sprintf("  floor %d: %s", c.Floor, map[bool]string{true: "BELOW", false: "ok"}[c.Clvl < c.Floor])
		}
		fmt.Fprintf(&b, "  clvl at %2d h  %d%s\n", c.Hour, c.Clvl, note)
	}
	for _, bd := range pr.Bands {
		hi := fmt.Sprint(bd.Hi)
		if bd.Hi == 99 {
			hi = "+"
		}
		fmt.Fprintf(&b, "  xp/h clvl %d-%s  %.0f XP/h (%d XP in %s)\n", bd.Lo, hi, bd.PerHour, bd.XP, mins(bd.Time))
	}
	if pr.Total > 0 {
		b.WriteString("  time in state")
		for _, g := range groupOrder {
			if d := pr.Groups[g]; d > 0 {
				fmt.Fprintf(&b, " %s %.0f%%", g, 100*float64(d)/float64(pr.Total))
			}
		}
		fmt.Fprintf(&b, " (%s of minute lines)\n", mins(pr.Total))
	}
	for _, u := range pr.Unmapped {
		fmt.Fprintf(&b, "  unmapped step text (group other): %q\n", u)
	}
	return b.String()
}
