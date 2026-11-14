package levelsim

import (
	"fmt"
	"strings"
)

// BestRow is one optimiser answer in the summary file.
type BestRow struct {
	H          float64      `json:"horizon_min"`
	XP         float64      `json:"xp"`
	Clvl       int          `json:"clvl"`
	AreaOrder  string       `json:"area_order"`
	Bands      []Band       `json:"bands"`
	Trajectory [][3]float64 `json:"trajectory_min_xp_clvl"`
	Restarts   int          `json:"restarts"`
}

// PlanRow is today's plan under the same model.
type PlanRow struct {
	H          float64      `json:"horizon_min"`
	XP         float64      `json:"xp"`
	Clvl       int          `json:"clvl"`
	Violations []string     `json:"violations_of_mode_floors"`
	Bands      []Band       `json:"bands"`
	Trajectory [][3]float64 `json:"trajectory_min_xp_clvl"`
}

// ModeSummary is everything for one danger mode.
type ModeSummary struct {
	Mode   string             `json:"mode"`
	Plan   []PlanRow          `json:"plan_today"`
	Best   []BestRow          `json:"best"`
	Ranked []BestRow          `json:"ranked_at_longest_horizon"`
	Gain   map[string]float64 `json:"gain_vs_plan"`
	Reach  map[string]float64 `json:"minutes_to_clvl"`
}

// Summary is the machine-readable result file (tools/levelsim/summary/best-strategies.json).
type Summary struct {
	Schema string        `json:"schema"`
	Note   string        `json:"note"`
	Params Params        `json:"params"`
	Modes  []ModeSummary `json:"modes"`
	Sens   []SensRow     `json:"sensitivity,omitempty"`
}

func restarts(steps []Step) int {
	n := 0
	for _, s := range steps {
		if s.Kind == "restart" {
			n++
		}
	}
	return n
}

// Summarise runs the plan and the optimiser for one mode.
func Summarise(d *Data, p Params, mode Mode, horizons []float64, width int, reachH float64, reachWidth int, denRefarm bool) ModeSummary {
	e := NewEnv(d, p, mode)
	e.AllowDenRefarm = denRefarm
	ms := ModeSummary{Mode: string(mode), Gain: map[string]float64{}, Reach: map[string]float64{}}
	for _, h := range horizons {
		pl := e.RunPolicy(PlanToday(d), h)
		ms.Plan = append(ms.Plan, PlanRow{H: h, XP: pl.End.XP, Clvl: pl.Clvl(), Violations: pl.Violations, Bands: e.Bands(pl.Steps), Trajectory: Trajectory(pl.Steps, h, 10)})
		b := e.OptimiseSeeded(h, width, PlanToday(d))
		ms.Best = append(ms.Best, BestRow{H: h, XP: b.XP, Clvl: b.Clvl, AreaOrder: e.Names(b.Signature()), Bands: e.Bands(b.Steps),
			Trajectory: Trajectory(b.Steps, h, 10), Restarts: restarts(b.Steps)})
		if pl.End.XP > 0 {
			ms.Gain[hkey(h)] = b.XP/pl.End.XP - 1
		}
		if h == horizons[len(horizons)-1] {
			for _, t := range b.Top {
				ms.Ranked = append(ms.Ranked, BestRow{H: h, XP: t.XP, Clvl: t.Clvl, AreaOrder: e.Names(t.Signature()), Bands: e.Bands(t.Steps),
					Trajectory: Trajectory(t.Steps, h, 10), Restarts: restarts(t.Steps)})
			}
		}
	}
	if reachH > 0 {
		rb := e.Optimise(reachH, reachWidth)
		for _, l := range ReachLevels {
			if m, ok := rb.Reach[l]; ok {
				ms.Reach[fmt.Sprintf("clvl%d", l)] = m
			}
		}
	}
	return ms
}

// FormatMode prints a ModeSummary as tables.
func FormatMode(e *Env, ms ModeSummary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== danger mode %s ===\n", ms.Mode)
	for i, pr := range ms.Plan {
		br := ms.Best[i]
		fmt.Fprintf(&b, "\nhorizon %g min: plan today %7.0f xp (clvl %d)   optimum %7.0f xp (clvl %d)   gain %+.0f%%\n", pr.H, pr.XP, pr.Clvl, br.XP, br.Clvl, 100*ms.Gain[hkey(pr.H)])
		if len(pr.Violations) > 0 {
			fmt.Fprintf(&b, "  plan entries below the mode's floors: %s\n", strings.Join(pr.Violations, "; "))
		}
		fmt.Fprintf(&b, "  best order: %s (%d restarts)\n", br.AreaOrder, br.Restarts)
		fmt.Fprintf(&b, "  best bands:\n")
		for _, x := range br.Bands {
			fmt.Fprintf(&b, "    %-22s clvl %2d -> %-3d min %5.1f-%-5.1f kills %-4.0f xp %-7.0f restarts before %d\n", x.Name, x.EnterClvl, x.LeaveClvl, x.From, x.To, x.Kills, x.XPGained, x.Restarts)
		}
		fmt.Fprintf(&b, "  best clvl by minute: %s\n", trajStr(br.Trajectory))
		fmt.Fprintf(&b, "  plan clvl by minute: %s\n", trajStr(pr.Trajectory))
	}
	if len(ms.Ranked) > 0 {
		fmt.Fprintf(&b, "\nranked candidates at %g min (distinct area orders):\n", ms.Ranked[0].H)
		for i, r := range ms.Ranked {
			fmt.Fprintf(&b, "  %d. %7.0f xp clvl %d  %s\n", i+1, r.XP, r.Clvl, r.AreaOrder)
		}
	}
	if len(ms.Reach) > 0 {
		fmt.Fprintf(&b, "\nminutes to reach (best path, same constraints):")
		for _, l := range ReachLevels {
			if m, ok := ms.Reach[fmt.Sprintf("clvl%d", l)]; ok {
				fmt.Fprintf(&b, " clvl %d: %.0f;", l, m)
			} else {
				fmt.Fprintf(&b, " clvl %d: not reached;", l)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func trajStr(tr [][3]float64) string {
	var parts []string
	for _, p := range tr {
		parts = append(parts, fmt.Sprintf("%g:%d", p[0], int(p[2])))
	}
	return strings.Join(parts, " ")
}
