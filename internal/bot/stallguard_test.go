package bot

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// The stall diagnostic (Questy death 11: 14 s with no log line and no survival action, seven hostiles): the guard logs one
// `stall: goroutine stack` after 60 s of silence with enough hostiles near, rate limited, counted, and fed to the soft
// invariant guard-stall. Quiet for less, fewer hostiles, in town or a recent line change nothing. Two classes, fake clock.
func TestGuardStallLogsOneStack(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, tc := range []struct {
			name     string
			hostiles int
			quiet    time.Duration
			town     bool
			want     bool
		}{
			{"silent with a pack", 3, 70 * time.Second, false, true},
			{"silent but one hostile", 1, 70 * time.Second, false, false},
			{"a recent log line", 3, 30 * time.Second, false, false},
			{"in town", 3, 70 * time.Second, true, false},
		} {
			g := newRig(t)
			buf := &bytes.Buffer{}
			g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
			g.w.set(func(w *world) { w.class, w.level, w.area = class, 12, 2 })
			for i := 0; i < tc.hostiles; i++ {
				g.w.addMonster(uint32(20+i), 10+i, 8, 0)
			}
			if tc.town {
				g.w.set(func(w *world) { w.area = 1 })
			}
			g.r.poll()
			now := g.e.Clock.Now()
			g.r.lastLogAt.Store(now.Add(-tc.quiet).UnixNano())
			got := g.r.stallCheck(g.r.latest(), now)
			if got != tc.want || strings.Contains(buf.String(), "stall: goroutine stack") != tc.want {
				t.Errorf("%v %s: stall %v (log %q), want %v", class, tc.name, got, buf.String(), tc.want)
			}
			if tc.want {
				if g.r.stalls.Load() != 1 || len(g.r.inv.pending) != 1 || g.r.inv.pending[0].Name != "guard-stall" {
					t.Errorf("%v: stalls %d pending %+v", class, g.r.stalls.Load(), g.r.inv.pending)
				}
				if again := g.r.stallCheck(g.r.latest(), now.Add(time.Minute)); again {
					t.Errorf("%v: a second stall line within the gap", class)
				}
				if later := g.r.stallCheck(g.r.latest(), now.Add(guardStallGap+time.Second)); !later {
					t.Errorf("%v: no stall line after the gap", class)
				}
			}
		}
	}
}

// attackFighter reports a fixed attack target and records the targets kept out of the fights.
type attackFighter struct {
	Fighter
	target  uint32
	ignored []uint32
}

func (f *attackFighter) Attacking() uint32                    { return f.target }
func (f *attackFighter) IgnoreFor(id uint32, _ time.Duration) { f.ignored = append(f.ignored, id) }

// Doofer 2026-11-26: a Fighting action ran 147 s with hostiles near and nothing changing. The guard breaks a Fighting action that
// is older than fight-stall Secs (120) and whose kills, xp and target life did not change for Count seconds (60); a kill resets
// it, a younger action, a boss run and other states are left alone; do() then keeps the target out. Two classes, fake clock.
func TestGuardBreaksAStalledFight(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		g := newRig(t)
		af := &attackFighter{Fighter: g.r.d.Fighter, target: 77}
		g.r.d.Fighter = af
		g.w.set(func(w *world) { w.class, w.level, w.area = class, 21, 5 })
		for i := 0; i < 3; i++ {
			g.w.addMonster(uint32(70+i+7), 10, 8+i, 0)
		}
		g.r.poll()
		s := g.r.latest()
		g.r.mu.Lock()
		g.r.activity = "Fighting in Dark Wood"
		g.r.mu.Unlock()
		t0 := g.e.Clock.Now()
		g.r.actionAt.Store(t0.UnixNano())
		step := func(d time.Duration, st State) bool { return g.r.fightStallCheck(s, t0.Add(d), st) }
		if step(0, StateClearing) {
			t.Fatalf("%v: broke at once", class)
		}
		if step(100*time.Second, StateClearing) {
			t.Errorf("%v: broke a fight that is younger than 120 s", class)
		}
		if step(130*time.Second, StateTraveling) {
			t.Errorf("%v: broke a traveling action", class)
		}
		g.r.bossRun.Store(true)
		if step(130*time.Second, StateClearing) {
			t.Errorf("%v: broke a boss run", class)
		}
		g.r.bossRun.Store(false)
		if !step(130*time.Second, StateClearing) {
			t.Fatalf("%v: did not break a fight with no progress for 130 s", class)
		}
		if g.r.fightTarget.Load() != 77 {
			t.Errorf("%v: blamed target %d, want 77", class, g.r.fightTarget.Load())
		}
		g.r.blameFightTarget()
		if len(af.ignored) != 1 || af.ignored[0] != 77 {
			t.Errorf("%v: ignored %v", class, af.ignored)
		}
		// a kill resets the clock of "no progress"
		g.r.actionAt.Store(t0.UnixNano())
		g.r.fightProg = fightProgress{}
		step(130*time.Second, StateClearing)
		g.w.set(func(w *world) { w.exp += 50 })
		g.r.poll()
		s = g.r.latest()
		if step(170*time.Second, StateClearing) {
			t.Errorf("%v: broke right after the xp changed", class)
		}
	}
}
