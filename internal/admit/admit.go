// Package admit is the one admission function (card admit1, review fable-logic-review-2026-11-25 P4): MayEnter says whether a
// hero may walk into an area for a purpose, MayLeaveTown whether it may leave town for one. The verdict is data: Allow, or
// Refused with a need and the fact that ends it. The package is pure (no I/O, no clock, no game): the caller fills a View from
// the snapshot and the knobs. The floors are rows (internal/plandata/admission.json) keyed by area and archetype; judges whose
// rules are not rows yet (the Monastery road, the Forgotten Tower, the boss floors) plug in through View.Legacy.
//
// Need codes are the closed codes of the typed outcome table in internal/bot/outcome.go (needTable); a test in that package
// keeps the two in step. This package does not import the bot, so the table stays there and nothing is forked.
package admit

import (
	"fmt"

	"d2lod-bot/internal/quest"
)

// Purpose is why the hero wants to move; the rows exempt or tighten by it.
type Purpose string

const (
	QuestWalk Purpose = "quest-walk" // a quest step walks into a field
	Farm      Purpose = "farm"       // gold or loot farm
	Level     Purpose = "level"      // the level plan picks an area
	Boss      Purpose = "boss"       // a boss or quest-boss walk
	CorpseRun Purpose = "corpse-run" // fetching the corpse: exempt from every row (Questy 2026-11-26)
	Trip      Purpose = "trip"       // going to town: always allowed
)

// The need codes this package emits (all rows of the bot's needTable).
const (
	NeedPotions  = "needs_potions"
	NeedGoldFarm = "needs_gold_farm"
	NeedTooWeak  = "too_weak"
	NeedRoute    = "route_closed"
	NeedNotReady = "not_ready"
)

// Wake is the fact that ends a refusal, by need (same words as the bot's needTable).
var Wake = map[string]string{
	NeedPotions:  "potions or portal scroll in the pack",
	NeedGoldFarm: "gold in purse or stash rises",
	NeedTooWeak:  "character level rises",
	NeedRoute:    "kit, level or gold change",
	NeedNotReady: "any of level, kit, gold or area changes",
}

// View is everything a verdict reads. Zero values mean "unknown, no rule" for numbers a row needs: a View built from an
// unreadable snapshot is allowed (the old judges did the same).
type View struct {
	Archetype  string // "melee" or "caster", resolved by the caller (a bow-less Amazon is a caster)
	Normal     bool   // rows apply in Normal only
	Level      int
	Life       int
	MaxLife    int
	Healing    int  // healing potions in belt and pack (rejuvenation counted)
	HasTP      bool // a town portal source in the pack
	Gold       int
	InTown     bool
	Area       int   // where the hero stands
	FirstField int   // the first non-town area of the hero's act
	Fields     []int // the act's fields in plan order (FirstField first); nil: only FirstField is known
	Try        bool  // the owner's try anyway: nothing is refused

	IsTown func(area int) bool // nil: no area is town
	MLvl   func(area int) int  // monster level of an area; nil: the poverty row is skipped
	// Knobs hands the existing knob values the knob rows name (for example "campaign.denMinLevelMelee"); a missing name is no rule.
	Knobs map[string]int
	// Legacy is the judge that is not a row yet (Monastery road, Barracks, Forgotten Tower): "" = open. It is asked last.
	Legacy func(area int) string
	// Prices of the kit for the quest-walk rule: a purse short of the missing potions farms gold first instead of a trip.
	PotionGold int
	TPGold     int
}

// Verdict is the answer. Allow is the zero value of Refused being false.
type Verdict struct {
	Refused bool
	Need    string // closed code (the bot's needTable)
	Wake    string // the fact that ends the refusal
	Why     string // words for the log and ops status --why
	Row     string // the row that refused (provider name), for explain
	// KitOnly: a chores trip fixes it (the purse can pay); false with needs_potions or needs_gold_farm means farm gold first.
	KitOnly bool
}

// Allow is the open verdict.
var Allow = Verdict{}

