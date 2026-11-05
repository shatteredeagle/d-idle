package env

// LadderTuning holds the automation-ladder switch (docs/t0-slice.md, ADR 0016). Off, the bot runs the legacy code
// paths byte for byte; on, the wired decision points ask internal/decide.
type LadderTuning struct {
	Enabled int // 1: wired decision points ask the decision-point store; 0: legacy code paths
}

func DefaultLadderTuning() LadderTuning { return LadderTuning{} }

func ladderKnobs() []Knob {
	return []Knob{
		{Key: "ladder.enabled", Label: "Decision ladder", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1: the wired decision points (t1.repair, t1.tp.lost; docs/t0-slice.md) build a context key and ask the decision-point store for the action: a learned rule, else the untrained default. The hero never waits for a question. 0: the legacy code paths, unchanged. Default off until the live drill passes."},
	}
}
