package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// skTable is the expected damage per hit in points for the SK drill (docs/probes/drills/2026-11-03-SK-design.txt,
// computed by hand from skills.txt, S: not checked against the game). Charged Bolt rows are per bolt.
var skTable = map[string]map[int][2]float64{
	"fire bolt":    {1: {3, 6}, 4: {7.5, 10.5}, 8: {13.5, 16.5}, 13: {23.5, 31.5}, 20: {45.5, 60.5}},
	"charged bolt": {1: {2, 4}, 4: {3.5, 5.5}, 8: {5.5, 7.5}, 13: {8, 10}, 20: {13.5, 15.5}},
}

const (
	skTolerance = 0.10 // PASS: mean within 10 percent of the table mean (U, proposed in the design file)
	skSlack     = 0.5  // each hit inside [min-0.5, max+0.5] (life truncation)
)

// skOpts are the flags of `combatlive sk`.
type skOpts struct {
	Skill   string        // skill name, for the table and the report
	Key     string        // hotkey the skill is bound to (bound beforehand: this command never opens the skill popup)
	Class   int           // monster class id to hit; 0 = nearest alive monster
	Level   int           // skill level of the character, selects the table row
	Hits    int           // stop after this many measured hits
	Casts   int           // hard cap on casts
	Wait    time.Duration // time between cast and the second life read
	Timeout time.Duration
}

// SKResult is the one JSON line printed.
type SKResult struct {
	Skill    string    `json:"skill"`
	Level    int       `json:"level"`
	Hits     int       `json:"hits"`
	Casts    int       `json:"casts"`
	Kills    int       `json:"kills"`  // casts that killed the target (not measured)
	Misses   int       `json:"misses"` // casts with no life change
	Mean     float64   `json:"mean"`   // points
	Min      float64   `json:"min"`    // points
	Max      float64   `json:"max"`    // points
	ExpMin   float64   `json:"expMin"` // table, 0 when the skill/level is not in the table
	ExpMax   float64   `json:"expMax"`
	ExpMean  float64   `json:"expMean"`
	MeanDev  float64   `json:"meanDev"`  // (mean-expMean)/expMean
	OutRange int       `json:"outRange"` // hits outside [expMin-0.5, expMax+0.5]
	Pass     *bool     `json:"pass,omitempty"`
	Samples  []float64 `json:"samples"`
	Note     string    `json:"note,omitempty"`
}

// skRig is what the sampler needs: reads only, plus one input call. Nothing here writes game memory.
type skRig struct {
	R     d2.StateReader
	In    input.Controller
	Sleep func(time.Duration)
	Now   func() time.Time
}

// life returns the unit's current life in points (stat 6 is 24.8 fixed point; U for monsters, see d2.MonsterInfo).
func (g skRig) life(u d2.Unit) (float64, error) {
	st, err := g.R.UnitStats(u.Addr)
	if err != nil {
		return 0, err
	}
	return float64(st.Get(d2.StatLife)) / 256, nil
}

// target picks the nearest alive monster of the class (any class when class is 0) that has life.
func (g skRig) target(class int) (d2.Unit, bool, error) {
	pl, err := g.R.Player()
	if err != nil {
		return d2.Unit{}, false, err
	}
	us, err := g.R.Units(d2.TypeMonster)
	if err != nil {
		return d2.Unit{}, false, err
	}
	var best d2.Unit
	bestD, found := math.MaxFloat64, false
	for _, u := range us {
		if !u.Alive() || (class != 0 && u.Class != class) {
			continue
		}
		if l, err := g.life(u); err != nil || l <= 0 {
			continue
		}
		if d := math.Hypot(float64(u.X-pl.X), float64(u.Y-pl.Y)); d < bestD {
			best, bestD, found = u, d, true
		}
	}
	return best, found, nil
}