func refuse(row, need, why string, kitOnly bool) Verdict {
	return Verdict{Refused: true, Need: need, Wake: Wake[need], Why: why, Row: row, KitOnly: kitOnly}
}

// exempt: a corpse run, a trip home, the owner's try anyway, a non-Normal difficulty.
func (v View) exempt(p Purpose) bool {
	return v.Try || !v.Normal || p == CorpseRun || p == Trip
}

// fields is Fields, or the first field alone.
func (v View) fields() []int {
	if len(v.Fields) > 0 {
		return v.Fields
	}
	return []int{v.FirstField}
}

func (v View) town(a int) bool { return a == 0 || v.IsTown != nil && v.IsTown(a) }

func (r Rows) has(areas []int, a int) bool {
	for _, x := range areas {
		if x == a {
			return true
		}
	}
	return false
}

// MayEnter says whether the hero may walk into area for purpose. A hero already standing in the area is not asked by the kit
// and life rows (the trip rules decide when it leaves). Error: the rows are unreadable.
func MayEnter(v View, area int, p Purpose) (Verdict, error) {
	rs, err := Load()
	if err != nil {
		return Allow, err
	}
	return rs.MayEnter(v, area, p), nil
}

// MayEnter is MayEnter on given rows (tests, shadow replays).
func (r Rows) MayEnter(v View, area int, p Purpose) Verdict {
	if v.exempt(p) || v.town(area) {
		return Allow
	}
	for _, rd := range r.Roads {
		if !r.has(rd.Areas, area) {
			continue
		}
		if min := rd.MinLevel[v.Archetype]; min > 0 && v.Level < min {
			return refuse(rd.ID, NeedTooWeak, fmt.Sprintf("level %d needed for the %s road (clvl %d)", min, rd.ID, v.Level), false)
		}
		if min := rd.MinMaxLife[v.Archetype]; min > 0 && v.MaxLife > 0 && v.MaxLife < min {
			return refuse(rd.ID, NeedTooWeak, fmt.Sprintf("life %d needed for the %s road (max life %d)", min, rd.ID, v.MaxLife), false)
		}
	}
	// The kit is asked after every row's level and life: a hero told "kit" makes the trip and must not be refused for
	// life by a later row of the same area (review 11 N8). It is an entry rule: a hero on the road passed it and goes
	// on with a kit that thinned, as the old judge does (earlyRouteCheck).
	for _, rd := range r.Roads {
		if !r.has(rd.Areas, area) {
			continue
		}
		if r.has(rd.Areas, v.Area) {
			// The exception to the entry rule (review 14 H3): the boss or chest level of the road (a quest handler row
			// works there) keeps the potion half. With a portal a trip restocks; without one the refusal is a hold, the
			// trip rules walk the hero out (U: not shown live).
			if area != v.Area && quest.ObjectiveAreas()[area] && v.Healing < rd.Kit.Healing {
				return refuse(rd.ID, NeedPotions, fmt.Sprintf("%d healing potions < %d for the %s road's objective level", v.Healing, rd.Kit.Healing, rd.ID), v.HasTP)
			}
			continue
		}
		if v.Healing < rd.Kit.Healing {
			return refuse(rd.ID, NeedPotions, fmt.Sprintf("%d healing potions < %d for the %s road", v.Healing, rd.Kit.Healing, rd.ID), true)
		}
		if rd.Kit.Portal > 0 && !v.HasTP {
			return refuse(rd.ID, NeedPotions, "no town portal for the "+rd.ID+" road", true)
		}
	}
	for _, kf := range r.KnobFloors {
		if !r.has(kf.Areas, area) {
			continue
		}
		if k := kf.LevelKnob[v.Archetype]; k != "" {
			if min := v.Knobs[k]; min > 0 && v.Level < min {
				return refuse(kf.ID, NeedTooWeak, fmt.Sprintf("level %d needed for %s (clvl %d)", min, kf.ID, v.Level), false)
			}
		}
	}
	if pv := r.Poverty; pv.NoPotionGap > 0 && v.Healing == 0 && v.Level >= pv.FromLevel && v.MLvl != nil {
		if ml := v.MLvl(area); ml > v.Level-pv.NoPotionGap {
			return refuse("poverty", NeedPotions, fmt.Sprintf("no healing potions: monster level %d is above clvl %d minus %d", ml, v.Level, pv.NoPotionGap), false)
		}
	}
	if area != v.Area {
		if kf := r.KitFloor; kf.PotionsBelow > 0 && v.Level >= kf.FromLevel && v.Healing < kf.PotionsBelow && !kf.Open(v.Healing, v.Level, area, v.fields(), v.MLvl) {
			return refuse("kit-floor", NeedPotions, fmt.Sprintf("%d healing potions < %d: the first field only (area %d) unless a tier opens more, not area %d", v.Healing, kf.PotionsBelow, v.FirstField, area), false)
		}
		if why := r.leanWhy(v); why != "" {
			return refuse("lean", NeedNotReady, why, false)
		}
	}
	if v.Legacy != nil {
		if why := v.Legacy(area); why != "" {
			return refuse("legacy", NeedRoute, why, false)
		}
	}
	return Allow
}

