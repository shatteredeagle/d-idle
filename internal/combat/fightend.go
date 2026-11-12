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

// FightSummary is the compact record of one finished fight for the watch minimap (OnSummary): unlike OnFight it is
// sent for every fight that had telemetry, however short. End is the telemetry `end=` reason, Left as in FightEnd.
type FightSummary struct {
	Area       int
	End        string
	Kills      int
	Remaining  int
	Damage     int
	MinLifePct int
	LifePct    int
	Potions    int
	Casts      int
	Kites      int
	Approaches int
	Dodges     int
	Sec        float64
	Left       string
}

func (f *Fighter) noteSummary(ft *FightTelemetry, res ClearResult, err error) {
	if f.OnSummary == nil || !ft.have {
		return
	}
	s := FightSummary{Area: ft.Area, End: ft.End, Kills: res.Kills, Remaining: ft.Remaining, Damage: ft.Damage, MinLifePct: ft.MinLifePct,
		LifePct: ft.LifePct, Potions: res.Potions, Casts: ft.Casts, Kites: ft.Kites, Approaches: ft.Approaches, Dodges: ft.Dodges, Sec: res.Elapsed.Seconds()}
	switch {
	case errors.Is(err, ErrChicken):
		s.Left = "chicken"
	case errors.Is(err, ErrTownPortal):
		s.Left = "portal"
	}
	f.OnSummary(s)
}

// Note is a passive remark of the fighter for the watch minimap: Kind is "ignore", "fallback" or "yield", Secs the
// length of an ignore window, X and Y the unit's place when HasPos.
type Note struct {
	Kind   string
	Text   string
	X, Y   int
	HasPos bool
	Secs   int
}

func (f *Fighter) note(n Note) {
	if f.OnNote != nil {
		f.OnNote(n)
	}
}
