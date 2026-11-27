package supply

import (
	"fmt"
	"strings"
)

// ViewSchema is the schema of the viewer's town plan (GET /api/townplan, town-planner section 8; additive).
const ViewSchema = "watch.townplan/1"

// ViewLine is one row of the shopping list.
type ViewLine struct {
	Line   string `json:"line"`
	Label  string `json:"label"`
	Have   int    `json:"have"`
	Min    int    `json:"min"`
	Target int    `json:"target"`
	Buy    int    `json:"buy"`
	Cost   int    `json:"cost"`
	Status string `json:"status"`        // ok, buy, short, dropped, clamped, blocked
	Why    string `json:"why,omitempty"` // the rule that set the min (tooltip)
}

// ViewBudget is the budget row.
type ViewBudget struct {
	Purse int `json:"purse"`
	Stash int `json:"stash"`
	Spent int `json:"spent"`
	Left  int `json:"left"`
}

// View is what the viewer shows: derived from a Plan by ViewOf, no clock, no input.
type View struct {
	Schema  string     `json:"schema"`
	For     string     `json:"for"`
	Profile string     `json:"profile"`
	Verdict string     `json:"verdict"`
	Chip    string     `json:"chip"`
	Lines   []ViewLine `json:"lines"`
	Budget  ViewBudget `json:"budget"`
	Stops   []string   `json:"stops"`
	Done    int        `json:"stopsDone"`
	Waiting string     `json:"waiting"`
	Minute  string     `json:"minute"`
	Known   bool       `json:"known"` // false: no visit has solved a plan yet
}

var lineLabels = map[Line]string{
	Healing:      "healing potions",
	Mana:         "mana potions",
	Rejuvenation: "rejuvenation potions",
	Portal:       "Town Portal scrolls",
	GoldCarry:    "gold to carry",
}

// Label is the player's word for a line.
func (l Line) Label() string {
	if s, ok := lineLabels[l]; ok {
		return s
	}
	return strings.ReplaceAll(string(l), "_", " ")
}

// Chip is the verdict as the header chip words it.
func (v Verdict) Chip() string {
	switch v {
	case Ready:
		return "Ready"
	case ReadyLean:
		return "Ready (lean)"
	case ShortGold:
		return "Short of gold"
	case Blocked:
		return "Blocked"
	}
	return string(v)
}

// Waiting is the one sentence of the card for a step that waits, the same words the guard says (Why) plus what the
// visit does about it; "" when nothing is open and nothing is to buy.
func (p Plan) Waiting() string {
	var buys []string
	first := -1
	for i, ln := range p.Lines {
		if ln.Buy > 0 {
			buys = append(buys, fmt.Sprintf("%d %s (%d gold)", ln.Buy, ln.Line.Label(), ln.Cost))
			if first < 0 {
				first = i
			}
		}
	}
	switch p.Verdict {
	case ShortGold:
		s := fmt.Sprintf("Short of gold: needs %d more for %s.", p.Short, Line(p.Open).Label())
		if len(buys) > 0 {
			s += " Buying " + strings.Join(buys, ", ") + " first."
		}
		return s
	case Blocked:
		return fmt.Sprintf("Blocked: %s cannot be bought here.", Line(p.Open).Label())
	}
	if first < 0 {
		return ""
	}
	ln := p.Lines[first]
	return fmt.Sprintf("Waiting: %s %d of %d. Buying %s.", ln.Line.Label(), ln.Have, ln.Min, strings.Join(buys, ", "))
}

// MinuteLine is the one line per solve for the minute line and the log (town-planner section 8).
func (p Plan) MinuteLine() string {
	var buy, dropped []string
	for _, ln := range p.Lines {
		if ln.Buy > 0 {
			buy = append(buy, fmt.Sprintf("%s x%d(%d)", ln.Line, ln.Buy, ln.Cost))
		}
		if ln.Status == "dropped" || ln.Status == "short" {
			dropped = append(dropped, fmt.Sprintf("%s(short %d)", ln.Line, max(ln.Min-ln.Have-ln.Buy, 0)))
		}
	}
	return fmt.Sprintf("town plan: for=%s verdict=%s buy=%s dropped=%s", p.Need.Excursion, p.Verdict, orNone(buy), orNone(dropped))
}

// ViewOf builds the viewer's plan. The count gives the purse and stash for the budget row; stops are the stops of the
// visit in order and done how many are ticked off.
func ViewOf(p Plan, c Count, stops []string, done int) View {
	v := View{Schema: ViewSchema, For: p.Need.Excursion, Profile: p.Need.Profile, Verdict: string(p.Verdict), Chip: p.Verdict.Chip(),
		Lines: []ViewLine{}, Stops: append([]string{}, stops...), Done: min(max(done, 0), len(stops)), Known: true,
		Waiting: p.Waiting(), Minute: p.MinuteLine()}
	for _, ln := range p.Lines {
		v.Lines = append(v.Lines, ViewLine{Line: string(ln.Line), Label: ln.Line.Label(), Have: ln.Have, Min: ln.Min, Target: ln.Target,
			Buy: ln.Buy, Cost: ln.Cost, Status: ln.Status, Why: ln.Why})
	}
	stash := 0
	if c.StashKnown {
		stash = c.Stash
	}
	v.Budget = ViewBudget{Purse: c.Purse, Stash: stash, Spent: p.Spent, Left: max(p.Budget-p.Spent, 0)}
	return v
}

// NoView is the view before any visit solved a plan.
func NoView() View {
	return View{Schema: ViewSchema, Lines: []ViewLine{}, Stops: []string{}, Waiting: "No town plan yet: the hero has not been to town since the bot started."}
}