// IsLean: no town portal and fewer healing potions than the lean row: the walk home is the hero's only way out of a field.
func (r Rows) IsLean(v View) bool {
	return v.Normal && r.Lean.PotionsBelow > 0 && !v.HasTP && v.Healing < r.Lean.PotionsBelow
}

// leanWhy is the life hold of a lean hero outside town: not another field below the entry life line (Questy 90/176).
func (r Rows) leanWhy(v View) string {
	if !r.IsLean(v) || v.InTown || v.MaxLife <= 0 {
		return ""
	}
	if pct := v.Life * 100 / v.MaxLife; pct < r.Lean.EntryLifePct {
		return fmt.Sprintf("no portal and %d%% life: not entering another field below %d%%", pct, r.Lean.EntryLifePct)
	}
	return ""
}

// MayLeaveTown says whether the hero may leave town for purpose. A quest walk needs the quest-walk kit; any outing of a lean hero
// needs the life to be above the entry line (a hero in town is healed by the heal chore, so the line is read on the way out).
func MayLeaveTown(v View, p Purpose) (Verdict, error) {
	rs, err := Load()
	if err != nil {
		return Allow, err
	}
	return rs.MayLeaveTown(v, p), nil
}

// MayLeaveTown is MayLeaveTown on given rows.
func (r Rows) MayLeaveTown(v View, p Purpose) Verdict {
	if v.exempt(p) {
		return Allow
	}
	if w := r.QuestWalk; p == QuestWalk && w.BelowLevel > 0 && v.Level >= w.FromLevel && v.Level < w.BelowLevel {
		switch {
		case v.Healing < w.Healing:
			need := (w.Healing - v.Healing) * v.PotionGold
			if v.Gold < need {
				return refuse("quest-walk", NeedGoldFarm, fmt.Sprintf("%d healing potions < %d for a quest walk, %d gold needed: farm gold first", v.Healing, w.Healing, need), false)
			}
			return refuse("quest-walk", NeedPotions, fmt.Sprintf("%d healing potions < %d for a quest walk", v.Healing, w.Healing), true)
		case w.Portal > 0 && !v.HasTP:
			if v.Gold < v.TPGold {
				return refuse("quest-walk", NeedGoldFarm, fmt.Sprintf("no town portal for a quest walk, %d gold needed: farm gold first", v.TPGold), false)
			}
			return refuse("quest-walk", NeedPotions, "no town portal for a quest walk", true)
		}
	}
	if r.IsLean(v) && v.MaxLife > 0 {
		if pct := v.Life * 100 / v.MaxLife; pct < r.Lean.EntryLifePct {
			return refuse("lean", NeedNotReady, fmt.Sprintf("no portal and %d%% life: heal before leaving town (line %d%%)", pct, r.Lean.EntryLifePct), false)
		}
	}
	return Allow
}
