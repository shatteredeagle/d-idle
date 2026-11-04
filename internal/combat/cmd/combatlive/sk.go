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

// The client-side life stat of a monster is a 0..128 percent scale (VERIFIED LIVE 2026-11-03, i2, see
// docs/probes/sksmoke/2026-11-03-SK-live.txt), so life deltas are not damage points. No exact (server-side) monster life
// read exists in the repo (UNVERIFIED that one can be built, docs/sksmoke-tools.md). This command therefore uses the
// "bolts to kill" fallback: cast at one monster until it dies, count the casts that hurt it, and bound the per-hit
// damage with the monster's max life from monstats. That is an ESTIMATE, not a measurement of single hits.

// skTable is the expected damage per hit in points for the SK drill (docs/probes/sksmoke/2026-11-03-SK-design.txt,
// computed by hand from skills.txt, S: not checked against the game). Charged Bolt rows are per bolt.
var skTable = map[string]map[int][2]float64{
	"fire bolt":    {1: {3, 6}, 4: {7.5, 10.5}, 8: {13.5, 16.5}, 13: {23.5, 31.5}, 20: {45.5, 60.5}},
	"charged bolt": {1: {2, 4}, 4: {3.5, 5.5}, 8: {5.5, 7.5}, 13: {8, 10}, 20: {13.5, 15.5}},
}

const skMinKills = 10 // kills needed for a verdict

// skOpts are the flags of `combatlive sk`.
type skOpts struct {
	Skill   string        // skill name, for the table and the report
	Key     string        // hotkey the skill is bound to (bound beforehand: this command never opens the skill popup)
	Class   int           // monster class id to hit; 0 = nearest alive monster
	Level   int           // skill level of the character, selects the table row
	Kills   int           // stop after this many usable kills
	Casts   int           // hard cap on casts
	Wait    time.Duration // time between cast and the second life read
	Timeout time.Duration
}

// SKKill is one monster killed by repeated casts.
type SKKill struct {
	Class   int     `json:"class"`
	ID      int     `json:"id"`
	Hits    int     `json:"hits"` // casts that lowered its life, the killing cast included
	MaxLife float64 `json:"maxLife"`
	Lo      float64 `json:"lo"` // maxLife/Hits: lower bound of the mean damage per hit (the last hit overkills)
	Hi      float64 `json:"hi"` // maxLife/(Hits-1): upper bound of the mean of the non-killing hits (Hits >= 2)
}

// SKResult is the one JSON line printed.
type SKResult struct {
	Skill   string   `json:"skill"`
	Level   int      `json:"level"`
	Casts   int      `json:"casts"`
	Misses  int      `json:"misses"`  // casts with no life change
	OneShot int      `json:"oneShot"` // kills in one hit: only prove damage >= max life, not used
	Used    []SKKill `json:"kills"`
	MeanLo  float64  `json:"meanLo"` // mean of Lo over the used kills (points)
	MeanHi  float64  `json:"meanHi"` // mean of Hi over the used kills (points)
	ExpMin  float64  `json:"expMin"` // table, 0 when the skill/level is not in the table
	ExpMax  float64  `json:"expMax"`
	ExpMean float64  `json:"expMean"`
	Pass    *bool    `json:"pass,omitempty"` // table mean inside [MeanLo, MeanHi]; an estimate, see Note
	Note    string   `json:"note,omitempty"`
	Trace   []string `json:"trace,omitempty"` // first raw readings: class id before after
}

// skRig is what the sampler needs: reads only, plus one input call. Nothing here writes game memory.
type skRig struct {
	R     d2.StateReader
	In    input.Controller
	Sleep func(time.Duration)
	Now   func() time.Time
	// MaxLife gives the monstats max life range of a class for a difficulty; ok false when unknown.
	MaxLife func(class, diff int) (lo, hi int, ok bool)
	Log     io.Writer // one progress line per cast; nil for none
}

func (g skRig) logf(format string, a ...any) {
	if g.Log != nil {
		fmt.Fprintf(g.Log, format+"\n", a...)
	}
}

