package combat

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

func tgAt(id uint32, x, y int) Target { return Target{Unit: mon(id, clsFallen, x, y), Name: "t"} }

// profileAim is the aim row of a shipped profile: the tests read the data, they do not restate it.
func profileAim(t *testing.T, class d2.Class, build, skill string) AimRule {
	t.Helper()
	pr, _ := SelectProfile(&d2.Player{Class: class}, testCatalog(), Build(build))
	for _, r := range pr.Aims {
		if r.Name == skill {
			return r
		}
	}
	t.Fatalf("profile %v/%q has no aim row %q", class, build, skill)
	return AimRule{}
}

// a lone hostile near the hero, a pack of four farther out
func strayAndPack() (Target, []Target) {
	stray := tgAt(1, 108, 100)
	return stray, []Target{stray, tgAt(2, 120, 110), tgAt(3, 121, 111), tgAt(4, 119, 111), tgAt(5, 120, 112)}
}

func TestAimClusterTwoBuilds(t *testing.T) {
	for _, c := range []struct {
		class        d2.Class
		build, skill string
	}{{d2.Sorceress, "blizzard", "Blizzard"}, {d2.Druid, "fire", "Eruption"}} {
		r := profileAim(t, c.class, c.build, c.skill)
		stray, hs := strayAndPack()
		out := aimPoint(r, aimIn{Me: input.Point{X: 100, Y: 100}, Target: stray, Hostiles: hs, Range: 30})
		if out.Refused != "" || out.Hits < 2 || pdist(out.At, input.Point{X: 120, Y: 111}) > 2 {
			t.Errorf("%s: aim %+v, want the pack (hits >= 2) near 120,111", c.skill, out)
		}
	}
}

func TestAimRules(t *testing.T) {
	me := input.Point{X: 100, Y: 100}
	tg := tgAt(1, 110, 100)
	cases := []struct {
		name string
		r    AimRule
		in   aimIn
		want input.Point
	}{
		{"lead", AimRule{Rule: AimLead, LeadFrames: 25}, aimIn{Vel: func(uint32) (float64, float64, bool) { return 0, 4, true }}, input.Point{X: 110, Y: 104}},
		{"lead unknown", AimRule{Rule: AimLead, LeadFrames: 25}, aimIn{}, input.Point{X: 110, Y: 100}},
		{"offset past", AimRule{Rule: AimOffset, Offset: 3}, aimIn{}, input.Point{X: 113, Y: 100}},
		{"offset short", AimRule{Rule: AimOffset, Offset: -4}, aimIn{}, input.Point{X: 106, Y: 100}},
		{"range clamp", AimRule{Rule: AimOffset, Offset: 10}, aimIn{Range: 15}, input.Point{X: 115, Y: 100}},
		{"packEdge", AimRule{Rule: AimPackEdge, Radius: 2}, aimIn{Hostiles: []Target{tg, tgAt(2, 116, 100)}}, input.Point{X: 112, Y: 100}},
	}
	for _, c := range cases {
		c.in.Me, c.in.Target = me, tg
		if out := aimPoint(c.r, c.in); out.At != c.want || out.Refused != "" {
			t.Errorf("%s: %+v, want %v", c.name, out, c.want)
		}
	}
}

// Point scoring is a gate, and the line rule falls back to the target or refuses.
func TestAimGateAndLOS(t *testing.T) {
	me, tg := input.Point{X: 100, Y: 100}, tgAt(1, 110, 100)
	out := aimPoint(AimRule{Rule: AimCluster, CrowdAtLeast: 2}, aimIn{Me: me, Target: tg})
	if out.Refused != aimRefusedCrowd {
		t.Fatalf("lone target passed a crowd gate of 2: %+v", out)
	}
	blockFar := func(a, b input.Point) bool { return b.X <= 110 }
	out = aimPoint(AimRule{Rule: AimOffset, Offset: 3}, aimIn{Me: me, Target: tg, LOS: blockFar})
	if !out.Fallback || out.At != unitPoint(tg.Unit) || out.Refused != "" {
		t.Fatalf("blocked point did not fall back to the target: %+v", out)
	}
	none := func(a, b input.Point) bool { return false }
	if out = aimPoint(AimRule{Rule: AimOffset, Offset: 3}, aimIn{Me: me, Target: tg, LOS: none}); out.Refused != aimRefusedLOS {
		t.Fatalf("no line at all not refused: %+v", out)
	}
	if out = aimPoint(AimRule{Rule: AimOffset, Offset: 3, LOS: AimLOSNone}, aimIn{Me: me, Target: tg, LOS: none}); out.Refused != "" {
		t.Fatalf("los none refused: %+v", out)
	}
	wall := func(p input.Point) bool { return p.X != 113 }
	if out = aimPoint(AimRule{Rule: AimOffset, Offset: 3, LOS: AimLOSWalk}, aimIn{Me: me, Target: tg, Walk: wall}); !out.Fallback {
		t.Fatalf("los walk aimed into a wall: %+v", out)
	}
}

