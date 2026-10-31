package boss

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// The Chaos Sanctuary seal executor (docs/act4.md section 6.1, work item A4-4). Geometry, layout detection and node
// paths are S (Kolbot Common.js `Common.Diablo`, copied in docs/prior-art/02-runs.md), never run against the live
// 1.14d game: everything here is U until the live checks A4-5 and A4-8. Object class ids are V (objects.txt, see
// quest/act4.go). The executor is pure data plus a loop over World/Actions/ObjectOps, so it is fully offline-testable.

// Seal object classes (V names "Seal", roles S).
const (
	SealInfector1 = 392
	SealInfector2 = 393
	SealSeis      = 394
	SealVizier1   = 395
	SealVizier2   = 396
)

// Super unique numbers of the three seal bosses (V superuniques: Infector 36, Lord De Seis 37, Grand Vizier 38).
const (
	uniqueInfector = 36
	uniqueSeis     = 37
	uniqueVizier   = 38
)

// ObjectOps is the optional capability behind seals, the Ancients' altar and Baal's portal: click a static object of
// the given class standing at (or near) at, and read its mode. Mode 0 = untouched, anything else = used (S for seals).
type ObjectOps interface {
	UseObject(ctx context.Context, class int, at Pt) error
	ObjectMode(ctx context.Context, class int, at Pt) (int, error)
}

// Chaos Sanctuary fixed points (S Kolbot).
var (
	ChaosEntrance = Pt{7790, 5544}
	ChaosStar     = Pt{7791, 5293}
)

// SealGroup is one of the three boss rooms.
type SealGroup struct {
	Name   string
	Seals  []int // objects clicked at the end of the path, in order
	Unique int   // super unique number of the boss that spawns
	Boss   string
	// Probe is the layout test: the preset of class Probe.Class sits at Probe.Axis (x or y) == Probe.Value in layout A.
	Probe Probe
	A, B  []Pt // paths from the star to the seal room, per layout
}

// Probe is the Kolbot getLayout test.
type Probe struct {
	Class int
	Y     bool // compare y instead of x
	Value int
}

// A reports whether the preset position is layout A.
func (p Probe) A(pos Pt) bool {
	if p.Y {
		return pos.Y == p.Value
	}
	return pos.X == p.Value
}

func pts(v ...int) []Pt {
	out := make([]Pt, 0, len(v)/2)
	for i := 0; i+1 < len(v); i += 2 {
		out = append(out, Pt{v[i], v[i+1]})
	}
	return out
}

// EntranceToStar is the walk from the Sanctuary entrance to the star (S).
var EntranceToStar = pts(7794, 5517, 7791, 5491, 7768, 5459, 7775, 5424, 7817, 5458, 7777, 5408, 7769, 5379, 7777, 5357, 7809, 5359, 7805, 5330, 7780, 5317, 7791, 5293)

// SealGroups in Kolbot's order: Vizier, Seis, Infector (S).
var SealGroups = []SealGroup{
	{Name: "vizier", Seals: []int{SealVizier1, SealVizier2}, Unique: uniqueVizier, Boss: "Grand Vizier of Chaos",
		Probe: Probe{Class: SealVizier2, Y: true, Value: 5275},
		A:     pts(7759, 5295, 7734, 5295, 7716, 5295, 7718, 5276, 7697, 5292, 7678, 5293, 7665, 5276, 7662, 5314),
		B:     pts(7759, 5295, 7734, 5295, 7716, 5295, 7701, 5315, 7666, 5313, 7653, 5284)},
	{Name: "seis", Seals: []int{SealSeis}, Unique: uniqueSeis, Boss: "Lord De Seis",
		Probe: Probe{Class: SealSeis, Value: 7773},
		A:     pts(7781, 5259, 7805, 5258, 7802, 5237, 7776, 5228, 7775, 5205, 7804, 5193, 7814, 5169, 7788, 5153),
		B:     pts(7781, 5259, 7805, 5258, 7802, 5237, 7776, 5228, 7811, 5218, 7807, 5194, 7779, 5193, 7774, 5160, 7803, 5154)},
	{Name: "infector", Seals: []int{SealInfector1, SealInfector2}, Unique: uniqueInfector, Boss: "Infector of Souls",
		Probe: Probe{Class: SealInfector1, Value: 7893},
		A:     pts(7809, 5268, 7834, 5306, 7852, 5280, 7852, 5310, 7869, 5294, 7895, 5295, 7919, 5290),
		B:     pts(7809, 5268, 7834, 5306, 7852, 5280, 7852, 5310, 7869, 5294, 7895, 5274, 7927, 5275, 7932, 5297, 7923, 5313)},
}

