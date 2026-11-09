package watch

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"d2lod-bot/internal/telemetry"
)

// GET /api/hero: the command centre of the watch screen (docs/ui-design-system.md section 15.2) in one read-only
// reply: who the hero is and how it is (vitals, gold), what it does now with the current order and the next ones,
// and the Why card's three lines. Everything comes from what the bot already reports (GoalsOrders.OnStatus, the order
// list, the whyline, the inventory desk for gold); nothing here reaches the bot.

const (
	HeroRoute  = "/api/hero"
	heroSchema = "watch.hero/1"
)

// HeroView is the /api/hero reply. Pointers are null when the bot has not reported the value.
type HeroView struct {
	Schema     string    `json:"schema"`
	Known      bool      `json:"known"`
	Name       string    `json:"name,omitempty"`
	Class      string    `json:"class,omitempty"`
	Level      int       `json:"level,omitempty"`
	Difficulty string    `json:"difficulty,omitempty"`
	Area       string    `json:"area,omitempty"`
	Life       int       `json:"life"`
	MaxLife    int       `json:"maxLife"`
	Mana       int       `json:"mana"`
	MaxMana    int       `json:"maxMana"`
	XP         int64     `json:"xp"`
	XPLevel    int64     `json:"xpLevel"` // experience at the start of this level (0 when unknown)
	XPNext     int64     `json:"xpNext"`  // experience at the next level (0 when unknown)
	XPPct      *float64  `json:"xpPct"`   // progress through this level
	Gold       *int      `json:"gold"`
	StashGold  *int      `json:"stashGold"`
	State      string    `json:"state"`
	StateLabel string    `json:"stateLabel"`
	Activity   string    `json:"activity"`
	Goal       *HeroGoal `json:"goal"`
	Queue      []string  `json:"queue"`
	Why        HeroWhy   `json:"why"`
}

// HeroGoal is the order under way.
type HeroGoal struct {
	Text    string   `json:"text"`
	Pct     *float64 `json:"pct"`     // share of the order's minutes gone; null for an order without a limit
	LeftMin *int     `json:"leftMin"` // minutes left
	Kills   int      `json:"kills"`
	XP      int64    `json:"xp"`
}

// HeroWhy is the Why card: the reason now, what would change it, what comes next.
type HeroWhy struct {
	Code   string `json:"code,omitempty"`
	Now    string `json:"now"`
	Change string `json:"change"`
	Next   string `json:"next"`
}

//go:embed whychange.json
var whyChangeJSON []byte

var whyChange = sync.OnceValue(func() map[string]string {
	var t struct {
		Codes map[string]string `json:"codes"`
	}
	if err := json.Unmarshal(whyChangeJSON, &t); err != nil {
		panic("watch: whychange.json: " + err.Error())
	}
	return t.Codes
})

// GoldSource reports the gold on the hero and in the stash; ok is false while it is not known.
type GoldSource func() (gold, stash int, ok bool)

// SetGoldSource sets where /api/hero reads gold from (the inventory desk in the bot; nil: unknown).
func (h *Hub) SetGoldSource(f GoldSource) { h.mu.Lock(); h.gold = f; h.mu.Unlock() }

// Hero builds the /api/hero reply.
func (h *Hub) Hero() HeroView {
	up := h.Uptime()
	ov := h.OrdersView()
	h.mu.Lock()
	hi, gold := h.hero, h.gold
	h.mu.Unlock()
	v := HeroView{Schema: heroSchema, Known: hi.Known || hi.Class != "", Name: safeText(hi.Name, 40), Class: hi.Class, Level: hi.Level,
		Difficulty: hi.Difficulty, Area: hi.Area, Life: hi.Life, MaxLife: hi.MaxLife, Mana: hi.Mana, MaxMana: hi.MaxMana,
		XP: hi.XP, XPNext: hi.XPNext, State: up.State, StateLabel: up.Label, Activity: hi.Activity, Queue: []string{}}
	if lo, ok := telemetry.DefaultExpForLevel(hi.Level); ok {
		v.XPLevel = int64(lo)
		if v.XPNext == 0 {
			if hi, ok := telemetry.DefaultExpForLevel(hi.Level + 1); ok {
				v.XPNext = int64(hi)
			}
		}
	}
	if v.XPNext > v.XPLevel && v.XP >= v.XPLevel {
		p := float64(int(1000*float64(v.XP-v.XPLevel)/float64(v.XPNext-v.XPLevel))) / 10
		v.XPPct = &p
	}
	if gold != nil {
		if g, s, ok := gold(); ok {
			v.Gold, v.StashGold = &g, &s
		}
	}
	for _, o := range ov.Orders {
		switch o.State {
		case OrderActive:
			g := &HeroGoal{Text: o.Text, Kills: o.Kills, XP: o.XPGained}
			if o.Minutes > 0 && o.StartedAt > 0 {
				gone := (ov.Now - o.StartedAt) / 60
				p := min(100, max(0, float64(int(1000*gone/float64(o.Minutes)))/10))
				left := max(0, int(float64(o.Minutes)-gone+0.5))
				g.Pct, g.LeftMin = &p, &left
			}
			v.Goal = g
		case OrderQueued:
			if len(v.Queue) < 3 {
				v.Queue = append(v.Queue, o.Text)
			}
		}
	}
	v.Why = h.heroWhy(v)
	return v
}

func (h *Hub) heroWhy(v HeroView) HeroWhy {
	cur, params := h.why.Detail()
	w := HeroWhy{Code: cur.Code, Now: cur.Text}
	if w.Now == "" && v.Activity != "" {
		w.Now = v.Activity + "."
	}
	if t := whyChange()[cur.Code]; t != "" {
		ok := true
		for k := range strings.SplitSeq(t, "{") {
			name, _, closed := strings.Cut(k, "}")
			if closed && params[name] == "" {
				ok = false
			}
		}
		if ok {
			for k, p := range params {
				t = strings.ReplaceAll(t, "{"+k+"}", safeText(p, whyParamMax))
			}
			w.Change = t
		}
	}
	switch {
	case len(v.Queue) > 0:
		w.Next = "Next: " + v.Queue[0] + "."
	case v.Goal != nil && v.Goal.LeftMin != nil:
		w.Next = fmt.Sprintf("After this order (about %d min left) the plan takes over again.", *v.Goal.LeftMin)
	case v.Goal == nil:
		w.Next = "No order from you: the hero follows its own plan."
	}
	return w
}

func (h *Hub) heroHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, h.Hero()) }
}

// goldFromInventory reads gold from the inventory desk's view (cmd/d2bot binds an internal/invdesk.Desk, whose view
// carries gold and stashGold).
func goldFromInventory(a *InvAPI) GoldSource {
	return func() (int, int, bool) {
		d := a.desk()
		if d == nil {
			return 0, 0, false
		}
		b, err := json.Marshal(d.View())
		if err != nil {
			return 0, 0, false
		}
		var g struct {
			Gold      *int `json:"gold"`
			StashGold int  `json:"stashGold"`
		}
		if json.Unmarshal(b, &g) != nil || g.Gold == nil {
			return 0, 0, false
		}
		return *g.Gold, g.StashGold, true
	}
}
