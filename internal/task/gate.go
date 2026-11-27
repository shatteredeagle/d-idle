package task

import (
	"fmt"

	"d2lod-bot/internal/env"
)

// BossGate is the readiness a boss quest needs before the planner hands it to the executor (WP1 of
// docs/soak-analysis.md). The campaign floor is the level part; the rest is the kit. A character that fails only the kit
// farms the next level instead and the bot's own town trips restock it.
type BossGate struct {
	MinLevel  int
	Potions   int  // healing potions in belt and pack
	NeedTP    bool // a town portal scroll or tome
	ManaPots  int  // mana potions a caster needs (0 = not checked)
	Antidotes int  // antidote potions wanted; only checked while the character can afford them (antidoteGold)
	Andy      bool // MinLevel is the Andariel knob, see andarielFloor
	Route     bool // the walk is the Monastery road: the route guard (route.go) applies on top
	Barracks  bool // the Smith's Barracks: the class floors World.BarracksMinLevel and BarracksMinLife apply on top
	BR        bool // Blood Raven: the floor, kit counts and gold come from World.BR (brFloor, brShort), not from the fields above
}

// BRKnobs are the Blood Raven gate numbers (Tuning.Boss.BR*); a zero field takes the default below.
type BRKnobs struct{ CasterLevel, MeleeLevel, Potions, ManaPotions, TPCharges, Gold int }

// Blood Raven defaults (docs/sme/blood-raven-caster.md section 7; U: judgment). Same values as env.DefaultBossTuning.
const (
	brCasterLevel, brMeleeLevel, brPotions, brManaPotions, brTPCharges, brGold = 9, 8, 4, 2, 2, 250
)

func orDef(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
}

// brFloor is the level a character needs for Blood Raven: the class floor, or the wait level of a failed attempt.
func brFloor(w World) int {
	f := orDef(w.BR.MeleeLevel, brMeleeLevel)
	if w.BRHigh {
		f = orDef(w.BR.CasterLevel, brCasterLevel)
	}
	return max(f, w.BRWaitLevel)
}

// brShort says what the Blood Raven kit lacks ("" = complete). Gold is separate (brGoldShort): a missing kit is fixed by
// a chores trip, missing gold by farming.
func brShort(w World) string {
	k := w.BR
	switch {
	case w.Potions < orDef(k.Potions, brPotions):
		return fmt.Sprintf("%d healing potions < %d", w.Potions, orDef(k.Potions, brPotions))
	case !w.HasTP:
		return "no town portal"
	case w.TPKnown && w.TPCharges < orDef(k.TPCharges, brTPCharges):
		return fmt.Sprintf("%d portal charges < %d", w.TPCharges, orDef(k.TPCharges, brTPCharges))
	case w.Caster && w.ManaPotions < orDef(k.ManaPotions, brManaPotions):
		return fmt.Sprintf("%d mana potions < %d (caster)", w.ManaPotions, orDef(k.ManaPotions, brManaPotions))
	}
	return ""
}

// brGoldShort says what the gold reserve lacks ("" = enough).
func brGoldShort(w World) string {
	if g := orDef(w.BR.Gold, brGold); w.Gold < g {
		return fmt.Sprintf("gold %d < %d", w.Gold, g)
	}
	return ""
}

// BRLine is the log line of the Blood Raven gate: `blood raven: gate clvl=N kit=ok|short(why)`.
func BRLine(w World) string {
	kit := "ok"
	if why := brShort(w); why != "" {
		kit = "short(" + why + ")"
	} else if why := brGoldShort(w); why != "" {
		kit = "short(" + why + ")"
	}
	return fmt.Sprintf("blood raven: gate clvl=%d kit=%s", w.Level, kit)
}

// antidoteGold is the gold from which a missing antidote counts as a kit shortfall (an antidote costs about 20 gold at
// Akara; U: price not read from the install here). Below it the gate passes without them.
const antidoteGold = 150

// MercAlive is the optional merc state reader hook (M3 fills it): alive is meaningful only when known. The default,
// nil, means unknown and the merc rule passes. Set it once at start-up, before the bot runs.
var MercAlive func() (alive bool, known bool)