// life returns the unit's stat 6 divided by 256. VERIFIED LIVE 2026-11-03 (i2): for client-side monsters this is a
// PERCENT scale (full health reads 128), so it only tells hurt / not hurt / dead, never points.
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

// sample casts at one monster at a time until it dies, until o.Kills usable kills, o.Casts casts or the deadline.
func (g skRig) sample(ctx context.Context, o skOpts) (*SKResult, error) {
	res := &SKResult{Skill: o.Skill, Level: o.Level, Used: []SKKill{}}
	if o.Key == "" {
		return nil, errors.New("sk: --key is required (the hotkey the skill is bound to)")
	}
	pl, err := g.R.Player()
	if err != nil {
		return nil, err
	}
	diff := int(pl.Difficulty)
	end := g.Now().Add(o.Timeout)
	var cur d2.Unit
	have, hits := false, 0
	for len(res.Used) < o.Kills && res.Casts < o.Casts && g.Now().Before(end) {
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
			cur, have, hits = u, true, 0
			g.logf("target class %d id %d", u.Class, u.ID)
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
		if len(res.Trace) < 25 {
			res.Trace = append(res.Trace, fmt.Sprintf("c%d id%d %.2f %.2f", cur.Class, cur.ID, before, after))
		}
		outcome := "hit"
		switch {
		case after <= 0:
			outcome = "kill"
			hits++
			have = false
			g.recordKill(res, cur, hits, diff)
		case before-after <= 0:
			outcome = "miss"
			res.Misses++
		default:
			hits++
		}
		g.logf("cast %d class %d id %d life %.2f -> %.2f %s hits %d kills %d", res.Casts, cur.Class, cur.ID, before, after, outcome, hits, len(res.Used))
	}
	res.finish()
	return res, nil
}

// recordKill keeps a kill that has a known max life and at least two hits.
func (g skRig) recordKill(res *SKResult, u d2.Unit, hits, diff int) {
	if g.MaxLife == nil {
		res.Note = strings.TrimSpace(res.Note + " no monster table: no max life")
		return
	}
	lo, hi, ok := g.MaxLife(u.Class, diff)
	if !ok || hi <= 0 {
		res.Note = strings.TrimSpace(res.Note + fmt.Sprintf(" class %d not in monstats", u.Class))
		return
	}
	if hits < 2 {
		res.OneShot++
		return
	}
	// The rolled max life is unknown: the midpoint of the table range stands in for it (U).
	ml := float64(lo+hi) / 2
	if lo != hi && !strings.Contains(res.Note, "max life is the midpoint") {
		res.Note = strings.TrimSpace(res.Note + " max life is the midpoint of the monstats range (U)")
	}
	res.Used = append(res.Used, SKKill{Class: u.Class, ID: int(u.ID), Hits: hits, MaxLife: ml, Lo: ml / float64(hits), Hi: ml / float64(hits-1)})
}

// finish fills the bounds and the verdict against the table.
func (r *SKResult) finish() {
	if len(r.Used) == 0 {
		return
	}
	for _, k := range r.Used {
		r.MeanLo += k.Lo
		r.MeanHi += k.Hi
	}
	r.MeanLo /= float64(len(r.Used))
	r.MeanHi /= float64(len(r.Used))
	row, ok := skTable[strings.ToLower(r.Skill)][r.Level]
	if !ok {
		r.Note = strings.TrimSpace(r.Note + " no table row for this skill/level: no verdict")
		return
	}
	r.ExpMin, r.ExpMax, r.ExpMean = row[0], row[1], (row[0]+row[1])/2
	pass := r.ExpMean >= r.MeanLo && r.ExpMean <= r.MeanHi && len(r.Used) >= skMinKills
	r.Pass = &pass
	if len(r.Used) < skMinKills {
		r.Note = strings.TrimSpace(r.Note + " fewer than 10 usable kills")
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
	fs.IntVar(&o.Kills, "kills", 10, "usable kills wanted")
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
