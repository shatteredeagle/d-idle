package boss

import (
	"context"
	"fmt"
	"time"
)

// The Throne of Destruction wave loop (docs/act5.md section 6, work item A5-5). Geometry and the clear nodes are S
// (Kolbot Baal.js, docs/prior-art/02-runs.md); wave 1, 2 and 4 class ids are U (never dumped), wave 3 (class 557) and
// wave 5 (571..573) are S. Nothing here ran against the live 1.14d game.

// Throne rectangle: x 15072..15118, y < 5080 is inside (S); the archway is at (15097,5099).
const (
	throneX0, throneX1 = 15072, 15118
	throneYMax         = 5080
)

// Wave classes of the later waves (S). Waves 1, 2 and 4 are counted by transition only.
const (
	waveCouncil = 3
	waveLister  = 5
)

// Throne points (S).
var (
	ThroneWait      = Pt{15095, 5029} // first stand point
	ThroneBaalSpot  = Pt{15090, 5014}
	ThronePrePortal = Pt{15090, 5008}
	ChamberStand    = Pt{15134, 5923}
	// ThroneClearNodes are visited every ClearEvery while no wave monster is in reach, skipping empty nodes (S).
	ThroneClearNodes = pts(15097, 5054, 15079, 5014, 15085, 5053, 15085, 5040, 15098, 5040, 15099, 5022, 15086, 5024, 15079, 5014)
)

// InThrone reports whether p is inside the throne rectangle.
func InThrone(p Pt) bool { return p.X >= throneX0 && p.X <= throneX1 && p.Y < throneYMax }

// WaveOpts tunes RunWaves. Zero values take the defaults.
type WaveOpts struct {
	Timeout      time.Duration // whole loop (default 30 min, S)
	ClearEvery   time.Duration // idle clear sweep period (default 20 s, S)
	NodeRadius   int           // monsters within this of a node make it worth a visit (default 30, S)
	ClearBudget  time.Duration // per node (default 10 s)
	Slice        time.Duration // one Attack call (default 3 s)
	Poll         time.Duration // default 250 ms
	BaalGoneWait time.Duration // wait for the throne Baal to leave after wave 5 (default 3 min, S)
	SeenWait     time.Duration // without a throne Baal in view after this long the loop assumes the waves are over (default 10 s)
	DrinkPct     int           // default 50
	ChickenPct   int           // default 12
	// AbortClasses end the run when one is in view: souls and dolls in the throne (S Kolbot quits on them; ids U).
	AbortClasses []int
}

func (o WaveOpts) withDefaults() WaveOpts {
	if o.Timeout <= 0 {
		o.Timeout = 30 * time.Minute
	}
	if o.ClearEvery <= 0 {
		o.ClearEvery = 20 * time.Second
	}
	if o.NodeRadius <= 0 {
		o.NodeRadius = 30
	}
	if o.ClearBudget <= 0 {
		o.ClearBudget = 10 * time.Second
	}
	if o.Slice <= 0 {
		o.Slice = 3 * time.Second
	}
	if o.Poll <= 0 {
		o.Poll = 250 * time.Millisecond
	}
	if o.BaalGoneWait <= 0 {
		o.BaalGoneWait = 3 * time.Minute
	}
	if o.SeenWait <= 0 {
		o.SeenWait = 10 * time.Second
	}
	if o.DrinkPct <= 0 {
		o.DrinkPct = 50
	}
	if o.ChickenPct <= 0 {
		o.ChickenPct = 12
	}
	return o
}

// WaveResult says what RunWaves saw.
type WaveResult struct {
	Waves    int   // transitions from "nothing to fight" to "something to fight" in the throne
	Named    []int // waves recognised by class: 3 (council), 5 (Lister)
	BaalGone bool
	Notes    []string
}