// Layout returns the node path of g for the given presets and whether layout A was detected. When the probe seal is
// missing from the presets (they may not stream before the room does, U) it reports ok = false and A is returned.
func (g SealGroup) Layout(ps []Preset) (path []Pt, a, ok bool) {
	for _, p := range ps {
		if p.Kind == PresetObject && p.ID == g.Probe.Class {
			if g.Probe.A(p.Pos) {
				return g.A, true, true
			}
			return g.B, false, true
		}
	}
	return g.A, true, false
}

// SealOpts tunes OpenSeals. Zero values take the defaults.
type SealOpts struct {
	ClearRadius int           // ClearAround radius at each node (default 30, S)
	ClearBudget time.Duration // per node (default 8 s)
	Tries       int           // clicks per seal (default 5, S)
	BossWait    time.Duration // wait for the seal boss to show after the last seal of a group (default 12 s)
	BossBudget  time.Duration // fight budget per seal boss (default 3 min)
	DiabloWait  time.Duration // wait for Diablo after the last seal (default 30 s, S)
	Slice       time.Duration // one Attack call (default 3 s)
	Poll        time.Duration // default 250 ms
}

func (o SealOpts) withDefaults() SealOpts {
	if o.ClearRadius <= 0 {
		o.ClearRadius = 30
	}
	if o.ClearBudget <= 0 {
		o.ClearBudget = 8 * time.Second
	}
	if o.Tries <= 0 {
		o.Tries = 5
	}
	if o.BossWait <= 0 {
		o.BossWait = 12 * time.Second
	}
	if o.BossBudget <= 0 {
		o.BossBudget = 3 * time.Minute
	}
	if o.DiabloWait <= 0 {
		o.DiabloWait = 30 * time.Second
	}
	if o.Slice <= 0 {
		o.Slice = 3 * time.Second
	}
	if o.Poll <= 0 {
		o.Poll = 250 * time.Millisecond
	}
	return o
}

// SealResult says what OpenSeals did.
type SealResult struct {
	Layouts  map[string]string // group name -> "A", "B" or "A?" when the probe preset was missing
	Opened   []int             // seal classes that read as open
	Bosses   []string          // seal bosses seen dead
	Unkilled []string          // seal bosses still alive after the fight budget and the retry
	Notes    []string
	Diablo   bool // Diablo showed up in the monster list
	DiabloAt Pt
}

// ErrSealStuck means a seal never read as open after all tries.
var ErrSealStuck = fmt.Errorf("boss: seal did not open")

