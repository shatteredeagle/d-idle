package pact

import (
	"fmt"
	"sort"
	"strings"
)

// The data behind the Pact offer page and the return screen's cost-and-paid section (TASK-168, docs/pact-ui.md): the
// catalogue by lever family with ranks, the heat budget, the incompatibility reasons, the unlock state and the current
// policy next to the proposed one. Pure data; the page and routes are in internal/watch/pactui.go.

// Span is one stretch of time a Pact acted at a rank. End < 0 means it still does. Times are Config.Rel seconds.
type Span struct {
	Pact  string  `json:"pact"`
	Rank  int     `json:"rank"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

func (c *Controller) trackSpans(liveNow map[string]int) {
	if c.cfg.Rel == nil {
		return
	}
	now := c.cfg.Rel()
	for i := range c.spans {
		sp := &c.spans[i]
		if sp.End >= 0 {
			continue
		}
		if r, on := liveNow[sp.Pact]; !on || r != sp.Rank {
			sp.End = now
		} else {
			delete(liveNow, sp.Pact)
		}
	}
	ids := make([]string, 0, len(liveNow))
	for id := range liveNow {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c.spans = append(c.spans, Span{Pact: id, Rank: liveNow[id], Start: now, End: -1})
	}
	if len(c.spans) > 200 {
		c.spans = append([]Span(nil), c.spans[len(c.spans)-200:]...)
	}
}

// Spans are the live stretches so far (a copy).
func (c *Controller) Spans() []Span {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Span(nil), c.spans...)
}

func (c *Controller) unlockedLocked(t string) bool {
	return c.cfg.Gate != nil && c.cfg.Gate.Allowed(LadderID(t))
}

// Legacy lists the unlock triggers the state file recorded before the unlock ladder took over (TASK-168 -> TASK-153).
// The caller adopts each into the ladder, then calls ClearLegacy.
func (c *Controller) Legacy() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.st.Unlocked...)
}

// ClearLegacy drops the pre-ladder unlock field from the state file once the ladder holds it. Idempotent.
func (c *Controller) ClearLegacy() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.st.Unlocked) == 0 {
		return nil
	}
	old := c.st
	c.st.Unlocked = nil
	if err := c.save(); err != nil {
		c.st = old
		return err
	}
	return nil
}

// unlockText says in plain words what opens a Pact. Counted triggers (hall_count:N, tally:N) are not seen by the bot
// yet, so they stay locked (U).
func unlockText(t string) string {
	switch t {
	case "andariel_first_kill":
		return "Kill Andariel once to learn this Pact."
	case "baal_normal":
		return "Defeat Baal in Normal to learn this Pact."
	}
	return "Locked (" + t + ")."
}

// Choose is Select with the unlock rule: a locked Pact cannot be chosen.
func (c *Controller) Choose(picks []Pick) error {
	c.mu.Lock()
	for _, pk := range picks {
		p, err := c.cfg.Cat.Get(pk.ID)
		if err != nil {
			c.mu.Unlock()
			return err
		}
		if !c.unlockedLocked(p.UnlockTrigger) {
			c.mu.Unlock()
			return fmt.Errorf("pact: %q is locked: %s", p.ID, unlockText(p.UnlockTrigger))
		}
	}
	c.mu.Unlock()
	return c.Select(picks)
}

// RankView is one rank on the page.
type RankView struct {
	Rank        int    `json:"rank"`
	Heat        int    `json:"heat"`
	Effect      string `json:"effect"`
	TallyPct    int    `json:"tallyPct"`
	CharmPoints int    `json:"charmPoints,omitempty"`
	Title       string `json:"title,omitempty"`
	Status      string `json:"status"` // "ok", or why it cannot be chosen now
}

// PactView is one catalogue entry; a locked one is a silhouette (no name, no effect lines).
type PactView struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Locked   bool       `json:"locked"`
	Unlock   string     `json:"unlock,omitempty"`
	Learn    string     `json:"learn,omitempty"`
	Away     string     `json:"away"`
	Chosen   int        `json:"chosen"`   // rank in the current set, 0 = none
	Proposed int        `json:"proposed"` // rank in the proposed set, 0 = none
	Ranks    []RankView `json:"ranks"`
}

// FamilyView groups Pacts by lever family.
type FamilyView struct {
	Family string     `json:"family"`
	Pacts  []PactView `json:"pacts"`
}

// LeverLine is one lever (a policy row or a tuner knob) with the value now and the value the proposed set asks for.
type LeverLine struct {
	Lever    string `json:"lever"`
	Now      string `json:"now"`
	Proposed string `json:"proposed"`
	Source   string `json:"source"` // the Pact (and rank) that sets it in the proposed set, "" = no Pact
}

// SetView is a chosen set with its heat and its check result.
type SetView struct {
	Picks []PickView `json:"picks"`
	Heat  int        `json:"heat"`
	// TallyPct and CharmPoints are the summed rewards of the set (meta layer only).
	TallyPct    int    `json:"tallyPct"`
	CharmPoints int    `json:"charmPoints"`
	Error       string `json:"error,omitempty"` // why the set is refused, "" = allowed
}

// PickView names a pick for the page.
type PickView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Rank int    `json:"rank"`
}

// View is the whole offer page.
type View struct {
	Class      string       `json:"class"`
	Heat       int          `json:"heat"`
	HeatCap    int          `json:"heatCap"`
	MaxCap     int          `json:"maxCap"`
	Slots      int          `json:"slots"`
	Current    SetView      `json:"current"`
	Proposed   *SetView     `json:"proposed,omitempty"`
	Levers     []LeverLine  `json:"levers,omitempty"` // only with a proposal
	Families   []FamilyView `json:"families"`
	AccountMsg string       `json:"note"`
}

// pickView describes a set; the error is the Check result for the class.
func (c *Controller) setView(picks []Pick, slots, masteries int) SetView {
	sv := SetView{Picks: []PickView{}}
	for _, pk := range picks {
		name := pk.ID
		if p, err := c.cfg.Cat.Get(pk.ID); err == nil {
			name = p.Name
		}
		sv.Picks = append(sv.Picks, PickView{ID: pk.ID, Name: name, Rank: pk.Rank})
		if p, err := c.cfg.Cat.Get(pk.ID); err == nil {
			if r, err := p.Rank(pk.Rank); err == nil {
				sv.TallyPct += r.Reward.TallyPct
				sv.CharmPoints += r.Reward.CharmPoints
			}
		}
	}
	if h, err := c.cfg.Cat.Heat(picks); err == nil {
		sv.Heat = h
	}
	if err := c.cfg.Cat.Check(picks, c.cfg.Class, slots, masteries); err != nil {
		sv.Error = err.Error()
	}
	return sv
}

// ParsePicks reads "id:rank,id:rank" (the proposal query). Empty means no picks.
func ParsePicks(s string) ([]Pick, error) {
	var out []Pick
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, r, ok := strings.Cut(part, ":")
		var rank int
		if !ok {
			return nil, fmt.Errorf("pact: %q is not id:rank", part)
		}
		if _, err := fmt.Sscanf(r, "%d", &rank); err != nil {
			return nil, fmt.Errorf("pact: %q has no rank number", part)
		}
		out = append(out, Pick{ID: id, Rank: rank})
	}
	return out, nil
}

// View builds the page. propose is nil for no proposal (an empty non-nil slice proposes the empty set).
func (c *Controller) View(propose []Pick) View {
	c.mu.Lock()
	defer c.mu.Unlock()
	cat := c.cfg.Cat
	slots, masteries := cat.Limits.SlotsAtUnlock, c.st.Masteries
	v := View{Class: c.cfg.Class, HeatCap: cat.HeatCap(masteries), MaxCap: cat.Limits.MaxCap, Slots: slots, Families: []FamilyView{},
		AccountMsg: "Pacts change the bot's play; rewards go to the tally, the Hall and the legacy charm only. Vanilla XP and loot never change."}
	v.Current = c.setView(c.st.Picks, slots, masteries)
	v.Heat = v.Current.Heat
	cur := map[string]int{}
	for _, pk := range c.st.Picks {
		cur[pk.ID] = pk.Rank
	}
	prop := map[string]int{}
	if propose != nil {
		pv := c.setView(propose, slots, masteries)
		v.Proposed = &pv
		for _, pk := range propose {
			prop[pk.ID] = pk.Rank
		}
		v.Levers = c.levers(propose)
	}
	ref := c.st.Picks
	if propose != nil {
		ref = propose
	}
	byFam := map[string]*FamilyView{}
	var order []string
	for i := range cat.Pacts {
		p := &cat.Pacts[i]
		if !p.AllowsClass(c.cfg.Class) {
			continue
		}
		pv := PactView{ID: p.ID, Name: p.Name, Away: p.AwaySafety, Chosen: cur[p.ID], Proposed: prop[p.ID], Learn: p.LearnText}
		if !c.unlockedLocked(p.UnlockTrigger) {
			pv.Locked, pv.Name, pv.Learn = true, "???", ""
			pv.Unlock = unlockText(p.UnlockTrigger)
		}
		for _, r := range p.Ranks {
			rv := RankView{Rank: r.Rank, Heat: r.Heat, TallyPct: r.Reward.TallyPct, CharmPoints: r.Reward.CharmPoints, Title: r.Reward.Title, Status: "ok"}
			if !pv.Locked {
				rv.Effect = r.EffectLine
				others := make([]Pick, 0, len(ref)+1)
				for _, pk := range ref {
					if pk.ID != p.ID {
						others = append(others, pk)
					}
				}
				if err := cat.Check(append(others, Pick{ID: p.ID, Rank: r.Rank}), c.cfg.Class, slots, masteries); err != nil {
					rv.Status = err.Error()
				}
			} else {
				rv.Status = "locked"
			}
			pv.Ranks = append(pv.Ranks, rv)
		}
		f := byFam[p.LeverFamily]
		if f == nil {
			f = &FamilyView{Family: p.LeverFamily}
			byFam[p.LeverFamily] = f
			order = append(order, p.LeverFamily)
		}
		f.Pacts = append(f.Pacts, pv)
	}
	sort.Strings(order)
	for _, k := range order {
		v.Families = append(v.Families, *byFam[k])
	}
	return v
}

// levers lists what the proposed set would set, next to what the sheet and the tuner hold now.
func (c *Controller) levers(propose []Pick) []LeverLine {
	var out []LeverLine
	seen := map[string]bool{}
	for _, pk := range propose {
		p, err := c.cfg.Cat.Get(pk.ID)
		if err != nil {
			continue
		}
		r, err := p.Rank(pk.Rank)
		if err != nil {
			continue
		}
		src := fmt.Sprintf("%s rank %d", p.Name, pk.Rank)
		if r.Params.Row != "" && !seen["row:"+r.Params.Row] {
			seen["row:"+r.Params.Row] = true
			now := "unknown"
			if c.cfg.Sheet != nil {
				now = c.cfg.Sheet.Current(r.Params.Row)
			}
			out = append(out, LeverLine{Lever: "policy row " + r.Params.Row, Now: now, Proposed: r.Params.Option, Source: src})
		}
		ks := make([]string, 0, len(r.Params.Knobs))
		for k := range r.Params.Knobs {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		for _, k := range ks {
			if seen["knob:"+k] {
				continue
			}
			seen["knob:"+k] = true
			now := "profile value"
			if v, ok := c.knobs[k]; ok {
				now = fmt.Sprint(v)
			}
			out = append(out, LeverLine{Lever: "knob " + k, Now: now, Proposed: fmt.Sprint(r.Params.Knobs[k]), Source: src})
		}
		if r.Params.DeathLimit > 0 {
			out = append(out, LeverLine{Lever: "death limit", Now: "none", Proposed: fmt.Sprint(r.Params.DeathLimit), Source: src})
		}
	}
	return out
}

// CostLine is one chosen Pact for the "what it cost and paid" section: what it asked, how long it acted (from the first
// stretch at the current rank to the end of the last) and what it promises.
type CostLine struct {
	Pact   string  `json:"pact"`
	Name   string  `json:"name"`
	Rank   int     `json:"rank"`
	Effect string  `json:"effect"`
	Row    string  `json:"row,omitempty"` // the policy row it sets, the key of the sheet's counters
	Start  float64 `json:"start"`         // first live moment at this rank; -1 = never acted
	End    float64 `json:"end"`           // end of the last stretch; -1 = acting now
	Reward Reward  `json:"reward"`
}

// Costs lists the chosen Pacts with their live stretches.
func (c *Controller) Costs() []CostLine {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []CostLine
	for _, pk := range c.st.Picks {
		p, err := c.cfg.Cat.Get(pk.ID)
		if err != nil {
			continue
		}
		r, err := p.Rank(pk.Rank)
		if err != nil {
			continue
		}
		cl := CostLine{Pact: p.ID, Name: p.Name, Rank: pk.Rank, Effect: r.EffectLine, Row: r.Params.Row, Start: -1, End: 0, Reward: r.Reward}
		for _, sp := range c.spans {
			if sp.Pact != p.ID || sp.Rank != pk.Rank {
				continue
			}
			if cl.Start < 0 {
				cl.Start = sp.Start
			}
			cl.End = sp.End
		}
		out = append(out, cl)
	}
	return out
}
