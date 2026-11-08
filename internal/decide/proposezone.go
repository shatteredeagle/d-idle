package decide

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// The hero proposes the next zone (TASK-131, ADR 0018 item F3; docs/proposezone.md). The proposer is a question with a
// key per (from-zone, to-zone). It lives beside the decision-point store, not in its catalogue: the key is a free zone
// pair, the catalogue keys are enumerated bands. It copies the store's repeat semantics (cooldown after an answer,
// quiet after repeated stays) and it never fades: nothing is lost while the player is away, the proposal waits.
//
// There is NO auto-advance: the proposer never places an order. Only an answer of "go" (the desk in internal/watch) does.

// ProposeSchema is the schema of the table file.
const ProposeSchema = "decide.proposezone/1"

// Answers of a proposal.
const (
	ProposeGo    = "go"
	ProposeStay  = "stay"
	ProposeLater = "later"
)

// Proposal states.
const (
	ProposalPending  = "pending"
	ProposalAccepted = "accepted"
	ProposalStayed   = "stayed"
	ProposalLater    = "later"
)

// ErrProposal wraps every refusal of the proposer.
var ErrProposal = errors.New("decide: zone proposal")

// ZoneNode is one zone and the zones that can follow it, best first.
type ZoneNode struct {
	Area string   `json:"area"`
	Next []string `json:"next"`
}

// ClassLead is how many levels above the zone minimum a class wants to be before the hero proposes.
type ClassLead struct {
	Lead int    `json:"lead"`
	Note string `json:"note,omitempty"`
}

// ProposeTable is the swappable data of the proposer.
type ProposeTable struct {
	Schema     string               `json:"schema"`
	Doc        string               `json:"doc,omitempty"`
	Zones      []ZoneNode           `json:"zones"`
	Classes    map[string]ClassLead `json:"classes"`
	MinPotions int                  `json:"minPotions"`
}

//go:embed proposezone.json
var shippedPropose []byte

// ShippedProposeTable loads the embedded table.
func ShippedProposeTable() (*ProposeTable, error) { return ParseProposeTable(shippedPropose) }

// ParseProposeTable decodes (unknown fields are an error) and validates.
func ParseProposeTable(b []byte) (*ProposeTable, error) {
	var t ProposeTable
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("%w: table: %v", ErrProposal, err)
	}
	if t.Schema != ProposeSchema {
		return nil, fmt.Errorf("%w: table schema %q, want %q", ErrProposal, t.Schema, ProposeSchema)
	}
	seen := map[string]bool{}
	for _, z := range t.Zones {
		k := strings.ToLower(z.Area)
		if z.Area == "" || len(z.Next) == 0 || seen[k] {
			return nil, fmt.Errorf("%w: table zone %q is empty, has no next zone or is listed twice", ErrProposal, z.Area)
		}
		seen[k] = true
	}
	if len(t.Classes) == 0 || t.MinPotions < 0 {
		return nil, fmt.Errorf("%w: table needs classes and a non-negative minPotions", ErrProposal)
	}
	return &t, nil
}

func (t *ProposeTable) node(area string) (ZoneNode, bool) {
	for _, z := range t.Zones {
		if strings.EqualFold(z.Area, strings.TrimSpace(area)) {
			return z, true
		}
	}
	return ZoneNode{}, false
}

// Knows says whether area is a zone the proposer proposes from.
func (t *ProposeTable) Knows(area string) bool { _, ok := t.node(area); return ok }

// RangeFn gives the recommended clvl range of a zone; the bot passes the tally zone table so there is one table.
// ok false means the zone has no range (the proposer then skips it and says so in Skipped).
type RangeFn func(area string) (min, max int, ok bool)

// ProposeInput is what the proposer reads each time it looks. Kit is the bot's kit verdict ("OK", "THIN", "NONE"), the
// same enum as the catalogue field kit.
type ProposeInput struct {
	Class   string
	Clvl    int
	Area    string // the field the hero is farming now
	Kit     string
	Potions int
	Deaths  int // deaths in the zone just now, display only
	Away    bool
}

