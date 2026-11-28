package resource

import (
	"encoding/json"
	"strings"
	"sync"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/plandata"
)

// Mana-heavy builds (docs/builds-endgame.md W7, docs/plan/data/potions-belt.json "mana_heavy_builds", U design, needs
// a live Hell check): War Cry and Blizzard packs drain the single mana column, so from Nightmare on these builds
// carry the mana-heavy columns and drink at 35% instead of the normal threshold. In Normal they get the default
// columns of mana need 2 (hp, hp, mp, mp: two healing columns stay) and the normal drink line (TASK-1117).

// ManaHeavyDrinkPct is the mana % below which a mana-heavy build drinks (json drink_mana_below_pct).
const ManaHeavyDrinkPct = 35

var (
	manaHeavyOnce   sync.Once
	manaHeavyBuilds map[string]bool
)

// loadManaHeavy reads the builds that flag themselves `"mana_heavy": true` in their embedded table, so a new
// build needs a data edit and no Go name (audit S4 seam; the json "builds" list mirrors it, a test compares both).
func loadManaHeavy() map[string]bool {
	manaHeavyOnce.Do(func() {
		manaHeavyBuilds = map[string]bool{}
		ents, _ := plandata.FS.ReadDir(".")
		for _, e := range ents {
			if !strings.HasPrefix(e.Name(), "build-") {
				continue
			}
			raw, err := plandata.FS.ReadFile(e.Name())
			if err != nil {
				continue
			}
			var h struct {
				Build     string `json:"build"`
				ManaHeavy bool   `json:"mana_heavy"`
			}
			if json.Unmarshal(raw, &h) == nil && h.ManaHeavy && h.Build != "" {
				manaHeavyBuilds[h.Build] = true
			}
		}
	})
	return manaHeavyBuilds
}

// ManaHeavyBuild reports whether the leveler table name is a mana-heavy build.
func ManaHeavyBuild(build string) bool { return loadManaHeavy()[build] }

// manaHeavyCols is the json by_clvl block (hp, mp, mp, hp / hp, mp, mp, rejuv / rejuv, mp, mp, rejuv).
func manaHeavyCols(level int) [4]Potion {
	hp, mp, rv := PotionLife, PotionMana, PotionRejuv
	switch {
	case level >= 61:
		return [4]Potion{rv, mp, mp, rv}
	case level >= 31:
		return [4]Potion{hp, mp, mp, rv}
	}
	return [4]Potion{hp, mp, mp, hp}
}

// manaHeavy selects the mana-heavy belt columns and drink line, Nightmare and Hell only (U design W7: the columns
// replace a healing column, which a Normal hero still needs).
func (in Input) manaHeavy() bool { return in.Diff != d2.Normal && ManaHeavyBuild(in.Build) }

// ManaNeedFor is the mana need of a hero: ManaNeed(class), raised to 2 for a table that flags itself mana_heavy, in
// every difficulty (review 8 C8: the flag was ignored in Normal, so a Hammerdin or a War Cry Barbarian got no mana
// kit line there). The need comes from the build, the class value (classtraits.json manaNeed) is only the default.
// This is the one source: the kit gate, the route guard, the belt columns (PlanForBuild), the shop rule, the town
// trip and the Mana Burn rules all ask it; ManaNeed(class) alone is for callers that know no build.
func ManaNeedFor(c d2.Class, build string) int {
	if ManaHeavyBuild(build) {
		return 2
	}
	return ManaNeed(c)
}

func (in Input) manaNeed() int { return ManaNeedFor(in.Class, in.Build) }

// PlanForBuild is PlanForDiff with the columns of the build's mana need. A mana-heavy build gets the mana-heavy
// columns in Nightmare and Hell (the Hell rejuvenation columns of PlanForDiff do not apply to it: its own rows already
// carry them from clvl 31) and in Normal the default columns of need 2, which keep two healing columns: the kit gate
// and the shop ask the same need (ManaNeedFor), so a mana minimum always has a column to sit in (review C B2).
func PlanForBuild(c d2.Class, level int, d d2.Difficulty, build string) BeltPlan {
	p := PlanForDiff(c, level, d)
	if ManaHeavyBuild(build) {
		if d != d2.Normal {
			p.Cols = manaHeavyCols(level)
		} else {
			p.Cols = colsForNeed(ManaNeedFor(c, build), level)
		}
	}
	return p
}
