package questboard

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// Slice S8: the opt-in auto-start and the away behaviour, with a fake clock, for a melee Paladin and a caster Sorceress.

type clk struct {
	mu sync.Mutex
	t  time.Time
}

func newClk() *clk { return &clk{t: time.Date(2026, 11, 17, 1, 0, 0, 0, time.UTC)} }

func (c *clk) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }

func (c *clk) add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

type autoRig struct {
	*rig
	c    *clk
	o    AutoOptions
	env  Env
	mode string
}

func newAutoRig(t *testing.T, class string, clvl int, o AutoOptions) *autoRig {
	t.Helper()
	r := &rig{t: t, b: mustBoard(t), dir: t.TempDir(), hero: "Testy"}
	r.f = NewFake(r.b, class, clvl)
	a := &autoRig{rig: r, c: newClk(), o: o, env: r.f}
	a.restart()
	return a
}

func (a *autoRig) restart() {
	st, err := Open(a.b, a.dir, a.hero)
	if err != nil {
		a.t.Fatal(err)
	}
	st.now = a.c.Now
	a.x = NewExec(a.b, st, a.env, a.f, Options{Enabled: true, Now: a.c.Now, Auto: a.o, Mode: a.mode})
}

func (a *autoRig) opt(quest string) {
	a.t.Helper()
	if err := a.x.st.SetAutoStart(quest, 0, true); err != nil {
		a.t.Fatal(err)
	}
}

var calm = AutoInput{Calm: true}

func (a *autoRig) tickAuto(in AutoInput) AutoAct {
	a.t.Helper()
	act, err := a.x.AutoTick(in)
	if err != nil {
		a.t.Fatal(err)
	}
	return act
}

func (a *autoRig) plan() Plan {
	p, _, err := a.x.st.Load()
	if err != nil {
		a.t.Fatal(err)
	}
	return p
}

func opts(delay int) AutoOptions {
	o := DefaultAutoOptions()
	o.DelaySec = delay
	return o
}

func TestAutoStartsOnlyASafeOffer(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			a := newAutoRig(t, bd.class, 6, opts(0))
			a.opt(den)
			act := a.tickAuto(calm)
			if act.Kind != AutoStarted || act.Quest != den || act.Chunk != clr {
				t.Fatalf("act %+v", act)
			}
			p := a.plan()
			if len(p.Accepted) != 1 || !p.Accepted[0].Auto || len(p.AutoLog) != 1 || p.AutoLog[0].Kind != EvAutoStart {
				t.Fatalf("plan %+v", p)
			}
			if a.chunk(den, clr).State != Accepted {
				t.Fatalf("state %v", a.chunk(den, clr).State)
			}
		})
	}
}

// Blood Raven at clvl 6 is risky for the Paladin (floor 7) and too hard for the Sorceress (floor 8): neither starts.
func TestAutoNeverOnRiskyOrTooHard(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			a := newAutoRig(t, bd.class, 6, opts(0))
			a.opt(brq)
			a.c.add(time.Hour)
			if act := a.tickAuto(calm); act.Kind != AutoNone {
				t.Fatalf("started %+v", act)
			}
			if len(a.plan().Accepted) != 0 {
				t.Fatal("accepted")
			}
			if bd.burialSeen && !strings.Contains(a.x.AutoState().Why, "not safe") {
				t.Fatalf("why %q", a.x.AutoState().Why)
			}
		})
	}
}

type unknownEnv struct{ *Fake }

func (unknownEnv) Ready(Quest, Chunk) Word { return ReadyUnknown }

func TestAutoNeverOnUnknownReadiness(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			a := newAutoRig(t, bd.class, 20, opts(0))
			a.env = unknownEnv{a.f}
			a.restart()
			a.opt(den)
			if act := a.tickAuto(calm); act.Kind != AutoNone || len(a.plan().Accepted) != 0 {
				t.Fatalf("act %+v", act)
			}
		})
	}
}

func TestAutoNeverWithoutTheFlagOrForANeverQuest(t *testing.T) {
	a := newAutoRig(t, "Paladin", 20, opts(0))
	if act := a.tickAuto(calm); act.Kind != AutoNone {
		t.Fatalf("no flag: %+v", act)
	}
	if err := a.x.st.SetAutoStart("search_for_cain", 0, true); err == nil {
		t.Fatal("a never quest took the flag")
	}
}

