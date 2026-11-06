package watch

import (
	"errors"
	"fmt"
	"strings"

	"d2lod-bot/internal/goals"
)

// T0 order kinds (docs/research/automation-ladder.md, T0 Orders: "go to AREA", "fight here", "come home").
const (
	OrderGo   = "go"    // go to Area and fight there for Minutes
	OrderHere = "fight" // fight where the hero stands for Minutes
	OrderHome = "home"  // go to town and rest
)

// Order states. The bot reports changes through Hub.OrderState.
const (
	OrderQueued    = "queued"
	OrderActive    = "active"
	OrderDone      = "done"
	OrderCancelled = "cancelled"
	OrderFailed    = "failed"
)

// Order is one task order of the orders box.
type Order struct {
	ID      int     `json:"id"`
	Kind    string  `json:"kind"`
	Area    string  `json:"area,omitempty"`
	Minutes int     `json:"minutes,omitempty"`
	Forever bool    `json:"forever,omitempty"` // runs until the player cancels it (all farm limits 0)
	State   string  `json:"state"`
	Note    string  `json:"note,omitempty"`
	Text    string  `json:"text"`
	GivenAt float64 `json:"givenAt"` // virtual seconds
	// Progress, for the Hero-now panel: set when the order starts, kills and xp counted until it ends.
	StartedAt float64 `json:"startedAt,omitempty"`
	Kills     int     `json:"kills,omitempty"`
	XPGained  int64   `json:"xpGained,omitempty"`
}

// OrderSink is what the bot implements to take an order. The Hub validates first; the sink may still refuse (error text
// is shown to the player). GoalsOrders is the real-bot mapping.
type OrderSink interface {
	Order(o Order) error
	Cancel(id int) error
}

// Limits of the orders box.
const (
	MaxOrderMinutes = 240
	MaxOpenOrders   = 5
)

var (
	ErrBadOrder    = errors.New("watch: bad order")
	ErrNoOrder     = errors.New("watch: no such order")
	ErrOrdersFull  = fmt.Errorf("watch: at most %d open orders", MaxOpenOrders)
	errOrderClosed = errors.New("watch: order already finished")
)

func orderText(o Order) string {
	how := fmt.Sprintf("for %d min", o.Minutes)
	if o.Forever {
		how = "until you cancel"
	}
	switch o.Kind {
	case OrderGo:
		return fmt.Sprintf("Farm %s %s", o.Area, how)
	case OrderHere:
		return "Fight here " + how
	default:
		return "Rest in town"
	}
}

// validateOrder normalises an order request against the area list (case-insensitive) and the limits.
// forever (no time limit) replaces the minutes; the two never combine.
func validateOrder(kind, area string, minutes int, forever bool, areas []string) (Order, error) {
	o := Order{Kind: kind}
	switch kind {
	case OrderGo:
		for _, a := range areas {
			if strings.EqualFold(a, strings.TrimSpace(area)) {
				o.Area = a
			}
		}
		if o.Area == "" {
			return o, fmt.Errorf("%w: area %q is not one of the orderable areas", ErrBadOrder, safeText(area, 40))
		}
		fallthrough
	case OrderHere:
		switch {
		case forever && minutes != 0:
			return o, fmt.Errorf("%w: until-you-cancel takes no minutes", ErrBadOrder)
		case forever:
			o.Forever = true
		case minutes < 1 || minutes > MaxOrderMinutes:
			return o, fmt.Errorf("%w: minutes must be 1 to %d", ErrBadOrder, MaxOrderMinutes)
		default:
			o.Minutes = minutes
		}
	case OrderHome:
		if area != "" || minutes != 0 || forever {
			return o, fmt.Errorf("%w: rest in town takes no area or minutes", ErrBadOrder)
		}
	default:
		return o, fmt.Errorf("%w: kind must be go, fight or home", ErrBadOrder)
	}
	o.Text = orderText(o)
	return o, nil
}

// OrderToGoal maps a T0 order to a goal of internal/goals (U: the goals manager is the bot's order queue today; the
// real bot may prefer its own T0 entry point, see the design doc "Bot integration"). here is the area the hero stands
// in, needed for "fight here". "Come home" becomes a short idle goal, which sends the hero to town (U).
func OrderToGoal(o Order, here string) (goals.Goal, error) {
	g := goals.Goal{Priority: 5}
	switch o.Kind {
	case OrderGo:
		g.Type, g.Params = goals.FarmArea, goals.Params{Area: o.Area, UntilMins: o.Minutes}
	case OrderHere:
		if here == "" {
			return g, fmt.Errorf("%w: the current area is unknown", ErrBadOrder)
		}
		g.Type, g.Params = goals.FarmArea, goals.Params{Area: here, UntilMins: o.Minutes}
	case OrderHome:
		g.Type, g.Params = goals.Idle, goals.Params{Minutes: 1}
		g.Title = "Come home"
	default:
		return g, fmt.Errorf("%w: kind %q", ErrBadOrder, o.Kind)
	}
	if err := g.Normalize(); err != nil {
		return g, fmt.Errorf("%w: %v", ErrBadOrder, err)
	}
	return g, nil
}

// safeText makes bot-supplied or player-supplied text plain: control characters and newlines become spaces, runs of
// spaces collapse, and the result is cut to max runes. JSON encoding escapes markup; the page also only uses textContent.
func safeText(s string, max int) string {
	var b strings.Builder
	n := 0
	space := true
	for _, r := range s {
		if r < 0x20 || r == 0x7f || r == 0x2028 || r == 0x2029 {
			r = ' '
		}
		if r == ' ' {
			if space {
				continue
			}
			space = true
		} else {
			space = false
		}
		if n >= max {
			break
		}
		b.WriteRune(r)
		n++
	}
	return strings.TrimSpace(b.String())
}
