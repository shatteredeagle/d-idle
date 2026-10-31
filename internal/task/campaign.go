package task

import (
	"fmt"
	"strings"

	"d2lod-bot/internal/env"
)

// Entry is one quest in a campaign with its level floor (soft gate, docs/goals/planner.md section 1).
type Entry struct {
	Quest    string
	Title    string
	MinLevel int
	// Act is the act (1-5) whose town and areas the quest needs; 0 = any. The planner does not start an entry of a
	// later act than the character's (a clvl 16 character in Act 1 must finish Andariel before Radament).
	Act int
	// NoKitMinLevel, when above MinLevel, is the floor for a character with neither a town portal nor kitMinPotions
	// healing potions (no way home, no heal): it levels first instead of walking into the quest area.
	NoKitMinLevel int
	// Andy: the floor is the Andariel knob (World.AndarielMinLevel), not MinLevel, which then only documents the default.
	Andy bool
	// BR: the floor is the Blood Raven gate's (class floor and wait level, gate.go brFloor).
	BR bool
	// Route: the quest walks the Monastery road; its floor is raised to the route guard's (route.go routeFloor).
	Route bool
	// Den: the floor is raised to World.DenMinLevel (class-dependent, the bot fills it from campaign.denMinLevel*).
	Den bool
	// Optional: the quest may be skipped by skipWhy (the Imbue quest, the Forgotten Tower).
	Optional bool
}

// kitMinPotions is the healing potion count that counts as a kit.
const kitMinPotions = 3

func (e Entry) floor(w World) int {
	f := e.baseFloor(w)
	if e.Route {
		f = max(f, routeFloor(w))
	}
	return f
}

func (e Entry) baseFloor(w World) int {
	if e.Den && normalDiff(w) {
		return max(w.DenMinLevel, e.MinLevel, e.noKitFloor(w))
	}
	if e.Andy && normalDiff(w) {
		return andarielFloor(w)
	}
	if e.BR && normalDiff(w) {
		return brFloor(w)
	}
	return max(e.MinLevel, e.noKitFloor(w))
}

func (e Entry) noKitFloor(w World) int {
	if e.NoKitMinLevel > e.MinLevel && !w.HasTP && w.Potions < kitMinPotions {
		return e.NoKitMinLevel
	}
	return 0
}

// skipWhy is the reason an optional entry is skipped for w ("" = not skipped).
func (e Entry) skipWhy(w World) string {
	if !e.Optional {
		return ""
	}
	switch e.Quest {
	case QTools:
		return ImbueSkipWhy(w)
	case QTower:
		return TowerSkipWhy(w)
	}
	return ""
}

// Campaign is an ordered quest list. Plan picks the next quest from the world each time (planner.md "next"):
// the first unsatisfied entry whose floor is met; if none qualifies, a level step toward the lowest floor.
type Campaign struct {
	Name    string
	Entries []Entry
	// Difficulty is the difficulty the campaign plays in ("" = any, the Normal templates). A world in another
	// difficulty gets a Stuck plan until the bot has switched (nightmare.go).
	Difficulty string
}

// Act1Options tunes the Normal Act 1 template.
type Act1Options struct {
	Cain      bool // add the Search for Cain chain (off in the critical path, planner.md section 2)
	SkipTools bool
	SkipTower bool
}

