package bot

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// g2d NOKILL fixes (docs/research/g2d-nokill-diagnosis.md): burst cap of rung 3, kill drought watchdog, empty quest
// field, failsafe visibility in the minute line. Each rule is tested with two classes.

var nokillClasses = []d2.Class{d2.Sorceress, d2.Barbarian}

// Seed 11: kills between the cycles refill the per-window counter, so only the time window can stop the cycling.
func TestRung3BurstIgnoresProgress(t *testing.T) {
	for _, cl := range nokillClasses {
		t.Run(cl.String(), func(t *testing.T) {
			g := ladderRig(t, cl, 6)
			g.r.e.Tune.Bot.Rung3WindowMax, g.r.e.Tune.Bot.Rung3WindowMin = 3, 20
			ctx := context.Background()
			cycles := 0
			for i := 0; i < 8; i++ {
				g.realProgress(realResetMin) // three kills: the ladder starts over
				g.clock.Sleep(2 * time.Minute)
				for k := 0; k < 2; k++ { // rung 1, then rung 3 (rung 2 does not apply in town)
					g.r.escalate(ctx, g.r.control(), "no progress in state farming for 61s", errStuck)
				}
				cycles = strings.Count(g.w.log(), "newgame")
			}
			if cycles != 3 {
				t.Fatalf("%d re-entries in 16 min, want the burst cap 3", cycles)
			}
			if g.r.ladderStep() == 3 {
				t.Fatal("the capped round must not end on rung 3")
			}
		})
	}
}

func TestRung3BurstWindowExpires(t *testing.T) {
	g := ladderRig(t, d2.Sorceress, 6)
	g.r.e.Tune.Bot.Rung3WindowMax, g.r.e.Tune.Bot.Rung3WindowMin = 2, 10
	for i := 0; i < 4; i++ {
		g.realProgress(realResetMin)
		g.clock.Sleep(11 * time.Minute) // each cycle is alone in its window
		g.r.escalate(context.Background(), g.r.control(), "stuck", errStuck)
		g.r.escalate(context.Background(), g.r.control(), "stuck", errStuck)
	}
	if n := strings.Count(g.w.log(), "newgame"); n != 4 {
		t.Fatalf("%d re-entries, want 4 (never two inside a window)", n)
	}
}

func (r *Runner) ladderStep() int {
	r.hmu.Lock()
	defer r.hmu.Unlock()
	return r.ladder
}

func droughtRig(t *testing.T, cl d2.Class, goalLevel bool) *rig {
	t.Helper()
	g := ladderRig(t, cl, 6)
	g.w.set(func(w *world) { w.area = 2; w.life, w.maxL = 100, 100; w.level = 10 })
	if goalLevel {
		if err := g.r.Start(reach(20)); err != nil {
			t.Fatal(err)
		}
	}
	g.r.poll()
	g.r.droughtCheck(context.Background(), g.r.latest(), g.r.control()) // arms the clock
	g.r.obMu.Lock()
	g.r.ob.fieldSince = g.r.e.Clock.Now()
	g.r.obMu.Unlock()
	return g
}

func (g *rig) droughtStep(d time.Duration) bool {
	g.clock.Sleep(d)
	g.r.poll()
	return g.r.droughtCheck(context.Background(), g.r.latest(), g.r.control())
}

func TestDroughtWatchdog(t *testing.T) {
	for _, cl := range nokillClasses {
		t.Run(cl.String(), func(t *testing.T) {
			g := droughtRig(t, cl, false) // a farm goal names its area: nothing to skip, so the first drought is a fresh game
			if g.droughtStep(7 * time.Minute) {
				t.Fatal("7 minutes is below the default 8")
			}
			if !g.droughtStep(2 * time.Minute) {
				t.Fatal("9 minutes without a kill must act")
			}
			if n := strings.Count(g.w.log(), "newgame"); n != 1 {
				t.Fatalf("%d fresh games, want 1", n)
			}
			if g.droughtStep(5 * time.Minute) {
				t.Fatal("the window restarts after the action")
			}
			// a kill resets the stage; without one the next drought is the next stage
			g.r.obMu.Lock()
			g.r.ob.killXPAt = g.r.e.Clock.Now()
			g.r.obMu.Unlock()
			g.droughtStep(0)
			g.r.obMu.Lock()
			st := g.r.ob.droughtStage
			g.r.obMu.Unlock()
			if st != 0 {
				t.Fatalf("stage %d after a kill, want 0", st)
			}
		})
	}
}

func TestDroughtSkipsTheAreaFirst(t *testing.T) {
	for _, cl := range nokillClasses {
		t.Run(cl.String(), func(t *testing.T) {
			g := droughtRig(t, cl, true)
			g.r.obMu.Lock()
			g.r.ob.targetArea = 2
			g.r.obMu.Unlock()
			if !g.droughtStep(9 * time.Minute) {
				t.Fatal("must act")
			}
			g.r.obMu.Lock()
			until, ok := g.r.avoid[2]
			g.r.obMu.Unlock()
			if !ok || !until.After(g.r.e.Clock.Now()) {
				t.Fatal("the area must be avoided after the first drought")
			}
			if strings.Contains(g.w.log(), "newgame") {
				t.Fatal("the first drought re-plans, it does not start a fresh game")
			}
			if !g.droughtStep(9*time.Minute) || !strings.Contains(g.w.log(), "newgame") {
				t.Fatal("a second drought in a row starts a fresh game")
			}
		})
	}
}

