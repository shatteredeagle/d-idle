package bot

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/task"
)

func controlRig(t *testing.T, class d2.Class, ctl, mask int) *rig {
	g := newRig(t)
	g.w.set(func(w *world) {
		w.area, w.life, w.class = 5, 30, class
		w.mons = []d2.Unit{{ID: 9, Type: d2.TypeMonster, Class: 10, Mode: d2.MonModeStand, X: 103, Y: 100}}
	})
	g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, ctl, mask
	g.fight.safety = func(*d2.Snapshot) SafetyAction { return SafetyDrinkLife }
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
			if tc.farm {
				g.w.set(func(w *world) { w.area = 4 })
			}
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

// A1 (fable review part 7): a broke hero below the resume line with an empty belt and nothing hostile near walks home
// for the free heal under control too, as the chain's low-life gate does before recovery or the gold farm order. Before
// the fix the controlled hook started the farm slice. Two classes; a healthy hero still farms (the gate must not eat the
// family).
func TestArbiterControlFarmHonoursTheLowLifeGate(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, tc := range []struct {
			name string
			mask int
			farm bool
			life int
			home bool
		}{
			{"recovery hurt", 3 | 32, false, 20, true},
			{"order hurt", 3 | 16, true, 20, true},
			{"recovery healthy", 3 | 32, false, 100, false},
		} {
			g, gf := farmRig(t, 10, class)
			g.w.set(func(w *world) { w.area, w.life = 4, tc.life })
			g.w.belt()
			g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, 1, tc.mask
			if tc.farm {
				gf.Start()
			} else {
				g.r.loop.broke.on = true
			}
			g.r.poll()
			g.clock.Sleep(time.Second)
			g.r.runGoalShadowed(context.Background(), g.r.latest(), campControl())
			d, _ := g.r.Arbiter()
			if home := g.r.currentWhy() == WhyHealTrip; home != tc.home {
				t.Fatalf("%v %s: heal trip %v, want %v (chose %+v, why %v, totown %d)", class, tc.name, home, tc.home, d.Chosen, g.r.currentWhy(), g.w.count("totown"))
			}
		}
	}
}

// A3 (fable review part 7): the loop is closed. A controlled hook that declines (chores on the trip cooldown) reports
// the failure, so the arbiter backs the intent off and chooses something else on the next tick, and the declined
// family does not veto the quest step. Before the fix the same tick was a 40 min "arbiter holds the quest step" slice
// and the arbiter chose chores again. Two classes; the uncontrolled chores family (mask without bit 4) still vetoes.
func TestArbiterControlDeclinedHookReportsAndDoesNotVeto(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, tc := range []struct {
			name string
			mask int
			held int64
		}{{"chores controlled, declines", 3 | 4 | 64, 0}, {"chores not controlled", 3 | 64, 1}} {
			g, _ := farmRig(t, 6, class)
			g.w.set(func(w *world) { w.gold0 = 900 })
			g.w.belt()
			g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, 1, tc.mask
			g.r.setLastTrip(g.clock.Now())
			g.r.poll()
			g.clock.Sleep(time.Second)
			g.r.runGoalShadowed(context.Background(), g.r.latest(), campControl())
			d, _ := g.r.Arbiter()
			if d.Chosen == nil || arbiter.Family(d.Chosen.Source) != "chores" {
				t.Fatalf("%v %s: setup, the arbiter should want the refill: %+v", class, tc.name, d.Chosen)
			}
			if n := g.r.arb.ctl.Load(); n != tc.held {
				t.Fatalf("%v %s: held or acted %d, want %d (why %s)", class, tc.name, n, tc.held, g.r.currentWhy())
			}
			if tc.held != 0 {
				continue
			}
			g.r.poll()
			g.clock.Sleep(time.Second)
			g.r.setLastTrip(g.clock.Now())
			g.r.runGoalShadowed(context.Background(), g.r.latest(), campControl())
			if d, _ := g.r.Arbiter(); d.Chosen != nil && arbiter.Family(d.Chosen.Source) == "chores" {
				t.Fatalf("%v: the declined refill was chosen again one second later (no back-off): %+v", class, d.Chosen)
			}
		}
	}
}