// sample casts at the target until o.Hits measured hits, o.Casts casts or the deadline.
func (g skRig) sample(ctx context.Context, o skOpts) (*SKResult, error) {
	res := &SKResult{Skill: o.Skill, Level: o.Level, Samples: []float64{}}
	if o.Key == "" {
		return nil, errors.New("sk: --key is required (the hotkey the skill is bound to)")
	}
	end := g.Now().Add(o.Timeout)
	var cur d2.Unit
	have := false
	for len(res.Samples) < o.Hits && res.Casts < o.Casts && g.Now().Before(end) {
		if err := ctx.Err(); err != nil {
			res.Note = err.Error()
			break
		}
		if !have {
			u, ok, err := g.target(o.Class)
			if err != nil {
				return res, err
			}
			if !ok {
				res.Note = "no live target"
				break
			}
			cur, have = u, true
		}
		before, err := g.life(cur)
		if err != nil {
			return res, err
		}
		if before <= 0 {
			have = false
			continue
		}
		us, err := g.R.Units(d2.TypeMonster) // fresh position: monsters walk
		if err != nil {
			return res, err
		}
		for _, u := range us {
			if u.ID == cur.ID {
				cur = u
			}
		}
		if err := g.In.CastAt(o.Key, input.Point{X: cur.X, Y: cur.Y}); err != nil {
			return res, err
		}
		res.Casts++
		g.Sleep(o.Wait)
		after, err := g.life(cur)
		if err != nil {
			return res, err
		}
		switch d := before - after; {
		case after <= 0:
			res.Kills++ // overkill: the real hit is unknown, discard
			have = false
		case d <= 0:
			res.Misses++
		default:
			res.Samples = append(res.Samples, d)
		}
	}
	res.finish()
	return res, nil
}

// finish fills the statistics and the verdict against the table.
func (r *SKResult) finish() {
	r.Hits = len(r.Samples)
	if r.Hits == 0 {
		return
	}
	var sum float64
	r.Min, r.Max = r.Samples[0], r.Samples[0]
	for _, s := range r.Samples {
		sum += s
		r.Min, r.Max = math.Min(r.Min, s), math.Max(r.Max, s)
	}
	r.Mean = sum / float64(r.Hits)
	row, ok := skTable[strings.ToLower(r.Skill)][r.Level]
	if !ok {
		r.Note = strings.TrimSpace(r.Note + " no table row for this skill/level: no verdict")
		return
	}
	r.ExpMin, r.ExpMax, r.ExpMean = row[0], row[1], (row[0]+row[1])/2
	r.MeanDev = (r.Mean - r.ExpMean) / r.ExpMean
	for _, s := range r.Samples {
		if s < r.ExpMin-skSlack || s > r.ExpMax+skSlack {
			r.OutRange++
		}
	}
	pass := math.Abs(r.MeanDev) <= skTolerance && r.OutRange == 0 && r.Hits >= 40
	r.Pass = &pass
	if r.Hits < 40 {
		r.Note = strings.TrimSpace(r.Note + " fewer than 40 hits")
	}
}

// parseSK reads the subcommand flags; classOf resolves a monster name to its class id (data tables).
func parseSK(args []string, classOf func(string) (int, bool)) (skOpts, error) {
	var o skOpts
	var mon string
	fs := flag.NewFlagSet("sk", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.Skill, "skill", "", "skill name, e.g. \"Fire Bolt\"")
	fs.StringVar(&o.Key, "key", "", "hotkey the skill is bound to, e.g. f1")
	fs.StringVar(&mon, "monster", "", "monster name (monstats id), default nearest")
	fs.IntVar(&o.Level, "level", 0, "skill level of the character (selects the expected row)")
	fs.IntVar(&o.Hits, "hits", 40, "measured hits wanted")
	fs.IntVar(&o.Casts, "casts", 200, "cast cap")
	fs.DurationVar(&o.Wait, "wait", 900*time.Millisecond, "wait after the cast before the second life read")
	fs.DurationVar(&o.Timeout, "timeout", 5*time.Minute, "overall limit")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if o.Skill == "" || o.Key == "" {
		return o, errors.New("sk: --skill and --key are required")
	}
	if mon != "" {
		c, ok := classOf(mon)
		if !ok {
			return o, fmt.Errorf("sk: unknown monster %q", mon)
		}
		o.Class = c
	}
	return o, nil
}

func skCmd(ctx context.Context, g skRig, o skOpts, w io.Writer) error {
	res, err := g.sample(ctx, o)
	if res != nil {
		b, _ := json.Marshal(res)
		fmt.Fprintln(w, string(b))
	}
	return err
}
