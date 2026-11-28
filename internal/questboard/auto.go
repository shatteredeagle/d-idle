package questboard

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Opt-in auto-start and the away behaviour (slice S8; design 1.1, 2 and 4.3; user decisions 2026-11-16).
//
// A quest the player marked `auto` starts by itself when its offer is SAFE (never risky, too hard or unknown), the hero
// is in a calm state and the limits below allow it. The start shows a countdown first (AutoOptions.DelaySec) that the
// player can cancel from the page. A start covers the current chunk only: the next chunk raises its own offer, which
// auto-starts only if the quest is still marked and the offer is still safe. The hand-in stays the player's tap while the
// player is present; only a chunk the board started by itself is handed in by the board, and only while the player is
// away: no watch client has asked for the board for AutoOptions.AwayMin minutes. Everything the controller did while the
// player was away is listed on the "while you were away" card until the player dismisses it.
//
// The controller holds no progress: the opt-in flags, the Auto mark on the plan record and the event log are in the plan
// file, so a restart loses nothing but the countdown (it starts again).

// AutoOptions are the data knobs of the controller (internal/env quests.auto*, quests.awayMin). Negative values are zero.
type AutoOptions struct {
	DelaySec int // countdown before an auto-start; 0 starts at once (quests.autoDelaySec)
	AwayMin  int // no watch activity for this long means the player is away (quests.awayMin)
	HoldMin  int // no auto-start this long after the player paused or reset (quests.autoHoldMin)
	GapMin   int // at least this long between two auto-started chunks (quests.autoGapMin)
	DayCap   int // at most this many auto-started chunks per 24 hours; 0 means none (quests.autoDayCap)
}

// DefaultAutoOptions are the values of the knobs.
func DefaultAutoOptions() AutoOptions {
	return AutoOptions{DelaySec: 20, AwayMin: 10, HoldMin: 15, GapMin: 30, DayCap: 6}
}

func mins(n int) time.Duration {
	if n < 0 {
		n = 0
	}
	return time.Duration(n) * time.Minute
}

// AutoInput is what the bot loop knows at a tick.
type AutoInput struct {
	Paused bool // the player's pause (or a freeze) is on
	Calm   bool // not in a fight: a start now disturbs nothing
}

// AutoKind is what a tick did.
type AutoKind string

const (
	AutoNone      AutoKind = ""
	AutoCountdown AutoKind = "countdown" // a start is counting down
	AutoStarted   AutoKind = "started"   // the chunk was accepted by itself this tick
	AutoHandingIn AutoKind = "handingin" // the board hands in a chunk it started, the player being away
)

// AutoAct is the answer of AutoTick.
type AutoAct struct {
	Kind         AutoKind
	Quest, Chunk string
}

// AutoPending is a start that counts down.
type AutoPending struct {
	Quest, Chunk, Title string
	Ends                time.Time
}

// autoState is the controller's memory of this process; nothing here is persisted.
type autoState struct {
	am      sync.Mutex // guards the fields; taken inside e.mu, never the other way round
	born    time.Time  // process start: with no watch client yet, away counts from here
	seen    time.Time  // last watch activity
	hold    time.Time  // last player pause or reset
	pending *AutoPending
	muted   map[string]bool // chunks whose countdown the player cancelled, until they leave the offer
	tried   map[string]bool // chunks the board handed in (or tried to) this process
	why     string          // why an opted-in offer does not start now, in words
	logged  string          // the last why that was logged, so a lasting reason logs once
}

// Touch records watch activity: the board was asked for by a client.
func (e *Exec) Touch() {
	now := e.o.Now()
	e.ax.am.Lock()
	e.ax.seen = now
	e.ax.am.Unlock()
}

// NoteHold records a player pause or reset: nothing starts by itself for HoldMin minutes and a countdown is dropped.
func (e *Exec) NoteHold() {
	now := e.o.Now()
	e.ax.am.Lock()
	e.ax.hold = now
	e.dropPendingLocked("you paused or reset")
	e.ax.am.Unlock()
}

