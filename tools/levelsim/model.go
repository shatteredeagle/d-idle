package levelsim

import "math"

// Params are the fitted and swept knobs of the model.
type Params struct {
	RateScale  float64 // multiplies every area's fresh kill rate
	Decay      float64 // exponent g: rate = r0 (floor + (1-floor) f^g), f = remaining fraction
	Floor      float64 // rate floor as a share of r0 (roaming to the last monsters)
	Yield      float64 // share of the area's units a sweep actually kills (163 of 198 in Cold Plains)
	RestartMin float64 // Save and Exit to the next town arrival (V: 15 s)
	TownTax    float64 // share of game time spent on chores (S: 5 to 8 percent), policies and the optimiser only
	WalkScale  float64 // multiplies the graph's walking minutes
	HopMin     float64 // waypoint hop or town portal, minutes
	XPScale    float64 // multiplies XP per kill
	Depleted   float64 // remaining fraction below which an area counts as swept
	DenClear   float64 // remaining fraction at which the Den quest is done
}

// DefaultParams are the values fitted by `levelsim -calibrate` (docs/levelsim.md section 3).
func DefaultParams() Params {
	return Params{RateScale: 1.0, Decay: 0.25, Floor: 0.5, Yield: 1.0, RestartMin: 0.25, TownTax: 0.065,
		WalkScale: 1, HopMin: 0.35, XPScale: 1, Depleted: 0.08, DenClear: 0.12}
}

// Mode picks the hard danger floors (docs/levelsim.md section 5).
type Mode string

const (
	// ModeStrict: monster level proxy (mean mlvl - 1) and the boss minimum safe levels of boss1-trace (I).
	ModeStrict Mode = "strict"
	// ModePlan: today's plan entry levels (melee) where a row exists, strict elsewhere.
	ModePlan Mode = "plan"
	// ModeBold: mean mlvl - 2, only the quest floors (Den 3, Blood Raven 8) from the bosses.
	ModeBold Mode = "bold"
)

// Env bundles the tables, the parameters and the danger mode.
type Env struct {
	D    *Data
	P    Params
	Mode Mode
	// AllowDenRefarm lets a strategy farm the Den again after the quest (the plan excludes it, levelplan.md change 1).
	AllowDenRefarm bool
	// NoDenRequired drops the constraint that the Den quest is done (toy tables).
	NoDenRequired bool
	adj           map[int][]edgeTo
	wp            uint64
	walk          [][nNodes]float64
	next          [][nNodes]int
}

type edgeTo struct {
	to  int
	min float64
}

// NewEnv builds an environment.
func NewEnv(d *Data, p Params, m Mode) *Env {
	e := &Env{D: d, P: p, Mode: m, adj: map[int][]edgeTo{}}
	for _, ed := range d.Graph.Edges {
		a, b := int(ed[0]), int(ed[1])
		e.adj[a] = append(e.adj[a], edgeTo{b, ed[2]})
		e.adj[b] = append(e.adj[b], edgeTo{a, ed[2]})
	}
	for _, w := range d.Graph.Waypoints {
		e.wp |= 1 << uint(w)
	}
	return e
}

// State is a point of a game: time, XP, place and what is cleared in the current game.
type State struct {
	T      float64
	XP     float64
	Loc    int
	Kills  [40]float64 // kills per area id in the current game (a restart zeroes it)
	WP     uint64      // activated waypoints (kept across games)
	Den    bool
	BR     bool
	Games  int
	TotalK float64
	cl     int                   // cached level
	LvlT   [MaxLevel + 1]float64 // first time each level was reached (0 = not yet)
}

// NewState is a fresh hero in the Rogue Encampment.
func NewState() State {
	s := State{Loc: 1, cl: 1}
	s.LvlT[1] = 0.0001
	return s
}

// Clvl is the character level.
func (s *State) Clvl() int {
	for s.cl < MaxLevel && s.XP >= expTable[s.cl+1] {
		s.cl++
	}
	return s.cl
}

func (e *Env) floorFor(a *Area) int {
	strict := int(math.Ceil(a.MLvl - 1))
	if a.UniqFloor > strict {
		strict = a.UniqFloor
	}
	var fl int
	switch e.Mode {
	case ModePlan:
		// today's entry levels where the plan has a row, else the same rule of thumb the rows follow
		// (mean monster level + 2.5, rounded up: rows run +2.4 to +4.3)
		fl = int(math.Ceil(a.MLvl + 2.5))
		if a.PlanEnter > 0 {
			fl = a.PlanMelee
		}
		if a.ID == 8 {
			fl = 3
		}
		if a.ID == 17 {
			fl = 8
		}
	case ModeBold:
		fl = int(math.Ceil(a.MLvl - 2))
		switch a.ID {
		case 8:
			fl = 3
		case 17:
			fl = 8
		}
	default:
		fl = strict
	}
	if fl < 1 {
		fl = 1
	}
	return fl
}

// Floor is the hard entry floor of an area in the environment's mode.
func (e *Env) Floor(id int) int { return e.floorFor(e.D.Areas[id]) }

// Remaining is the share of the area still alive in the current game (0..1).
func (e *Env) Remaining(s *State, a *Area) float64 {
	f := 1 - s.Kills[a.ID]/(e.P.Yield*a.Units)
	if f < 0 {
		return 0
	}
	return f
}

func (e *Env) rate(a *Area, f float64) float64 {
	return e.P.RateScale * a.Rate * (e.P.Floor + (1-e.P.Floor)*math.Pow(f, e.P.Decay))
}

