package combat

import (
	"errors"
	"time"
)

// FightMinSec: a fight shorter than this that cost no life is not reported to OnFight (rate limit by construction:
// one event per fight, and none for a one-hit pull).
const FightMinSec = 3

// FightEnd is what OnFight gets: the fight telemetry line as data.
type FightEnd struct {
	Area, Diff int
	MinLifePct int // lowest life percent seen in the fight (the "minlife=" of the telemetry line)
	Kills      int
	Potions    int
	Damage     int
	Sec        int
	Left       string // "chicken" (left the game), "portal" (town portal opened) or ""
}

func (f *Fighter) noteFightEnd(ft *FightTelemetry, res ClearResult, err error) {
	if f.OnFight == nil || !ft.have {
		return
	}
	if res.Elapsed < FightMinSec*time.Second && ft.Damage == 0 {
		return
	}
	end := FightEnd{Area: ft.Area, Diff: ft.Diff, MinLifePct: ft.MinLifePct, Kills: res.Kills, Potions: res.Potions,
		Damage: ft.Damage, Sec: int(res.Elapsed.Seconds())}
	switch {
	case errors.Is(err, ErrChicken):
		end.Left = "chicken"
	case errors.Is(err, ErrTownPortal):
		end.Left = "portal"
	}
	f.OnFight(end)
}
