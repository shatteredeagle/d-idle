package levelsim

import (
	"fmt"
	"sort"
)

// Step is one slice of a trajectory.
type Step struct {
	Kind         string // farm, restart
	Area         int
	T0, T1       float64
	Travel       float64 // minutes of the walk or hop before the slice
	XP0, XP1     float64
	Clvl0, Clvl1 int
	Kills        float64
	Exit         string
}

// Row is one line of a policy: farm Area while Enter <= clvl < Leave and more than MinF of it is alive.
type Row struct {
	Area  int     `json:"area"`
	Enter int     `json:"enter"`
	Leave int     `json:"leave"`
	MinF  float64 `json:"min_f"` // leave at this remaining share; 0 = the model's swept threshold
}

// Policy is an ordered row list plus the restart rule.
type Policy struct {
	Name    string `json:"name"`
	Rows    []Row  `json:"rows"`
	Restart bool   `json:"restart"` // save and exit when no row is open (a fresh game respawns everything)
}

// Result is one simulated trajectory.
type Result struct {
	Name       string
	H          float64
	End        State
	Steps      []Step
	Violations []string // entries below the mode's hard floor
}

// Clvl at the end of the horizon.
func (r *Result) Clvl() int { return r.End.Clvl() }

// PlanToday is progression-normal.json (v2 plus the levelplan changes): the farm rows in file order, the Den
// at the campaign floor 3, Burial Grounds at the melee entry 8. docs/levelplan.md.
func PlanToday(d *Data) Policy {
	p := Policy{Name: "plan today", Restart: true}
	for _, id := range d.Order {
		a := d.Areas[id]
		if !a.PlanFarm {
			continue
		}
		enter := a.PlanMelee
		if id == 8 {
			enter = 3
		}
		p.Rows = append(p.Rows, Row{Area: id, Enter: enter, Leave: a.PlanLeave})
	}
	// file order is the plan's order, not the area id order
	order := map[int]int{2: 0, 8: 1, 3: 2, 17: 3, 9: 4, 4: 5, 5: 6, 6: 7, 13: 8, 18: 9}
	sort.SliceStable(p.Rows, func(i, j int) bool { return order[p.Rows[i].Area] < order[p.Rows[j].Area] })
	return p
}

// RunPolicy plays a policy for H minutes.
func (e *Env) RunPolicy(pol Policy, H float64) Result {
	res := Result{Name: pol.Name, H: H}
	s := NewState()
	seen := map[string]bool{}
	justRestarted := false
	for guard := 0; s.T < H && guard < 2000; guard++ {
		cl := s.Clvl()
		var row *Row
		for i := range pol.Rows {
			r := &pol.Rows[i]
			a, ok := e.D.Areas[r.Area]
			if !ok || cl < r.Enter || cl >= r.Leave {
				continue
			}
			if r.Area == 8 && s.Den && !e.AllowDenRefarm {
				continue
			}
			mf := r.MinF
			if mf == 0 {
				mf = e.P.Depleted
			}
			if e.Remaining(&s, a) <= mf {
				continue
			}
			row = r
			break
		}
		if row == nil {
			if pol.Restart && !justRestarted {
				st := Step{Kind: "restart", T0: s.T, XP0: s.XP, Clvl0: cl}
				e.Restart(&s)
				st.T1, st.XP1, st.Clvl1, st.Area = s.T, s.XP, cl, 1
				res.Steps = append(res.Steps, st)
				justRestarted = true
				continue
			}
			// nothing open even in a fresh game: keep clearing the best open row at the floor rate
			for i := range pol.Rows {
				if pol.Rows[i].Enter <= cl && (row == nil || pol.Rows[i].Leave > row.Leave) {
					row = &pol.Rows[i]
				}
			}
			if row == nil {
				s.T += 1
				continue
			}
		}
		justRestarted = false
		a := e.D.Areas[row.Area]
		if f := e.Floor(a.ID); cl < f {
			k := fmt.Sprintf("%s at clvl %d (floor %d)", a.Name, cl, f)
			if !seen[k] {
				seen[k] = true
				res.Violations = append(res.Violations, k)
			}
		}
		st := Step{Kind: "farm", Area: a.ID, T0: s.T, XP0: s.XP, Clvl0: cl}
		k0 := s.TotalK
		st.Travel = e.Travel(&s, a.ID)
		mf := row.MinF
		if mf == 0 {
			mf = e.P.Depleted
		}
		e.Farm(&s, a, Exit{MinF: mf, Until: row.Leave, OnLevel: true, TMax: H}, nil)
		st.T1, st.XP1, st.Clvl1, st.Kills = s.T, s.XP, s.Clvl(), s.TotalK-k0
		st.Exit = fmt.Sprintf("f<=%.2f or clvl %d", mf, row.Leave)
		res.Steps = append(res.Steps, st)
	}
	res.End = s
	return res
}

// XPAt interpolates the XP the steps reached at minute t.
func XPAt(steps []Step, t float64) float64 {
	xp := 0.0
	for _, st := range steps {
		if t < st.T0 {
			return xp
		}
		if t <= st.T1 && st.T1 > st.T0 {
			return st.XP0 + (st.XP1-st.XP0)*(t-st.T0)/(st.T1-st.T0)
		}
		xp = st.XP1
	}
	return xp
}
