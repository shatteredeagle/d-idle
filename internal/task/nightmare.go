package task

import (
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

// Campaign templates for Nightmare and Hell: the same quest nodes as Normal, replayed in the higher difficulty.
//
// Quests are reset per difficulty (V code: quest.Source.Read takes the difficulty slot, bot.QuestReader.Read(difficulty),
// so World.Quests of a Nightmare game shows Nightmare words only; U: that every act quest, including Den of Evil, Tools
// of the Trade and the Radament/Summoner chain, really is offered again in Nightmare and Hell - community knowledge,
// not seen live). The nodes themselves do not change; only the floors do, and the character must be in the matching
// difficulty (a Normal character is held with Stuck until the bot's ensureDifficulty has started the new game).
//
// Floors come from combat.EntryGate (the act floor, docs/difficulty.md 2.2, R) and from the boss table below
// (docs/difficulty.md 2.2 "Bosses (min clvl, R)"). Below a floor the Plan is a level step, which the bot answers from
// progression-<difficulty>.json (the farming areas between quests).

// bossFloors is the minimum clvl for a quest's fight per difficulty (R; quests the table does not name fall back to
// the act floor, U).
var bossFloors = map[string][2]int{ // quest -> {nightmare, hell}
	QAndariel: {45, 65},
	QArcane:   {45, 72},
	QSummoner: {50, 70},
	QDuriel:   {50, 72},
	QCouncil:  {52, 75},
	QGuardian: {55, 75},
	QDiablo:   {55, 80},
	QBaal:     {58, 82},
}

// difficultyAll is Acts 1 to 5 (same set as normal_all) for difficulty diff, with every
// entry floored by the entry gate.
func difficultyAll(name string, diff d2.Difficulty, cain bool) *Campaign {
	a1, a2, a3, a4, a5 := defaultOptions(cain)
	c := &Campaign{Name: name, Difficulty: diffName(diff)}
	for act, part := range [][]Entry{NormalAct1(a1).Entries, NormalAct2(a2).Entries, NormalAct3(a3).Entries, NormalAct4(a4).Entries, NormalAct5(a5).Entries} {
		top := 0
		for _, e := range part {
			e.MinLevel = difficultyFloor(diff, act+1, e.Quest, e.MinLevel)
			e.NoKitMinLevel = 0 // the act floor is already far above the kit-less level
			top = max(top, e.MinLevel)
			c.Entries = append(c.Entries, e)
		}
		// The hop entry may re-fight the act's last boss (the Act 3 red portal), so it takes the act's top floor.
		if len(part) > 0 {
			c.Entries[len(c.Entries)-1].MinLevel = top
		}
	}
	return c
}

// difficultyFloor is the level floor of quest q in act (1-5) for diff. Hell act 1 quests all sit in caves, crypts or
// the Barracks, so they take the dungeon floor (U: Burial Grounds is partly open ground; a wrong guess costs a few
// levels).
func difficultyFloor(diff d2.Difficulty, act int, q string, normal int) int {
	d := combat.EntryGate(diff, act, 0, 0, diff == d2.Hell && act == 1)
	floor := max(normal, d.Floor)
	if b, ok := bossFloors[q]; ok {
		floor = max(floor, b[int(diff)-1])
	}
	return floor
}

func diffName(d d2.Difficulty) string { return [...]string{"normal", "nightmare", "hell"}[d] }

// wrongDifficulty reports a world in another difficulty than the campaign's ("" counts as normal).
func (c *Campaign) wrongDifficulty(w World) bool {
	if c.Difficulty == "" {
		return false
	}
	have := w.Difficulty
	if have == "" {
		have = "normal"
	}
	return have != c.Difficulty
}