// OpenSeals walks entrance -> star -> each seal room, clears around every node, opens the seals, kills the room's
// boss, and finally waits for Diablo. It starts from wherever the character is in the Chaos Sanctuary: seals that
// already read as open are skipped, so a restart mid-sequence resumes. A Diablo already in the monster list ends it
// at once. Errors from Actions pass through (ErrDead and friends via passFatal).
func OpenSeals(ctx context.Context, w World, a Actions, ops ObjectOps, o SealOpts) (SealResult, error) {
	o = o.withDefaults()
	res := SealResult{Layouts: map[string]string{}}
	if v, err := w.View(ctx); err != nil {
		return res, err
	} else if m, ok := findClass(v, ClassDiablo); ok && m.Alive() {
		res.Diablo, res.DiabloAt = true, m.Pos
		return res, nil
	}
	ps, err := w.Presets(ctx)
	if err != nil {
		res.Notes = append(res.Notes, fmt.Sprintf("presets: %v", err))
	}
	walk := func(path []Pt) error {
		for _, p := range path {
			if v, err := w.View(ctx); err != nil {
				return err
			} else if v.Dead {
				return ErrDead
			}
			if err := a.GoTo(ctx, p); err != nil {
				if fe := passFatal(err); fe != nil {
					return fe
				}
				res.Notes = append(res.Notes, fmt.Sprintf("goto %v: %v", p, err))
			}
			if v, err := w.View(ctx); err == nil && v.Dead {
				return ErrDead
			}
			if err := a.ClearAround(ctx, p, o.ClearRadius, o.ClearBudget); err != nil {
				if fe := passFatal(err); fe != nil {
					return fe
				}
			}
		}
		return nil
	}
	v, err := w.View(ctx)
	if err != nil {
		return res, err
	}
	paths := make([][]Pt, len(SealGroups))
	for i, g := range SealGroups {
		path, isA, ok := g.Layout(ps)
		tag := "B"
		if isA {
			tag = "A"
		}
		if !ok {
			tag += "?"
			res.Notes = append(res.Notes, g.Name+": probe seal not in the presets, assuming layout A")
		}
		res.Layouts[g.Name] = tag
		paths[i] = path
	}
	if v.Pos.Dist(ChaosStar) > 60 {
		if err := walk(routeToStar(v.Pos, paths)); err != nil {
			return res, err
		}
	}
	var stubborn []int // group indexes whose boss outlived its fight budget
	for gi, g := range SealGroups {
		path := paths[gi]
		// Probe presets give the seal positions, so each seal is clicked where the game put it.
		open := 0
		for _, s := range g.Seals {
			at := sealPos(ps, s, path)
			if mode, err := ops.ObjectMode(ctx, s, at); err == nil && mode != 0 {
				res.Opened = append(res.Opened, s)
				open++
			}
		}
		if open == len(g.Seals) {
			continue
		}
		if err := walk(path); err != nil {
			return res, err
		}
		for _, s := range g.Seals {
			if containsInt(res.Opened, s) {
				continue
			}
			if err := openSeal(ctx, w, a, ops, o, s, sealPos(ps, s, path)); err != nil {
				if fe := passFatal(err); fe != nil {
					return res, fe // death, chicken and escape are not a stuck seal
				}
				return res, fmt.Errorf("%w: %s seal %d: %v", ErrSealStuck, g.Name, s, err)
			}
			res.Opened = append(res.Opened, s)
		}
		killed, err := killSealBoss(ctx, w, a, o, g)
		switch {
		case errors.Is(err, ErrTimeout):
			res.Notes = append(res.Notes, g.Name+": boss not killed in budget, retrying at the end")
			stubborn = append(stubborn, gi)
		case err != nil:
			return res, err
		case killed:
			res.Bosses = append(res.Bosses, g.Boss)
		default:
			res.Notes = append(res.Notes, g.Name+": boss not seen (a pack may have killed it or the spawn is U)")
		}
		if err := walk(reverse(path)); err != nil { // back to the star
			return res, err
		}
		if err := a.GoTo(ctx, ChaosStar); err != nil {
			if fe := passFatal(err); fe != nil {
				return res, fe
			}
		}
	}
	// One bounded retry per stubborn boss; Diablo only needs the seals, so a boss that stays alive is not fatal.
	for _, gi := range stubborn {
		g, path := SealGroups[gi], paths[gi]
		if err := walk(path); err != nil {
			return res, err
		}
		killed, err := killSealBoss(ctx, w, a, o, g)
		switch {
		case errors.Is(err, ErrTimeout):
			res.Unkilled = append(res.Unkilled, g.Boss)
			res.Notes = append(res.Notes, g.Name+": boss still alive after the retry")
		case err != nil:
			return res, err
		case killed:
			res.Bosses = append(res.Bosses, g.Boss)
		}
		if err := walk(reverse(path)); err != nil {
			return res, err
		}
		if err := a.GoTo(ctx, ChaosStar); err != nil {
			if fe := passFatal(err); fe != nil {
				return res, fe
			}
		}
	}
	// Diablo: Kolbot waits for the lights packet; the monster list poll is our substitute (U).
	end := w.Now().Add(o.DiabloWait)
	for {
		v, err := w.View(ctx)
		if err != nil {
			return res, err
		}
		if v.Dead {
			return res, ErrDead
		}
		if m, ok := findClass(v, ClassDiablo); ok && m.Alive() {
			res.Diablo, res.DiabloAt = true, m.Pos
			return res, nil
		}
		if !w.Now().Before(end) {
			return res, ErrBossNotFound
		}
		if err := w.Sleep(ctx, o.Poll); err != nil {
			return res, err
		}
	}
}

