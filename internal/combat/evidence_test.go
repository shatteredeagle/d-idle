package combat

import (
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

func evidenceLines(log string, rule string) []string {
	var out []string
	for _, l := range strings.Split(log, "\n") {
		if strings.Contains(l, `msg="combat evidence"`) && strings.Contains(l, "rule="+rule+" ") {
			out = append(out, l)
		}
	}
	return out
}

func TestEvidenceRateLimit(t *testing.T) {
	h := newHarness(t)
	buf := logTo(h)
	for range 5 {
		h.f.evidence("standOK", "kept", "unit", 7)
	}
	h.f.evidence("standOK", "other")
	if got := evidenceLines(buf.String(), "standOK"); len(got) != 2 || !strings.Contains(got[0], "outcome=kept unit=7") {
		t.Fatalf("want one line per outcome inside the gap: %q", got)
	}
	h.s.Sleep(evidenceGap + time.Second)
	h.f.evidence("standOK", "kept", "unit", 7)
	if got := evidenceLines(buf.String(), "standOK"); len(got) != 3 {
		t.Fatalf("a new line after the gap: %q", got)
	}
}

func TestMissileDodgeEvidence(t *testing.T) {
	missileKinds = func(c int) d2.MissileKind { return d2.MissileFireWall }
	defer func() { missileKinds = nil }()
	snap := &d2.Snapshot{Player: &d2.Player{ID: 1, X: 100, Y: 100, Life: 100, MaxLife: 100}, Missiles: []d2.Unit{
		{ID: 9, Class: 1, OwnerID: 50, OwnerType: 1, X: 102, Y: 100},
	}}
	for _, c := range []struct {
		knob int
		want []string
	}{{0, nil}, {1, []string{"outcome=sidestep", "outcome=held"}}} {
		h := newHarness(t, func(t *env.Tuning) { t.Combat.MissileDodge = c.knob })
		buf := logTo(h)
		n := 0
		h.f.sidestepMissiles(t.Context(), snap, &n) // moves
		h.f.lastDodge = h.f.now()
		h.f.sidestepMissiles(t.Context(), snap, &n) // inside the 800 ms gap
		h.f.sidestepMissiles(t.Context(), &d2.Snapshot{Player: snap.Player}, &n)
		got := evidenceLines(buf.String(), "missileDodge")
		if len(got) != len(c.want) {
			t.Fatalf("knob %d: %q", c.knob, got)
		}
		for i, w := range c.want {
			if !strings.Contains(got[i], w) {
				t.Errorf("knob %d line %d: %q lacks %s", c.knob, i, got[i], w)
			}
		}
	}
}

// The guard's pass in mode 1 and its refusal each write one line, for a caster and a melee class.
func TestPortalGuardEvidenceTwoBuilds(t *testing.T) {
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		h := newHarness(t)
		buf := logTo(h)
		a, in := portalDecision(cls, 1, 200, guardTargets(3, guardMon(3), false))
		h.f.noteCastGuard(in, a)
		if got := evidenceLines(buf.String(), "portalGuard"); len(got) != 1 || !strings.Contains(got[0], "outcome=allowed adjacent=3") || !strings.Contains(got[0], "mode=1") {
			t.Fatalf("%v allowed: %q", cls, buf.String())
		}
		a, in = portalDecision(cls, 1, 200, guardTargets(1, guardMon(120), true))
		h.f.noteCastGuard(in, a)
		if !strings.Contains(buf.String(), "portal cast refused: expected damage") {
			t.Fatalf("%v refused: %q", cls, buf.String())
		}
		buf.Reset()
		a, in = portalDecision(cls, 0, 200, guardTargets(1, guardMon(3), false))
		h.f.noteCastGuard(in, a)
		if buf.Len() != 0 {
			t.Fatalf("%v mode 0 pass must stay silent: %q", cls, buf.String())
		}
	}
}

func TestThreatDodgeEvidence(t *testing.T) {
	h := newHarness(t)
	buf := logTo(h)
	h.f.noteThreat(Incoming{}, false)
	h.f.noteThreat(Incoming{Dodge: []input.Point{{X: 1, Y: 1}}}, true)
	h.f.noteThreat(Incoming{Dodge: []input.Point{{X: 1, Y: 1}}}, false)
	h.f.noteThreat(Incoming{Shooters: []uint32{5}}, false)
	got := evidenceLines(buf.String(), "threatDodge")
	if len(got) != 3 {
		t.Fatalf("%q", got)
	}
	for i, w := range []string{"outcome=dodge missiles=1", "outcome=no-room", "outcome=kill-shooter missiles=0 shooters=1"} {
		if !strings.Contains(got[i], w) {
			t.Errorf("line %d: %q lacks %s", i, got[i], w)
		}
	}
}

func TestStandKept(t *testing.T) {
	fallen, zombie := &data.Monster{AI: "Fallen"}, &data.Monster{AI: "Zombie"}
	if !standKept(1, fallen) || standKept(0, fallen) || standKept(1, zombie) || standKept(1, nil) {
		t.Fatal("only a stander is kept, and only with the knob on")
	}
}