// andarielFloor is the clvl the campaign wants before it tries Andariel: the no-merc knob (World.AndarielMinLevel,
// default env.DefaultAndarielMinLevel), or with a living merc the lower World.AndarielMinLevelMerc (0 = the no-merc
// floor). A dead or unknown merc counts as none: the old +2 for a dead merc is replaced by the no-merc floor (ADR 0011).
func andarielFloor(w World) int {
	f := w.AndarielMinLevel
	if f <= 0 {
		f = env.DefaultAndarielMinLevel
	}
	if routeMercOK(w) && w.AndarielMinLevelMerc > 0 {
		f = min(f, w.AndarielMinLevelMerc)
	}
	return f
}

// AndarielWarn is the why-line of a character that passes the Andariel level floor but is under the class's recommended
// level: a warning, not a wall ("" = at or over it, or no recommendation).
func AndarielWarn(w World) string {
	rec := w.AndarielRec
	if routeMercOK(w) && w.AndarielRecMerc > 0 {
		rec = min(rec, w.AndarielRecMerc)
	}
	if rec <= 0 || w.Level >= rec || w.Level < andarielFloor(w) {
		return ""
	}
	return fmt.Sprintf("going on below the recommended level %d (clvl %d, merc=%s)", rec, w.Level, MercState(w))
}

// andarielHeld reports the unattended hold: the campaign may not enter Andariel while the knob is on, the fight has not
// been won once (ADR 0011 gate 3) and she is not dead.
func andarielHeld(w World) bool {
	return !TryActive() && w.AndarielHold && normalDiff(w) && !w.Killed[BossAndy] && !QuestDone(w, QAndariel)
}

const andarielHeldWhy = "unattended hold: the throwaway fight has not won yet (boss.andarielHold)"

// normalDiff: the knob is a Normal rule; Nightmare and Hell keep their static floors (nightmare.go).
func normalDiff(w World) bool { return w.Difficulty == "" || w.Difficulty == "normal" }

func (g BossGate) floor(w World) int {
	if TryActive() {
		return 0 // the owner's try anyway: the level floor is a readiness word
	}
	f := g.baseFloor(w)
	if g.Route {
		f = max(f, routeFloor(w))
	}
	if g.Barracks && normalDiff(w) {
		f = max(f, w.BarracksMinLevel)
	}
	return f
}

// barracksShort says why the Barracks is closed to w beyond the route guard ("" = open): the class level floor, then the
// max life floor (only while the life is readable). The text is what the board shows as the hold.
func (g BossGate) barracksShort(w World) string {
	if !g.Barracks {
		return ""
	}
	return BarracksWhy(w)
}

func (g BossGate) baseFloor(w World) int {
	if g.BR && normalDiff(w) {
		return brFloor(w)
	}
	if g.Andy && normalDiff(w) {
		if andarielHeld(w) {
			return max(andarielFloor(w), w.Level+1)
		}
		return andarielFloor(w)
	}
	return g.MinLevel
}

// AndarielLine is the log line of the Andariel gate for w ("" when the character is not in the Normal Act 1 band):
// the level verdict and, once the level passes, whether the kit is complete.
func AndarielLine(w World) string {
	g := bossGates[QAndariel]
	f := g.floor(w)
	if andarielHeld(w) {
		return fmt.Sprintf("boss: held for andariel (%s) clvl=%d floor=%d merc=%s", andarielHeldWhy, w.Level, AndarielFloor(w), MercState(w))
	}
	if w.Level < f {
		return fmt.Sprintf("boss: held for andariel (clvl %d < floor %d)", w.Level, f)
	}
	kit := "ok"
	if why := g.kitShort(w); why != "" {
		kit = "short(" + why + ")"
	}
	line := fmt.Sprintf("boss: andariel gate pass clvl=%d kit=%s", w.Level, kit)
	if warn := AndarielWarn(w); warn != "" {
		line += " warning: " + warn
	}
	return line
}

// bossGates is keyed by quest id. Numbers: Burial Grounds is the Soaky soak case (V, 2026-10-29 log 11:05 to 11:12: clvl
// 6, belt 0, 5 attempts, 11 potions in 358 s, 0 kills, the only death; 181 more aborted attempts at clvl 4 to 5). The floor
// 8 and the kit of 4 potions plus a TP are U: a judgment from that log, not measured (runs.json has no Blood Raven row;
// the boss script's own floor, internal/boss/act1.go, is 6). The later quests keep the floors and potion counts their
// Ready gates already enforce (act1.go) and add the TP.
var bossGates = map[string]BossGate{
	QBurial: {MinLevel: brMeleeLevel, Potions: brPotions, NeedTP: true, ManaPots: brManaPotions, BR: true},
	// Barracks / Monastery Gate: V, G1 attempts 11 to 13 died there 10 times (docs/deathautopsy.md), once with 2 minor
	// potions and no scroll. Kit of 4 healing potions, a scroll or tome and 2 mana potions for a caster is a supply
	// gate (U: the counts are judgment), not a level floor.
	QTools:    {MinLevel: 8, Potions: 4, NeedTP: true, ManaPots: 2, Route: true, Barracks: true},
	QTower:    {MinLevel: 14, Potions: 4, NeedTP: true, ManaPots: 2},
	QAndariel: {MinLevel: env.DefaultAndarielMinLevel, Potions: minPotions, NeedTP: true, ManaPots: 3, Antidotes: 2, Andy: true, Route: true, Barracks: true},
}

