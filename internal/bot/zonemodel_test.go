package bot

import (
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/task"
)

// modelRig is a runner with the model knob on and a scripted ZoneDecide.
func modelRig(t *testing.T, cl d2.Class, dec func(area int, cands []ZoneCand) (ZoneVerdict, error)) (*rig, *int) {
	t.Helper()
	g := farmLoopRig(t, cl, 10)
	calls := new(int)
	g.r.e.Tune.Bot.ZoneModel = 1
	g.r.d.ZoneDecide = func(s *d2.Snapshot, area int, diff string, potions int, dial string, cands []ZoneCand) (ZoneVerdict, error) {
		*calls++
		if dial != "balanced" {
			t.Errorf("dial %q, want the default balanced", dial)
		}
		return dec(area, cands)
	}
	return g, calls
}

func (g *rig) modelPick(level int, cleared map[int]bool) (Step, bool) {
	s := g.r.latest()
	ex := map[int]bool{}
	for a := range cleared {
		ex[a] = true
	}
	o := PickOptions{Acts: map[int]bool{1: true}, Exclude: ex, Cleared: cleared, FullKit: true, Class: planClass(s.Player.Class), Melee: MeleeTank(s.Player.Class)}
	o.Model = g.r.zoneModelPick(s, "normal", cleared)
	return g.r.d.Plan.Pick("normal", level, o)
}

func legacy(g *rig, level int) Step {
	st, _ := g.r.d.Plan.Pick("normal", level, PickOptions{Acts: map[int]bool{1: true}, FullKit: true, Class: planClass(g.r.latest().Player.Class), Melee: MeleeTank(g.r.latest().Player.Class)})
	return st
}

func TestZoneModelPicksLowerZoneThanLegacy(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(cl.String(), func(t *testing.T) {
			var offered []ZoneCand
			low := 0
			g, _ := modelRig(t, cl, func(area int, c []ZoneCand) (ZoneVerdict, error) {
				offered = c
				return ZoneVerdict{Area: low, Margin: "safe", Line: "zone choice: test"}, nil
			})
			leg := legacy(g, 12)
			if leg.Area == 2 {
				t.Skip("legacy already picks the lowest zone")
			}
			low = 2 // the margin is negative further out: the model walks back to the first field
			st, ok := g.modelPick(12, nil)
			if !ok || st.Area != 2 || st.Area >= leg.Area {
				t.Fatalf("model %d (%v), legacy %d: want the lower zone", st.Area, ok, leg.Area)
			}
			if len(offered) == 0 {
				t.Fatal("the controller saw no candidates")
			}
			if w, _ := g.r.ZoneMargin(); w != "safe" {
				t.Errorf("status word %q", w)
			}
		})
	}
}

func TestZoneModelFallsBackOnErrorAndOff(t *testing.T) {
	g, calls := modelRig(t, d2.Paladin, func(int, []ZoneCand) (ZoneVerdict, error) { return ZoneVerdict{}, errors.New("no profile") })
	want := legacy(g, 12)
	for i := 0; i < 3; i++ {
		if st, ok := g.modelPick(12, nil); !ok || st.Area != want.Area {
			t.Fatalf("error: got %d (%v), want the plan's %d", st.Area, ok, want.Area)
		}
	}
	if *calls != 3 {
		t.Errorf("asked %d times", *calls)
	}
	// an area the plan does not offer: also the plan rule
	g.r.d.ZoneDecide = func(*d2.Snapshot, int, string, int, string, []ZoneCand) (ZoneVerdict, error) {
		return ZoneVerdict{Area: 9999}, nil
	}
	if st, _ := g.modelPick(12, nil); st.Area != want.Area {
		t.Errorf("unknown area: got %d", st.Area)
	}
	g.r.e.Tune.Bot.ZoneModel = 0 // knob off: the hook is never built
	if g.r.zoneModelPick(g.r.latest(), "normal", nil) != nil {
		t.Error("model hook built with the knob off")
	}
}

