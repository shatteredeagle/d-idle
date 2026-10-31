package sentinel

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Rule is one named condition with a duration, a cooldown, an attempt limit and an escalation.
//
//	Cond must hold continuously for For before the rule fires; after firing it waits Cooldown.
//	After MaxAttempts firings within Window without the condition clearing, Escalate fires once instead
//	(then ActAlert once per Window). A Ladder replaces Act/Escalate: step n fires Ladder[n], the next step
//	needs the condition to hold for StepWait more; the last step repeats at Window intervals.
//	The condition being false for Th.ResetAfter clears attempts and the ladder.
type Rule struct {
	Name        string
	Act         Action
	Escalate    Action
	Ladder      []Action
	For         time.Duration
	Cooldown    time.Duration
	MaxAttempts int
	Window      time.Duration
	Cond        func(c *Ctx) (bool, string)
	NeedsChar   bool // skipped for an instance without a configured character
}

// Ctx is what a condition sees.
type Ctx struct {
	O     *Observation
	T     Thresholds
	Inst  InstanceConf
	S     *InstState
	Now   time.Time
	Keep_ []string
}

// FrameStale is how long the frame hash has been unchanged (0 when unknown).
func (c *Ctx) FrameStale() time.Duration { return c.S.frameStale(c.Now) }

// CPUFlat is how long the CPU tick counter has been unchanged.
func (c *Ctx) CPUFlat() time.Duration { return c.S.cpuFlat(c.Now) }

// Idle is how long nothing observable changed (memory fingerprint, lock, bot heartbeat).
func (c *Ctx) Idle() time.Duration {
	if c.S.idleSince.IsZero() {
		return 0
	}
	return c.Now.Sub(c.S.idleSince)
}

// Keep reports whether the instance is protected from the idle reaper.
func (c *Ctx) Keep() bool {
	for _, k := range c.Keep_ {
		if k == c.Inst.ID {
			return true
		}
	}
	return false
}

// SinceProgress is how long without xp / gold / area / movement while in game.
func (c *Ctx) SinceProgress() time.Duration { return c.S.sinceProgress(c.Now) }

// GoalActive: a goal is running (bot says so) or the instance is configured to expect progress.
func (c *Ctx) GoalActive() bool {
	if c.O.Bot != nil && c.O.Bot.Reachable {
		return c.O.Bot.GoalActive
	}
	return c.Inst.ExpectProgress
}

const moveEps = 3 // subtiles that count as movement

// InstState are the engine's per-instance trackers and per-rule counters.
type InstState struct {
	frameHash  string
	frameSince time.Time
	ticks      uint64
	ticksSince time.Time

	progAnchorX, progAnchorY int
	progExp                  uint32
	progGold, progArea       int
	progSince                time.Time
	progValid                bool

	idleFP    string
	idleSince time.Time

	lastAction time.Time
	rules      map[string]*ruleState
}

type ruleState struct {
	since      time.Time // condition true since (zero = false)
	falseSince time.Time
	last       time.Time // last firing
	fired      []time.Time
	escalated  bool
	step       int
	alerted    time.Time
}

func (s *InstState) frameStale(now time.Time) time.Duration {
	if s.frameSince.IsZero() {
		return 0
	}
	return now.Sub(s.frameSince)
}

func (s *InstState) cpuFlat(now time.Time) time.Duration {
	if s.ticksSince.IsZero() {
		return 0
	}
	return now.Sub(s.ticksSince)
}

func (s *InstState) sinceProgress(now time.Time) time.Duration {
	if !s.progValid {
		return 0
	}
	return now.Sub(s.progSince)
}

