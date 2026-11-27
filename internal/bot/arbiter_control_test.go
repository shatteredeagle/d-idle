package bot

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
)

func controlRig(t *testing.T, class d2.Class, ctl, mask int) *rig {
	g := newRig(t)
	g.w.set(func(w *world) {
		w.area, w.life, w.class = 5, 30, class
		w.mons = []d2.Unit{{ID: 9, Type: d2.TypeMonster, Class: 10, Mode: d2.MonModeStand, X: 103, Y: 100}}
	})
	g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, ctl, mask
	g.r.poll()
	g.clock.Sleep(time.Second)
	return g
}

// Survive under control: at low life with hostiles near the arbiter's reflex family acts (fights where it stands) and
// the tick counts as controlled. Two classes. The off cases are the must-fail half: knob 0, or the family bit not set,
// leave the chain alone.
func TestArbiterControlSurvive(t *testing.T) {
	for _, class := range []d2.Class{0, 1} {
		g := controlRig(t, class, 1, 3)
		g.r.runGoalShadowed(context.Background(), g.r.latest(), levelGoalControl())
		d, ok := g.r.Arbiter()
		if !ok || d.Chosen == nil || d.Chosen.Source != arbiter.SrcSurvive {
			t.Fatalf("class %d: the arbiter should choose survive: %v", class, d)
		}
		if n := g.r.arb.ctl.Load(); n != 1 {
			t.Fatalf("class %d: controlled ticks %d, want 1", class, n)
		}
		for name, mask := range map[string]int{"knob off": 3, "bit not set": 2} {
			ctl := 1
			if name == "knob off" {
				ctl = 0
			}
			h := controlRig(t, class, ctl, mask)
			h.r.runGoalShadowed(context.Background(), h.r.latest(), levelGoalControl())
			if n := h.r.arb.ctl.Load(); n != 0 {
				t.Fatalf("class %d, %s: the arbiter acted (%d)", class, name, n)
			}
		}
	}
}

func TestArbiterControlOffByDefault(t *testing.T) {
	g := newRig(t)
	if g.r.e.Tune.Bot.ArbiterControl != 0 {
		t.Fatal("bot.arbiterControl must default to 0")
	}
}

func fieldRig(t *testing.T, class d2.Class, ctl, mask int) *rig {
	g := newRig(t)
	g.w.set(func(w *world) { w.area, w.class, w.level = 5, class, 6 })
	g.w.belt()
	g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, ctl, mask
	g.r.poll()
	g.clock.Sleep(time.Second)
	return g
}

// Chores under control: an empty belt in a quiet field, the arbiter picks the refill and the chores hook starts the trip
// (family bit 4 = chores). Two classes; knob off and bit not set are the must-fail half.
func TestArbiterControlChores(t *testing.T) {
	for _, class := range []d2.Class{0, 1} {
		g := fieldRig(t, class, 1, 7)
		g.r.runGoalShadowed(context.Background(), g.r.latest(), levelGoalControl())
		d, ok := g.r.Arbiter()
		if !ok || d.Chosen == nil || arbiter.Family(d.Chosen.Source) != "chores" {
			t.Fatalf("class %d: the arbiter should choose chores: %v", class, d)
		}
		if n := g.r.arb.ctl.Load(); n != 1 || g.r.loop.trip == nil && g.w.count("totown") == 0 {
			t.Fatalf("class %d: controlled ticks %d, no trip started", class, n)
		}
		for name, c := range map[string][2]int{"knob off": {0, 7}, "bit not set": {1, 3}} {
			h := fieldRig(t, class, c[0], c[1])
			h.r.runGoalShadowed(context.Background(), h.r.latest(), levelGoalControl())
			if n := h.r.arb.ctl.Load(); n != 0 {
				t.Fatalf("class %d, %s: the arbiter acted (%d)", class, name, n)
			}
		}
	}
}

// Claim under control: a pending reward in the field, the arbiter picks the claim and the claim hook goes to town.
func TestArbiterControlClaim(t *testing.T) {
	for _, class := range []d2.Class{0, 1} {
		for _, mask := range []int{3 | 1<<3, 3} {
			g := fieldRig(t, class, 1, mask)
			g.w.belt(101, 102, 103, 104)
			g.r.poll()
			g.r.d.Quester = fakeQuester{w: g.w}
			g.r.loop.questPending = map[string]bool{"den_of_evil": true}
			g.r.runGoalShadowed(context.Background(), g.r.latest(), levelGoalControl())
			want := int64(0)
			if mask&(1<<3) != 0 {
				want = 1
			}
			d, _ := g.r.Arbiter()
			if n := g.r.arb.ctl.Load(); n != want {
				t.Logf("decision %+v", d.Chosen)
				t.Fatalf("class %d mask %d: controlled ticks %d, want %d", class, mask, n, want)
			}
		}
	}
}

func campControl() control {
	return control{goal: &goals.Goal{}, spec: goalSpec{kind: goals.Campaign, camp: &task.Campaign{}}}
}