// SetAutoOptions follows the knobs at run time.
func (e *Exec) SetAutoOptions(o AutoOptions) {
	e.mu.Lock()
	e.o.Auto = o
	e.mu.Unlock()
}

// awayAt reports away at now (callers hold ax.am). With no client seen since the process started, away counts from the
// start: a hero left running overnight is away after AwayMin minutes without anyone opening the page.
func (e *Exec) awayAt(now time.Time, o AutoOptions) bool {
	ref := e.ax.seen
	if ref.IsZero() || ref.Before(e.ax.born) {
		ref = e.ax.born
	}
	return now.Sub(ref) >= mins(o.AwayMin)
}

// Away reports whether no watch client has asked for the board for AwayMin minutes.
func (e *Exec) Away() bool { return e.AutoState().Away }

// CancelAuto ends a running countdown (the player's Cancel): the offer does not auto-start again until it has left the
// offer list or the process restarts; the opt-in flag stays. chunk "" matches the countdown of the quest.
func (e *Exec) CancelAuto(quest, chunk string) bool {
	e.ax.am.Lock()
	defer e.ax.am.Unlock()
	p := e.ax.pending
	if p == nil || p.Quest != quest || (chunk != "" && p.Chunk != chunk) {
		return false
	}
	if e.ax.muted == nil {
		e.ax.muted = map[string]bool{}
	}
	e.ax.muted[ChunkKey(p.Quest, p.Chunk)] = true
	e.dropPendingLocked("you cancelled it")
	return true
}

// setWhyLocked sets the held-back reason (callers hold ax.am) and logs it once per change.
func (e *Exec) setWhyLocked(why string) {
	// a cancelled countdown already says why; the plain gate reason of the same tick must not replace it
	if why != "" && strings.HasPrefix(e.ax.why, "countdown for ") && strings.HasSuffix(e.ax.why, ": "+why) {
		return
	}
	e.ax.why = why
	if why != e.ax.logged {
		e.ax.logged = why
		if why != "" {
			e.o.Log("quest auto-start held back", "why", why)
		}
	}
}

// dropPendingLocked ends a running countdown and records why (callers hold ax.am); without one it does nothing.
func (e *Exec) dropPendingLocked(reason string) {
	p := e.ax.pending
	if p == nil {
		return
	}
	e.ax.pending = nil
	e.setWhyLocked("countdown for " + p.Title + " cancelled: " + reason)
}

// AutoSnapshot is what the API shows of the controller.
type AutoSnapshot struct {
	Pending *AutoPending
	Away    bool
	Why     string
	Opt     AutoOptions
}

// AutoState is read by the desk for GET /api/quests.
func (e *Exec) AutoState() AutoSnapshot {
	e.mu.Lock()
	o := e.o.Auto
	e.mu.Unlock()
	now := e.o.Now()
	e.ax.am.Lock()
	defer e.ax.am.Unlock()
	s := AutoSnapshot{Away: e.awayAt(now, o), Why: e.ax.why, Opt: o}
	if p := e.ax.pending; p != nil {
		c := *p
		s.Pending = &c
	}
	return s
}

// AwayEvents are the controller events the player has not dismissed yet that happened while the player was away,
// oldest first.
func (e *Exec) AwayEvents() ([]AutoEvent, error) {
	plan, _, err := e.st.Load()
	if err != nil {
		return nil, err
	}
	var out []AutoEvent
	for _, ev := range plan.AutoLog {
		if ev.Away && ev.At.After(plan.AwaySeen) {
			out = append(out, ev)
		}
	}
	return out, nil
}