// RunWaves fights the five waves from the throne until the throne Baal (class 543) has left, then steps to the
// pre-portal point. It must be called inside the Throne of Destruction. The loop: attack the nearest living hostile in
// the rectangle; when none is left, every ClearEvery visit the clear nodes that still hold monsters; drink and chicken
// by life. Errors: ErrDead, ErrChicken, ErrLeftArena, ErrAborted (abort class seen), ErrTimeout.
// U: wave 3's hydras (not ours, they expire) are fought like any monster here; Kolbot waits in a corner instead.
func RunWaves(ctx context.Context, w World, a Actions, o WaveOpts) (WaveResult, error) {
	o = o.withDefaults()
	var res WaveResult
	start := w.Now()
	deadline := start.Add(o.Timeout)
	var lastClear, lastDrink time.Time
	var baalSeen bool
	var lister5 time.Time // when wave 5 was last seen cleared while the throne Baal still stands
	fighting := false
	if err := a.GoTo(ctx, ThroneWait); err != nil {
		if fe := passFatal(err); fe != nil {
			return res, fe
		}
		res.Notes = append(res.Notes, fmt.Sprintf("goto wait spot: %v", err))
	}
	lastClear = w.Now()
	for {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		v, err := w.View(ctx)
		if err != nil {
			return res, err
		}
		switch {
		case v.Dead:
			return res, ErrDead
		case v.Area != AreaThrone:
			return res, ErrLeftArena
		case !w.Now().Before(deadline):
			return res, ErrTimeout
		}
		for _, m := range v.Hostiles() {
			if containsInt(o.AbortClasses, m.Class) {
				return res, fmt.Errorf("%w: class %d in the throne", ErrAborted, m.Class)
			}
		}
		if p := v.LifePct(); p < o.ChickenPct {
			if err := a.Chicken(ctx); err != nil {
				return res, err
			}
			return res, ErrChicken
		} else if p < o.DrinkPct && v.Potions > 0 && w.Now().Sub(lastDrink) > 3*time.Second {
			lastDrink = w.Now()
			if err := a.DrinkHealing(ctx); err != nil {
				if fe := passFatal(err); fe != nil {
					return res, fe
				}
			}
			continue
		}
		baal, baalHere := Mon{}, false
		var foes []Mon
		for _, m := range v.Hostiles() {
			switch {
			case m.Class == ClassBaalThrone:
				baal, baalHere = m, true
			case InThrone(m.Pos):
				foes = append(foes, m)
			}
		}
		if baalHere {
			baalSeen = true
		}
		if len(foes) > 0 {
			if !fighting {
				fighting = true
				res.Waves++
			}
			noteWave(&res, foes)
			lister5 = time.Time{}
			if err := a.Attack(ctx, nearest(foes, v.Pos).ID, o.Slice); err != nil {
				if fe := passFatal(err); fe != nil {
					return res, fe
				}
				if err := w.Sleep(ctx, o.Poll); err != nil { // a failing Attack must not become a hot loop
					return res, err
				}
			}
			lastClear = w.Now()
			continue
		}
		fighting = false
		if !baalHere {
			if baalSeen || w.Now().Sub(start) >= o.SeenWait {
				if !baalSeen {
					res.Notes = append(res.Notes, "throne Baal never seen: assuming the waves are over")
				}
				res.BaalGone = true
				break
			}
		} else if lister5.IsZero() && containsInt(res.Named, waveLister) {
			lister5 = w.Now()
		}
		if baalHere && !lister5.IsZero() && w.Now().Sub(lister5) >= o.BaalGoneWait {
			res.Notes = append(res.Notes, fmt.Sprintf("throne Baal still at %v after wave 5 and %v", baal.Pos, o.BaalGoneWait))
			return res, ErrTimeout
		}
		if w.Now().Sub(lastClear) >= o.ClearEvery {
			lastClear = w.Now()
			if err := sweep(ctx, w, a, v, o); err != nil {
				return res, err
			}
			continue
		}
		if err := w.Sleep(ctx, o.Poll); err != nil {
			return res, err
		}
	}
	if err := a.GoTo(ctx, ThronePrePortal); err != nil {
		if fe := passFatal(err); fe != nil {
			return res, fe
		}
	}
	return res, nil
}

func noteWave(res *WaveResult, foes []Mon) {
	for _, m := range foes {
		switch m.Class {
		case ClassCouncilA5:
			addNamed(res, waveCouncil)
		case ClassLister, 572, 573:
			addNamed(res, waveLister)
		}
	}
}

func addNamed(res *WaveResult, n int) {
	if !containsInt(res.Named, n) {
		res.Named = append(res.Named, n)
	}
}

