package env

// LadderTuning holds the automation-ladder switch (docs/t0-slice.md, ADR 0016). Off, the bot runs the legacy code
// paths byte for byte; on, the wired decision points ask internal/decide.
type LadderTuning struct {
	Enabled int // 1: wired decision points ask the decision-point store; 0: legacy code paths
	// RepeatCooldownMin: minutes a question for the same key is not put again (and the answer given is reused); 0 = ask every time.
	RepeatCooldownMin int
	// QuietAfter: unanswered questions in a row after which a key stops asking; 0 = never.
	QuietAfter int
	// AwayMin: minutes without a look at the watch screen after which the player counts as away (zone proposals wait for the return); 0 = the built-in 20.
	AwayMin int
}

func DefaultLadderTuning() LadderTuning {
	return LadderTuning{RepeatCooldownMin: 30, QuietAfter: 3, AwayMin: 20}
}

func ladderKnobs() []Knob {
	return []Knob{
		{Key: "ladder.enabled", Label: "Decision ladder", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1: the wired decision points (t1.repair, t1.tp.lost; docs/t0-slice.md) build a context key and ask the decision-point store for the action: a learned rule, else the untrained default. The hero never waits for a question. 0: the legacy code paths, unchanged. Default off until the live drill passes."},
		{Key: "ladder.repeatCooldownMin", Label: "Ladder: same question cooldown", Unit: "min", Default: 30, Min: 0, Max: 600, NoHandicap: true,
			Desc: "The same question (same decision point and context key) is not put again for this long after it was asked or answered; if the player answered it, that answer acts meanwhile (no re-ask). 0: ask every time it comes up (the old behaviour, which re-asked about every minute while the situation lasted). TASK-137."},
		{Key: "ladder.quietAfter", Label: "Ladder: stop asking after", Unit: "questions", Default: 3, Min: 0, Max: 20, NoHandicap: true,
			Desc: "After this many questions for one key faded unanswered in a row the bot uses the default, stops asking about it and writes one journal line. An answer resets the count. 0: never stop. Needs ladder.repeatCooldownMin above 0. TASK-137."},
		{Key: "ladder.awayMin", Label: "Ladder: away after", Unit: "min", Default: 20, Min: 1, Max: 1440, NoHandicap: true,
			Desc: "Minutes since the player last looked at the watch screen after which the player counts as away. The hero keeps farming where it is; a zone proposal waits for the return and is never accepted by itself (docs/proposezone.md). TASK-131."},
	}
}
