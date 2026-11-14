package levelsim

import (
	"fmt"
	"sort"
	"strings"
)

// Band is one stretch of a strategy in one area.
type Band struct {
	Area       int     `json:"area"`
	Name       string  `json:"name"`
	EnterClvl  int     `json:"enter_clvl"`
	LeaveClvl  int     `json:"leave_clvl"`
	From       float64 `json:"from_min"`
	To         float64 `json:"to_min"`
	Kills      float64 `json:"kills"`
	XPGained   float64 `json:"xp"`
	Restarts   int     `json:"restarts_before"`
	ExitPolicy string  `json:"exit"`
}

// Bands compresses a step list into area stretches (consecutive slices in one area merge; a restart counts).
func (e *Env) Bands(steps []Step) []Band {
	var out []Band
	restarts := 0
	for _, s := range steps {
		if s.Kind == "restart" {
			restarts++
			continue
		}
		if n := len(out); n > 0 && out[n-1].Area == s.Area {
			b := &out[n-1]
			b.LeaveClvl, b.To = s.Clvl1, s.T1
			b.Kills += s.Kills
			b.XPGained += s.XP1 - s.XP0
			b.Restarts += restarts
			restarts = 0
			continue
		}
		out = append(out, Band{Area: s.Area, Name: e.D.Areas[s.Area].Name, EnterClvl: s.Clvl0, LeaveClvl: s.Clvl1,
			From: s.T0, To: s.T1, Kills: s.Kills, XPGained: s.XP1 - s.XP0, Restarts: restarts, ExitPolicy: s.Exit})
		restarts = 0
	}
	return out
}

// FormatBands renders the stretches as a table.
func (e *Env) FormatBands(steps []Step) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %-22s %-9s %-11s %-7s %-8s %s\n", "area", "clvl", "minutes", "kills", "xp", "restarts before")
	for _, x := range e.Bands(steps) {
		fmt.Fprintf(&b, "  %-22s %2d -> %-3d %5.1f-%-5.1f %-7.0f %-8.0f %d\n", x.Name, x.EnterClvl, x.LeaveClvl, x.From, x.To, x.Kills, x.XPGained, x.Restarts)
	}
	return b.String()
}

// Trajectory samples clvl and xp every `every` minutes up to H.
func Trajectory(steps []Step, H, every float64) [][3]float64 {
	var out [][3]float64
	for t := every; t <= H+1e-9; t += every {
		xp := XPAt(steps, t)
		out = append(out, [3]float64{t, xp, float64(LevelFor(xp))})
	}
	return out
}

// FormatTrajectory renders "clvl by minute".
func FormatTrajectory(steps []Step, H, every float64) string {
	var parts []string
	for _, p := range Trajectory(steps, H, every) {
		parts = append(parts, fmt.Sprintf("%g:%d", p[0], int(p[2])))
	}
	return strings.Join(parts, " ")
}

// Names renders an area order as names.
func (e *Env) Names(sig string) string {
	var out []string
	for _, f := range strings.Split(strings.TrimSuffix(sig, ">"), ">") {
		var id int
		fmt.Sscanf(f, "%d", &id)
		if a, ok := e.D.Areas[id]; ok {
			out = append(out, a.Name)
		}
	}
	return strings.Join(out, " > ")
}

// Distinct returns the sorted set of areas a step list farms.
func Distinct(steps []Step) []int {
	m := map[int]bool{}
	for _, s := range steps {
		if s.Kind == "farm" {
			m[s.Area] = true
		}
	}
	var out []int
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
