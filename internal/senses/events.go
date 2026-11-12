package senses

import "time"

// Event retention and cap (the history ring of the minimap, docs/senses-map.md "History").
const (
	EventRetention = 30 * time.Minute
	EventCap       = 500
)

// Event kinds.
const (
	EvKill     = "kill"
	EvDamage   = "damage"
	EvPotion   = "potion"
	EvLoot     = "loot"
	EvLevelUp  = "levelup"
	EvDeath    = "death"
	EvFight    = "fight"
	EvChicken  = "chicken"
	EvRoamFail = "roamfail"
	EvGateHold = "gatehold"
	EvIgnore   = "ignore"
	EvFallback = "fallback"
	EvYield    = "yield"
	EvPortal   = "portal"
	EvWaypoint = "waypoint"
)

// FightInfo is the compact fight summary an event may carry (combat.FightSummary, copied by the wiring).
type FightInfo struct {
	End      string  `json:"end,omitempty"`
	Kills    int     `json:"kills"`
	Damage   int     `json:"damage,omitempty"`
	MinLife  int     `json:"min"`
	Life     int     `json:"life"`
	Sec      float64 `json:"sec"`
	Potions  int     `json:"potions,omitempty"`
	Casts    int     `json:"casts,omitempty"`
	Approach int     `json:"approach,omitempty"`
	Left     string  `json:"left,omitempty"`
	Kites    int     `json:"kites,omitempty"`
	Dodges   int     `json:"dodges,omitempty"`
}

// Event is one thing the bot did or suffered, at the hero's position.
type Event struct {
	At    time.Time
	X, Y  int
	Area  int
	Kind  string
	Text  string
	Big   bool // boss or unique kill: drawn larger
	Secs  int  // length of the window the event opens (an ignore), 0 for none
	Fight *FightInfo
}

// Events is the ring of recent events. Not safe for concurrent use; Publisher holds its lock.
type Events struct{ evs []Event }

// Add stores e, drops the expired and the oldest beyond the cap. A fight event also gives its summary to the kills
// that happened during it, so a kill marker can say how the fight went.
func (r *Events) Add(e Event) {
	if e.Kind == EvFight && e.Fight != nil {
		from := e.At.Add(-time.Duration(e.Fight.Sec*float64(time.Second)) - 2*time.Second)
		for i := range r.evs {
			if v := &r.evs[i]; v.Kind == EvKill && v.Fight == nil && v.Area == e.Area && !v.At.Before(from) {
				v.Fight = e.Fight
			}
		}
	}
	r.evs = append(r.evs, e)
	cut := 0
	for cut < len(r.evs) && e.At.Sub(r.evs[cut].At) > EventRetention {
		cut++
	}
	if over := len(r.evs) - cut - EventCap; over > 0 {
		cut += over
	}
	if cut > 0 {
		r.evs = append(r.evs[:0], r.evs[cut:]...)
	}
}

// Area returns the events of one area inside the retention at now, oldest first.
func (r *Events) Area(now time.Time, area int) []Event {
	var out []Event
	for _, e := range r.evs {
		if e.Area == area && now.Sub(e.At) <= EventRetention {
			out = append(out, e)
		}
	}
	return out
}

// SetActivity records what the bot is doing now (an Act* code); later trail samples carry it.
func (p *Publisher) SetActivity(act uint8) {
	p.mu.Lock()
	if act < actCount {
		p.act = act
	}
	p.mu.Unlock()
}

// AddEvent records an event at the hero's position in the current area (x, y of the event are ignored).
func (p *Publisher) AddEvent(kind, text string, big bool, f *FightInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.have {
		return
	}
	if kind == EvKill {
		p.heat.cell(p.areaNo, p.snap.Hero.X, p.snap.Hero.Y).kills++
	}
	p.events.Add(Event{At: p.clock(), X: p.snap.Hero.X, Y: p.snap.Hero.Y, Area: p.areaNo, Kind: kind, Text: text, Big: big, Fight: f})
}

// EventsOf returns the events of the current area.
func (p *Publisher) EventsOf() []Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.events.Area(p.clock(), p.areaNo)
}

// AddEventAt is AddEvent at a given place instead of the hero's (a failed roam target, an ignored monster). Events
// of one kind and place within 5 s are merged so a retrying loop does not fill the ring.
func (p *Publisher) AddEventAt(kind, text string, x, y, secs int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.have {
		return
	}
	now := p.clock()
	if n := len(p.events.evs); n > 0 {
		if l := p.events.evs[n-1]; l.Kind == kind && l.X == x && l.Y == y && now.Sub(l.At) < 5*time.Second {
			return
		}
	}
	p.events.Add(Event{At: now, X: x, Y: y, Area: p.areaNo, Kind: kind, Text: text, Secs: secs})
}
