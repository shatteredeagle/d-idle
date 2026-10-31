package combat

import (
	"fmt"
	"strings"

	"d2lod-bot/internal/d2"
)

// Death and fight tags for the logs and DeathWeights (docs/difficulty.md P3 item 27; DIFFP2). The line is key=value
// and stable for parsers.

// LevelGap is monster level minus character level at the death (positive: the killer out-levelled us). Zero when the
// monster level is unknown.
func (di DeathInfo) LevelGap() int {
	if di.MonsterLevel == 0 {
		return 0
	}
	return di.MonsterLevel - di.CharLevel
}

// MonsterAffixes is the affix set of a monster unit (zero for a plain one), for logs outside this package.
func MonsterAffixes(m *d2.MonsterInfo) Affix { return affixesOf(m) }

// Line is the death log line: difficulty, area, levels, gap, killer, affixes and distance.
func (di DeathInfo) Line() string {
	aff := di.Affixes.String()
	if aff == "" {
		aff = "-"
	}
	name := strings.ReplaceAll(di.MonsterName, " ", "_")
	if name == "" {
		name = "-"
	}
	return fmt.Sprintf("death diff=%s area=%d clvl=%d mlvl=%d gap=%d killer=%s class=%d boss=%v affixes=%s dist=%.0f",
		di.Difficulty, di.Area, di.CharLevel, di.MonsterLevel, di.LevelGap(), name, di.MonsterClass, di.Boss, aff, di.Distance)
}

// Weights turns a death into the DeathWeights increments the bot adds: one point on the monster class and on each
// of its affixes, doubled when the killer out-levelled the character by 10 or more in Hell (R: those deaths are the
// ones the entry gate should have prevented, so they weigh more).
func (di DeathInfo) Weights(monKey string) (monster float64, affix map[string]float64) {
	w := 1.0
	if di.Difficulty == d2.Hell && di.LevelGap() >= 10 {
		w = 2
	}
	affix = map[string]float64{}
	for _, n := range strings.Split(di.Affixes.String(), ",") {
		if n != "" {
			affix[n] = w
		}
	}
	if monKey != "" {
		monster = w
	}
	return monster, affix
}

// AffixTags is the sorted-by-first-seen list of distinct affix names in a pack, "-" when none (for fight lines).
func AffixTags(ts []Target) string {
	var all Affix
	for _, t := range ts {
		all |= t.Affixes
	}
	if s := all.String(); s != "" {
		return s
	}
	return "-"
}
