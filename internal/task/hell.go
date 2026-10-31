package task

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/leveler"
)

// NightmareAll is the Nightmare campaign over every act that has nodes (nightmare_all), see nightmare.go.
func NightmareAll(cain bool) *Campaign { return difficultyAll("nightmare_all", d2.Nightmare, cain) }

// HellAll is the Hell campaign over every act that has nodes (hell_all). The Hell resist penalty (-100) and the
// immunity work are the fighter's business (docs/difficulty.md); this only orders the quests and gates them.
func HellAll(cain bool) *Campaign { return difficultyAll("hell_all", d2.Hell, cain) }

// NextDifficulty names the difficulty to start once the current one is finished: "nightmare" after Normal, "hell"
// after Nightmare, "" after Hell or while Baal is not dead. baalDone is the Eve of Destruction state read for the
// current difficulty (the Baal kill unlocks the next one, S docs/progression/difficulty.md section 3, G1). The bot
// switches by starting a new game with that difficulty (bot.ensureDifficulty -> Cycler.NewGame), driven by the goal's
// Params.Difficulty; the level gate is ReadyForNext.
func NextDifficulty(cur string, baalDone bool) string {
	if !baalDone {
		return ""
	}
	switch cur {
	case "", "normal":
		return "nightmare"
	case "nightmare":
		return "hell"
	}
	return ""
}

// EntryLevel is the clvl at which ReadyForNext turns true for next (0 for an unknown difficulty).
func EntryLevel(next string) int {
	switch next {
	case "nightmare":
		return NightmareAll(false).Entries[0].MinLevel
	case "hell":
		return HellAll(false).Entries[0].MinLevel
	}
	return 0
}

// ReadyForNext says whether a character of clvl may hop into next: the first entry floor of that difficulty's
// campaign (combat.EntryGate) is met and, when res is not nil, leveler.ResistGate passes. res are the raw (pre-penalty)
// resists, see leveler.RawResists; nil means unknown and skips the resist gate. why explains a refusal; a refused
// character keeps farming the old difficulty.
func ReadyForNext(next string, clvl int, res *d2.Resists) (ok bool, why string) {
	var c *Campaign
	var target d2.Difficulty
	switch next {
	case "nightmare":
		c, target = NightmareAll(false), d2.Nightmare
	case "hell":
		c, target = HellAll(false), d2.Hell
	default:
		return false, "no next difficulty"
	}
	if f := c.Entries[0].MinLevel; clvl < f {
		return false, "clvl below the " + next + " act 1 floor"
	}
	if res != nil {
		if g := leveler.ResistGate(target, *res); !g.OK {
			return false, g.Why
		}
	}
	return true, ""
}