func TestZoneModelFreshGameAndDepleted(t *testing.T) {
	g, _ := modelRig(t, d2.Sorceress, func(area int, c []ZoneCand) (ZoneVerdict, error) {
		dep := false
		for _, z := range c {
			dep = dep || z.Depleted
		}
		if !dep {
			t.Error("cleared zone not offered as depleted")
		}
		return ZoneVerdict{Fresh: true}, nil
	})
	leg := legacy(g, 12)
	if _, ok := g.modelPick(12, map[int]bool{leg.Area: true}); ok {
		t.Fatal("fresh verdict must report nothing eligible so the guarded fresh game starts")
	}
	// nothing cleared or no out-of-game flow: no fresh game, the plan decides
	g.r.d.ZoneDecide = func(*d2.Snapshot, int, string, int, string, []ZoneCand) (ZoneVerdict, error) {
		return ZoneVerdict{Fresh: true}, nil
	}
	if _, ok := g.modelPick(12, nil); !ok {
		t.Error("fresh with nothing cleared must fall back to the plan")
	}
}

// Hysteresis: the pick is kept for the dwell; the model is asked again only after it.
func TestZoneModelDwell(t *testing.T) {
	g, _ := modelRig(t, d2.Paladin, func(int, []ZoneCand) (ZoneVerdict, error) { return ZoneVerdict{Area: 2}, nil })
	if g.r.zoneRecheckDue() {
		t.Fatal("due before any verdict")
	}
	g.modelPick(8, nil)
	if g.r.zoneRecheckDue() {
		t.Fatal("due at once: the zone would thrash")
	}
	g.clock.Sleep(zoneRecheck + time.Second)
	if !g.r.zoneRecheckDue() {
		t.Fatal("not due after the dwell")
	}
	g.r.e.Tune.Bot.ZoneModel = 0
	if g.r.zoneRecheckDue() {
		t.Fatal("due with the model off")
	}
}

func TestZoneModelMercLeftDeadAndDial(t *testing.T) {
	g, _ := modelRig(t, d2.Paladin, func(int, []ZoneCand) (ZoneVerdict, error) {
		return ZoneVerdict{Area: 2, MercDead: true, MercRevive: false, MercWhy: "the zone kills him"}, nil
	})
	if g.r.zoneLeaveMercDead() {
		t.Fatal("no verdict yet")
	}
	g.modelPick(8, nil)
	if !g.r.zoneLeaveMercDead() {
		t.Fatal("the model said leave him dead")
	}
	g.clock.Sleep(5 * zoneRecheck)
	if g.r.zoneLeaveMercDead() {
		t.Fatal("a stale verdict must not keep suppressing the revive")
	}
	for i, id := range zoneDialIDs {
		g.r.e.Tune.Bot.ZoneDial = i
		if g.r.zoneDial() != id {
			t.Errorf("dial %d is %q", i, g.r.zoneDial())
		}
	}
	g.r.e.Tune.Bot.ZoneDial = 9
	if g.r.zoneDial() != "balanced" {
		t.Error("out of range dial is not balanced")
	}
}

// The boss margin lowers the Andariel wall only when it is safe; risky or too hard keeps the legacy floor and warns; no
// data or an explicit knob leaves everything alone.
func TestZoneBossFloor(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(cl.String(), func(t *testing.T) {
			g, _ := modelRig(t, cl, nil)
			lvl := g.r.latest().Player.Level
			base := func() task.World { return task.World{AndarielMinLevel: lvl + 8, AndarielMinLevelMerc: lvl + 6} }
			run := func(word string, ok bool, bt env.BossTuning) task.World {
				g.r.d.BossMargin = func(*d2.Snapshot, string, string, string) (string, bool) { return word, ok }
				w := base()
				g.r.zoneBossFloor(&w, bt)
				return w
			}
			if w := run("safe", true, env.BossTuning{}); w.AndarielMinLevel != lvl || w.AndarielMinLevelMerc != lvl {
				t.Errorf("safe: floors %d/%d, want the level %d", w.AndarielMinLevel, w.AndarielMinLevelMerc, lvl)
			}
			for _, word := range []string{"risky", "too hard"} {
				if w := run(word, true, env.BossTuning{}); w.AndarielMinLevel != lvl+8 || w.AndarielRec != lvl+1 {
					t.Errorf("%s: floor %d rec %d: legacy floor stays and the warning is on", word, w.AndarielMinLevel, w.AndarielRec)
				}
			}
			if w := run("safe", false, env.BossTuning{}); w.AndarielMinLevel != lvl+8 {
				t.Error("no margin data: the legacy floor must stay")
			}
			if w := run("safe", true, env.BossTuning{AndarielMinLevel: 30}); w.AndarielMinLevel != lvl+8 {
				t.Error("an explicit knob floor must stay")
			}
			g.r.e.Tune.Bot.ZoneModel = 0
			if w := run("safe", true, env.BossTuning{}); w.AndarielMinLevel != lvl+8 {
				t.Error("knob off: legacy floor")
			}
		})
	}
}

