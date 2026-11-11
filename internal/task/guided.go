package task

import (
	"fmt"
	"strconv"

	"d2lod-bot/internal/campaign"
)

// Guide is the guided journey (docs/campaign.md): the campaign spine decides which chapter the Normal Act 1 campaign
// runs next, the quest handlers, gates and safety walls of this package still run it. Nil fields are not allowed except
// Opts and OnDecision.
type Guide struct {
	Spine *campaign.Spine
	State func() *campaign.HeroState // the hero's persisted state (read only here)
	Class func() string              // lowercased class name
	Opts  func() campaign.Opts       // time, veto window, farm budget, readiness checks
	// OnDecision gets every decision with the progress it was made from (the bot persists Observe, queues the floor
	// question, writes the Chronicle).
	OnDecision func(campaign.Decision, campaign.Progress, campaign.Opts)
}

// WithGuide makes c follow g in Normal Act 1 (nil g: the plain floor planner).
func (c *Campaign) WithGuide(g *Guide) *Campaign {
	if c != nil {
		c.guide = g
	}
	return c
}

// chapterGroup is the plan group of a chapter without a quest handler (start, the Cold Plains waypoint).
func chapterGroup(id string) string { return "chapter." + id }

// guidedPlan is the plan the guide picks; ok false hands over to the floor planner (Free mode, the spine done, another
// act or difficulty).
func (c *Campaign) guidedPlan(w World, s Session) (Plan, bool) {
	g := c.guide
	if g == nil || !normalDiff(w) || (w.Act > 1) {
		return Plan{}, false
	}
	// Entries the spine has no chapter for (the free Rogue claim) keep their place: they are the camp's own chores.
	mapped := map[string]bool{}
	for _, ch := range g.Spine.Chapters {
		mapped[ch.Quest] = true
	}
	for _, e := range c.Entries {
		if mapped[e.Quest] || e.Act != 1 || e.Optional || e.Andy || c.complete(w, s, e.Quest) || w.Level < e.floor(w) {
			continue
		}
		if s != nil {
			if _, b := s.Blocked(e.Quest); b {
				continue
			}
		}
		return Plan{ID: "quest." + e.Quest, Group: e.Quest, Nodes: Handler(e.Quest), Why: "next: " + e.Title}, true
	}
	p, st := c.guideInputs(w, s)
	o := campaign.Opts{}
	if g.Opts != nil {
		o = g.Opts()
	}
	o.Wall = func(ch *campaign.Chapter) int {
		for _, e := range c.Entries {
			if e.Quest != "" && e.Quest == ch.Quest {
				return e.floor(w)
			}
		}
		return 0
	}
	d, err := g.Spine.Next(p, st, o)
	if err != nil {
		return Plan{Stuck: true, Why: "campaign: " + err.Error()}, true
	}
	if g.OnDecision != nil {
		g.OnDecision(d, p, o)
	}
	ch := g.Spine.Chapter(d.Chapter)
	switch d.Action {
	case campaign.ActGo:
		why := d.Why
		if d.Warn != "" {
			why += " (" + d.Warn + ")"
		}
		grp := chapterPlanGroup(ch)
		id := "quest." + grp
		if ch.Quest == "" {
			id = grp
		}
		return Plan{ID: id, Group: grp, Nodes: chapterNodes(ch), Why: why}, true
	case campaign.ActFarm, campaign.ActAsk:
		return Plan{ID: "level." + strconv.Itoa(d.Level), Nodes: nodes(StepLevelFor(d.Level, d.Title)), Why: d.Why}, true
	case campaign.ActOrders:
		return Plan{ID: "level." + strconv.Itoa(w.Level+1), Nodes: nodes(StepLevel(w.Level + 1)), Why: d.Why}, true
	}
	return Plan{}, false // free mode or the spine is done: the floor planner takes the hop to Act 2 and beyond
}

// guideInputs builds the decision inputs from the world: done quests (a session-finished group counts), known
// waypoints, the merc, and the hero state with session-blocked chapters on Hold (a copy: the persisted state is not changed).
func (c *Campaign) guideInputs(w World, s Session) (campaign.Progress, *campaign.HeroState) {
	g := c.guide
	p := campaign.Progress{Hero: campaign.Hero{Class: g.Class(), Level: w.Level, Merc: routeMercOK(w)},
		Quests: map[string]bool{}, Waypoints: map[int]bool{}, MercOwned: w.HasMerc || w.MercKnown}
	for a, ok := range w.WPs {
		p.Waypoints[int(a)] = ok
	}
	orig := g.State()
	st := *orig
	st.Intents = map[string]campaign.Intent{}
	for k, v := range orig.Intents {
		st.Intents[k] = v
	}
	for i := range g.Spine.Chapters {
		ch := &g.Spine.Chapters[i]
		grp := chapterPlanGroup(ch)
		if q := ch.Done.Quest; q != "" && (QuestDone(w, q) || (s != nil && s.Finished(q))) {
			p.Quests[q] = true
		}
		if s == nil {
			continue
		}
		if ch.Done.Merc && s.Finished(grp) {
			p.MercOwned = true // the claim ran and found nothing to claim (no merc reader): do not repeat it this session
		}
		if wp := ch.Done.Waypoint; wp > 0 && s.Finished(grp) {
			p.Waypoints[wp] = true // the waypoint step is optional: a finished group without proof counts
		}
		if _, b := s.Blocked(grp); b && st.GoNow != ch.ID {
			st.Intents[ch.ID] = campaign.Hold
		}
	}
	// A later quest done proves the waypoint chapters before it (the reader may not know the waypoint list).
	for i := range g.Spine.Chapters {
		ch := &g.Spine.Chapters[i]
		if wp := ch.Done.Waypoint; wp > 0 && !p.Waypoints[wp] {
			for _, later := range g.Spine.Chapters[i+1:] {
				if later.Done.Quest != "" && p.Quests[later.Done.Quest] {
					p.Waypoints[wp] = true
				}
			}
		}
	}
	return p, &st
}

func chapterPlanGroup(ch *campaign.Chapter) string {
	if ch.Quest != "" {
		return ch.Quest
	}
	return chapterGroup(ch.ID)
}

// chapterNodes are the nodes of a chapter: its quest handler, else a waypoint or a level step from its done rule.
func chapterNodes(ch *campaign.Chapter) []Node {
	switch {
	case ch.Quest != "":
		return Handler(ch.Quest)
	case ch.Done.Waypoint > 0:
		a := Area(ch.Done.Waypoint)
		return nodes(StepWaypoint(chapterGroup(ch.ID)+"/wp", a, func(w World) bool { return w.WPs[a] }))
	}
	return nodes(StepLevelFor(ch.Done.Level, ch.Title))
}

// StepLevelFor is StepLevel whose title names the chapter it unlocks ("Level up to 13 for Charsi's hammer"); the id
// stays level.N so the progress list and the watch read it as before.
func StepLevelFor(n int, chapter string) Step {
	s := StepLevel(n)
	if chapter != "" {
		s.Title = fmt.Sprintf("Level up to %d for %s", n, chapter)
	}
	return s
}

// NewGuide is the guide of a journey controller; class gives the lowercased class name of the hero.
func NewGuide(c *campaign.Controller, class func() string) *Guide {
	return &Guide{Spine: c.Spine(), State: c.State, Class: class, Opts: c.Opts, OnDecision: c.Decided}
}
