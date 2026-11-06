package nav

import (
	"testing"

	"d2lod-bot/internal/input"
)

// g2e: the wanted exit lies behind an unwanted one; the clip crept to the boundary and the next click was refused
// ("nav: no path ... unwanted exit", 104 / 295 / 47 per seed). The gate must walk around the disc in a few clicks.
func TestGateGoesAroundTheDisc(t *testing.T) {
	for _, c := range []struct {
		name     string
		from, to input.Point
	}{
		{"west to east through the Den", denAt(-40, 0), denAt(45, 0)},
		{"north to south, off centre", denAt(3, -38), denAt(-2, 50)},
		{"starting on the boundary", denAt(-15, 1), denAt(45, 2)},
	} {
		_, n, _ := gateNav(t, c.from)
		lm, exits := n.gatedExits()
		ex := mp(denExit.Pos)
		pos := c.from
		for i := 0; i < gateMaxRewrites && !clickOKAll(pos, c.to, exits); i++ {
			q, how := n.gateRewrite(lm, pos, c.to, exits)
			if how == "" {
				t.Fatalf("%s: held at %v (step %d)", c.name, pos, i)
			}
			if !clickOKAll(pos, q, exits) {
				t.Fatalf("%s: %s click %v from %v breaks the invariant", c.name, how, q, pos)
			}
			pos = q
		}
		if !clickOKAll(pos, c.to, exits) {
			t.Fatalf("%s: still blocked at %v (%.1f from the exit) after the rewrite cap", c.name, pos, dist(pos, ex))
		}
	}
}

// A target inside the bound has no way through: the gate still holds.
func TestGateDetourNotForATargetInsideTheBound(t *testing.T) {
	_, n, _ := gateNav(t, denAt(-30, 0))
	lm, exits := n.gatedExits()
	if _, how := n.gateRewrite(lm, denAt(-30, 0), denAt(3, 0), exits); how == "detour" {
		t.Fatal("a target inside the bound must not be reached by a detour")
	}
}