// NormalAct1 is the Normal Act 1 campaign: Den, Blood Raven, Smith (clvl 8), Countess (clvl 14), Andariel (Tuning.Boss.AndarielMinLevel, default 15),
// Warriv hop. Floors are from catalog.md / runs.json (U for 6 and 14).
func NormalAct1(o Act1Options) *Campaign {
	c := &Campaign{Name: "normal_act1"}
	add := func(q, title string, min int) {
		c.Entries = append(c.Entries, Entry{Quest: q, Title: title, MinLevel: min, Act: 1, Route: q == QTools || q == QAndariel || q == QAct1Done, Optional: q == QTools || q == QTower})
	}
	add(QDen, "Den of Evil", 1)
	// Found offline 2026-10-29: a kit-less clvl 1-3 character died or stalled in the Den (U: 5 is a judgment, not measured).
	c.Entries[0].NoKitMinLevel = 5
	c.Entries[0].Den = true
	add(QBurial, "Sisters' Burial Grounds", bossGates[QBurial].MinLevel)
	c.Entries[len(c.Entries)-1].BR = true
	if o.Cain {
		add(QCain, "The Search for Cain", 10)
	}
	if !o.SkipTools {
		add(QTools, "Tools of the Trade", 8)
	}
	if !o.SkipTower {
		add(QTower, "The Forgotten Tower", 14)
	}
	add(QAndariel, "Sisters to the Slaughter", env.DefaultAndarielMinLevel)
	add(QAct1Done, "Travel to Act 2", env.DefaultAndarielMinLevel)
	for i := len(c.Entries) - 2; i < len(c.Entries); i++ {
		c.Entries[i].Andy = true
	}
	return c
}

// complete reports whether the entry needs nothing more this session.
func (c *Campaign) complete(w World, s Session, q string) bool {
	if QuestDone(w, q) || (s != nil && s.Finished(q)) {
		return true
	}
	for _, n := range Handler(q) { // pure form of "finished": no step has work left
		if need, _ := n.Step.Need(w); need {
			return false
		}
	}
	return true
}

// Plan implements Planner. It is a pure function of (w, s); s only routes around finished/blocked groups.
func (c *Campaign) Plan(w World, s Session) Plan {
	if c.wrongDifficulty(w) {
		return Plan{Stuck: true, Why: "wrong difficulty: the " + c.Name + " campaign needs " + c.Difficulty}
	}
	lowest, stuck, kitOnly, denWait := 0, "", true, false
	for _, e := range c.Entries {
		if c.complete(w, s, e.Quest) {
			continue
		}
		if why := e.skipWhy(w); why != "" {
			continue // skipped: optional (why); never blocks the chain
		}
		if w.Act > 0 && e.Act > w.Act {
			continue // a later act: its predecessor's hop entry is still pending, so the campaign is not done
		}
		if s != nil {
			if why, b := s.Blocked(e.Quest); b {
				stuck += e.Quest + " (" + why + "); "
				kitOnly = kitOnly && strings.Contains(why, "not ready: ") && !strings.Contains(why, "clvl")
				continue
			}
		}
		if e.BR && w.Act > e.Act && normalDiff(w) {
			// Blood Raven was skipped for its kit or gold and the character has left Act 1: the way back is a Warriv
			// trip, so the campaign moves on instead of walking back to Burial Grounds (EARLYGAME sim: a gold-0 melee
			// character looped on it from Act 2).
			continue
		}
		if w.Level >= e.floor(w) {
			return Plan{ID: "quest." + e.Quest, Group: e.Quest, Nodes: Handler(e.Quest),
				Why: fmt.Sprintf("next unfinished quest: %s", e.Title)}
		}
		if fl := e.floor(w); lowest == 0 || fl < lowest {
			lowest = fl
			denWait = e.Den && fl == w.DenMinLevel
		}
	}
	if lowest > 0 {
		why := fmt.Sprintf("auto: need clvl %d for the next quest", lowest)
		if denWait {
			why = fmt.Sprintf("den: waiting for clvl %d in Blood Moor", lowest)
		}
		return Plan{ID: fmt.Sprintf("level.%d", lowest), Nodes: nodes(StepLevel(lowest)), Why: why}
	}
	if stuck != "" && kitOnly {
		// Every quest waits for supplies (mana potions, a scroll, potions) and the character is above every floor: farm for gold
		// instead of pausing; the executor releases the blocks once the gold or the level has grown (kitRetryGold).
		return Plan{ID: fmt.Sprintf("level.%d", w.Level+1), Nodes: nodes(StepLevel(w.Level + 1)),
			Why: "every quest waits for supplies: farming gold at clvl " + fmt.Sprint(w.Level)}
	}
	if stuck != "" {
		return Plan{Stuck: true, Why: "blocked: " + stuck}
	}
	return Plan{Done: true}
}
