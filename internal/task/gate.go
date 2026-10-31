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

// andarielHeld reports the unattended hold: the campaign may not enter Andariel while the knob is on, the fight has not
// been won once (ADR 0011 gate 3) and she is not dead.
func andarielHeld(w World) bool {
	return w.AndarielHold && normalDiff(w) && !w.Killed[BossAndy] && !QuestDone(w, QAndariel)
}

const andarielHeldWhy = "unattended hold: the throwaway fight has not won yet (boss.andarielHold)"

// normalDiff: the knob is a Normal rule; Nightmare and Hell keep their static floors (nightmare.go).
func normalDiff(w World) bool { return w.Difficulty == "" || w.Difficulty == "normal" }

func (g BossGate) floor(w World) int {
	f := g.baseFloor(w)
	if g.Route {
		f = max(f, routeFloor(w))
	}
	return f
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
		return "boss: held for andariel (" + andarielHeldWhy + ")"
	}
	if w.Level < f {
		return fmt.Sprintf("boss: held for andariel (clvl %d < floor %d)", w.Level, f)
	}
	kit := "ok"
	if why := g.kitShort(w); why != "" {
		kit = "short(" + why + ")"
	}
	return fmt.Sprintf("boss: andariel gate pass clvl=%d kit=%s", w.Level, kit)
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
	QTools:    {MinLevel: 8, Potions: 4, NeedTP: true, ManaPots: 2, Route: true},
	QTower:    {MinLevel: 14, Potions: 4, NeedTP: true, ManaPots: 2},
	QAndariel: {MinLevel: env.DefaultAndarielMinLevel, Potions: minPotions, NeedTP: true, ManaPots: 3, Antidotes: 2, Andy: true, Route: true},
}

// NotReady says why w may not start the boss quest q ("" = ready or no gate for q).
func (g BossGate) NotReady(w World) string {
	if g.Andy && andarielHeld(w) {
		return "andariel " + andarielHeldWhy
	}
	if g.Route && w.Level < routeFloor(w) {
		return "route guard: blocked monastery (" + RouteLevelWhy(w) + ")"
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
		case w.Level < g.floor(w):
			return &Fault{Kind: KindDanger, Why: "not ready: " + why}
		case g.BR && normalDiff(w) && brShort(w) == "":
			// only gold is short: a chores trip buys nothing, the planner levels and farms first (recovery)
			return &Fault{Kind: KindDanger, Why: "not ready: " + why}
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
			return &Fault{Kind: KindDanger, Why: "not ready: area banned after repeated deaths"}
		case w.RouteBlocked != "" && RouteGuarded(a):
			return &Fault{Kind: KindDanger, Why: "not ready: route guard (" + w.RouteBlocked + ")"}
		case w.TowerBlocked != "" && TowerGuarded(a):
			return &Fault{Kind: KindDanger, Why: "not ready: tower guard (" + w.TowerBlocked + ")"}
		case w.Gated && a != w.SafeArea:
			return &Fault{Kind: KindDanger, Why: "not ready: " + w.GatedWhy}
		}
		return nil
	}
}

// gatedReady is the Ready of the quest steps that start a boss quest (greeting, waypoint, walk): a gear-gated character
// cannot finish the quest, so it must not start it and loop through talk, skip and retry in town (LIVE G1 attempt 4:
// talk-charsi retry/skipped, then 'Go to the Barracks' not ready, twice). The planner levels it in its safe area instead.
func gatedReady(w World) *Fault {
	if w.Gated {
		return &Fault{Kind: KindDanger, Why: "not ready: " + w.GatedWhy}
	}
	return nil
}