// A3: the chain declined recovery on purpose (gear gate, or a board chunk that needs no gold: recoveryDue) before it
// asks arbHoldsQuest; the arbiter's choice of recovery or the gold farm must not turn that chunk into a leveling slice.
// Another family (an uncontrolled refill) still holds the quest step. Two classes.
func TestArbiterRecoveryChoiceDoesNotVetoTheChunkTheChainAllows(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g, _ := farmRig(t, 6, class)
		g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, 1, 3|64
		for fam, want := range map[string]bool{arbiter.SrcRecovery: false, arbiter.SrcGoldFarm: false, "chores": true, arbiter.SrcQuest: false} {
			g.r.arb.chose = fam
			if got := g.r.arbHoldsQuest(); got != want {
				t.Fatalf("%v: arbiter chose %s: holds the quest step %v, want %v", class, fam, got, want)
			}
		}
	}
}

// A2 (fable review part 7): survive has one danger rule, the fighter's safety verdict, and yields to a running
// emergency trip. Life under the old drink line with a quiet verdict is not danger; a verdict at any life is; with a trip running
// the trip goes on (the chain's order) instead of a sweep per tick. Two classes. Before the fix the first and third case
// chose survive.
func TestArbiterSurviveReadsTheSafetyVerdictAndYieldsToATrip(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, tc := range []struct {
			name    string
			verdict SafetyAction
			trip    bool
			survive bool
		}{
			{"low life, verdict none", SafetyNone, false, false},
			{"verdict drink, hostile near", SafetyDrinkLife, false, true},
			{"verdict rejuv, hostile near", SafetyDrinkRejuv, false, true},
			{"verdict mana only", SafetyDrinkMana, false, false},
			{"verdict chicken, trip running", SafetyChicken, true, false},
		} {
			g := controlRig(t, class, 1, 1)
			g.fight.safety = func(*d2.Snapshot) SafetyAction { return tc.verdict }
			if tc.trip {
				g.r.loop.trip = &trip{plan: ChorePlan{Heal: true, Reason: "test"}, emergency: true}
			}
			g.r.runGoalShadowed(context.Background(), g.r.latest(), levelGoalControl())
			d, _ := g.r.Arbiter()
			if got := d.Chosen != nil && d.Chosen.Source == arbiter.SrcSurvive; got != tc.survive {
				t.Fatalf("%v %s: survive chosen %v, want %v (%+v)", class, tc.name, got, tc.survive, d.Chosen)
			}
			if tc.trip && g.r.arb.ctl.Load() != 0 {
				t.Fatalf("%v %s: the survive hook ran over the trip", class, tc.name)
			}
		}
	}
}

// Review 11, N1: a declined refill (trip cooldown) backs the upkeep intent off. An emergency trip that then runs in
// town, hero hurt and broke, must go on: before the fix the arbiter refused the backed-off upkeep intent, chose recovery
// and its hook walked the hero out to farm unhealed. The direct half asks the farm hooks themselves: whatever the arbiter
// chose, a running trip goes on and due chores start (the chain's prefix). Two classes.
func TestArbiterControlBackoffNeverTakesARunningTrip(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g, _ := farmRig(t, 6, class)
		g.w.set(func(w *world) { w.gold0 = 900 })
		g.w.belt()
		// the upkeep source (one key for "the plan would buy" and "a trip runs") needs a town with a planner
		g.r.d.Town = fixedPlanner{Townsman: g.r.d.Town, pl: supply.Plan{Lines: []supply.PlanLine{{Line: supply.Healing, Have: 1, Min: 3, Buy: 2}}}}
		g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, 1, 255
		g.r.setLastTrip(g.clock.Now())
		g.r.poll()
		g.clock.Sleep(time.Second)
		g.r.runGoalShadowed(context.Background(), g.r.latest(), campControl())
		if d, _ := g.r.Arbiter(); d.Chosen == nil || d.Chosen.Source != arbiter.SrcUpkeep || g.r.arb.ctl.Load() != 0 {
			t.Fatalf("%v: setup, the upkeep hook should be chosen and decline: %+v (acted %d)", class, d.Chosen, g.r.arb.ctl.Load())
		}
		g.w.set(func(w *world) { w.area, w.life = 1, 30 })
		g.r.loop.broke.on = true
		g.r.loop.trip = &trip{plan: ChorePlan{Heal: true, Reason: "test"}, emergency: true}
		g.r.poll()
		g.clock.Sleep(time.Second)
		g.r.runGoalShadowed(context.Background(), g.r.latest(), campControl())
		d, _ := g.r.Arbiter()
		if d.Chosen == nil || arbiter.Family(d.Chosen.Source) != "chores" {
			t.Fatalf("%v: the running emergency trip was not chosen: %+v, rejected %+v", class, d.Chosen, d.Rejected)
		}
		if why := g.r.currentWhy(); why == WhyFarmGold || g.w.count("chores") == 0 {
			t.Fatalf("%v: the trip did not go on (why %s, chores %d)", class, why, g.w.count("chores"))
		}
	}
}