// routeToStar is the walk from pos back to the star, starting at the nearest known node: the entrance route, or the
// reverse of a seal room path when the character stands in one.
func routeToStar(pos Pt, paths [][]Pt) []Pt {
	bd := 1 << 30
	var bestPath []Pt
	pick := func(nodes []Pt, rev bool) {
		for i, n := range nodes {
			if d := pos.Dist(n); d < bd {
				bd = d
				if rev {
					bestPath = reverse(nodes[:i+1])
				} else {
					bestPath = nodes[i:]
				}
			}
		}
	}
	pick(EntranceToStar, false)
	for _, p := range paths {
		pick(p, true)
	}
	if bd > 150 { // nowhere near the Sanctuary route: start at the entrance
		return EntranceToStar
	}
	return append(bestPath, ChaosStar)
}

// sealPos is the preset position of the seal, else the last node of the group's path.
func sealPos(ps []Preset, class int, path []Pt) Pt {
	for _, p := range ps {
		if p.Kind == PresetObject && p.ID == class {
			return p.Pos
		}
	}
	return path[len(path)-1]
}

func openSeal(ctx context.Context, w World, a Actions, ops ObjectOps, o SealOpts, class int, at Pt) error {
	if err := a.GoTo(ctx, at); err != nil {
		if fe := passFatal(err); fe != nil {
			return fe
		}
	}
	var last error
	for i := 0; i < o.Tries; i++ {
		if err := ops.UseObject(ctx, class, at); err != nil {
			last = err
			if fe := passFatal(err); fe != nil {
				return fe
			}
			continue
		}
		mode, err := ops.ObjectMode(ctx, class, at)
		if err == nil && mode != 0 {
			return nil
		}
		last = fmt.Errorf("mode %d err %v", mode, err)
	}
	if v, err := w.View(ctx); err == nil && v.Dead {
		return ErrDead // a dead character cannot click: report the death, not a broken seal
	}
	return last
}

func killSealBoss(ctx context.Context, w World, a Actions, o SealOpts, g SealGroup) (bool, error) {
	match := Match{UniqueNo: g.Unique, Name: g.Boss}
	wait := w.Now().Add(o.BossWait)
	deadline := w.Now().Add(o.BossBudget)
	seen := false
	for {
		v, err := w.View(ctx)
		if err != nil {
			return seen, err
		}
		if v.Dead {
			return seen, ErrDead
		}
		var boss *Mon
		for i := range v.Monsters {
			if match.Is(v.Monsters[i]) {
				boss = &v.Monsters[i]
			}
		}
		switch {
		case boss != nil && !boss.Alive():
			return true, nil
		case boss != nil:
			seen = true
			if !w.Now().Before(deadline) {
				return seen, ErrTimeout
			}
			if err := a.Attack(ctx, boss.ID, o.Slice); err != nil {
				if fe := passFatal(err); fe != nil {
					return seen, fe
				}
				if err := w.Sleep(ctx, o.Poll); err != nil { // a failing Attack must not become a hot loop
					return seen, err
				}
			}
		case seen:
			return true, nil // vanished after being seen: treated as dead
		case !w.Now().Before(wait):
			return false, nil
		default:
			if err := a.ClearAround(ctx, v.Pos, o.ClearRadius, o.ClearBudget); err != nil {
				if fe := passFatal(err); fe != nil {
					return seen, fe
				}
			}
			if err := w.Sleep(ctx, o.Poll); err != nil {
				return seen, err
			}
		}
	}
}

func findClass(v View, class int) (Mon, bool) {
	for _, m := range v.Monsters {
		if m.Class == class {
			return m, true
		}
	}
	return Mon{}, false
}

func containsInt(s []int, x int) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

func reverse(p []Pt) []Pt {
	out := make([]Pt, len(p))
	for i, x := range p {
		out[len(p)-1-i] = x
	}
	return out
}
