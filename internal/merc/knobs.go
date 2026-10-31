package merc

import "d2lod-bot/internal/env"

// Knobs is the policy configuration, the merc.* tuning group (docs/knobs.md).
type Knobs struct {
	ReviveMaxPerHour  int
	ReviveGoldMult    int
	StopDeathsPerHour int
	StopGoldMult      int
	HireGoldMult      int
	FreeRogueOnly     bool
	ReplaceAtAct2     bool
	SwapGainPct       int
	ReqStrDex         bool
	PotionLifePct     int
}

// KnobsFrom converts the env tuning group.
func KnobsFrom(t env.MercTuning) Knobs {
	return Knobs{t.ReviveMaxPerHour, t.ReviveGoldMult, t.StopDeathsPerHour, t.StopGoldMult, t.HireGoldMult,
		t.FreeRogueOnly != 0, t.ReplaceAtAct2 != 0, t.SwapGainPct, t.ReqStrDex != 0, t.PotionLifePct}
}

// DefaultKnobs are the defaults of env.DefaultMercTuning.
func DefaultKnobs() Knobs { return KnobsFrom(env.DefaultMercTuning()) }
