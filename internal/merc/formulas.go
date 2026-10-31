package merc

// Formulas from docs/sme/d2decomped-findings.md section 2. S (secondary source, a reimplementation's notes); the
// live probe (docs/plan/plan-v2.md M4) is the acceptance test.

// OfferedLevel is the level a hire row shows for a player of level clvl: clvl-5+r with r in 0..4, at least 2.
func OfferedLevel(clvl, r int) int { return max(2, clvl-5+r) }

// HireCost is Gold x (15 d + 100) / 100 with d = offered level - the row's base level, never below Gold (hireling
// table column). Payment source for a hire (carried vs stash) is U.
func HireCost(baseGold, offerLevel, rowLevel int) int {
	d := max(0, offerLevel-rowLevel)
	return max(baseGold, baseGold*(15*d+100)/100)
}

// ReviveCost is level^2/2 x 15 capped at 50,000 (operator order as written in the notes: integer halving first).
// Carried gold is paid first, then the stash.
func ReviveCost(level int) int {
	if level < 1 {
		level = 1
	}
	return min(level*level/2*15, 50000)
}