// Exit is when a farm slice ends.
type Exit struct {
	MinF    float64 // leave when the remaining share is at or below this
	Until   int     // leave when clvl reaches this (0 = no)
	OnLevel bool    // leave at the next level-up
	MaxMin  float64 // field minutes cap (0 = none)
	TMax    float64 // absolute clock cap (0 = none)
	NoTax   bool    // replay of an observed segment: no chore time added
}

const stepMin = 0.1

// Farm advances the state by killing in area a until the exit fires. It returns the field minutes used.
func (e *Env) Farm(s *State, a *Area, ex Exit, sample func(t, xp, kills float64)) float64 {
	start := s.Clvl()
	tax := e.P.TownTax
	if ex.NoTax {
		tax = 0
	}
	var field float64
	for {
		if s.Kills[a.ID] >= a.Units {
			break
		}
		f := e.Remaining(s, a)
		if f <= ex.MinF {
			break
		}
		if ex.MaxMin > 0 && field >= ex.MaxMin-1e-9 {
			break
		}
		if ex.TMax > 0 && s.T >= ex.TMax-1e-9 {
			break
		}
		dt := stepMin
		if ex.MaxMin > 0 && field+dt > ex.MaxMin {
			dt = ex.MaxMin - field
		}
		if ex.TMax > 0 && s.T+dt/(1-tax) > ex.TMax {
			dt = (ex.TMax - s.T) * (1 - tax)
		}
		if dt <= 0 {
			break
		}
		k := e.rate(a, f) * dt
		if left := a.Units - s.Kills[a.ID]; k > left {
			k = left
			dt = k / e.rate(a, f)
		}
		cl := s.Clvl()
		s.XP += k * a.XPPerKill(cl) * e.P.XPScale
		s.Kills[a.ID] += k
		s.TotalK += k
		field += dt
		s.T += dt / (1 - tax)
		e.noteLevels(s)
		if a.ID == 8 && !s.Den && e.Remaining(s, a) <= e.P.DenClear {
			s.Den = true
		}
		if a.ID == 17 && !s.BR && e.Remaining(s, a) <= 0.3 {
			s.BR = true
			s.XP += 181 // Blood Raven's own kill (V, boss1-trace)
			s.T += 1.0  // claim the Rogue
			e.noteLevels(s)
		}
		if sample != nil {
			sample(s.T, s.XP, s.TotalK)
		}
		if ex.OnLevel && s.Clvl() > start {
			break
		}
		if ex.Until > 0 && s.Clvl() >= ex.Until {
			break
		}
	}
	return field
}

func (e *Env) noteLevels(s *State) {
	for l := s.Clvl(); l > 1 && s.LvlT[l] == 0; l-- {
		s.LvlT[l] = s.T
	}
}

// Restart saves and exits: every area respawns, the hero stands in town.
func (e *Env) Restart(s *State) {
	s.Kills = [40]float64{}
	s.Loc = 1
	s.T += e.P.RestartMin
	s.Games++
}

// Travel moves the hero to area `to` and returns the minutes it took (graph walk or waypoint).
func (e *Env) Travel(s *State, to int) float64 {
	if s.Loc == to {
		return 0
	}
	cost, path := e.shortest(s, s.Loc, to)
	s.T += cost
	for _, n := range path {
		if e.wp&(1<<uint(n)) != 0 {
			s.WP |= 1 << uint(n)
		}
	}
	s.Loc = to
	return cost
}

// shortest is the cheapest way from `from` to `to`: walk the graph, or take a town portal and then a
// waypoint (or the walk from town / an activated waypoint). It returns the minutes and the nodes walked.
func (e *Env) shortest(s *State, from, to int) (float64, []int) {
	e.paths()
	best, route := e.walk[from][to], e.walkPath(from, to)
	if from != 1 {
		hop := e.P.HopMin
		if c := hop + e.walk[1][to]; c < best {
			best, route = c, e.walkPath(1, to)
		}
		for w := 2; w < 64; w++ {
			if s.WP&(1<<uint(w)) != 0 {
				if c := hop + e.P.HopMin + e.walk[w][to]; c < best {
					best, route = c, e.walkPath(w, to)
				}
			}
		}
	} else {
		for w := 2; w < 64; w++ {
			if s.WP&(1<<uint(w)) != 0 {
				if c := e.P.HopMin + e.walk[w][to]; c < best {
					best, route = c, e.walkPath(w, to)
				}
			}
		}
	}
	return best, route
}

const nNodes = 64

// paths fills the all-pairs walking minutes (Floyd-Warshall over the graph, scaled by WalkScale).
func (e *Env) paths() {
	if e.walk != nil {
		return
	}
	inf := math.Inf(1)
	e.walk = make([][nNodes]float64, nNodes)
	e.next = make([][nNodes]int, nNodes)
	for i := range e.walk {
		for j := range e.walk[i] {
			e.walk[i][j] = inf
			e.next[i][j] = -1
		}
		e.walk[i][i] = 0
		e.next[i][i] = i
	}
	for a, list := range e.adj {
		for _, ed := range list {
			e.walk[a][ed.to] = ed.min * e.P.WalkScale
			e.next[a][ed.to] = ed.to
		}
	}
	for k := 0; k < nNodes; k++ {
		for i := 0; i < nNodes; i++ {
			for j := 0; j < nNodes; j++ {
				if c := e.walk[i][k] + e.walk[k][j]; c < e.walk[i][j] {
					e.walk[i][j] = c
					e.next[i][j] = e.next[i][k]
				}
			}
		}
	}
}

func (e *Env) walkPath(from, to int) []int {
	var out []int
	if e.next[from][to] < 0 {
		return out
	}
	for n := from; n != to; {
		n = e.next[n][to]
		out = append(out, n)
	}
	return out
}