// Proposal is one open or answered question.
type Proposal struct {
	ID      int      `json:"id"`
	Key     string   `json:"key"`
	From    string   `json:"from"`
	To      string   `json:"to"`
	Class   string   `json:"class"`
	Clvl    int      `json:"clvl"`
	Min     int      `json:"min"`
	Max     int      `json:"max"`
	Text    string   `json:"text"`
	Why     []string `json:"why"`
	State   string   `json:"state"`
	Asked   float64  `json:"askedAt"` // clock seconds when raised
	ReadyAt float64  `json:"readyAt"` // clock seconds when the hero first met the range for this pair
	Away    bool     `json:"wasAway"` // raised while the player was away
	Options []string `json:"options"` // go, stay, later
}

type proposeRec struct {
	ReadyAt  float64 `json:"readyAt"`
	Answered float64 `json:"answeredAt,omitempty"`
	Answer   string  `json:"answer,omitempty"`
	Stays    int     `json:"stays,omitempty"` // stays in a row (an answer of go resets, later does not count)
}

type proposeFile struct {
	Next  int                    `json:"next"`
	Recs  map[string]*proposeRec `json:"recs"`
	Items []*Proposal            `json:"items"`
}

// Proposer raises at most one pending proposal. Safe for concurrent use.
type Proposer struct {
	mu         sync.Mutex
	tbl        *ProposeTable
	ranges     RangeFn
	clk        Clock
	path       string
	cooldown   time.Duration
	quietAfter int
	f          proposeFile
}

// NewProposer makes a proposer; path "" keeps it in memory. A file that exists but cannot be read is an error.
func NewProposer(tbl *ProposeTable, ranges RangeFn, clk Clock, path string) (*Proposer, error) {
	if tbl == nil || ranges == nil || clk == nil {
		return nil, fmt.Errorf("%w: needs a table, a range function and a clock", ErrProposal)
	}
	p := &Proposer{tbl: tbl, ranges: ranges, clk: clk, path: path, cooldown: DefaultRepeatCooldown, quietAfter: DefaultQuietAfter,
		f: proposeFile{Next: 1, Recs: map[string]*proposeRec{}}}
	if path != "" {
		b, err := os.ReadFile(path)
		switch {
		case err == nil:
			if err := json.Unmarshal(b, &p.f); err != nil {
				return nil, fmt.Errorf("%w: %s: %v", ErrProposal, path, err)
			}
			if p.f.Recs == nil {
				p.f.Recs = map[string]*proposeRec{}
			}
		case !errors.Is(err, os.ErrNotExist):
			return nil, fmt.Errorf("%w: %v", ErrProposal, err)
		}
	}
	return p, nil
}

// SetRepeatPolicy is Store.SetRepeatPolicy for proposals: after a stay the pair is not proposed again for cooldown;
// after quietAfter stays in a row it is not proposed again at all (cooldown 0: no suppression).
func (p *Proposer) SetRepeatPolicy(cooldown time.Duration, quietAfter int) {
	p.mu.Lock()
	p.cooldown, p.quietAfter = cooldown, quietAfter
	p.mu.Unlock()
}

func (p *Proposer) now() float64 { return p.clk.Now().Seconds() }

func (p *Proposer) saveLocked() error {
	if p.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(&p.f, "", " ")
	if err != nil {
		return err
	}
	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p.path)
}