func TestAutoCountdownAndCancel(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			a := newAutoRig(t, bd.class, 6, opts(20))
			a.opt(den)
			if act := a.tickAuto(calm); act.Kind != AutoCountdown {
				t.Fatalf("act %+v", act)
			}
			a.c.add(10 * time.Second)
			if act := a.tickAuto(calm); act.Kind != AutoCountdown || a.x.AutoState().Pending == nil {
				t.Fatalf("act %+v", act)
			}
			if !a.x.CancelAuto(den, "") || a.x.CancelAuto(den, "") {
				t.Fatal("cancel must succeed once")
			}
			a.c.add(time.Minute)
			if act := a.tickAuto(calm); act.Kind != AutoNone || len(a.plan().Accepted) != 0 {
				t.Fatalf("after cancel %+v", act)
			}
			if !a.plan().AutoStart(0, den) {
				t.Fatal("cancel cleared the opt-in")
			}
			// the offer goes away (the hero is hurt, say) and comes back: it counts down again
			a.f.Level = 0
			a.tickAuto(calm)
			a.f.Level = 6
			if act := a.tickAuto(calm); act.Kind != AutoCountdown {
				t.Fatalf("re-offer %+v", act)
			}
			a.c.add(21 * time.Second)
			if act := a.tickAuto(calm); act.Kind != AutoStarted {
				t.Fatalf("after delay %+v", act)
			}
		})
	}
}

func TestAutoHeldBackStates(t *testing.T) {
	a := newAutoRig(t, "Sorceress", 6, opts(0))
	a.opt(den)
	if act := a.tickAuto(AutoInput{Paused: true, Calm: true}); act.Kind != AutoNone || a.x.AutoState().Why != "the bot is paused" {
		t.Fatalf("paused %+v %q", act, a.x.AutoState().Why)
	}
	// a pause starts the hold: still nothing after the resume
	if act := a.tickAuto(calm); act.Kind != AutoNone || !strings.Contains(a.x.AutoState().Why, "paused or reset") {
		t.Fatalf("hold %+v %q", act, a.x.AutoState().Why)
	}
	a.c.add(16 * time.Minute)
	if act := a.tickAuto(AutoInput{}); act.Kind != AutoNone || a.x.AutoState().Why != "a fight is on" {
		t.Fatalf("fight %+v %q", act, a.x.AutoState().Why)
	}
	a.x.SetMode("idle")
	if act := a.tickAuto(calm); act.Kind != AutoNone || !strings.Contains(a.x.AutoState().Why, "idle") {
		t.Fatalf("idle %+v", act)
	}
	a.x.SetMode("farm")
	if act := a.tickAuto(calm); act.Kind != AutoStarted {
		t.Fatalf("farm %+v", act)
	}
}

func TestAutoResetHoldsToo(t *testing.T) {
	a := newAutoRig(t, "Paladin", 6, opts(0))
	a.opt(den)
	a.x.Reset()
	if act := a.tickAuto(calm); act.Kind != AutoNone {
		t.Fatalf("after reset %+v", act)
	}
	a.c.add(15 * time.Minute)
	if act := a.tickAuto(calm); act.Kind != AutoStarted {
		t.Fatalf("after hold %+v", act)
	}
}

// finish plays a started chunk to the end: the steps, then the hand-in by hand.
func (a *autoRig) finish(quest, chunk string) {
	a.t.Helper()
	a.tick(8)
	res, err := a.x.HandIn(context.Background(), quest, chunk)
	if err != nil || res.Outcome != HandInDone {
		a.t.Fatalf("hand-in %+v %v", res, err)
	}
}

