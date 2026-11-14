package levelplan

import (
	"fmt"
	"math"

	"d2lod-bot/internal/campaign"
	"d2lod-bot/internal/telemetry"
)

// Rule is how the simulation picks the next farm area.
type Rule int

const (
	RuleBest  Rule = iota // Choose: best XP per minute, advance before re-clearing (the v3 rule)
	RuleFirst             // the first in-band fresh area in plan order, else re-clear the first in-band area (the AREAMON "plan rows" route)
)

// SimOpts tune Simulate. Zero values take the AREAMON model defaults (re-clear 0.5, 1 minute per area change).
type SimOpts struct {
	Minutes float64
	Class   string          // lowercased class, for the chapter level rows
	Spine   *campaign.Spine // quest slices take their recommended level from it; nil: quests are skipped
	Reclear float64
	Switch  float64
	Rule    Rule
}

// Point is the state at the end of one activity.
type Point struct {
	Minute float64
	XP     float64
	Clvl   int
	What   string // "farm <slice> <area>", "quest <chapter>", "reclear <slice> <area>"
}

// Result is a simulated trajectory.
type Result struct {
	Trace []Point
	Done  map[string]float64 // quest chapter -> minute it finished
}

// At is the XP and clvl at minute m (linear inside an activity).
func (r Result) At(m float64) (xp float64, clvl int) {
	prev := Point{Clvl: 1}
	for _, p := range r.Trace {
		if p.Minute >= m {
			if p.Minute == prev.Minute {
				return p.XP, p.Clvl
			}
			x := prev.XP + (p.XP-prev.XP)*(m-prev.Minute)/(p.Minute-prev.Minute)
			return x, LevelOf(x)
		}
		prev = p
	}
	return prev.XP, prev.Clvl
}

// LevelOf is the clvl of total experience xp (Experience.txt, D, internal/telemetry).
func LevelOf(xp float64) int {
	l := 1
	for {
		need, ok := telemetry.DefaultExpForLevel(l + 1)
		if !ok || xp < float64(need) {
			return l
		}
		l++
	}
}

// Simulate plays the plan minute by minute on the XP table (I: the AREAMON time model; no deaths, no town trips, a
// full kit always, the Rogue alive from the chapter that hands her out). Quest slices go in plan order at the
// chapter's recommended level (Guided default goAt) and clear the chapter's area once; optional chapters are skipped
// (their default is ask or skip). The run ends at Minutes or when the last quest slice is done.
func Simulate(c *ClassPlan, t Table, o SimOpts) (Result, error) {
	if o.Reclear == 0 {
		o.Reclear = 0.5
	}
	if o.Switch == 0 {
		o.Switch = 1
	}
	if o.Spine != nil && o.Class != c.Class {
		return Result{}, fmt.Errorf("levelplan: the %q plan simulated for class %q", c.Class, o.Class)
	}
	res := Result{Done: map[string]float64{}}
	var (
		now, xp float64
		last    int
		merc    bool
		cleared = map[int]bool{}
	)
	clvl := func() int { return LevelOf(xp) }
	// spend works through an area clear in steps of at most one minute so the clvl, and with it XP per kill, updates.
	spend := func(area int, factor float64, what string) {
		a, ok := t[area]
		if area != last {
			now += o.Switch
			last = area
		}
		left := 1.0
		if ok {
			left = a.ClearMin
		}
		for left > 0 && now < o.Minutes {
			dt := math.Min(math.Min(1, left), o.Minutes-now)
			if ok {
				xp += a.PerMin(clvl()) * factor * dt
			}
			now += dt
			left -= dt
		}
		cleared[area] = true
		res.Trace = append(res.Trace, Point{Minute: now, XP: xp, Clvl: clvl(), What: what})
	}
	quests := c.questsIn(o.Spine)
	for now < o.Minutes {
		q, ok, err := nextQuest(quests, res.Done, o, clvl(), merc)
		if err != nil {
			return res, err
		}
		if ok {
			spend(q.AreaID, freshFactor(cleared[q.AreaID], o.Reclear), "quest "+q.ID)
			res.Done[q.ID] = now
			merc = merc || mercFrom(o.Spine, q.ID)
			if q.ID == quests[len(quests)-1].ID {
				break
			}
			continue
		}
		h := Hero{Level: clvl(), Merc: merc, FullKit: true}
		fresh := func(a int) bool { return !cleared[a] }
		var ch Choice
		if o.Rule == RuleFirst {
			ch, ok = c.chooseFirst(h, fresh)
		} else {
			ch, ok = c.Choose(h, fresh, t, o.Reclear)
		}
		if !ok {
			return res, fmt.Errorf("levelplan: no farm slice open at clvl %d (minute %.0f)", h.Level, now)
		}
		what := "farm "
		if !ch.Fresh {
			what = "reclear "
		}
		spend(ch.Area, freshFactor(!ch.Fresh, o.Reclear), fmt.Sprintf("%s%s %d", what, ch.Slice, ch.Area))
	}
	return res, nil
}

func freshFactor(clearedBefore bool, reclear float64) float64 {
	if clearedBefore {
		return reclear
	}
	return 1
}

// questsIn are the non-optional chapters of the quest slices, in plan order.
func (c *ClassPlan) questsIn(sp *campaign.Spine) []*campaign.Chapter {
	if sp == nil {
		return nil
	}
	var out []*campaign.Chapter
	for _, s := range c.Slices {
		if ch := sp.Chapter(s.Chapter); s.Kind == KindQuest && ch != nil && !ch.Optional {
			out = append(out, ch)
		}
	}
	return out
}

// nextQuest is the first undone quest; it goes when the hero is at its recommended level.
func nextQuest(qs []*campaign.Chapter, done map[string]float64, o SimOpts, clvl int, merc bool) (*campaign.Chapter, bool, error) {
	for _, q := range qs {
		if _, ok := done[q.ID]; ok {
			continue
		}
		l, err := q.LevelsFor(o.Class)
		if err != nil {
			return nil, false, err
		}
		_, rec := l.For(merc)
		return q, clvl >= rec, nil
	}
	return nil, false, nil
}

// mercFrom: a chapter whose done rule is "merc" and that requires id hands out the merc once id is done.
func mercFrom(sp *campaign.Spine, id string) bool {
	for _, ch := range sp.Chapters {
		if !ch.Done.Merc {
			continue
		}
		for _, r := range ch.Requires {
			if r == id {
				return true
			}
		}
	}
	return false
}

func (c *ClassPlan) chooseFirst(h Hero, fresh func(int) bool) (Choice, bool) {
	var first *Choice
	for _, s := range c.Farms() {
		if ok, _ := Gate(s, h); !ok || h.Level > s.Exit {
			continue
		}
		for _, a := range s.Areas {
			if fresh(a) {
				return Choice{Slice: s.ID, Area: a, Fresh: true}, true
			}
			if first == nil {
				first = &Choice{Slice: s.ID, Area: a}
			}
		}
	}
	if first == nil {
		return Choice{}, false
	}
	return *first, true
}
