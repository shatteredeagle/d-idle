package levelsim

import (
	"fmt"
	"math"
	"sort"
)

// Replay is the model's prediction for one logged run.
type Replay struct {
	RunID               int
	PredXP, PredKills   float64
	PredClvl            int
	ObsXP, ObsKills     float64
	ObsClvl             int
	XPErr, KillsErr     float64 // signed relative error at the end of the hour
	MinuteMAE           float64 // mean |pred - obs| xp over the logged minute rows, relative to the final observed xp (0 if no rows)
	Series              [][2]float64
	HasSeries           bool
	Skipped, SkipReason string
}

// overlap returns the length of [a0,a1] inside [b0,b1].
func overlap(a0, a1, b0, b1 float64) float64 {
	return math.Max(0, math.Min(a1, b1)-math.Max(a0, b0))
}

// ReplayRun feeds the logged area sequence of a run through the kill model.
func (e *Env) ReplayRun(r Run) Replay {
	rp := Replay{RunID: r.ID, ObsXP: r.Observed.XP, ObsKills: r.Observed.Kills, ObsClvl: r.Observed.Clvl}
	if len(r.Arrivals) == 0 {
		rp.Skipped, rp.SkipReason = "summary", "no area sequence retained"
		return rp
	}
	s, series := e.replay(r)
	rp.PredXP, rp.PredKills, rp.PredClvl, rp.Series = s.XP, s.TotalK, s.Clvl(), series
	rp.XPErr = relErr(rp.PredXP, rp.ObsXP)
	rp.KillsErr = relErr(rp.PredKills, rp.ObsKills)
	rp.HasSeries = len(r.Minutes) > 0
	if rp.HasSeries {
		var sum float64
		for _, m := range r.Minutes {
			sum += math.Abs(seriesAt(series, m[0]) - m[3])
		}
		rp.MinuteMAE = sum / float64(len(r.Minutes)) / r.Observed.XP
	}
	return rp
}

func relErr(pred, obs float64) float64 {
	if obs == 0 {
		return 0
	}
	return (pred - obs) / obs
}

// replay runs the logged sequence and samples XP every step.
func (e *Env) replay(r Run) (State, [][2]float64) {
	end := r.EndMin
	if end == 0 {
		end = 60
	}
	s := NewState()
	rs := append([]float64(nil), r.Restarts...)
	sort.Float64s(rs)
	ri := 0
	series := [][2]float64{{0, 0}}
	for i, a := range r.Arrivals {
		t0 := a[0]
		t1 := end
		if i+1 < len(r.Arrivals) {
			t1 = r.Arrivals[i+1][0]
		}
		if t0 >= end {
			break
		}
		if t1 > end {
			t1 = end
		}
		for ri < len(rs) && rs[ri] <= t0 {
			s.Kills = [40]float64{}
			ri++
		}
		id := int(a[1])
		ar, ok := e.D.Areas[id]
		if id == 1 || !ok || t1 <= t0 {
			series = append(series, [2]float64{t1, s.XP})
			continue
		}
		dur := t1 - t0
		for _, d := range r.Dead {
			dur -= overlap(t0, t1, d[0], d[1])
		}
		if dur < 0 {
			dur = 0
		}
		s.T = t0
		elapsed := 0.0
		e.Farm(&s, ar, Exit{MaxMin: dur, NoTax: true}, func(_, xp, _ float64) {
			elapsed += stepMin
			series = append(series, [2]float64{t0 + math.Min(elapsed, dur), xp})
		})
		series = append(series, [2]float64{t1, s.XP})
	}
	return s, series
}

func seriesAt(series [][2]float64, t float64) float64 {
	v := 0.0
	for _, p := range series {
		if p[0] > t {
			break
		}
		v = p[1]
	}
	return v
}

// KillsCheck is the only test for a run without an area sequence: observed kills times the lowest and
// highest XP per kill of the starter fields must put the XP inside the observed level band (I).
func (e *Env) KillsCheck(r Run) (lo, hi float64, ok bool) {
	lo = r.Observed.Kills * e.D.Areas[2].XK[2]
	hi = r.Observed.Kills * e.D.Areas[3].XK[2] * 1.1
	l := r.Observed.Clvl
	ok = hi >= XPForLevel(l) && lo < XPForLevel(l+1)
	return
}

// Fit is one point of the calibration grid.
type Fit struct {
	P    Params
	Loss float64
}