// Look is called with the hero's state. It returns the pending proposal (a new one, or the one still waiting) or nil.
// An unknown class or a farm area the table does not know is an error only for the class; an unknown area simply has no
// next zone. Nothing is ever placed or advanced here.
func (p *Proposer) Look(in ProposeInput) (*Proposal, error) {
	cl, ok := p.tbl.Classes[strings.ToLower(in.Class)]
	if !ok {
		return nil, fmt.Errorf("%w: no lead for class %q in the table", ErrProposal, in.Class)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, it := range p.f.Items {
		if it.State == ProposalPending {
			// The hero moved on its own (an order of the player) and left the pair: the question is stale and closes.
			if !strings.EqualFold(it.From, in.Area) && in.Area != "" {
				it.State = ProposalStayed
				continue
			}
			c := *it
			return &c, p.saveLocked()
		}
	}
	node, ok := p.tbl.node(in.Area)
	if !ok || in.Clvl < 1 || in.Kit != "OK" || in.Potions < p.tbl.MinPotions {
		return nil, nil
	}
	now := p.now()
	for _, to := range node.Next {
		min, max, ok := p.ranges(to)
		if !ok || in.Clvl < min+cl.Lead || in.Clvl > max {
			continue
		}
		key := node.Area + ">" + to
		rec := p.f.Recs[key]
		if rec == nil {
			rec = &proposeRec{ReadyAt: now}
			p.f.Recs[key] = rec
		}
		if p.quietAfter > 0 && rec.Stays >= p.quietAfter {
			continue
		}
		if p.cooldown > 0 && rec.Answer != "" && now-rec.Answered < p.cooldown.Seconds() {
			continue
		}
		pr := &Proposal{ID: p.f.Next, Key: key, From: node.Area, To: to, Class: strings.ToLower(in.Class), Clvl: in.Clvl, Min: min, Max: max,
			State: ProposalPending, Asked: now, ReadyAt: rec.ReadyAt, Away: in.Away, Options: []string{ProposeGo, ProposeStay, ProposeLater}}
		pr.Text = fmt.Sprintf("I am level %d and ready for %s (recommended %d to %d, kit ok). Go there?", in.Clvl, to, min, max)
		pr.Why = []string{fmt.Sprintf("level %d is inside %d to %d (%s wants %d more level%s first)", in.Clvl, min, max, pr.Class, cl.Lead, plural(cl.Lead)),
			fmt.Sprintf("%d potion%s in the belt", in.Potions, plural(in.Potions))}
		if in.Deaths > 0 {
			pr.Why = append(pr.Why, fmt.Sprintf("%d death%s in %s lately", in.Deaths, plural(in.Deaths), node.Area))
		} else {
			pr.Why = append(pr.Why, "no deaths lately")
		}
		p.f.Next++
		p.f.Items = append(p.f.Items, pr)
		c := *pr
		return &c, p.saveLocked()
	}
	return nil, p.saveLocked()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// Pending lists the open proposals, oldest first.
func (p *Proposer) Pending() []Proposal {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := []Proposal{}
	for _, it := range p.f.Items {
		if it.State == ProposalPending {
			out = append(out, *it)
		}
	}
	return out
}

// Get returns one proposal by id.
func (p *Proposer) Get(id int) (Proposal, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, it := range p.f.Items {
		if it.ID == id {
			return *it, true
		}
	}
	return Proposal{}, false
}

// Answer records the player's answer to a pending proposal. "go" is recorded by the desk only after the order was
// placed; "stay" counts toward quiet; "later" starts the cooldown but does not count.
func (p *Proposer) Answer(id int, answer string) (Proposal, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var it *Proposal
	for _, x := range p.f.Items {
		if x.ID == id {
			it = x
		}
	}
	if it == nil {
		return Proposal{}, fmt.Errorf("%w: no proposal %d", ErrProposal, id)
	}
	if it.State != ProposalPending {
		return *it, fmt.Errorf("%w: proposal %d is already %s", ErrProposal, id, it.State)
	}
	if answer != ProposeGo && answer != ProposeStay && answer != ProposeLater {
		return *it, fmt.Errorf("%w: answer %q is not go, stay or later", ErrProposal, answer)
	}
	rec := p.f.Recs[it.Key]
	if rec == nil {
		rec = &proposeRec{ReadyAt: it.ReadyAt}
		p.f.Recs[it.Key] = rec
	}
	rec.Answer, rec.Answered = answer, p.now()
	switch answer {
	case ProposeGo:
		it.State, rec.Stays = ProposalAccepted, 0
	case ProposeStay:
		it.State = ProposalStayed
		rec.Stays++
	case ProposeLater:
		it.State = ProposalLater
	}
	return *it, p.saveLocked()
}
