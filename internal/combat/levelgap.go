package combat

import "d2lod-bot/internal/d2"

// LevelGapOK is docs/difficulty.md P2 item 23: in Hell do not leave town for an area whose monster level is 10 or
// more above the character, and in general require clvl >= mlvl - margin (Normal 2, Nightmare 5, Hell 8, P1 item 3).
// The margins are R. mlvl 0 (unknown) passes.
func LevelGapOK(diff d2.Difficulty, clvl, mlvl int) bool {
	if mlvl <= 0 {
		return true
	}
	margin := [...]int{2, 5, 8}[min(max(int(diff), 0), 2)]
	if diff == d2.Hell && mlvl-clvl >= 10 {
		return false
	}
	return clvl >= mlvl-margin
}