// Loss scores parameters on the runs that have an area sequence: the final XP and kill errors of every
// run plus the per-minute mean error where minute rows exist, plus a weak prior (the logged hour 4:
// a Cold Plains sweep ends at 163 of 198 units, the tail runs under half the fresh rate, so yield about
// 0.82 and floor about 0.4; decay 1 is the linear default). The data alone barely pins decay, floor and yield.
func Loss(d *Data, p Params, runs []Run) float64 {
	env := NewEnv(d, p, ModeStrict)
	var loss float64
	for _, r := range runs {
		rp := env.ReplayRun(r)
		if rp.Skipped != "" {
			continue
		}
		loss += rp.XPErr*rp.XPErr + 0.5*rp.KillsErr*rp.KillsErr
		if rp.HasSeries {
			loss += rp.MinuteMAE * rp.MinuteMAE
		}
	}
	sq := func(x float64) float64 { return x * x }
	return loss + priorWeight*(sq((p.Yield-0.82)/0.1)+sq((p.Floor-0.4)/0.2)+sq(p.Decay-1))
}

const priorWeight = 0.002

// FitSet is the grid search result: the best point and the spread of the near-optimal ones.
type FitSet struct {
	Best             Fit
	Near             int // grid points within 25 percent of the best loss
	Lo, Hi           Params
	GridPoints       int
	BestWithoutPrior Fit
}

// Calibrate searches the grid for Decay, Floor and Yield (RateScale stays at the base value: the areas'
// fresh rates are observed or derived, see docs/levelsim.md section 3).
func Calibrate(d *Data, base Params, runs []Run) FitSet {
	var all []Fit
	for _, g := range []float64{0.25, 0.5, 0.75, 1, 1.5, 2, 3} {
		for _, fl := range []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7} {
			for _, y := range []float64{0.7, 0.75, 0.8, 0.85, 0.9, 0.95, 1.0} {
				p := base
				p.Decay, p.Floor, p.Yield = g, fl, y
				all = append(all, Fit{p, Loss(d, p, runs)})
			}
		}
	}
	fs := FitSet{GridPoints: len(all)}
	fs.Best = all[0]
	for _, f := range all {
		if f.Loss < fs.Best.Loss {
			fs.Best = f
		}
	}
	fs.Lo, fs.Hi = fs.Best.P, fs.Best.P
	for _, f := range all {
		if f.Loss > fs.Best.Loss*1.25 {
			continue
		}
		fs.Near++
		fs.Lo.Decay, fs.Hi.Decay = math.Min(fs.Lo.Decay, f.P.Decay), math.Max(fs.Hi.Decay, f.P.Decay)
		fs.Lo.Floor, fs.Hi.Floor = math.Min(fs.Lo.Floor, f.P.Floor), math.Max(fs.Hi.Floor, f.P.Floor)
		fs.Lo.Yield, fs.Hi.Yield = math.Min(fs.Lo.Yield, f.P.Yield), math.Max(fs.Hi.Yield, f.P.Yield)
	}
	return fs
}

// CalibReport renders the replay table of all runs under parameters p.
func CalibReport(d *Data, p Params) string {
	env := NewEnv(d, p, ModeStrict)
	out := fmt.Sprintf("params: rate x%.2f, decay g %.2f, floor %.2f, yield %.2f\n", p.RateScale, p.Decay, p.Floor, p.Yield)
	out += fmt.Sprintf("%-4s %-9s %-9s %-8s %-7s %-7s %-6s %-6s %s\n", "run", "obs xp", "pred xp", "xp err", "obs k", "pred k", "k err", "clvl", "per-minute MAE")
	for _, r := range d.Runs {
		rp := env.ReplayRun(r)
		if rp.Skipped != "" {
			lo, hi, ok := env.KillsCheck(r)
			out += fmt.Sprintf("%-4d no area sequence (%s): kills %.0f x xp/kill = %.0f..%.0f, observed clvl %d, band check %v\n",
				r.ID, rp.SkipReason, r.Observed.Kills, lo, hi, r.Observed.Clvl, ok)
			continue
		}
		mae := "-"
		if rp.HasSeries {
			mae = fmt.Sprintf("%.1f%%", 100*rp.MinuteMAE)
		}
		xp := "-"
		if rp.ObsXP > 0 {
			xp = fmt.Sprintf("%.0f", rp.ObsXP)
		}
		out += fmt.Sprintf("%-4d %-9s %-9.0f %-8s %-7.0f %-7.0f %-6s %d/%d  %s\n", r.ID, xp, rp.PredXP,
			fmt.Sprintf("%+.1f%%", 100*rp.XPErr), rp.ObsKills, rp.PredKills, fmt.Sprintf("%+.1f%%", 100*rp.KillsErr), rp.PredClvl, rp.ObsClvl, mae)
	}
	return out
}