// NotReady says why w may not start the boss quest q ("" = ready or no gate for q).
func (g BossGate) NotReady(w World) string {
	if g.Andy && andarielHeld(w) {
		return "andariel " + andarielHeldWhy
	}
	if g.Route && w.Level < routeFloor(w) {
		return "route guard: blocked monastery (" + RouteLevelWhy(w) + ")"
	}
	if why := g.barracksShort(w); why != "" {
		return why
	}
	if w.Level < g.floor(w) {
		return fmt.Sprintf("clvl %d < %d", w.Level, g.floor(w))
	}
	if g.Route {
		if why := RouteKitWhy(w); why != "" {
			return "route guard: kit (" + why + ")"
		}
	}
	if g.BR && normalDiff(w) {
		if why := brShort(w); why != "" {
			return why
		}
		return brGoldShort(w)
	}
	return g.kitShort(w)
}

// kitShort is the supply part of the gate: "" when the kit is complete.
func (g BossGate) kitShort(w World) string {
	switch {
	case w.Potions < g.Potions:
		return fmt.Sprintf("%d healing potions < %d", w.Potions, g.Potions)
	case g.NeedTP && !w.HasTP:
		return "no town portal"
	case g.ManaPots > 0 && w.Caster && w.ManaPotions < g.ManaPots:
		return fmt.Sprintf("%d mana potions < %d (caster)", w.ManaPotions, g.ManaPots)
	case g.Antidotes > 0 && w.Antidotes < g.Antidotes && w.Gold >= antidoteGold:
		return fmt.Sprintf("%d antidotes < %d", w.Antidotes, g.Antidotes)
	}
	return ""
}

// GateFor returns the gate of quest q.
func GateFor(q string) (BossGate, bool) { g, ok := bossGates[q]; return g, ok }

// gateReady is the Ready check of a boss quest's travel step: too low a level is a danger fault (the planner levels first),
// a missing kit inserts a chores trip, and when the chores cannot fix it the executor blocks the quest and the planner
// falls to the next entry or the level step ("blocked: not ready").
func gateReady(q string) func(World) *Fault {
	g := bossGates[q]
	return func(w World) *Fault {
		if f := gatedReady(w); f != nil {
			return f
		}
		why := g.NotReady(w)
		switch {
		case why == "":
			return nil
		case w.Level < g.floor(w), g.barracksShort(w) != "":
			return refusal(w, "too_weak", "not ready: "+why)
		case g.BR && normalDiff(w) && brShort(w) == "":
			// only gold is short: a chores trip buys nothing, the planner levels and farms first (recovery)
			return refusal(w, "needs_gold_farm", "not ready: "+why)
		}
		return &Fault{Kind: KindResource, Why: "not ready: " + why, Insert: []Step{StepChores(id(q, "chores"))}}
	}
}

// isTown reports the five act towns (levels.txt ids, V: world.go constants).
func isTown(a Area) bool {
	return a == RogueEncampment || a == LutGholein || a == 75 || a == 103 || a == 109
}