func TestAutoRateLimits(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			o := opts(0)
			o.DayCap = 2
			a := newAutoRig(t, bd.class, 20, o)
			a.opt(den)
			a.opt(brq)
			if act := a.tickAuto(calm); act.Kind != AutoStarted || act.Quest != den {
				t.Fatalf("first %+v", act)
			}
			a.finish(den, clr)
			a.c.add(5 * time.Minute)
			if act := a.tickAuto(calm); act.Kind != AutoNone || !strings.Contains(a.x.AutoState().Why, "30 minutes") {
				t.Fatalf("gap %+v %q", act, a.x.AutoState().Why)
			}
			// a restart does not reset the limits: they are counted from the plan's log
			a.restart()
			if act := a.tickAuto(calm); act.Kind != AutoNone {
				t.Fatalf("gap after restart %+v", act)
			}
			a.c.add(30 * time.Minute)
			if act := a.tickAuto(calm); act.Kind != AutoStarted || act.Quest != brq {
				t.Fatalf("second %+v", act)
			}
			a.f.ChunkDone[ChunkKey(brq, brc)] = TierMemory
			a.f.DoneQuest[brq] = TierMemory
			// a third quest is held by the day cap, whatever the gap
			a.opt("forgotten_tower")
			a.c.add(2 * time.Hour)
			if act := a.tickAuto(calm); act.Kind != AutoNone || !strings.Contains(a.x.AutoState().Why, "limit of 2") {
				t.Fatalf("cap %+v %q", act, a.x.AutoState().Why)
			}
			a.c.add(24 * time.Hour)
			if act := a.tickAuto(calm); act.Kind != AutoStarted {
				t.Fatalf("next day %+v", act)
			}
		})
	}
}

func TestAutoAwayDetection(t *testing.T) {
	a := newAutoRig(t, "Sorceress", 6, opts(0))
	if a.x.Away() {
		t.Fatal("away at once")
	}
	a.c.add(9 * time.Minute)
	if a.x.Away() {
		t.Fatal("away after 9 minutes")
	}
	a.x.Touch()
	a.c.add(9 * time.Minute)
	if a.x.Away() {
		t.Fatal("away 9 minutes after a touch")
	}
	a.c.add(2 * time.Minute)
	if !a.x.Away() {
		t.Fatal("not away after 11 minutes")
	}
	a.x.Touch()
	if a.x.Away() {
		t.Fatal("still away after a touch")
	}
}

func waitEvents(t *testing.T, a *autoRig, n int) []AutoEvent {
	t.Helper()
	for i := 0; i < 300; i++ {
		if p := a.plan(); len(p.AutoLog) >= n {
			return p.AutoLog
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("event never came")
	return nil
}

func TestAutoHandInOnlyWhenAwayAndOnlyForAutoStarts(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			a := newAutoRig(t, bd.class, 6, opts(0))
			a.opt(den)
			a.x.Touch()
			if act := a.tickAuto(calm); act.Kind != AutoStarted {
				t.Fatalf("start %+v", act)
			}
			a.tick(8)
			if a.chunk(den, clr).State != Awaiting {
				t.Fatalf("state %v", a.chunk(den, clr).State)
			}
			// the player is here: the tap is theirs
			a.c.add(2 * time.Minute)
			a.x.Touch()
			if act := a.tickAuto(calm); act.Kind != AutoNone || a.f.Count("handin") != 0 {
				t.Fatalf("present %+v", act)
			}
			// the player leaves
			a.c.add(11 * time.Minute)
			if act := a.tickAuto(AutoInput{}); act.Kind != AutoNone { // mid-fight: wait
				t.Fatalf("fight %+v", act)
			}
			if act := a.tickAuto(calm); act.Kind != AutoHandingIn {
				t.Fatalf("away %+v", act)
			}
			evs := waitEvents(t, a, 2)
			last := evs[len(evs)-1]
			if last.Kind != EvAutoHandIn || !last.Away || evs[0].Kind != EvAutoStart || evs[0].Away {
				t.Fatalf("events %+v", evs)
			}
			if a.chunk(den, clr).State != Done || a.f.Count("handin") != 1 {
				t.Fatalf("not handed in once: %v %d", a.chunk(den, clr).State, a.f.Count("handin"))
			}
			if act := a.tickAuto(calm); act.Kind != AutoNone || a.f.Count("handin") != 1 {
				t.Fatalf("again %+v", act)
			}
		})
	}
}

func TestAutoNeverHandsInAHandStart(t *testing.T) {
	a := newAutoRig(t, "Paladin", 6, opts(0))
	if err := a.x.Accept(den, clr); err != nil {
		t.Fatal(err)
	}
	a.tick(8)
	a.c.add(time.Hour)
	if act := a.tickAuto(calm); act.Kind != AutoNone || a.f.Count("handin") != 0 {
		t.Fatalf("hand-started chunk handed in: %+v", act)
	}
}

