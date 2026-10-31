package boss

import (
	"strings"
	"time"
)

// Builtin hazard constructors: data for docs/progression/bosses.md section 3, evaluated by the runner.

// Danger kind keys the adapter must produce (U: reading missiles/objects into these is new work).
const (
	DangerPoisonCloud = "poison-cloud" // Andariel's poison missile pools
	DangerFireWall    = "fire-wall"    // Diablo, not used by Act 1 scripts
)

// PoisonStatus drinks an antidote while poisoned and a potion exists (Andariel; prior-art 02: antidotes when
// poison resist is below 75). Cooldown stops a second potion while the first still works.
func PoisonStatus() Hazard {
	return Hazard{Name: "poisoned", Cooldown: 20 * time.Second, Do: Response{Kind: RespAntidote},
		Detect: func(c Ctx) bool { return c.V.Poisoned && c.V.Antidotes > 0 }}
}

// PoisonCloud steps away from a poison pool when the player stands within near subtiles of it.
func PoisonCloud(near, away int) Hazard {
	return Hazard{Name: "poison-cloud", Cooldown: 1500 * time.Millisecond, Do: Response{Kind: RespStepAway, Radius: away},
		Detect: func(c Ctx) bool { _, ok := nearestDanger(c.V, DangerPoisonCloud, near); return ok }}
}

// RaisedAdds fires when at least min non-boss hostiles stand within radius of the player (Blood Raven's raised
// skeletons; Fallen packs) and clears them before the boss is pressed.
func RaisedAdds(min, radius int, budget time.Duration) Hazard {
	return Hazard{Name: "adds", Cooldown: 12 * time.Second, Do: Response{Kind: RespClearAdds, Radius: radius, Budget: budget},
		Detect: func(c Ctx) bool { return countAdds(c, radius) >= min }}
}

// Skeletons fires on raised skeleton minions by name (Blood Raven); U: the name or minion flag of raised units.
func Skeletons(min, radius int, budget time.Duration) Hazard {
	return Hazard{Name: "raised-skeletons", Cooldown: 10 * time.Second, Do: Response{Kind: RespClearAdds, Radius: radius, Budget: budget},
		Detect: func(c Ctx) bool {
			n := 0
			for _, m := range c.V.Hostiles() {
				if (m.Minion || strings.Contains(strings.ToLower(m.Name), "skeleton")) && c.V.Pos.Dist(m.Pos) <= radius && !isBoss(c, m) {
					n++
				}
			}
			return n >= min
		}}
}

// CrowdedAndHurt retreats to the arrival point when many hostiles are close and life is low (Barracks).
func CrowdedAndHurt(min, radius, lifePct int) Hazard {
	return Hazard{Name: "crowded", Cooldown: 8 * time.Second, Do: Response{Kind: RespRetreat},
		Detect: func(c Ctx) bool { return c.V.LifePct() < lifePct && countAdds(c, radius) >= min }}
}

// NoPotionLowLife escapes when life is under pct and nothing is left to drink.
func NoPotionLowLife(pct int) Hazard {
	return Hazard{Name: "no-potion", Do: Response{Kind: RespEscape},
		Detect: func(c Ctx) bool { return c.V.LifePct() < pct && c.V.Potions == 0 }}
}

func isBoss(c Ctx, m Mon) bool { return c.Script.Match.Is(m) }

func countAdds(c Ctx, radius int) int {
	n := 0
	for _, m := range c.V.Hostiles() {
		if !isBoss(c, m) && c.V.Pos.Dist(m.Pos) <= radius {
			n++
		}
	}
	return n
}

func nearestDanger(v View, kind string, within int) (Danger, bool) {
	best, bd := Danger{}, within+1
	for _, d := range v.Dangers {
		if d.Kind != kind {
			continue
		}
		if dist := v.Pos.Dist(d.Pos); dist < bd {
			best, bd = d, dist
		}
	}
	return best, bd <= within
}

// awayPoint is the point r subtiles from p on the ray from `from` through p.
func awayPoint(p, from Pt, r int) Pt {
	dx, dy := float64(p.X-from.X), float64(p.Y-from.Y)
	d := isqrt(dx*dx + dy*dy)
	if d == 0 {
		return Pt{p.X + r, p.Y}
	}
	return Pt{p.X + int(dx/d*float64(r)), p.Y + int(dy/d*float64(r))}
}
