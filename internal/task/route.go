package task

import "fmt"

// Route guard and the optional Imbue quest (SME decision EXP after Soaky died three times on the same stretch of the
// Monastery road, docs/sme/play-safety.md). Both are pure functions of the World; a zero knob field disables the rule,
// so a World built without the bot (tests, sims) keeps the old behaviour. The bot fills the knobs from Tuning.Campaign
// and Tuning.Route (defaults 20 and on).

// Monastery road: Tamoe Highland (7), Monastery Gate (26), Outer Cloister (27), Barracks (28), Jail 1 to 3 (29-31), Inner
// Cloister (32), Cathedral (33) and Catacombs 1 to 4 (34-37), the way to Andariel. V: levels.txt ids, same list as the
// progression-normal steps.
const (
	routeFirstArea = 26
	routeLastArea  = 37
	tamoeHighland  = 7
)

// RouteGuarded reports whether a is on the guarded Monastery road.
func RouteGuarded(a Area) bool {
	return a == tamoeHighland || (a >= routeFirstArea && a <= routeLastArea)
}

// Tower guard (TOWERGUARD, G1 attempt 17: Soaky, clvl 14 caster, no merc, empty belt, died twice in the Tower Cellar for
// the Countess, a no-reward quest in Normal, docs/plan/plan-v2.md 10.5): the Forgotten Tower (20) and Tower Cellar 1 to
// 5 (21-25). V: levels.txt ids, same as TowerCellar5 = 25 in world.go.
const (
	towerFirstArea = 20
	towerLastArea  = 25
)

// TowerGuarded reports whether a is the Forgotten Tower or one of its cellars.
func TowerGuarded(a Area) bool { return a >= towerFirstArea && a <= towerLastArea }

// AreaGuarded reports whether a is under the Monastery or the tower guard.
func AreaGuarded(a Area) bool { return RouteGuarded(a) || TowerGuarded(a) }

// towerFloor is the clvl the tower needs: 0 when the guard is off or the difficulty is not Normal. A living merc opens
// it at any level with TowerNeedsMerc (the old rule), else lowers the floor to TowerMinLevelMerc (0 = no lowering).
func towerFloor(w World) int {
	if w.TowerMinLevel <= 0 || !normalDiff(w) {
		return 0
	}
	if routeMercOK(w) {
		if w.TowerNeedsMerc {
			return 0
		}
		if w.TowerMinLevelMerc > 0 {
			return min(w.TowerMinLevelMerc, w.TowerMinLevel)
		}
	}
	return w.TowerMinLevel
}

// TowerLevelWhy says why the tower is closed on level grounds ("" = open): clvl below towerFloor.
func TowerLevelWhy(w World) string {
	f := towerFloor(w)
	if w.Level >= f {
		return ""
	}
	return levelWhy(w, f)
}

// levelWhy words a level verdict: `no merc, clvl 14 < 16` or, with a living merc, `clvl 13 < 14 (with merc)`.
func levelWhy(w World, f int) string {
	if routeMercOK(w) {
		return fmt.Sprintf("clvl %d < %d (with merc)", w.Level, f)
	}
	return fmt.Sprintf("no merc, clvl %d < %d", w.Level, f)
}

// MercState is the merc word of the floor log lines: alive, dead (the reader knows it died) or none (no merc, unknown).
func MercState(w World) string {
	switch {
	case routeMercOK(w):
		return "alive"
	case w.MercKnown:
		return "dead"
	}
	return "none"
}

// FloorLine is the log line of the effective floor of one guard: `route guard: monastery floor=16 (merc=none)`.
func FloorLine(gate string, floor int, w World) string {
	return fmt.Sprintf("route guard: %s floor=%d (merc=%s)", gate, floor, MercState(w))
}

// MonasteryFloor, TowerFloor and AndarielFloor are the effective floors of the three guards for w (0 = guard off or
// not Normal); the bot logs them with FloorLine.
func MonasteryFloor(w World) int { return routeFloor(w) }
func TowerFloor(w World) int     { return towerFloor(w) }
func AndarielFloor(w World) int {
	if !normalDiff(w) {
		return 0
	}
	return max(andarielFloor(w), routeFloor(w))
}