// fieldReady is the Ready of every step that ends in a field area: the death-spiral guards. A gated character (no weapon,
// no body armor, or no potions and no gold) may only go to the safe farm area, and nobody walks into an area the
// death-spiral breaker banned. Both are danger faults: the executor blocks the quest and the planner levels (the bot's
// leveling slice then farms the safe area) until the world changes. V: Soaky 2026-10-29 walked a naked character through
// Underground Passage Level 1 four times.
func fieldReady(a Area) func(World) *Fault {
	return func(w World) *Fault {
		if isTown(a) {
			return nil
		}
		switch {
		case w.Banned[a]:
			why := "area banned after repeated deaths"
			if n := w.BanNote[a]; n != "" {
				why = "area banned after " + n
			}
			return refusal(w, "area_banned", "not ready: "+why)
		case w.RouteBlocked != "" && RouteGuarded(a) && !(!isTown(w.Area) && w.Area != 0 && RouteLeaveWhy(w) == ""):
			// a hero already out of town (any area, it passed the gate when it left) goes on with a kit that merely thinned: RouteLeaveWhy
			return guardFault(w, "route guard", "route", w.RouteBlocked, RouteLevelWhy(w) == "")
		case w.TowerBlocked != "" && TowerGuarded(a):
			return guardFault(w, "tower guard", "ftower", w.TowerBlocked, TowerLevelWhy(w) == "")
		case w.WalkBlocked.Why != "" && !isTown(a) && a != w.FirstField:
			return guardFault(w, "quest walk", "walk", w.WalkBlocked.Why, w.WalkBlocked.KitOnly)
		case w.EarlyBlocked[a].Why != "":
			// the early road (Underground Passage level 1, Dark Wood): a hold, never a strike (Questy bed 2026-11-25: the quest walk was
			// refused at the hop, counted as "too weak" three times and parked for 1h30m)
			return guardFault(w, "early road", "early", w.EarlyBlocked[a].Why, w.EarlyBlocked[a].KitOnly)
		case w.Gated && a != w.SafeArea:
			if w.BoardSM && w.GatedFix == "potions" {
				// bug 4: a missing potion kit is fixed by a town visit, so the hero goes now and nothing is counted
				return &Fault{Kind: KindResource, Code: "needs_potions", Why: "not ready: " + w.GatedWhy,
					Insert: []Step{StepChores(fmt.Sprintf("field.%d/chores", a))}}
			}
			return refusal(w, gatedCode(w), "not ready: "+w.GatedWhy)
		}
		return nil
	}
}

// refusal is the danger fault of a guard. With the quest board's state machine (World.BoardSM) it also carries the
// reason code and is a hold; without it the fault is exactly the old one.
func refusal(w World, code, why string) *Fault {
	f := &Fault{Kind: KindDanger, Why: why}
	if w.BoardSM {
		f.Code, f.Hold = code, true
	}
	return f
}

// gatedCode is the code of a gear gate that no town visit fixes: gold first when the purse and the potions are both
// empty, else the hero is too weak for the area.
func gatedCode(w World) string {
	if w.Potions == 0 && w.Gold == 0 {
		return "needs_gold_farm"
	}
	return "too_weak"
}

// guardFault is the fault of a closed guarded road. A level shortage is danger (the planner levels first); a kit-only
// shortage (potions, portal, mana) is a resource fault that inserts a chores trip, as gateReady does for the boss
// quests. Before, a hero at 3 healing potions of 4 was refused with danger, the planner went on leveling and the kit
// was never restocked (Proving Ground act1more, act1.tools.malus Paladin 11).
func guardFault(w World, name, stepID, why string, kitOnly bool) *Fault {
	msg := "not ready: " + name + " (" + why + ")"
	if !kitOnly {
		return refusal(w, "route_closed", msg)
	}
	if k := w.KitPlan; k != nil && k.Short != "" {
		// the plan cannot close the line with the gold owned: a chores trip would end "ran but still needed", so say what is
		// missing. With the board SM this is the hold: the plan's open line is the reason (gold first).
		if k.Hand != "" {
			return refusal(w, "needs_gold_farm", msg+"; "+k.Hand)
		}
		return refusal(w, "needs_gold_farm", msg+"; "+k.Short)
	}
	f := &Fault{Kind: KindResource, Why: msg, Insert: []Step{StepChores("guard/" + stepID + "/chores")}}
	if w.BoardSM {
		f.Code = "needs_potions" // a fixable kit shortage is a restock trip, never counted
	}
	return f
}

// gatedReady is the Ready of the quest steps that start a boss quest (greeting, waypoint, walk): a gear-gated character
// cannot finish the quest, so it must not start it and loop through talk, skip and retry in town (LIVE G1 attempt 4:
// talk-charsi retry/skipped, then 'Go to the Barracks' not ready, twice). The planner levels it in its safe area instead.
func gatedReady(w World) *Fault {
	if w.Gated {
		if w.BoardSM && w.GatedFix == "potions" {
			return &Fault{Kind: KindResource, Code: "needs_potions", Why: "not ready: " + w.GatedWhy, Insert: []Step{StepChores("gate.chores")}}
		}
		return refusal(w, gatedCode(w), "not ready: "+w.GatedWhy)
	}
	return nil
}