// autoGate is why nothing may start by itself now ("" when the state allows it). The rate limits are counted from the
// plan's event log, so a restart cannot reset them.
func (e *Exec) autoGate(plan Plan, in AutoInput, now time.Time) string {
	o := e.o.Auto
	switch {
	case in.Paused:
		return "the bot is paused"
	case !e.ax.hold.IsZero() && now.Sub(e.ax.hold) < mins(o.HoldMin):
		return "you paused or reset a moment ago"
	case !in.Calm:
		return "a fight is on"
	case e.o.Mode == "idle":
		return "the mode is idle, which waits for you"
	case o.DayCap <= 0:
		return "auto-start is limited to zero starts a day"
	}
	var day int
	var last time.Time
	for _, ev := range plan.AutoLog {
		if ev.Kind != EvAutoStart {
			continue
		}
		if now.Sub(ev.At) < 24*time.Hour {
			day++
		}
		if ev.At.After(last) {
			last = ev.At
		}
	}
	if !last.IsZero() && now.Sub(last) < mins(o.GapMin) {
		return fmt.Sprintf("the last auto-start was less than %d minutes ago", o.GapMin)
	}
	if day >= o.DayCap {
		return fmt.Sprintf("the limit of %d auto-starts a day is reached", o.DayCap)
	}
	return ""
}

// autoCandidate is the first opted-in chunk whose offer is safe, and a held-back reason for the first opted-in chunk
// that is not. Only the word `safe` qualifies: risky, too hard and unknown never start by themselves.
func (e *Exec) autoCandidate(vs []QuestView, plan Plan, diff int) (Ref, string, string) {
	var held string
	for _, v := range vs {
		if !plan.AutoStart(diff, v.ID) {
			continue
		}
		q, err := e.b.Quest(v.ID)
		if err != nil || q.AutoStart != "allowed" {
			if held == "" {
				held = v.Title + " is marked auto but the quest is never auto-started"
			}
			continue
		}
		ready := false
		for _, cv := range v.Chunks {
			if cv.State != Ready {
				continue
			}
			ready = true
			if cv.Reason != ReadySafe {
				if held == "" {
					held = v.Title + " is not safe enough to start by itself; it waits for you"
				}
				continue
			}
			return Ref{v.ID, cv.ID}, v.Title, ""
		}
		if !ready && held == "" {
			held = v.Title + " has no ready chunk (" + v.Why + ")"
		}
	}
	return Ref{}, "", held
}

