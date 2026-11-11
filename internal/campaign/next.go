package campaign

import (
	"fmt"
	"time"
)

// Progress is what the hero has done: the inputs of the chapter done rules and the bands.
type Progress struct {
	Hero
	Quests    map[string]bool // task-layer quest ids that are done
	Waypoints map[int]bool    // area ids whose waypoint is known
	MercOwned bool            // a merc is employed, alive or dead
}

// IsDone reports whether chapter c is done for p.
func (c *Chapter) IsDone(p Progress) bool {
	switch {
	case c.Done.Quest != "":
		return p.Quests[c.Done.Quest]
	case c.Done.Waypoint > 0:
		return p.Waypoints[c.Done.Waypoint]
	case c.Done.Merc:
		return p.MercOwned || p.Merc
	}
	return p.Level >= c.Done.Level
}

// Action is what the decision asks the bot to do.
type Action string

const (
	ActGo     Action = "go"     // run the chapter (its quest handler)
	ActAsk    Action = "ask"    // queue the floor question for the chapter; farm meanwhile (Farm is set)
	ActFarm   Action = "farm"   // level in Farm until Level, then the chapter's band changes
	ActDone   Action = "done"   // every chapter is done or skipped
	ActFree   Action = "free"   // Free mode: the caller runs its own planner
	ActOrders Action = "orders" // Orders mode: only the player's orders
)

// Decision is the next step of the journey.
type Decision struct {
	Action    Action    `json:"action"`
	Chapter   string    `json:"chapter,omitempty"`
	Title     string    `json:"title,omitempty"`
	Quest     string    `json:"quest,omitempty"`
	Readiness Readiness `json:"readiness"`
	Farm      *FarmZone `json:"farm,omitempty"`
	Level     int       `json:"level,omitempty"` // the level the farm step aims at
	Why       string    `json:"why"`
	Warn      string    `json:"warn,omitempty"` // going below the band (Go now, a yes at the floor)
}

// Opts are the timing knobs of the decision.
type Opts struct {
	Now time.Time
	// AskWait is the veto window: an unanswered floor question counts as yes after it (0 = never by default).
	AskWait time.Duration
	// FarmBudget is how long a level step for a Ready chapter runs before the hero offers to go at the floor (0 = never).
	FarmBudget time.Duration
	// Checks gives the readiness checks for a chapter's area (nil = none).
	Checks func(c *Chapter) []Check
	// Wall gives the safety floor of a chapter from the live knobs (the route and tower guards, the Blood Raven wait
	// level, the Andariel hold; 0 = none). A wall raises the band's floor and recommended level: bands never send the
	// hero into a walk the safety rules refuse.
	Wall func(c *Chapter) int
	// Zone names the area the bot's own level planner farms in ("" = use the spine's farm table).
	Zone func() string
}

// Next is the guided-journey decision for p in state st. Pure: it reads st and never changes it (Observe does).
func (s *Spine) Next(p Progress, st *HeroState, o Opts) (Decision, error) {
	switch st.Mode {
	case Free:
		return Decision{Action: ActFree, Why: "free mode: the hero plans on its own"}, nil
	case Orders:
		return Decision{Action: ActOrders, Why: "orders mode: waiting for your orders"}, nil
	}
	satisfied := func(id string) bool {
		c := s.Chapter(id)
		return c.IsDone(p) || st.Intent(c, p.Class) == Skip
	}
	if c := s.Chapter(st.GoNow); c != nil && !c.IsDone(p) {
		r, err := s.rate(c, p, o)
		if err != nil {
			return Decision{}, err
		}
		d := Decision{Action: ActGo, Chapter: c.ID, Title: c.Title, Quest: c.Quest, Readiness: r, Why: "your order: " + c.Title + " now"}
		if r.Band != Comfortable {
			d.Warn = fmt.Sprintf("going in %s (clvl %d, floor %d, recommended %d)", r.Band, p.Level, r.Floor, r.Recommended)
		}
		return d, nil
	}
	var ask, target *Decision
	targetNeed := 0
	for i := range s.Chapters {
		c := &s.Chapters[i]
		in := st.Intent(c, p.Class)
		if c.IsDone(p) || in == Skip || in == Hold {
			continue
		}
		ok := true
		for _, r := range c.Requires {
			ok = ok && satisfied(r)
		}
		if !ok {
			continue
		}
		r, err := s.rate(c, p, o)
		if err != nil {
			return Decision{}, err
		}
		d := Decision{Chapter: c.ID, Title: c.Title, Quest: c.Quest, Readiness: r}
		atFloor := r.Band == Ready
		goable := st.GoAt.enough(r.Band)
		offer := atFloor && !goable && o.FarmBudget > 0 && st.Farm != nil && st.Farm.Chapter == c.ID && o.Now.Sub(st.Farm.Since) >= o.FarmBudget
		if (goable && atFloor && in == Ask) || offer {
			switch a, answered := st.Answers[c.ID]; {
			case answered && a.Yes:
				d.Action, d.Why, d.Warn = ActGo, "you said go: "+c.Title, "going at the floor: "+joinReasons(r)
				return d, nil
			case answered && a.Level >= p.Level:
				goable = false // a no holds until the next level-up
			case !st.Asked[c.ID].IsZero() && o.AskWait > 0 && o.Now.Sub(st.Asked[c.ID]) >= o.AskWait:
				d.Action, d.Why = ActGo, fmt.Sprintf("no answer in %s: going to %s at the floor", o.AskWait, c.Title)
				d.Warn = "going at the floor: " + joinReasons(r)
				return d, nil
			default:
				if ask == nil {
					a := d
					a.Action, a.Why = ActAsk, fmt.Sprintf("Ready for %s at the floor (clvl %d, recommended %d): go now?", c.Title, p.Level, r.Recommended)
					ask = &a
				}
				goable = false
			}
		}
		if goable {
			d.Action, d.Why = ActGo, "next chapter: "+c.Title
			if c.Why != "" {
				d.Why += ". " + c.Why
			}
			return d, nil
		}
		need := r.Recommended
		if st.GoAt == GoFloor {
			need = r.Floor
		}
		need = max(need, p.Level+1)
		if target == nil || need < targetNeed {
			t := d
			target, targetNeed = &t, need
		}
	}
	if target == nil && ask == nil {
		return Decision{Action: ActDone, Why: "the " + s.Difficulty + " journey is done"}, nil
	}
	if target == nil {
		target, targetNeed = ask, p.Level+1
	}
	z := s.FarmFor(p, satisfied)
	if o.Zone != nil {
		if n := o.Zone(); n != "" {
			z = &FarmZone{Area: n}
		}
	}
	d := *target
	if ask != nil {
		d = *ask
	} else {
		d.Action = ActFarm
		d.Why = fmt.Sprintf("Farming %s for level %d to unlock %s", zoneName(z), targetNeed, target.Title)
		if p.Level >= target.Readiness.Recommended { // level is enough, a readiness check holds it
			d.Why = fmt.Sprintf("Farming %s until %s is ready (%s)", zoneName(z), target.Title, joinReasons(target.Readiness))
		}
	}
	d.Farm, d.Level = z, targetNeed
	return d, nil
}