func TestArbiterFarmHooksHonourTheTripPrefix(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, fam := range []string{arbiter.SrcRecovery, arbiter.SrcGoldFarm} {
			for _, running := range []bool{true, false} {
				g, gf := farmRig(t, 6, class)
				g.w.set(func(w *world) { w.gold0 = 900 })
				g.w.belt()
				g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, 1, 255
				if fam == arbiter.SrcGoldFarm {
					gf.Start()
				} else {
					g.r.loop.broke.on = true
				}
				if running {
					g.w.set(func(w *world) { w.area, w.life = 1, 30 })
					g.r.loop.trip = &trip{plan: ChorePlan{Heal: true, Reason: "test"}, emergency: true}
				}
				g.r.poll()
				g.clock.Sleep(time.Second)
				d := arbiter.Decision{Chosen: &arbiter.Intent{Source: fam, Kind: "farm_gold", Class: arbiter.Objective}}
				_, took := g.r.arbAct(context.Background(), g.r.latest(), campControl(), d, 255)
				why := g.r.currentWhy()
				if !took || why == WhyFarmGold || g.w.count("chores")+g.w.count("totown") == 0 {
					t.Fatalf("%v %s trip running %v: took %v why %s chores %d totown %d: the farm ran over the trip or the due chores",
						class, fam, running, took, why, g.w.count("chores"), g.w.count("totown"))
				}
			}
		}
	}
}

// Review 11, N3: survive is "a drink verdict with a hostile near", nothing else. A quiet hurt hero (the verdict drinks
// or asks for the portal with nothing in the snapshot) and a leave verdict (portal, chicken, crowd retreat or abandon)
// must not get the survive hook's full sweep of the area: the chain's own step runs (a plain chores trip goes on) and
// the guard carries the verdict out. Before the fix every case here chose survive and started a fight. Two classes.
func TestArbiterSurviveNeedsAHostileAndNeverSweepsOnALeaveVerdict(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, tc := range []struct {
			name    string
			verdict SafetyAction
			quiet   bool
			trip    bool
		}{
			{"quiet, drink verdict, plain trip running", SafetyDrinkLife, true, true},
			{"quiet, portal verdict", SafetyTownPortal, true, false},
			{"hostile, portal verdict", SafetyTownPortal, false, false},
			{"hostile, chicken", SafetyChicken, false, false},
			{"hostile, crowd retreat", SafetyCrowdRetreat, false, false},
			{"hostile, crowd abandon", SafetyCrowdAbandon, false, false},
		} {
			g := controlRig(t, class, 1, 1)
			g.w.set(func(w *world) {
				w.life = 40
				if tc.quiet {
					w.mons = nil
				}
			})
			g.w.belt(101)
			g.fight.safety = func(*d2.Snapshot) SafetyAction { return tc.verdict }
			if tc.trip {
				g.r.loop.trip = &trip{plan: ChorePlan{Heal: true, Reason: "test"}}
			}
			g.r.poll()
			g.r.runGoalShadowed(context.Background(), g.r.latest(), levelGoalControl())
			d, _ := g.r.Arbiter()
			if d.Chosen != nil && d.Chosen.Source == arbiter.SrcSurvive || g.r.arb.ctl.Load() != 0 {
				t.Fatalf("%v %s: survive chosen (%+v), controlled ticks %d", class, tc.name, d.Chosen, g.r.arb.ctl.Load())
			}
			if tc.trip && (g.r.currentWhy() == WhyFight || g.w.count("totown") == 0) {
				t.Fatalf("%v %s: the trip did not go on (why %s, calls %s)", class, tc.name, g.r.currentWhy(), g.w.log())
			}
		}
	}
}