func TestDroughtQuietCases(t *testing.T) {
	g := droughtRig(t, d2.Sorceress, false)
	g.w.set(func(w *world) { w.life = 50 }) // 50 percent: the fight rules own this, not the watchdog
	if g.droughtStep(20 * time.Minute) {
		t.Fatal("low life")
	}
	g.w.set(func(w *world) { w.life = 100; w.area = 1 })
	if g.droughtStep(20 * time.Minute) {
		t.Fatal("town")
	}
	g.w.set(func(w *world) { w.area = 2 })
	g.r.e.Tune.Bot.DroughtMin = 0
	g.r.obMu.Lock()
	g.r.ob.fieldSince = g.r.e.Clock.Now().Add(-time.Hour)
	g.r.obMu.Unlock()
	if g.droughtStep(20 * time.Minute) {
		t.Fatal("knob 0 is off")
	}
}

// Seed 17: a quest-run area with no hostile in sight is left after bot.emptyFieldSec; hostiles in sight keep it.
func TestEmptyQuestFieldLeaves(t *testing.T) {
	for _, cl := range nokillClasses {
		t.Run(cl.String(), func(t *testing.T) {
			g := newRig(t)
			r := g.r
			g.w.set(func(w *world) { w.class = cl; w.state = d2.StateInGame; w.area = 2 })
			r.poll()
			r.loop.cleared = map[int]bool{}
			run := LevelRun{Area: 2}
			r.loop.run = &run
			step := func(d time.Duration) bool { g.clock.Sleep(d); r.poll(); return r.emptyAreaLeave(2) }
			if step(0) || step(50*time.Second) || step(50*time.Second) {
				t.Fatal("too early")
			}
			if !step(60*time.Second) || !r.loop.cleared[2] {
				t.Fatal("160 s with no hostile must leave the quest field")
			}
			r.loop.cleared = map[int]bool{}
			g.w.addMonster(9, 10, 5, 5)
			r.poll()
			r.e.Tune.Bot.EmptyFieldSec = 150
			step(0)
			if step(50*time.Second) || step(50*time.Second) || step(60*time.Second) || r.loop.cleared[2] {
				t.Fatal("a hostile in sight keeps the quest area")
			}
		})
	}
}

func TestFailsafeTag(t *testing.T) {
	for in, want := range map[string]string{
		"no progress in state farming for 61s":        "np_farming",
		"no progress for 600s (state clearing)":       "np_global",
		"walk (to Blood Moor)":                        "walk",
		"a very long action name that goes on and on": "a_very_long_action_name_",
	} {
		if got := failsafeTag(in); got != want {
			t.Errorf("failsafeTag(%q) = %q, want %q", in, got, want)
		}
	}
}

type peakMover struct {
	*fakeMover
	peak int
}

func (p *peakMover) TakePeakRung() int { v := p.peak; p.peak = 0; return v }

// The minute line carries the last failsafe reason and the nav stuck-ladder peak, and stays short.
func TestMinuteLineShowsFailsafeAndNavRung(t *testing.T) {
	g := newRig(t)
	pm := &peakMover{fakeMover: g.mover, peak: 4}
	g.r.d.Mover = pm
	var buf strings.Builder
	g.e.Log = slog.New(slog.NewJSONHandler(&buf, nil))
	g.r.obMu.Lock()
	g.r.ob.fsWhy = failsafeTag("no progress in state farming for 61s")
	g.r.obMu.Unlock()
	g.w.set(func(w *world) { w.state = d2.StateInGame; w.area = 2; w.life, w.maxL = 90, 100 })
	g.r.poll()
	s := g.r.latest()
	g.r.obMu.Lock()
	g.r.minuteLine(s, s.Player, g.clock.Now())
	g.r.obMu.Unlock()
	line := buf.String()
	if !strings.Contains(line, `"fsWhy":"np_farming"`) || !strings.Contains(line, `"navRung":4`) {
		t.Fatalf("minute line lacks the new keys: %s", line)
	}
	if pm.peak != 0 {
		t.Fatal("the peak must be taken (reset) by the minute line")
	}
}

type refuseMover struct {
	*fakeMover
	n int
}

func (p *refuseMover) TakeGateRefusals() int { v := p.n; p.n = 0; return v }

// F7: the minute line names the nokillfix mechanism that fired last and the mouth-gate refusals of the minute; both
// are absent when there is nothing to say, and a kill clears the mechanism.
func TestMinuteLineShowsNokillMechanismAndMouthRefusals(t *testing.T) {
	g := newRig(t)
	rm := &refuseMover{fakeMover: g.mover, n: 7}
	g.r.d.Mover = rm
	var buf strings.Builder
	g.e.Log = slog.New(slog.NewJSONHandler(&buf, nil))
	g.r.noteNokill("emptyfield")
	g.w.set(func(w *world) { w.state = d2.StateInGame; w.area = 2; w.life, w.maxL = 90, 100 })
	g.r.poll()
	s := g.r.latest()
	line := func() string {
		buf.Reset()
		g.r.obMu.Lock()
		g.r.minuteLine(s, s.Player, g.clock.Now())
		g.r.obMu.Unlock()
		return buf.String()
	}
	if l := line(); !strings.Contains(l, `"nkMech":"emptyfield"`) || !strings.Contains(l, `"mouthRef":7`) {
		t.Fatalf("minute line lacks the keys: %s", l)
	}
	g.r.obMu.Lock()
	g.r.ob.bumpReal(g.clock.Now())
	g.r.obMu.Unlock()
	if l := line(); strings.Contains(l, "nkMech") || strings.Contains(l, "mouthRef") {
		t.Fatalf("keys must be absent when empty: %s", l)
	}
}
