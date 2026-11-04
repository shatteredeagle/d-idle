package combat

import (
	"os"
	"time"

	"d2lod-bot/internal/buildinfo"
	"d2lod-bot/internal/d2"
)

// Log-only helpers for the per-skill soak fields (card logfields, docs/logfields.md). Nothing here reads memory or
// sends input: it looks at the snapshot the fight loop already has and writes lines. castLog.off exists for the test
// that proves the action sequence does not depend on the logging.

// castLog is the pending cast waiting for its outcome.
type castLog struct {
	off       bool
	buildDone bool
	targets   int // monsters counted for the next cast (nova), 0 = unknown; consumed by castLogStart
	pend      *pendCast
}

type pendCast struct {
	name       string
	at         time.Time
	mana, cost int
	ticks      int
	sawCast    bool // the player was seen in the cast mode after the click (U live)
	sawMana    bool // the pool dropped by the cost (the mana leaves at the action frame, S)
	sawHit     bool // hit recovery seen before either of the above
	targets    int  // monsters in the area the cast was aimed at (S: counted before the cast, not confirmed hits); 0 = unknown
	kills      int  // kills noted while this cast was the pending one (S: the hit that killed is not observable)
}

// castLogStart records a cast just sent. A cast still pending is closed with what was seen (the next cast came first).
func (f *Fighter) castLogStart(snap *d2.Snapshot, sp SkillSpec) {
	c := &f.clog
	if c.off {
		return
	}
	f.castLogBuild()
	f.castLogFlush()
	c.pend = &pendCast{name: sp.Name, at: f.now(), mana: snap.Player.Mana, cost: sp.Cost,
		ticks: castTicks(sp.Name, f.tune().CastFCR, f.D2D), targets: c.targets}
	c.targets = 0
}

// castLogTargets tells the log how many monsters the next cast is aimed at (the crowd nova counts them). The combat
// code has no missile-hit signal, so this is the only per-cast hit figure it knows (S).
func (f *Fighter) castLogTargets(n int) {
	if !f.clog.off {
		f.clog.targets = n
	}
}

// castLogKill credits a kill to the pending cast, if its window is still open.
func (f *Fighter) castLogKill() {
	if pc := f.clog.pend; pc != nil && !f.clog.off {
		pc.kills++
	}
}

// castLogTick is called once per fight tick with the fresh player: it watches the pending cast and closes it when the
// cast length (plus slack) has passed.
func (f *Fighter) castLogTick(p *d2.Player) {
	pc := f.clog.pend
	if pc == nil || f.clog.off {
		return
	}
	switch {
	case p.Mode == modeCast:
		pc.sawCast = true
	case p.Mode == modeGetHit && !pc.sawCast && !pc.sawMana:
		pc.sawHit = true
	}
	if pc.cost > 0 && pc.mana-p.Mana >= pc.cost {
		pc.sawMana = true
	}
	window := time.Duration(max(pc.ticks, 10)+4) * tickMs * time.Millisecond
	if f.now().Sub(pc.at) >= window {
		f.castLogFlush()
	}
}

// castLogFlush writes the `cast result` line of the pending cast. made = the cast mode or the mana drop was seen;
// cut = hit recovery came first and neither was seen. Both are samples at the fight tick rate, so a short cast can
// read made=false (U live). cost is whole mana (the 256ths are not logged); ticks is the modelled cast length.
func (f *Fighter) castLogFlush() {
	pc := f.clog.pend
	f.clog.pend = nil
	if pc == nil {
		return
	}
	made := pc.sawCast || pc.sawMana
	args := []any{"skill", pc.name, "made", made, "cut", pc.sawHit && !made, "cost", pc.cost, "ticks", pc.ticks, "kills", pc.kills}
	if pc.targets > 0 {
		args = append(args, "targets", pc.targets)
	}
	f.log().Debug("cast result", args...)
}

// castLogBuild writes the one `build` line (branch from D2LOD_BRANCH, commit from the build stamp; n/a-safe: a key
// with no value is left out).
func (f *Fighter) castLogBuild() {
	if f.clog.buildDone {
		return
	}
	f.clog.buildDone = true
	args := []any{}
	if b := os.Getenv("D2LOD_BRANCH"); b != "" {
		args = append(args, "branch", b)
	}
	if c := buildinfo.Current(); c != "dev" {
		args = append(args, "commit", c)
	}
	f.log().Info("build", args...)
}

// lastSkillKV is the key/value pair for the kill line: the skill cast most recently (the log credits by last cast,
// not by hit).
func (f *Fighter) lastSkillKV() []any {
	if f.clog.off || f.lastCastName == "" {
		return nil
	}
	return []any{"last_skill", f.lastCastName, "since_cast_ms", f.now().Sub(f.lastCast).Milliseconds()}
}

// ArmorState names the armor state bit set on the player ("Frozen Armor", ...; "none" when the state array has bits
// but no armor, "?" when it is empty or unreadable). For the minute line's armor sample.
func ArmorState(p *d2.Player) string {
	if p == nil || p.States == (d2.StateSet{}) {
		return "?"
	}
	for _, a := range []struct {
		id   int
		name string
	}{{d2.StateFrozenArmor, "Frozen Armor"}, {d2.StateChillingArmor, "Chilling Armor"}, {d2.StateShiverArmor, "Shiver Armor"}} {
		if p.States.Has(a.id) {
			return a.name
		}
	}
	return "none"
}