// Goldfarm and recovery under control (bits 16 and 32): the player's farm order and the purse recovery are the chosen
// family and their hooks run; the order beats recovery when both are on, as in the chain. Two classes; the bit not set is
// the must-fail half. U: the fake world only proves the hook is reached, not the farm itself.
func TestArbiterControlGoldFarmAndRecovery(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, tc := range []struct {
			name        string
			mask        int
			farm, broke bool
			want        string
			ctl         int64
		}{
			{"order", 3 | 16, true, false, "goldfarm", 1},
			{"order bit not set", 3, true, false, "goldfarm", 0},
			{"recovery", 3 | 32, false, true, "recovery", 1},
			{"recovery bit not set", 3, false, true, "recovery", 0},
			{"order beats recovery", 3 | 16 | 32, true, true, "goldfarm", 1},
		} {
			g, gf := farmRig(t, 10, class)
			// Stand in the farm zone (Stony Field, TestGoldFarmTargetBothClasses). From another area the guard goroutine
			// raced the fake walk: when it looked first it cancelled the slice as "crossed an area border" (~5% under load).
			g.w.set(func(w *world) { w.area = 4 })
			g.w.belt(101, 102, 103, 104)
			g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, 1, tc.mask
			if tc.farm {
				gf.Start()
			}
			if tc.broke {
				g.w.belt()
				g.r.loop.broke.on = true
			}
			g.r.poll()
			g.clock.Sleep(time.Second)
			g.r.runGoalShadowed(context.Background(), g.r.latest(), campControl())
			d, ok := g.r.Arbiter()
			if !ok || d.Chosen == nil || arbiter.Family(d.Chosen.Source) != tc.want {
				t.Fatalf("%v %s: chose %+v, want %s", class, tc.name, d.Chosen, tc.want)
			}
			if n := g.r.arb.ctl.Load(); n != tc.ctl {
				t.Fatalf("%v %s: controlled ticks %d, want %d", class, tc.name, n, tc.ctl)
			}
		}
	}
}

// Quest under control (bit 64): the arbiter wants a refill, the chores hook is not under control and the trip cooldown
// holds the chain's own chores back, so the chain would walk the quest step with an empty belt (incident 0319). With the
// bit the quest step does not start and the hero levels meanwhile; without it (or with the knob off) the chain is
// untouched. Two classes. U: the fake world only proves the quest step is held, not what the leveling does.
func TestArbiterControlQuestHeldWhileRefillIsWanted(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, tc := range []struct {
			name      string
			ctl, mask int
			want      int64
		}{{"bit set", 1, 3 | 64, 1}, {"bit not set", 1, 3, 0}, {"knob off", 0, 3 | 64, 0}} {
			g, _ := farmRig(t, 6, class)
			g.w.set(func(w *world) { w.gold0 = 900 })
			g.w.belt()
			g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, tc.ctl, tc.mask
			g.r.setLastTrip(g.clock.Now())
			g.r.poll()
			g.clock.Sleep(time.Second)
			g.r.runGoalShadowed(context.Background(), g.r.latest(), campControl())
			if n := g.r.arb.ctl.Load(); n != tc.want {
				d, _ := g.r.Arbiter()
				t.Logf("chosen %+v why %s", d.Chosen, g.r.currentWhy())
				t.Fatalf("%v %s: held %d, want %d", class, tc.name, n, tc.want)
			}
		}
	}
}

// A standing typed refusal silences the quest source until a wake fact changes (arbRefused records the facts).
func TestQuestSourceReadsTheTypedRefusal(t *testing.T) {
	g, _ := farmRig(t, 6, d2.Sorceress)
	g.r.e.Tune.Bot.ArbiterShadow = 1
	g.r.poll()
	s := g.r.latest()
	if g.r.questRefusalStands(s) {
		t.Fatal("no refusal yet")
	}
	g.r.arbRefused("not_ready", "any of level, kit, gold or area changes")
	if !g.r.questRefusalStands(s) {
		t.Fatal("the refusal must stand while the wake facts are unchanged")
	}
	g.w.set(func(w *world) { w.level = 7 })
	g.r.poll()
	if g.r.questRefusalStands(g.r.latest()) {
		t.Fatal("a level change is a wake")
	}
}

// Level under control (bit 128): a standing typed refusal silences the quest source, the arbiter chooses the filler and
// the quest step is replaced by a leveling slice. Two classes; bit not set and knob off must not act.
func TestArbiterControlLevel(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, tc := range []struct {
			name      string
			ctl, mask int
			want      int64
		}{{"bit set", 1, 3 | 128, 1}, {"bit not set", 1, 3, 0}, {"knob off", 0, 3 | 128, 0}} {
			g, _ := farmRig(t, 6, class)
			g.w.set(func(w *world) { w.gold0 = 900 })
			g.w.belt(101, 102, 103, 104)
			g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, tc.ctl, tc.mask
			g.r.poll()
			g.r.arbRefused("not_ready", "any of level, kit, gold or area changes")
			g.clock.Sleep(time.Second)
			g.r.runGoalShadowed(context.Background(), g.r.latest(), campControl())
			if n := g.r.arb.ctl.Load(); n != tc.want {
				d, _ := g.r.Arbiter()
				t.Fatalf("%v %s: level ticks %d, want %d (chose %+v)", class, tc.name, n, tc.want, d.Chosen)
			}
		}
	}
}