func TestAutoFailedHandInIsLoggedWithReason(t *testing.T) {
	a := newAutoRig(t, "Sorceress", 6, opts(0))
	a.opt(den)
	a.tickAuto(calm)
	a.tick(8)
	a.f.HandInErrs = 5
	a.c.add(time.Hour)
	if act := a.tickAuto(calm); act.Kind != AutoHandingIn {
		t.Fatalf("%+v", act)
	}
	evs := waitEvents(t, a, 2)
	if k := evs[1].Kind; k != EvAutoHandInFailed || evs[1].Note == "" {
		t.Fatalf("events %+v", evs)
	}
	evs2, _ := a.x.AwayEvents()
	if len(evs2) != 1 || evs2[0].Kind != EvAutoHandInFailed {
		t.Fatalf("away events %+v", evs2) // the start was before the player left; only the failure is on the card
	}
}

// The restart in the middle of an away stretch: the flag, the Auto mark and the log are in the plan, so the board
// resumes the chunk and still hands it in by itself.
func TestAutoRestartPersistence(t *testing.T) {
	a := newAutoRig(t, "Paladin", 6, opts(0))
	a.opt(den)
	a.tickAuto(calm)
	a.tick(8)
	a.restart()
	if !a.plan().AutoStart(0, den) || !a.plan().autoRecord(0, den, clr) {
		t.Fatalf("plan %+v", a.plan())
	}
	a.c.add(time.Hour)
	if act := a.tickAuto(calm); act.Kind != AutoHandingIn {
		t.Fatalf("after restart %+v", act)
	}
	waitEvents(t, a, 2)
	if a.chunk(den, clr).State != Done {
		t.Fatal("not done")
	}
}

func TestAutoKnobOffChangesNothing(t *testing.T) {
	a := newAutoRig(t, "Sorceress", 6, opts(0))
	a.opt(den)
	a.x.SetEnabled(false)
	a.c.add(time.Hour)
	if act := a.tickAuto(calm); act.Kind != AutoNone {
		t.Fatalf("%+v", act)
	}
	if p := a.plan(); len(p.Accepted) != 0 || len(p.AutoLog) != 0 {
		t.Fatalf("plan %+v", p)
	}
	if _, err := a.x.State(); err != ErrOff {
		t.Fatal(err)
	}
}

func TestAwayEventsAndDismiss(t *testing.T) {
	a := newAutoRig(t, "Paladin", 6, opts(0))
	a.opt(den)
	a.c.add(time.Hour) // nobody looked: away
	a.tickAuto(calm)
	evs, _ := a.x.AwayEvents()
	if len(evs) != 1 || !evs[0].Away {
		t.Fatalf("events %+v", evs)
	}
	a.c.add(time.Minute)
	if err := a.x.st.DismissAway(a.c.Now()); err != nil {
		t.Fatal(err)
	}
	if evs, _ = a.x.AwayEvents(); len(evs) != 0 {
		t.Fatalf("after dismiss %+v", evs)
	}
}

func TestPlanRoundTripsAutoFields(t *testing.T) {
	a := newAutoRig(t, "Sorceress", 6, opts(0))
	a.opt(den)
	a.tickAuto(calm)
	a.restart()
	p := a.plan()
	if len(p.AutoLog) != 1 || p.AutoLog[0].Quest != den || !p.Accepted[0].Auto {
		t.Fatalf("plan %+v", p)
	}
}

// A cancelled countdown records its reason (once in the log, then in AutoState().Why) for every way it ends.
func TestAutoCancelledCountdownRecordsReason(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			a := newAutoRig(t, bd.class, 6, opts(20))
			var logs []string
			a.x.o.Log = func(msg string, kv ...any) {
				if msg == "quest auto-start held back" {
					logs = append(logs, fmt.Sprint(kv[1]))
				}
			}
			a.opt(den)
			a.tickAuto(calm)
			a.x.NoteHold()
			if w := a.x.AutoState().Why; !strings.Contains(w, "cancelled: you paused or reset") {
				t.Fatalf("why %q", w)
			}
			a.tickAuto(calm) // the hold gate: must not replace or re-log the cancel reason
			a.tickAuto(calm)
			if len(logs) != 2 || !strings.Contains(logs[0], "cancelled") {
				t.Fatalf("logs %q, want the cancel line then the hold line", logs)
			}
			a.c.add(time.Hour)
			a.tickAuto(calm)
			a.f.Level = 0 // the offer goes away mid-countdown
			a.tickAuto(calm)
			if w := a.x.AutoState().Why; !strings.Contains(w, "cancelled: the offer is gone") {
				t.Fatalf("why %q", w)
			}
		})
	}
}