// TowerKitWhy is the supply the tower needs on top (same kit as the Monastery road); "" = complete or guard off.
func TowerKitWhy(w World) string {
	if w.TowerMinLevel <= 0 || !normalDiff(w) {
		return ""
	}
	return kitWhy(w)
}

// TowerWhy is the full tower verdict: "" = open, else level or kit.
func TowerWhy(w World) string {
	if why := TowerLevelWhy(w); why != "" {
		return why
	}
	return TowerKitWhy(w)
}

// TowerSkipWhy says why the optional Forgotten Tower quest is skipped ("" = do it): level grounds only, a thin kit is a
// chores trip, not a skip.
func TowerSkipWhy(w World) string { return TowerLevelWhy(w) }

// TowerLine is the log line of the tower verdict: `route guard: blocked tower (why)`, "" when the guard is off or open.
func TowerLine(w World) string {
	if why := TowerWhy(w); why != "" {
		return "route guard: blocked tower (" + why + ")"
	}
	return ""
}

// Route kit: the supplies the road needs on top of the level or merc condition (U: judgment).
const (
	routeHealing = 4
	routePortals = 2
	routeMana    = 2
)

func routeMercOK(w World) bool { return w.HasMerc || (w.MercKnown && w.MercAlive) }

// routeFloor is the clvl the road needs: 0 when the guard is off or the difficulty is not Normal; with a living merc
// RouteMinLevelMerc (0 = any level, the old rule), else RouteMinLevel.
func routeFloor(w World) int {
	if w.RouteMinLevel <= 0 || !normalDiff(w) {
		return 0
	}
	if routeMercOK(w) {
		return min(w.RouteMinLevelMerc, w.RouteMinLevel)
	}
	return w.RouteMinLevel
}

// RouteLevelWhy says why the road is closed on level grounds ("" = open): clvl below routeFloor.
func RouteLevelWhy(w World) string {
	if f := routeFloor(w); w.Level < f {
		return levelWhy(w, f)
	}
	return ""
}

// RouteKitWhy says what supply the road lacks ("" = complete or guard off). A chores trip fixes it.
func RouteKitWhy(w World) string {
	if w.RouteMinLevel <= 0 || !normalDiff(w) {
		return ""
	}
	return kitWhy(w)
}

func kitWhy(w World) string {
	switch {
	case w.Potions < routeHealing:
		return fmt.Sprintf("%d healing potions < %d", w.Potions, routeHealing)
	case !w.HasTP:
		return "no town portal"
	case w.TPKnown && w.TPCharges < routePortals:
		return fmt.Sprintf("%d portal charges < %d", w.TPCharges, routePortals)
	case w.Caster && w.ManaPotions < routeMana:
		return fmt.Sprintf("%d mana potions < %d (caster)", w.ManaPotions, routeMana)
	}
	return ""
}

// RouteWhy is the full verdict: "" = the road is open, else the reason (level or kit).
func RouteWhy(w World) string {
	if why := RouteLevelWhy(w); why != "" {
		return why
	}
	return RouteKitWhy(w)
}

// RouteLine is the log line of the guard verdict: `route guard: blocked monastery (why)` or `route guard: open`.
func RouteLine(w World) string {
	if w.RouteMinLevel <= 0 {
		return "route guard: off"
	}
	if why := RouteWhy(w); why != "" {
		return "route guard: blocked monastery (" + why + ")"
	}
	return "route guard: open"
}

// ImbueSkipWhy says why the optional Imbue quest (Tools of the Trade: Barracks Malus and the Charsi imbue) is skipped
// ("" = do it). Never for non-melee classes (Sorceress, Necromancer, Druid, Amazon: World.ImbueUseful false); melee
// classes only from ImbueMinLevel and, with ImbueNeedsMerc, with a living merc.
func ImbueSkipWhy(w World) string {
	if w.ImbueMinLevel <= 0 || !normalDiff(w) {
		return ""
	}
	switch {
	case !w.ImbueUseful:
		return "class has no use for the imbue"
	case w.Level < w.ImbueMinLevel:
		return fmt.Sprintf("clvl %d < %d", w.Level, w.ImbueMinLevel)
	case w.ImbueNeedsMerc && !routeMercOK(w):
		return "no merc"
	}
	return ""
}