func TestZoneKnobsDefault(t *testing.T) {
	d := env.DefaultTuning().Bot
	if d.ZoneModel != 0 || d.ZoneDial != 1 {
		t.Fatalf("defaults zoneModel %d zoneDial %d, want 0 and 1 until the A/B passes", d.ZoneModel, d.ZoneDial)
	}
}

// The revive loop breaker: three quick merc deaths in the planned zone close it to reviving even though the model says
// "revive"; a long life resets; the knob 0 turns it off. Same rule for both classes.
func TestZoneMercReviveLoopBreaker(t *testing.T) {
	for _, cl := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(cl.String(), func(t *testing.T) {
			g, _ := modelRig(t, cl, func(int, []ZoneCand) (ZoneVerdict, error) {
				return ZoneVerdict{Area: 2, MercDead: true, MercRevive: true, MercWhy: "worth reviving"}, nil
			})
			g.modelPick(8, nil)
			p := g.r.latest().Player
			p.Area = 2
			edge := func(alive bool) { g.r.zoneMercEdge(p, alive, g.r.e.Clock.Now()) }
			quickDeath := func() { // revived, lives a minute, dies
				edge(false)
				g.clock.Sleep(time.Second)
				edge(true)
				g.clock.Sleep(time.Minute)
				edge(false)
			}
			edge(true)
			quickDeath() // each call: the previous death, a revive, a minute of life, the next death (the first has no revive before it)
			quickDeath()
			if g.r.zoneLeaveMercDead() {
				t.Fatal("two quick deaths are below the limit")
			}
			quickDeath()
			if !g.r.zoneLeaveMercDead() {
				t.Fatal("three quick deaths must stop the revive")
			}
			g.clock.Sleep(61 * time.Minute)
			g.modelPick(8, nil) // fresh verdict (the old one went stale)
			if g.r.zoneLeaveMercDead() {
				t.Fatal("the closure expires")
			}
		})
	}
	g, _ := modelRig(t, d2.Paladin, func(int, []ZoneCand) (ZoneVerdict, error) { return ZoneVerdict{Area: 2}, nil })
	g.r.e.Tune.Bot.MercReviveLoopN = 0
	p := g.r.latest().Player
	p.Area = 2
	for i := 0; i < 5; i++ {
		g.r.zoneMercEdge(p, true, g.r.e.Clock.Now())
		g.r.zoneMercEdge(p, false, g.r.e.Clock.Now())
		g.r.zoneMercEdge(p, true, g.r.e.Clock.Now())
		g.clock.Sleep(time.Second)
	}
	if g.r.mercLoop().BlockedArea(0, g.r.e.Clock.Now()) {
		t.Fatal("N 0 is off")
	}
}

// holereach: the picker never offers an area the hero cannot reach, and a hole fallback needs the kit and logs once an hour.
func TestZoneModelDropsUnreachableCandidates(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(cl.String(), func(t *testing.T) {
			var offered []ZoneCand
			g, _ := modelRig(t, cl, func(area int, c []ZoneCand) (ZoneVerdict, error) {
				offered = c
				return ZoneVerdict{Area: 2, Margin: "safe", Line: "zone choice: test"}, nil
			})
			want := legacy(g, 12).Area
			g.mover.routeFn = func(from, to int) ([]int, error) {
				if to == want && from != to {
					return nil, errors.New("no route")
				}
				return covMoverRoute(g, from, to)
			}
			g.modelPick(12, nil)
			for _, c := range offered {
				if c.Area == want && want != g.r.latest().Player.Area {
					t.Fatalf("unreachable area %d was offered: %v", want, offered)
				}
			}
		})
	}
}