// AutoTick is one tick of the controller: it counts a start down, starts the chunk, or hands in a chunk the board
// started while the player is away. It never blocks (the hand-in dialogue runs on its own goroutine, like the desk's) and
// does nothing while the knob is off.
func (e *Exec) AutoTick(in AutoInput) (AutoAct, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.o.Enabled {
		return AutoAct{}, nil
	}
	now := e.o.Now()
	vs, plan, diff, err := e.derive()
	if err != nil {
		return AutoAct{}, err
	}
	e.ax.am.Lock()
	if in.Paused {
		e.ax.hold = now
		e.dropPendingLocked("the bot is paused")
	}
	away := e.awayAt(now, e.o.Auto)
	e.ax.am.Unlock()

	o := e.open(vs, plan, diff)
	if len(o) > 0 {
		e.ax.am.Lock()
		e.dropPendingLocked("a quest dialogue or hand-in is open")
		if e.ax.pending == nil && e.ax.why == "" {
			e.ax.logged = ""
		}
		e.ax.am.Unlock()
		return e.autoHandInLocked(vs, plan, diff, o, in, away), nil
	}
	cand, title, held := e.autoCandidate(vs, plan, diff)
	e.ax.am.Lock()
	// a muted countdown stays muted while its chunk is still on offer
	for k := range e.ax.muted {
		if k != ChunkKey(cand.Quest, cand.Chunk) {
			delete(e.ax.muted, k)
		}
	}
	if cand.Quest == "" || e.ax.muted[ChunkKey(cand.Quest, cand.Chunk)] {
		reason := held
		if cand.Quest != "" {
			reason = "you cancelled it"
		} else if e.ax.pending != nil {
			reason = "the offer is gone (" + held + ")"
			if held == "" {
				reason = "the offer is gone"
			}
		}
		if e.ax.pending != nil {
			e.dropPendingLocked(reason)
		} else {
			e.setWhyLocked(reason)
		}
		e.ax.am.Unlock()
		return AutoAct{}, nil
	}
	if why := e.autoGate(plan, in, now); why != "" {
		if e.ax.pending != nil {
			e.dropPendingLocked(why)
		} else {
			e.setWhyLocked(why)
		}
		e.ax.am.Unlock()
		return AutoAct{}, nil
	}
	e.setWhyLocked("")
	p := e.ax.pending
	if p == nil || p.Quest != cand.Quest || p.Chunk != cand.Chunk {
		p = &AutoPending{Quest: cand.Quest, Chunk: cand.Chunk, Title: title, Ends: now.Add(time.Duration(maxInt(e.o.Auto.DelaySec, 0)) * time.Second)}
		e.ax.pending = p
		e.o.Log("quest auto-start counting down", "quest", cand.Quest, "chunk", cand.Chunk, "ends", p.Ends)
	}
	if now.Before(p.Ends) {
		e.ax.am.Unlock()
		return AutoAct{Kind: AutoCountdown, Quest: cand.Quest, Chunk: cand.Chunk}, nil
	}
	e.ax.pending = nil
	e.ax.am.Unlock()
	ev := AutoEvent{At: now.UTC(), Quest: cand.Quest, Chunk: cand.Chunk, Kind: EvAutoStart, Away: away}
	if err := e.acceptLocked(vs, plan, diff, cand.Quest, cand.Chunk, &ev); err != nil {
		return AutoAct{}, err
	}
	e.o.Log("quest auto-started", "quest", cand.Quest, "chunk", cand.Chunk, "away", away)
	return AutoAct{Kind: AutoStarted, Quest: cand.Quest, Chunk: cand.Chunk}, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// autoHandInLocked hands in the one accepted chunk when the board started it, it waits at the NPC and the player is
// away. A present player taps Hand in; if they leave, the next tick after AwayMin minutes does it.
func (e *Exec) autoHandInLocked(vs []QuestView, plan Plan, diff int, o []Ref, in AutoInput, away bool) AutoAct {
	if len(o) != 1 || !away || in.Paused || !in.Calm || e.handing {
		return AutoAct{}
	}
	r := o[0]
	_, cv, ok := chunkView(vs, r.Quest, r.Chunk)
	if !ok || cv.State != Awaiting || !plan.autoRecord(diff, r.Quest, r.Chunk) {
		return AutoAct{}
	}
	k := ChunkKey(r.Quest, r.Chunk)
	e.ax.am.Lock()
	if e.ax.tried[k] {
		e.ax.am.Unlock()
		return AutoAct{}
	}
	if e.ax.tried == nil {
		e.ax.tried = map[string]bool{}
	}
	e.ax.tried[k] = true
	e.ax.am.Unlock()
	go e.autoHandIn(r)
	return AutoAct{Kind: AutoHandingIn, Quest: r.Quest, Chunk: r.Chunk}
}

func (e *Exec) autoHandIn(r Ref) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res, err := e.HandIn(ctx, r.Quest, r.Chunk)
	ev := AutoEvent{At: e.o.Now().UTC(), Quest: r.Quest, Chunk: r.Chunk, Kind: EvAutoHandIn, Away: true, Awarded: res.Awarded}
	switch {
	case err != nil:
		ev.Kind, ev.Note = EvAutoHandInFailed, cleanErr(err)
	case res.Outcome == HandInDone:
	case res.Outcome == HandInConfirming:
		ev.Note = "the reward was seen; the game has not confirmed it yet"
	default:
		ev.Kind, ev.Note = EvAutoHandInFailed, res.Why
	}
	if err := e.st.AddAutoEvent(ev); err != nil {
		e.o.Log("quest auto event not saved", "err", err)
	}
}

// autoRecord reports whether the accepted record of the chunk was written by the auto-start controller.
func (p Plan) autoRecord(diff int, quest, chunk string) bool {
	for _, r := range p.Accepted {
		if r.Quest == quest && r.Chunk == chunk && r.Diff == Difficulties[diff] {
			return r.Auto
		}
	}
	return false
}
