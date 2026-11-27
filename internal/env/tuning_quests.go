package env

// QuestsTuning holds the quest board switch (docs/questboard.md, design docs/design/quest-board-2026-11-16.md).
type QuestsTuning struct {
	// BoardSM 1: the board runs on the run state machine (runs and latches in the plan file, docs/design/fable-review-2026-11-21); 0 (default): the old derive-every-tick paths.
	BoardSM int

	// StepExec 1: the generic step executor (internal/questspec/exec.go) runs the quest steps that carry an exec row in data/questspec (Cain scroll and reading, Horadric Staff transmute, the orifice); 0 (default): the old handlers.
	StepExec int

	Board int // 1 (default): the quest board drives quests (player taps Start and Hand in); 0: the old floor planner drives them (measured runs)

	// Opt-in auto-start (slice S8); nothing here acts while Board is 0 or while no quest is marked auto.
	AutoDelaySec int // countdown before an auto-start, in seconds; 0 starts at once
	AwayMin      int // minutes without a watch client before the player counts as away
	AutoHoldMin  int // minutes after the player paused or reset when nothing starts by itself
	AutoGapMin   int // minutes between two auto-started chunks
	AutoDayCap   int // auto-started chunks per 24 hours

	// LayoutRead 1: the d2d backend reads the fork's read-only `debug levelmap` and walks straight to known quest objects (ADR 0022); 0 (default): explore as before. PREREQUISITE of the Smith (and Countess) step: their spawn is a superunique preset read by name, so with 0 the run only explores for them and ends "needs quests.layoutRead 1".
	LayoutRead int

	// LegacyWall: levels under a quest's legacy level floor at which a chunk judged by that floor alone reads too hard and is not
	// offered; 0 = warn only (ADR 0011, "warn, never wall"). Default 2: a hero more than two levels under the floor waits.
	LegacyWall int
}

func DefaultQuestsTuning() QuestsTuning {
	return QuestsTuning{Board: 1, BoardSM: 0, AutoDelaySec: 20, AwayMin: 10, AutoHoldMin: 15, AutoGapMin: 30, AutoDayCap: 6, LayoutRead: 0, LegacyWall: 2}
}

func questsKnobs() []Knob {
	return []Knob{
		{Key: "quests.board", Label: "Quest board drives quests", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 (default since the Journey was retired, docs/questboard.md S9): a quest runs only after the player accepts its current chunk (Start); the bot does that chunk, walks to the NPC and stops, and the player taps Hand in. Nothing else starts quests: between chunks the hero acts on the player's mode (Train levels toward the next quest, Idle parks in town, Farm farms loot and gold). Quests the bot cannot do yet show 'not supported yet' and the hero trains instead. 0: the old floor planner plays the quests as before (used by measured runs and gates, no Journey); a hero on 0 has no Start button."},
		{Key: "quests.boardSM", Label: "Quest board state machine", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1: the quest board stores each accepted chunk as a run with a status and latches completion in the hero's plan file (a park survives a restart, accept implies the earlier parts, Stop implies the parts that wait for it, a chunk seen done stays done), design docs/design/fable-review-2026-11-21/quest-board.md. 0 (default): the board derives its state from the evidence every tick, as before. The plan file keeps its old accepted records either way."},
		{Key: "quests.stepExec", Label: "Generic step executor", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "[dark-launch] 1: quest steps that carry an exec row in data/questspec run on the generic step executor of internal/questspec (approach, act, wait for the declared fact read from the game, bounded retries with backoff, a typed outcome): the Cain scroll (take_scroll), Akara reading it (read_scroll), the Horadric Staff transmute and placing the Staff in the orifice; 0 (default): the old handlers. Card stepexec1, ledger row in data/darklaunch.json."},
		{Key: "quests.autoDelaySec", Label: "Auto-start countdown", Unit: "s", Default: 20, Min: 0, Max: 300, NoHandicap: true,
			Desc: "Seconds an opted-in quest counts down (shown on the page, with Cancel) before it starts by itself; 0 starts at once. Only a SAFE offer ever auto-starts."},
		{Key: "quests.awayMin", Label: "Away after", Unit: "min", Default: 10, Min: 1, Max: 240, NoHandicap: true,
			Desc: "Minutes without any watch client asking for the board before the player counts as away: then a chunk the board started by itself is also handed in by the board. A present player taps Hand in."},
		{Key: "quests.autoHoldMin", Label: "No auto-start after pause or reset", Unit: "min", Default: 15, Min: 0, Max: 240, NoHandicap: true,
			Desc: "No quest starts by itself this many minutes after the player paused or reset the bot."},
		{Key: "quests.autoGapMin", Label: "Minutes between auto-starts", Unit: "min", Default: 30, Min: 0, Max: 600, NoHandicap: true,
			Desc: "At least this long between two auto-started chunks (a bug can then not chain starts)."},
		{Key: "quests.autoDayCap", Label: "Auto-starts per day", Unit: "count", Default: 6, Min: 0, Max: 100, NoHandicap: true,
			Desc: "At most this many auto-started chunks in 24 hours; 0 turns auto-start off."},
		{Key: "quests.legacyWall", Label: "Level wall for floor-only quests", Unit: "levels", Default: 2, Min: 0, Max: 10, NoHandicap: true,
			Desc: "A chunk judged only by the old level guide (no boss margin) reads too hard, and is not startable, when the hero is more than this many levels under its floor: the board said risky for a clvl 4 Paladin and the Cain scroll walk through the Underground Passage killed it (Questy bed 2026-11-25, floor 9). 0: warn only, as ADR 0011 had it (the owner's Start anyway still starts it)."},
		{Key: "quests.layoutRead", Label: "Read the level layout", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1: ask the private d2d fork for the current level's room layout and quest-object positions (read-only verb `debug levelmap`, ADR 0022: a read never makes a run CHEATED) and walk straight to the Tree of Inifuss, the cairn stones or Cain's cage instead of exploring for them; a second read (`debug levelmonsters`) gives a superunique's spawn point (the Countess in Tower Cellar 5) from the level's build; stock d2d or an error falls back to exploring. 0 (default): explore as before."},
	}
}
