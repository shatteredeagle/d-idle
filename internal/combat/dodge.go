package combat

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

// Curse awareness, bounded missile sidestep and the per-fight telemetry line (docs/difficulty.md P2/P3).
// Nothing here is called by the fight loop yet; the wiring points are listed in docs/difficulty.md "Wiring (MISSILES)".

var curseIDs = d2.DefaultCurseIDs

// missileKinds maps a missile class to its kind; nil means unknown, which never dodges.
var missileKinds func(class int) d2.MissileKind

// SetTables resolves the curse state ids and the missile names from the loaded data tables. A nil DB keeps the S
// defaults and disables the sidestep.
func SetTables(db *data.DB) {
	curseIDs, missileKinds = d2.DefaultCurseIDs, nil
	if db == nil {
		return
	}
	curseIDs = d2.ResolveCurseIDs(func(name string) (int, bool) {
		if s := db.StateByName(name); s != nil {
			return s.ID, true
		}
		return 0, false
	})
	missileKinds = func(class int) d2.MissileKind {
		if m := db.MissileByID(class); m != nil {
			return d2.MissileKindByName(m.Name)
		}
		return d2.MissileOther
	}
}

// CurseDanger raises a pack danger when the player carries an enemy curse: Amplify Damage, Decrepify, Iron Maiden and
// Weaken make any pack at least DangerElite, Lower Resist too (resists drop, so the elemental affixes hit harder).
// The state ids are S until ResolveCurseIDs ran against states.txt.
func CurseDanger(p *d2.Player, danger int) int {
	if p != nil && p.Curses(curseIDs).Any() {
		return max(danger, DangerElite)
	}
	return danger
}

// CurseRetreat reports whether the bot should back off and drink: Amplify Damage (+100% physical damage taken) or
// Iron Maiden (reflects damage) is on the player and life is below pct. Zero pct disables it.
func CurseRetreat(p *d2.Player, pct int) bool {
	if p == nil || pct <= 0 || p.MaxLife <= 0 {
		return false
	}
	c := p.Curses(curseIDs)
	return (c.Amplify || c.IronMaiden) && p.Life*100 < pct*p.MaxLife
}

const maxMissileDodges = 6 // per fight: a bounded sidestep, never a dance

// dangerousMissiles are the hostile missiles of a dangerous kind within radius of me.
func dangerousMissiles(snap *d2.Snapshot, me input.Point, radius int) []input.Point {
	if missileKinds == nil || snap == nil || snap.Player == nil || radius <= 0 {
		return nil
	}
	friendly := Friendlies(snap, nil)
	var out []input.Point
	for _, m := range snap.Missiles {
		if _, ok := friendly[m.OwnerID]; ok || (m.OwnerID == snap.Player.ID && m.OwnerType == 0) {
			continue
		}
		if missileKinds(m.Class) == d2.MissileOther {
			continue
		}
		pt := input.Point{X: m.X, Y: m.Y}
		if pdist(me, pt) < float64(radius) {
			out = append(out, pt)
		}
	}
	return out
}

// sidestepMissiles moves out of the way of a dangerous missile when the MissileDodge knob is on. It returns true
// when it moved. done is the caller's per-fight counter: at most maxMissileDodges per fight and one per 800 ms.
func (f *Fighter) sidestepMissiles(ctx context.Context, snap *d2.Snapshot, done *int) (bool, error) {
	t := f.tune()
	if t.MissileDodge == 0 || f.mv == nil || !f.mayEvade() || *done >= maxMissileDodges || f.now().Sub(f.lastDodge) < 800*time.Millisecond {
		return false, nil
	}
	me := input.Point{X: snap.Player.X, Y: snap.Player.Y}
	near := dangerousMissiles(snap, me, max(t.DodgeRadius, 6))
	pt, ok := awayPoint(me, near, nil, max(t.DodgeRadius, 6)+4, f.valid)
	if !ok {
		return false, nil
	}
	f.lastDodge = f.now()
	*done++
	f.info("sidestepping missile", "to", pt, "missiles", len(near))
	return true, f.evade(ctx, me, pt)
}

// FightTelemetry accumulates one fight for the soak report: kills, damage taken, potions, sidesteps.
type FightTelemetry struct {
	Area, Diff int
	Danger     int
	Kills      int
	Potions    int
	Dodges     int
	Casts      int // attack casts
	Kites      int // kite steps
	Approaches int // moves to get in range
	Novas      int // Frost Nova casts on a crowd
	Forced     int // casts forced by the watchdog
	Damage     int // sum of life drops between samples (healing is not netted out)
	MinLifePct int
	last       int
	have       bool
}

func perSec(n int, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(n) / d.Seconds()
}

// Sample feeds one player reading; life drops add to Damage.
func (t *FightTelemetry) Sample(p *d2.Player) {
	if p == nil || p.MaxLife <= 0 {
		return
	}
	pc := p.Life * 100 / p.MaxLife
	if !t.have || pc < t.MinLifePct {
		t.MinLifePct = pc
	}
	if t.have && p.Life < t.last {
		t.Damage += t.last - p.Life
	}
	t.last, t.have = p.Life, true
}

// Finish copies the counters of a ClearResult and returns the telemetry line.
func (t *FightTelemetry) Finish(res ClearResult) string {
	t.Kills, t.Potions = res.Kills, res.Potions
	return t.Line(res.Elapsed)
}

// Line is a single key=value log line, stable for the soak report parser.
func (t *FightTelemetry) Line(elapsed time.Duration) string {
	return fmt.Sprintf("fight area=%d diff=%d danger=%d kills=%d damage=%d minlife=%d potions=%d dodges=%d sec=%d casts=%d kites=%d approaches=%d novas=%d forced=%d castsPerSec=%.2f",
		t.Area, t.Diff, t.Danger, t.Kills, t.Damage, t.MinLifePct, t.Potions, t.Dodges, int(elapsed.Seconds()), t.Casts, t.Kites, t.Approaches,
		t.Novas, t.Forced, perSec(t.Casts, elapsed))
}