func covMoverRoute(g *rig, from, to int) ([]int, error) {
	if from == to {
		return nil, nil
	}
	return []int{to}, nil
}

func TestZoneHoleNeedsKitAndLogsOncePerHour(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(cl.String(), func(t *testing.T) {
			g, _ := modelRig(t, cl, func(area int, c []ZoneCand) (ZoneVerdict, error) {
				return ZoneVerdict{Area: 3, Hole: true, Margin: "too hard", Line: "zone choice: hole test"}, nil
			})
			g.w.set(func(w *world) { w.area = 2 })
			g.r.poll()
			g.w.belt()
			g.r.poll()
			if g.r.fullKit(g.r.latest()) {
				t.Skip("rig cannot express a short kit")
			}
			if st, ok := g.modelPick(12, nil); ok && st.Area == 3 {
				t.Fatalf("hole fallback to area 3 with a short kit must not be chosen")
			}
			if _, ok := g.r.zst.noted1["holekit:3"]; !ok {
				t.Fatal("refusal not noted")
			}
			g.r.zoneNoteHour("hole:3", "x")
			n := len(g.r.zst.noted1)
			g.r.zoneNoteHour("hole:3", "x")
			if len(g.r.zst.noted1) != n {
				t.Fatal("repeat within the hour must be silent")
			}
			g.clock.Sleep(61 * time.Minute)
			at := g.r.zst.noted1["hole:3"]
			g.r.zoneNoteHour("hole:3", "x")
			if !g.r.zst.noted1["hole:3"].After(at) {
				t.Fatal("note after an hour must log again")
			}
		})
	}
}

// L2 (live review 2026-11-28): a hole names the best of the zones that are left. After bans for the bot's own faults
// that was Tower Cellar Level 4; the hero must not walk to it, and not to a zone with no margin either.
func TestZoneHoleNotWalkedAfterFaultBansOrWithoutMargin(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, tc := range []struct {
			name string
			hold bool
			skip string
			want bool // the hole zone is walked to
		}{{"plain hole", false, "", true}, {"no margin", true, "", false}, {"unreachable ban", false, whyUnreachable, false}, {"death ban", false, whyDeath, true}} {
			t.Run(cl.String()+"/"+tc.name, func(t *testing.T) {
				g, _ := modelRig(t, cl, func(area int, c []ZoneCand) (ZoneVerdict, error) {
					return ZoneVerdict{Area: 3, Hole: true, Hold: tc.hold, Margin: "too hard", Line: "zone choice: hole test"}, nil
				})
				g.w.belt(901, 902, 903, 904)
				g.w.set(func(w *world) {
					w.items = append(w.items, d2.Unit{ID: 900, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
				})
				g.r.poll()
				if !g.r.fullKit(g.r.latest()) && tc.want {
					t.Fatal("rig has a short kit")
				}
				if safe := g.r.safeFarmArea(g.r.latest()); safe == 3 || safe == 4 {
					t.Fatalf("rig: first field %d collides with the test zones", safe)
				}
				ex := map[int]bool{}
				if tc.skip != "" {
					g.r.skipArea(4, time.Minute, tc.skip)
					ex[4] = true
				}
				o := PickOptions{Acts: map[int]bool{1: true}, Exclude: ex, FullKit: true, Class: planClass(cl), Melee: MeleeTank(cl)}
				o.Model = g.r.zoneModelPick(g.r.latest(), "normal", nil)
				st, ok := g.r.d.Plan.Pick("normal", 12, o)
				if got := ok && st.Area == 3; got != tc.want {
					t.Fatalf("picked %+v ok %v, want the hole zone walked to: %v", st.Area, ok, tc.want)
				}
			})
		}
	}
}
