package combat

import (
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// Why a hostile unit was left alone. A target is never fought forever: every one of these ends the effort, is
// reported in ClearResult.Ignored with the reason, and is remembered for IgnoreMemorySec so the next Clear does
// not walk back to it (the first live bot runs looped on three such units).
type IgnoreReason string

const (
	IgnoreUnreachable IgnoreReason = "unreachable" // no path on the collision grid and no firing position with a clear line of sight
	IgnoreNoSkill     IgnoreReason = "no-skill"    // the class profile has no learned, usable attack skill
	IgnoreNoShot      IgnoreReason = "no-shot"     // reachable on paper, but ReachSec passed without a single cast at it
	IgnoreApproaches  IgnoreReason = "approaches"  // MaxApproaches walks toward it failed to bring it into range
	IgnoreAvoid       IgnoreReason = "avoid"       // the danger table or the death weights say do not fight it
	IgnoreImmune      IgnoreReason = "immune"      // immune to every attack skill we have
	IgnoreNoProgress  IgnoreReason = "no-progress" // casts landed but its life did not drop for NoProgressSec
	IgnoreTime        IgnoreReason = "time"        // MaxTargetSec engaged on this one target
)

// Ignored is one hostile the fighter gave up on (or never engaged on purpose).
type Ignored struct {
	ID     uint32
	Class  int
	Name   string
	Pos    input.Point
	Why    IgnoreReason
	Detail string
	Dist   float64 // from the player when it was ignored
	At     time.Time
	// Strikes counts how often this unit was ignored; the memory time doubles with each.
	Strikes int
}

func (i Ignored) String() string {
	return fmt.Sprintf("%s#%d at (%d,%d) %s %s", i.Name, i.ID, i.Pos.X, i.Pos.Y, i.Why, i.Detail)
}

// ignoreRec is the fighter's memory of an ignored unit across Clear calls.
type ignoreRec struct {
	Ignored
	until time.Time
}

// reengageDist: an ignored monster this close is fought again (it is on top of us, so the reasons no longer hold).
const reengageDist = 7

// IsIgnored reports whether the unit is currently remembered as ignored (and not close enough to be fought again).
func (f *Fighter) IsIgnored(id uint32, dist float64) bool { return f.ignoredNow(id, dist) }

func (f *Fighter) ignoredNow(id uint32, dist float64) bool {
	r, ok := f.ignores[id]
	if !ok {
		return false
	}
	if !f.now().Before(r.until) || dist <= reengageDist {
		return false
	}
	return true
}

// ignore ends the effort on a target: blacklists it for this fight, adds it to the result and remembers it.
func (f *Fighter) ignore(st *fightState, tg Target, why IgnoreReason, detail string) {
	if st.blacklist[tg.Unit.ID] && why != IgnoreAvoid {
		return
	}
	st.blacklist[tg.Unit.ID] = true
	rec := f.ignores[tg.Unit.ID]
	rec.Ignored = Ignored{ID: tg.Unit.ID, Class: tg.Unit.Class, Name: tg.Name, Pos: input.Point{X: tg.Unit.X, Y: tg.Unit.Y},
		Why: why, Detail: detail, Dist: tg.Dist, At: f.now(), Strikes: rec.Strikes + 1}
	rec.until = f.now().Add(time.Duration(f.tune().IgnoreMemorySec) * time.Second * time.Duration(min(rec.Strikes, 4)))
	if f.ignores == nil {
		f.ignores = map[uint32]ignoreRec{}
	}
	f.ignores[tg.Unit.ID] = rec
	st.ignored = append(st.ignored, rec.Ignored)
	f.info("ignoring target", "monster", tg.Name, "id", tg.Unit.ID, "why", string(why), "detail", detail, "dist", int(tg.Dist))
}

// IgnoredTargets lists the units currently remembered as ignored (not expired).
func (f *Fighter) IgnoredTargets() []Ignored {
	var out []Ignored
	for _, r := range f.ignores {
		if f.now().Before(r.until) {
			out = append(out, r.Ignored)
		}
	}
	return out
}

// ForgetIgnored clears the memory (the bot calls it when it changes area).
func (f *Fighter) ForgetIgnored() { f.ignores = nil }

// ---- line of sight and firing positions ---------------------------------------------------------------

// Sighter is an optional extension of Mover: whether a missile flies from a to b. Without it the fighter reads the
// collision grid of the environment's map tracker (cells with the BlockLOS bit stop a shot; unknown cells do not).
type Sighter interface {
	LineOfSight(a, b input.Point) bool
}

// los reports an open line between two subtile positions.
func (f *Fighter) los(a, b input.Point) bool {
	if f.sight != nil {
		return f.sight(a, b)
	}
	if s, ok := f.mv.(Sighter); ok {
		return s.LineOfSight(a, b)
	}
	if f.e == nil || f.e.World == nil {
		return true
	}
	lm := f.e.World.Current()
	if lm == nil || lm.Grid == nil {
		return true
	}
	return GridLOS(lm.Grid, a, b)
}

// known reports whether the collision grid has data at p (true when there is no grid to ask).
func (f *Fighter) known(p input.Point) bool {
	if f.e == nil || f.e.World == nil {
		return true
	}
	lm := f.e.World.Current()
	return lm == nil || lm.Grid == nil || lm.Grid.Known(p.X, p.Y)
}

// GridLOS walks the straight line between two subtile points over the collision grid; any known cell with the
// BlockLOS bit (other than the two end cells) blocks it. Pure given the grid.
func GridLOS(g *d2map.Grid, a, b input.Point) bool {
	dx, dy := b.X-a.X, b.Y-a.Y
	n := max(abs(dx), abs(dy))
	for i := 1; i < n; i++ {
		x := a.X + int(math.Round(float64(dx)*float64(i)/float64(n)))
		y := a.Y + int(math.Round(float64(dy)*float64(i)/float64(n)))
		if fl, ok := g.Flags(x, y); ok && fl&d2map.BlockLOS != 0 {
			return false
		}
	}
	return true
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// FiringPosition finds the nearest point from which the target can be shot: reachable (valid), a clear line of
// sight to the target, and within reach of it. Rings of 16 points at 100%, 75%, 50% and 30% of reach. ok is false when
// nothing qualifies, which means the target cannot be engaged at all.
func FiringPosition(me, target input.Point, reach int, valid func(input.Point) bool, los func(a, b input.Point) bool) (input.Point, bool) {
	best, bd, ok := input.Point{}, math.MaxFloat64, false
	for _, frac := range []float64{1, 0.75, 0.5, 0.3} {
		r := float64(reach) * frac
		if r < 2 {
			continue
		}
		for k := 0; k < 16; k++ {
			ang := float64(k) * math.Pi / 8
			p := input.Point{X: target.X + int(math.Round(r*math.Cos(ang))), Y: target.Y + int(math.Round(r*math.Sin(ang)))}
			if !los(p, target) || !valid(p) {
				continue
			}
			if d := pdist(me, p); d < bd {
				best, bd, ok = p, d, true
			}
		}
		if ok {
			return best, true // an outer ring that works beats walking closer
		}
	}
	return best, ok
}

// assessMaxDist: farther than this the grid between us and the target is usually still unexplored, so a missing path
// proves nothing; walk toward it and let the approach limit decide.
const assessMaxDist = 35

// plan says what to do about a target outside the casting range (or off screen).
type reachPlan int

const (
	planWalk reachPlan = iota // a path exists: approach as usual
	planFire                  // no path to it, but a firing position exists: walk there and shoot
	planNone                  // cannot be engaged
)

// assess decides, at most every 2 s per target, how to engage a target that is not in range yet.
func (f *Fighter) assess(tr *track, me input.Point, tg Target, reach int) (reachPlan, input.Point) {
	now := f.now()
	if !tr.assessAt.IsZero() && now.Sub(tr.assessAt) < 2*time.Second {
		return tr.plan, tr.firePos
	}
	tr.assessAt = now
	target := input.Point{X: tg.Unit.X, Y: tg.Unit.Y}
	tr.plan, tr.firePos = planWalk, input.Point{}
	if f.mv == nil {
		return tr.plan, tr.firePos
	}
	if _, err := f.mv.PathTo(target); err == nil || !f.known(target) || tg.Dist > assessMaxDist {
		return tr.plan, tr.firePos // a path, or the map is not loaded there yet: walk and see
	}
	// No path to the monster itself. Shoot across the gap if a point on our side of it works.
	if p, ok := FiringPosition(me, target, reach, f.valid, f.los); ok {
		tr.plan, tr.firePos = planFire, p
	} else {
		tr.plan = planNone
	}
	return tr.plan, tr.firePos
}

// inFiringSpot: close enough and with an open line, so a missing path does not matter.
func (f *Fighter) inFiringSpot(me, target input.Point, dist float64, reach int) bool {
	return dist <= float64(reach) && f.los(me, target)
}