func TestAimValidate(t *testing.T) {
	sk := loadClassData().skills
	ok := AimRule{Name: "Blizzard", Rule: AimCluster, Source: "U"}
	bad := map[string]AimRule{
		"no point":  {Name: "Zeal", Rule: AimCluster, Source: "U"}, // a melee build has nothing to aim
		"unknown":   {Name: "Blizzard", Rule: "spiral", Source: "U"},
		"los":       {Name: "Blizzard", Rule: AimCluster, LOS: "radar", Source: "U"},
		"lead":      {Name: "Blizzard", Rule: AimLead, Source: "U"},
		"source":    {Name: "Blizzard", Rule: AimCluster},
		"no skill":  {Name: "Nope", Rule: AimCluster, Source: "U"},
		"radius":    {Name: "Blizzard", Rule: AimCluster, Radius: -1, Source: "U"},
		"twice key": ok,
	}
	if err := validateAims([]Profile{{Class: "sorceress", Aims: []AimRule{ok}}}, sk); err != nil {
		t.Fatal(err)
	}
	for why, r := range bad {
		rows := []AimRule{r}
		if why == "twice key" {
			rows = append(rows, r)
		}
		if err := validateAims([]Profile{{Class: "paladin", Aims: rows}}, sk); err == nil || !strings.Contains(err.Error(), "aim row") {
			t.Errorf("%s: err %v", why, err)
		}
	}
}

func aimHarness(t *testing.T, class d2.Class, build string, mode int) (*harness, *d2.Snapshot) {
	t.Helper()
	h := newHarness(t)
	h.f.Build = Build(build)
	h.f.e.Tune.Combat.DeliveryUpkeep = mode
	h.f.sight = func(a, b input.Point) bool { return true }
	snap := baseSnap()
	snap.Player.Class, snap.Player.Skills = class, nil // the class picks the profile, not a Sorceress skill
	h.f.mv = nil                                       // every cell reachable
	return h, snap
}

// Scenario combat.point-spell-hits (offline form; the scenario runner is not built yet): with the knob at 2 the
// Sorceress and Druid rows aim at the pack, at 1 the old aim acts and the move is counted, at 0 nothing changes. A
// Paladin (melee, no rows) is never re-aimed.
func TestAimForModes(t *testing.T) {
	stray, hs := strayAndPack()
	cell := unitPoint(stray.Unit)
	for _, c := range []struct {
		class        d2.Class
		build, skill string
	}{{d2.Sorceress, "blizzard", "Blizzard"}, {d2.Druid, "fire", "Eruption"}} {
		sp := SkillSpec{Name: c.skill, Range: 30}
		for mode, moved := range []bool{false, false, true} {
			h, snap := aimHarness(t, c.class, c.build, mode)
			at, ok := h.f.aimFor(snap, sp, stray, hs, true)
			if !ok || (at != cell) == !moved {
				t.Errorf("%s mode %d: at %v ok %v, moved want %v", c.skill, mode, at, ok, moved)
			}
			n := h.f.AimCounts()
			if mode == 1 && n.ShadowMoved != 1 || mode == 2 && n.Aimed != 1 || n.Refused != 0 {
				t.Errorf("%s mode %d: counts %+v", c.skill, mode, n)
			}
		}
	}
	h, snap := aimHarness(t, d2.Paladin, "", 2)
	for _, name := range []string{"Zeal", "Blessed Hammer", "Attack"} {
		if at, ok := h.f.aimFor(snap, SkillSpec{Name: name, Range: 20}, stray, hs, true); !ok || at != cell {
			t.Errorf("paladin %s re-aimed to %v", name, at)
		}
	}
	if n := h.f.AimCounts(); n != (AimCounts{}) {
		t.Errorf("paladin counts %+v", n)
	}
}

// refused_at_most: a refusal is counted, and the watchdog path (gate false) never refuses for the crowd.
func TestAimRefusedCounted(t *testing.T) {
	h, snap := aimHarness(t, d2.Sorceress, "blizzard", 2)
	h.f.sight = func(a, b input.Point) bool { return false }
	sp, tg := SkillSpec{Name: "Blizzard", Range: 30}, tgAt(1, 110, 100)
	// los none: a blocked line does not matter to Blizzard
	if _, ok := h.f.aimFor(snap, sp, tg, nil, true); !ok {
		t.Fatal("Blizzard refused for a line it does not need")
	}
	h, snap = aimHarness(t, d2.Druid, "fire", 2)
	h.f.sight = func(a, b input.Point) bool { return false }
	sp.Name = "Eruption"
	if _, ok := h.f.aimFor(snap, sp, tg, nil, true); ok {
		t.Fatal("Eruption cast with no line to any cell")
	}
	if at, ok := h.f.aimFor(snap, sp, tg, nil, false); !ok || at != unitPoint(tg.Unit) {
		t.Fatalf("watchdog refused: %v %v", at, ok)
	}
	if n := h.f.AimCounts(); n.Refused != 2 {
		t.Fatalf("refused %d, want 2", n.Refused)
	}
}

// Velocity memory: two sightings a second apart give the lead its speed; a stale sighting gives none.
func TestAimVelocity(t *testing.T) {
	h, _ := aimHarness(t, d2.Sorceress, "blizzard", 2)
	a := &h.f.aim
	a.observe(h.f.now(), []Target{tgAt(7, 100, 100)})
	if _, _, ok := a.vel(7); ok {
		t.Fatal("velocity from one sighting")
	}
	h.s.Sleep(500 * 1e6)
	a.observe(h.f.now(), []Target{tgAt(7, 102, 100)})
	if vx, vy, ok := a.vel(7); !ok || vx != 4 || vy != 0 {
		t.Fatalf("vel %v %v %v, want 4 0", vx, vy, ok)
	}
	h.s.Sleep(3 * 1e9)
	a.observe(h.f.now(), []Target{tgAt(7, 120, 100)})
	if _, _, ok := a.vel(7); ok {
		t.Fatal("velocity from a stale sighting")
	}
}