func nearest(ms []Mon, p Pt) Mon {
	best := ms[0]
	for _, m := range ms[1:] {
		if m.Pos.Dist(p) < best.Pos.Dist(p) {
			best = m
		}
	}
	return best
}

// sweep visits every clear node with a monster within NodeRadius, in order (S clearThrone).
func sweep(ctx context.Context, w World, a Actions, v View, o WaveOpts) error {
	for _, n := range ThroneClearNodes {
		busy := false
		for _, m := range v.Hostiles() {
			if m.Class != ClassBaalThrone && m.Pos.Dist(n) <= o.NodeRadius {
				busy = true
				break
			}
		}
		if !busy {
			continue
		}
		if err := a.GoTo(ctx, n); err != nil {
			if fe := passFatal(err); fe != nil {
				return fe
			}
		}
		if err := a.ClearAround(ctx, n, o.NodeRadius, o.ClearBudget); err != nil {
			if fe := passFatal(err); fe != nil {
				return fe
			}
		}
	}
	return nil
}

// RunBaal is the whole Eve of Destruction leg: reach the Throne (WSK2 waypoint, two hops), fight the waves, step
// through the Worldstone Chamber portal (object 563, preset position when listed), then kill Baal with the shared
// runner. From the chamber it goes straight to the fight; from the throne it skips the walk. Opts.Hooks apply to the
// Baal fight only.
func RunBaal(ctx context.Context, w World, a Actions, ops ObjectOps, wo WaveOpts, o Opts) (Result, error) {
	s := baal()
	out := Result{Boss: s.Name}
	v, err := w.View(ctx)
	if err != nil {
		return out, err
	}
	if v.Dead {
		return out, ErrDead
	}
	if f := s.MinLevel[clampDiff(v.Difficulty)]; f > 0 && v.Level < f {
		return out, fmt.Errorf("%w: %s needs clvl %d, have %d", ErrTooWeak, s.Name, f, v.Level)
	}
	t0 := w.Now()
	if v.Area != AreaWorldstoneChamb {
		if v.Area != AreaThrone {
			if err := walkChain(ctx, s, w, a, v, s.Route); err != nil {
				return out, err
			}
		}
		wr, err := RunWaves(ctx, w, a, wo)
		out.Phases = append(out.Phases, "waves")
		out.Notes = append(out.Notes, wr.Notes...)
		out.Notes = append(out.Notes, fmt.Sprintf("waves %d named %v", wr.Waves, wr.Named))
		if err != nil {
			out.Elapsed = w.Now().Sub(t0)
			return out, err
		}
		if ops == nil {
			return out, fmt.Errorf("%w: no object capability for the portal", ErrAborted)
		}
		ps, _ := w.Presets(ctx)
		at, ok := presetPos(ps, objBaalPortal)
		if !ok {
			out.Notes = append(out.Notes, "Baal portal preset not found, clicking at the pre-portal point")
			at = ThronePrePortal
		}
		if err := a.GoTo(ctx, at); err != nil {
			if fe := passFatal(err); fe != nil {
				return out, fe
			}
		}
		if err := ops.UseObject(ctx, objBaalPortal, at); err != nil {
			return out, fmt.Errorf("boss: Baal portal: %w", err)
		}
		if err := waitArea(ctx, w, AreaWorldstoneChamb, 10*time.Second); err != nil {
			return out, err
		}
		out.Phases = append(out.Phases, "portal")
	}
	res, err := Run(ctx, s, w, a, o)
	res.Phases = append(out.Phases, res.Phases...)
	res.Notes = append(out.Notes, res.Notes...)
	res.Elapsed = w.Now().Sub(t0)
	return res, err
}

func waitArea(ctx context.Context, w World, area int, d time.Duration) error {
	end := w.Now().Add(d)
	for {
		v, err := w.View(ctx)
		if err != nil {
			return err
		}
		if v.Area == area {
			return nil
		}
		if !w.Now().Before(end) {
			return fmt.Errorf("%w: wanted area %d, in %d", ErrLeftArena, area, v.Area)
		}
		if err := w.Sleep(ctx, 250*time.Millisecond); err != nil {
			return err
		}
	}
}
