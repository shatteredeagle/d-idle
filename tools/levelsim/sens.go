package levelsim

import (
	"fmt"
	"sort"
	"strings"
)

// Scenario is one sensitivity case.
type Scenario struct {
	Name  string
	Apply func(p *Params, e *Env)
}

// Scenarios are the swept assumptions (docs/levelsim.md section 6).
func Scenarios() []Scenario {
	return []Scenario{
		{"baseline", func(*Params, *Env) {}},
		{"kill rate -20%", func(p *Params, _ *Env) { p.RateScale = 0.8 }},
		{"kill rate +20%", func(p *Params, _ *Env) { p.RateScale = 1.2 }},
		{"walking and chores -50% (walk, hop, town tax)", func(p *Params, _ *Env) { p.WalkScale, p.HopMin, p.TownTax = 0.5, p.HopMin/2, p.TownTax/2 }},
		{"walking and chores +100%", func(p *Params, _ *Env) { p.WalkScale, p.HopMin, p.TownTax = 2, p.HopMin*2, p.TownTax*2 }},
		{"restart 30 s", func(p *Params, _ *Env) { p.RestartMin = 0.5 }},
		{"restart 75 s", func(p *Params, _ *Env) { p.RestartMin = 1.25 }},
		{"restart 150 s", func(p *Params, _ *Env) { p.RestartMin = 2.5 }},
		{"XP per kill -15% (fewer uniques)", func(p *Params, _ *Env) { p.XPScale = 0.85 }},
		{"XP per kill +15%", func(p *Params, _ *Env) { p.XPScale = 1.15 }},
		{"supply decays (decay 1.5, floor 0.3, yield 0.85)", func(p *Params, _ *Env) { p.Decay, p.Floor, p.Yield = 1.5, 0.3, 0.85 }},
		{"Den may be farmed again", func(_ *Params, e *Env) { e.AllowDenRefarm = true }},
	}
}

// SensRow is the outcome of one scenario.
type SensRow struct {
	Name string             `json:"name"`
	Opt  map[string]float64 `json:"opt_xp"`
	Plan map[string]float64 `json:"plan_xp"`
	Gain map[string]float64 `json:"gain_vs_plan"`
	Sig  map[string]string  `json:"area_order"`
}

func hkey(h float64) string { return fmt.Sprintf("%g", h) }

// Sensitivity runs every scenario for every horizon in the given danger mode.
func Sensitivity(d *Data, base Params, mode Mode, horizons []float64, width int) []SensRow {
	var rows []SensRow
	for _, sc := range Scenarios() {
		p := base
		flags := &Env{}
		sc.Apply(&p, flags)
		e := NewEnv(d, p, mode)
		e.AllowDenRefarm = flags.AllowDenRefarm
		r := SensRow{Name: sc.Name, Opt: map[string]float64{}, Plan: map[string]float64{}, Gain: map[string]float64{}, Sig: map[string]string{}}
		for _, h := range horizons {
			pl := e.RunPolicy(PlanToday(d), h)
			b := e.OptimiseSeeded(h, width, PlanToday(d))
			k := hkey(h)
			r.Opt[k], r.Plan[k] = b.XP, pl.End.XP
			if pl.End.XP > 0 {
				r.Gain[k] = b.XP/pl.End.XP - 1
			}
			r.Sig[k] = e.Names(b.Signature())
		}
		rows = append(rows, r)
	}
	return rows
}

// FormatSens renders the sensitivity table.
func FormatSens(rows []SensRow, horizons []float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-50s", "scenario")
	for _, h := range horizons {
		fmt.Fprintf(&b, " | opt@%-4g plan@%-4g gain", h, h)
	}
	b.WriteString("\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-50s", r.Name)
		for _, h := range horizons {
			k := hkey(h)
			fmt.Fprintf(&b, " | %-8.0f %-9.0f %+4.0f%%", r.Opt[k], r.Plan[k], 100*r.Gain[k])
		}
		b.WriteString("\n")
	}
	if len(rows) > 0 && len(horizons) > 0 {
		k := hkey(horizons[len(horizons)-1])
		type dv struct {
			name string
			d    float64
		}
		var ds []dv
		base := rows[0].Opt[k]
		for _, r := range rows[1:] {
			if base > 0 {
				ds = append(ds, dv{r.Name, r.Opt[k]/base - 1})
			}
		}
		sort.Slice(ds, func(i, j int) bool { return abs(ds[i].d) > abs(ds[j].d) })
		b.WriteString("largest moves of the optimum at the longest horizon:")
		for i := 0; i < len(ds) && i < 3; i++ {
			fmt.Fprintf(&b, " %s %+.0f%%;", ds[i].name, 100*ds[i].d)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