func (s *InstState) observe(o *Observation) {
	now := o.Time
	// Idle fingerprint: anything a bot or an agent would change. A held game lock counts as activity.
	fp := fmt.Sprintf("%v|%s|%d|%d|%d|%d|%d|%d|%d|%v|%v|%v|%s", o.ProcAlive, o.State, o.X, o.Y, o.Area, o.Exp, o.Gold, o.Life, o.Mode, o.EscMenu, o.Panels, o.LockHeld, o.MenuScr)
	if o.Bot != nil && o.Bot.Reachable {
		fp += "|bot"
	}
	if fp != s.idleFP || s.idleSince.IsZero() {
		s.idleFP, s.idleSince = fp, now
	}
	if !o.ProcAlive {
		s.frameSince, s.ticksSince, s.frameHash, s.progValid = time.Time{}, time.Time{}, "", false
		return
	}
	if o.FrameOK {
		if o.FrameHash != s.frameHash || s.frameSince.IsZero() {
			s.frameHash, s.frameSince = o.FrameHash, now
		}
	}
	if o.CPUTicks != s.ticks || s.ticksSince.IsZero() {
		s.ticks, s.ticksSince = o.CPUTicks, now
	}
	if o.State != "ingame" || !o.MemOK {
		// Not playing: progress is not measured, and the clock restarts when play resumes.
		s.progValid = false
		return
	}
	moved := abs(o.X-s.progAnchorX)+abs(o.Y-s.progAnchorY) > moveEps
	if !s.progValid || moved || o.Exp != s.progExp || o.Gold != s.progGold || o.Area != s.progArea {
		s.progAnchorX, s.progAnchorY = o.X, o.Y
		s.progExp, s.progGold, s.progArea = o.Exp, o.Gold, o.Area
		s.progSince, s.progValid = now, true
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// Decision is a rule firing (the engine's output). Mode and Result are filled in by the runner.
type Decision struct {
	Time      time.Time `json:"time"`
	Instance  string    `json:"instance"`
	Rule      string    `json:"rule"`
	Action    Action    `json:"action"`
	Evidence  string    `json:"evidence"`
	Attempt   int       `json:"attempt"`
	Step      int       `json:"step,omitempty"` // ladder step (1-based) or 0
	Escalated bool      `json:"escalated,omitempty"`
}

// Engine is the pure rule evaluator. Not safe for concurrent use per instance; Step on different
// instances may run from one goroutine each if the maps are pre-created via State.
type Engine struct {
	Cfg   Config
	Rules []Rule
	// Disabled rule names (never fire, still tracked for status).
	Disabled map[string]bool

	insts map[string]*InstState
	byID  map[string]InstanceConf
}

func NewEngine(cfg Config) *Engine {
	e := &Engine{Cfg: cfg, Disabled: map[string]bool{}, insts: map[string]*InstState{}, byID: map[string]InstanceConf{}}
	e.Rules = DefaultRules(cfg.Th)
	for i := range e.Rules {
		ov, ok := cfg.Rules[e.Rules[i].Name]
		if !ok {
			continue
		}
		if ov.Enabled != nil && !*ov.Enabled {
			e.Disabled[e.Rules[i].Name] = true
		}
		if ov.For != nil {
			e.Rules[i].For = ov.For.D()
		}
		if ov.Cooldown != nil {
			e.Rules[i].Cooldown = ov.Cooldown.D()
		}
		if ov.MaxAttempts != nil {
			e.Rules[i].MaxAttempts = *ov.MaxAttempts
		}
	}
	for _, in := range cfg.Instances {
		e.byID[in.ID] = in
	}
	return e
}

// State returns (creating) the tracker of an instance.
func (e *Engine) State(id string) *InstState {
	s := e.insts[id]
	if s == nil {
		s = &InstState{rules: map[string]*ruleState{}}
		e.insts[id] = s
	}
	return s
}

// Step feeds one observation and returns the decision to act on (at most one per step: the first
// eligible rule in priority order). Conditions of lower rules keep their timers running.
func (e *Engine) Step(o Observation) []Decision {
	s := e.State(o.Instance)
	s.observe(&o)
	inst := e.byID[o.Instance]
	if inst.ID == "" {
		inst = InstanceConf{ID: o.Instance}
	}
	ctx := &Ctx{O: &o, T: e.Cfg.Th, Inst: inst, S: s, Now: o.Time, Keep_: e.Cfg.Keep}
	var out []Decision
	for i := range e.Rules {
		r := &e.Rules[i]
		rs := s.rules[r.Name]
		if rs == nil {
			rs = &ruleState{}
			s.rules[r.Name] = rs
		}
		ok, ev := r.Cond(ctx)
		if r.NeedsChar && inst.Character == "" {
			ok = false
		}
		if !ok {
			if !rs.since.IsZero() {
				rs.since = time.Time{}
				rs.falseSince = o.Time
			}
			if !rs.falseSince.IsZero() && o.Time.Sub(rs.falseSince) >= e.Cfg.Th.ResetAfter.D() {
				rs.fired, rs.escalated, rs.step, rs.falseSince = nil, false, 0, time.Time{}
			}
			continue
		}
		rs.falseSince = time.Time{}
		if rs.since.IsZero() {
			rs.since = o.Time
		}
		if len(out) > 0 || e.Disabled[r.Name] {
			continue
		}
		if o.Time.Sub(rs.since) < r.For {
			continue
		}
		if !rs.last.IsZero() && o.Time.Sub(rs.last) < r.Cooldown {
			continue
		}
		if !s.lastAction.IsZero() && o.Time.Sub(s.lastAction) < e.Cfg.Th.Settle.D() {
			continue
		}
		win := r.Window
		if win == 0 {
			win = 10 * time.Minute
		}
		kept := rs.fired[:0]
		for _, t := range rs.fired {
			if o.Time.Sub(t) < win {
				kept = append(kept, t)
			}
		}
		rs.fired = kept
		d := Decision{Time: o.Time, Instance: o.Instance, Rule: r.Name, Evidence: ev}
		switch {
		case len(r.Ladder) > 0:
			if rs.step >= len(r.Ladder) {
				// Past the last step: repeat only every Window.
				if !rs.alerted.IsZero() && o.Time.Sub(rs.alerted) < win {
					continue
				}
				rs.alerted = o.Time
				d.Action, d.Step = r.Ladder[len(r.Ladder)-1], len(r.Ladder)
			} else {
				d.Action, d.Step = r.Ladder[rs.step], rs.step+1
				rs.step++
			}
			// The Cooldown (= StepWait) before the next step is enforced above.
			rs.last = o.Time
			d.Attempt = d.Step
		case r.MaxAttempts > 0 && len(rs.fired) >= r.MaxAttempts:
			switch {
			case r.Escalate != "" && !rs.escalated:
				rs.escalated = true
				d.Action, d.Escalated = r.Escalate, true
			case !rs.alerted.IsZero() && o.Time.Sub(rs.alerted) < win:
				continue
			default:
				rs.alerted = o.Time
				d.Action, d.Escalated = ActAlert, true
				d.Evidence += fmt.Sprintf(" (attempt limit %d in %s reached)", r.MaxAttempts, win)
			}
			rs.fired = append(rs.fired, o.Time)
			d.Attempt = len(rs.fired)
			rs.last = o.Time
		default:
			d.Action = r.Act
			rs.fired = append(rs.fired, o.Time)
			d.Attempt = len(rs.fired)
			rs.last = o.Time
		}
		s.lastAction = o.Time
		out = append(out, d)
	}
	return out
}

// RuleStatus is the state of one rule for an instance, for the API.
type RuleStatus struct {
	Rule     string  `json:"rule"`
	Active   bool    `json:"active"`           // condition currently true
	ForSec   float64 `json:"forSec,omitempty"` // how long it has been true
	Attempts int     `json:"attempts"`
	Step     int     `json:"step,omitempty"`
}

// Status lists the rules whose condition is currently true (or that have attempts recorded).
func (e *Engine) Status(id string, now time.Time) []RuleStatus {
	s := e.insts[id]
	if s == nil {
		return nil
	}
	var out []RuleStatus
	for _, r := range e.Rules {
		rs := s.rules[r.Name]
		if rs == nil {
			continue
		}
		active := !rs.since.IsZero()
		if !active && len(rs.fired) == 0 {
			continue
		}
		st := RuleStatus{Rule: r.Name, Active: active, Attempts: len(rs.fired), Step: rs.step}
		if active {
			st.ForSec = now.Sub(rs.since).Seconds()
		}
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rule < out[j].Rule })
	return out
}

func joinInts(v []int) string {
	s := make([]string, len(v))
	for i, n := range v {
		s[i] = fmt.Sprint(n)
	}
	return strings.Join(s, ",")
}

// DefaultRules is the rule table, highest priority first. Durations come from the thresholds.
func DefaultRules(t Thresholds) []Rule {
	min10 := 10 * time.Minute
	return []Rule{
		{Name: "proc-gone", Act: ActRestartGame, Escalate: ActAlert, For: t.ProcGone.D(), Cooldown: 90 * time.Second, MaxAttempts: 3, Window: 15 * time.Minute,
			Cond: func(c *Ctx) (bool, string) {
				if !c.O.ProcAlive || c.O.ProcState == "Z" {
					return true, "no Game.exe process for the instance"
				}
				return false, ""
			}},
		{Name: "crash-dialog", Act: ActDismiss, Escalate: ActRestartGame, For: t.Dialog.D(), Cooldown: 8 * time.Second, MaxAttempts: 2, Window: min10,
			Cond: func(c *Ctx) (bool, string) {
				if len(c.O.Dialogs) > 0 {
					return true, "dialog: " + strings.Join(c.O.Dialogs, "; ")
				}
				return false, ""
			}},
		{Name: "helper-gone", Act: ActRestartHelp, Escalate: ActAlert, For: t.HelperGone.D(), Cooldown: 45 * time.Second, MaxAttempts: 3, Window: min10,
			Cond: func(c *Ctx) (bool, string) {
				if c.O.ProcAlive && !c.O.HelperUp {
					return true, "input helper not listening"
				}
				return false, ""
			}},
		{Name: "window-gone", Act: ActRestartGame, Escalate: ActAlert, For: t.WindowGone.D(), Cooldown: 3 * time.Minute, MaxAttempts: 2, Window: 20 * time.Minute,
			Cond: func(c *Ctx) (bool, string) {
				if c.O.ProcAlive && c.O.HelperUp && !c.O.HelperWindow {
					return true, "Game.exe alive but the helper sees no Diablo II window"
				}
				return false, ""
			}},
		{Name: "game-stopped", Act: ActCont, Escalate: ActRestartGame, For: t.Stopped.D(), Cooldown: 15 * time.Second, MaxAttempts: 2, Window: min10,
			Cond: func(c *Ctx) (bool, string) {
				if c.O.ProcAlive && c.O.ProcState == "T" {
					return true, "Game.exe is stopped (state T, SIGSTOP)"
				}
				return false, ""
			}},
		{Name: "game-hung", Act: ActRestartGame, Escalate: ActAlert, For: t.Hung.D(), Cooldown: 3 * time.Minute, MaxAttempts: 2, Window: 20 * time.Minute,
			Cond: func(c *Ctx) (bool, string) {
				if c.O.ProcAlive && c.O.ProcState != "T" && c.O.Hung {
					return true, "IsHungAppWindow: the game window does not answer"
				}
				return false, ""
			}},
		{Name: "game-frozen", Act: ActRestartGame, Escalate: ActAlert, For: 0, Cooldown: 3 * time.Minute, MaxAttempts: 2, Window: 20 * time.Minute,
			Cond: func(c *Ctx) (bool, string) {
				if !c.O.ProcAlive || c.O.ProcState == "T" {
					return false, ""
				}
				q := t.FrozenQuiet.D()
				if c.FrameStale() >= q && c.CPUFlat() >= q {
					return true, fmt.Sprintf("frame unchanged %s and CPU ticks flat %s", c.FrameStale().Round(time.Second), c.CPUFlat().Round(time.Second))
				}
				return false, ""
			}},
		{Name: "death-overlay", Act: ActRespawn, Escalate: ActRestartGame, For: t.DeathOverlay.D(), Cooldown: 12 * time.Second, MaxAttempts: 3, Window: min10,
			Cond: func(c *Ctx) (bool, string) {
				if c.O.MemOK && c.O.Dead() {
					return true, fmt.Sprintf("player dead: mode=%d life=%d/%d area=%d", c.O.Mode, c.O.Life, c.O.MaxLife, c.O.Area)
				}
				return false, ""
			}},
		{Name: "esc-menu-stuck", Act: ActEsc, Escalate: ActRestartGame, For: t.EscMenu.D(), Cooldown: 20 * time.Second, MaxAttempts: 3, Window: min10,
			Cond: func(c *Ctx) (bool, string) {
				if c.O.MemOK && c.O.State == "ingame" && c.O.EscMenu && !c.O.Dead() {
					return true, "Esc menu (UI[9]) open"
				}
				return false, ""
			}},
		{Name: "panel-stuck", Act: ActEsc, Escalate: ActAlert, For: t.PanelOpen.D(), Cooldown: 30 * time.Second, MaxAttempts: 3, Window: min10,
			Cond: func(c *Ctx) (bool, string) {
				if c.O.MemOK && c.O.State == "ingame" && !c.O.EscMenu && len(c.O.Panels) > 0 && !c.O.Dead() {
					return true, "panel flags open: " + joinInts(c.O.Panels)
				}
				return false, ""
			}},
		{Name: "out-of-game", Act: ActOOGStart, Escalate: ActRestartGame, For: t.OutOfGame.D(), Cooldown: 60 * time.Second, MaxAttempts: 3, Window: min10, NeedsChar: true,
			Cond: func(c *Ctx) (bool, string) {
				if c.O.ProcAlive && c.O.MemOK && c.O.State == "menu" && len(c.O.Dialogs) == 0 {
					return true, "at menu screen " + c.O.MenuScr + " while the character should be in game"
				}
				return false, ""
			}},
		{Name: "loading-stuck", Act: ActRestartGame, Escalate: ActAlert, For: t.LoadingStuck.D(), Cooldown: 3 * time.Minute, MaxAttempts: 2, Window: 20 * time.Minute,
			Cond: func(c *Ctx) (bool, string) {
				if c.O.ProcAlive && c.O.MemOK && c.O.State == "loading" {
					return true, "client state loading/ambiguous"
				}
				return false, ""
			}},
		{Name: "bot-silent", Act: ActRestartBot, Escalate: ActAlert, For: t.BotSilent.D(), Cooldown: 5 * time.Minute, MaxAttempts: 2, Window: 30 * time.Minute,
			Cond: func(c *Ctx) (bool, string) {
				b := c.O.Bot
				if b == nil {
					return false, ""
				}
				if !b.Reachable {
					return true, "bot health unreachable: " + b.Err
				}
				return false, ""
			}},
		{Name: "bot-stuck", Act: ActRestartBot, Escalate: ActAlert, For: t.BotStuck.D(), Cooldown: 5 * time.Minute, MaxAttempts: 2, Window: 30 * time.Minute,
			Cond: func(c *Ctx) (bool, string) {
				b := c.O.Bot
				if b != nil && b.Reachable && b.StuckReason != "" && b.StuckReason != "none" {
					return true, "bot reports stuck: " + b.StuckReason + " (state " + b.State + ")"
				}
				return false, ""
			}},
		{Name: "idle-reaper", Act: ActStopInstance, Escalate: ActAlert, For: 0, Cooldown: 10 * time.Minute, MaxAttempts: 2, Window: time.Hour,
			Cond: func(c *Ctx) (bool, string) {
				if c.Inst.ID == "i0" || c.Keep() || !c.O.ProcAlive || c.O.LockHeld {
					return false, ""
				}
				if b := c.O.Bot; b != nil && b.Reachable {
					return false, ""
				}
				if c.Idle() >= t.IdleReap.D() {
					return true, fmt.Sprintf("idle %s: no lock, no bot heartbeat, memory state unchanged", c.Idle().Round(time.Second))
				}
				return false, ""
			}},
		{Name: "no-progress", Ladder: []Action{ActEsc3, ActBotReset, ActLeaveRejoin, ActRestartGame, ActAlert},
			For: 0, Cooldown: t.StepWait.D(), Window: 30 * time.Minute,
			Cond: func(c *Ctx) (bool, string) {
				if c.O.MemOK && c.O.State == "ingame" && !c.O.Dead() && c.GoalActive() && c.SinceProgress() >= t.NoProgress.D() {
					return true, fmt.Sprintf("no xp/gold/area/position change for %s while a goal is active", c.SinceProgress().Round(time.Second))
				}
				return false, ""
			}},
	}
}