// rate is Rate with the readiness checks and the safety wall of o.
func (s *Spine) rate(c *Chapter, p Progress, o Opts) (Readiness, error) {
	var checks []Check
	if o.Checks != nil {
		checks = o.Checks(c)
	}
	r, err := Rate(c, p.Hero, checks)
	if err != nil || o.Wall == nil {
		return r, err
	}
	return r.withWall(o.Wall(c), p.Level), nil
}

// FarmFor picks the farm zone for p: the lowest open zone with mlvl >= clvl-5 (efficient XP, docs/sme/pace-and-formulas.md
// section 1), else the highest open one. open(id) says whether a chapter the zone requires is done.
func (s *Spine) FarmFor(p Progress, open func(id string) bool) *FarmZone {
	var best, top *FarmZone
	for i := range s.Farm {
		z := &s.Farm[i]
		if z.Requires != "" && !open(z.Requires) {
			continue
		}
		if top == nil || z.MLvl > top.MLvl {
			top = z
		}
		if z.MLvl >= p.Level-5 && (best == nil || z.MLvl < best.MLvl) {
			best = z
		}
	}
	if best == nil {
		return top
	}
	return best
}

func zoneName(z *FarmZone) string {
	if z == nil {
		return "the current zone"
	}
	return z.Area
}

func joinReasons(r Readiness) string {
	out := ""
	for i, s := range r.Reasons {
		if i > 0 {
			out += "; "
		}
		out += s
	}
	if out == "" {
		return string(r.Band)
	}
	return out
}

// Observe updates st after a decision at o.Now: the farm clock of the level step, the time a question was queued, and the
// chapters first seen done (returned, for the Chronicle and the unlock ladder). GoNow is cleared once its chapter is done.
func (s *Spine) Observe(p Progress, st *HeroState, d Decision, now time.Time) (newlyDone []string) {
	for i := range s.Chapters {
		c := &s.Chapters[i]
		if !c.IsDone(p) {
			continue
		}
		if _, seen := st.Done[c.ID]; !seen {
			if st.Done == nil {
				st.Done = map[string]time.Time{}
			}
			st.Done[c.ID] = now
			newlyDone = append(newlyDone, c.ID)
		}
		if st.GoNow == c.ID {
			st.GoNow = ""
		}
		delete(st.Asked, c.ID)
	}
	switch d.Action {
	case ActFarm, ActAsk:
		if st.Farm == nil || st.Farm.Chapter != d.Chapter {
			st.Farm = &Farming{Chapter: d.Chapter, Since: now}
		}
	default:
		st.Farm = nil
	}
	if d.Action == ActAsk {
		if st.Asked == nil {
			st.Asked = map[string]time.Time{}
		}
		if st.Asked[d.Chapter].IsZero() {
			st.Asked[d.Chapter] = now
		}
	}
	return newlyDone
}
