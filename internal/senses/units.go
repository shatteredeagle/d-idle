package senses

import "sort"

// Unit kinds of the panel. Colours follow the game's own automap: monsters red, mercenary blue, NPCs green. KindPlayer
// is reserved (other players are not in the monster table, so nothing fills it today).
const (
	KindMonster = "monster"
	KindMerc    = "merc"
	KindNPC     = "npc"
	KindPlayer  = "player"
)

// Elite kinds (docs/senses-units.md). A boss is a superunique that is not a plain pack leader; the reader cannot tell
// the two apart yet (U), so it reports superunique.
const (
	EliteChampion    = "champion"
	EliteUnique      = "unique"
	EliteSuperunique = "superunique"
	EliteBoss        = "boss"
)

// UnitRadius is how far from the hero a unit is published, in subtiles (about the revealed fog; S, a value to tune live).
// Units the client has loaded beyond it, or behind walls the hero never saw, stay out of the panel.
const UnitRadius = 40

// MaxUnits bounds the list (the nearest win).
const MaxUnits = 120

// Unit is the bot's full view of one unit. It never leaves the process as is: the watch side (watch.Senses) cuts it
// down per unlocked tier, so a locked tier's fields are not in the bytes.
type Unit struct {
	ID      uint32   `json:"id"`
	Kind    string   `json:"kind"`
	X       int      `json:"x"`
	Y       int      `json:"y"`
	Class   int      `json:"class"`
	Name    string   `json:"name,omitempty"`    // class display name
	Title   string   `json:"title,omitempty"`   // a superunique's or rare's own name, when the reader has it
	LifePct int      `json:"lifePct,omitempty"` // 1..100; 0 = unknown
	Target  bool     `json:"target,omitempty"`  // the monster the hero is attacking
	Elite   string   `json:"elite,omitempty"`
	Mods    []string `json:"mods,omitempty"` // display names; an unknown id reads "unknown mod #N"
}

// SetUnits replaces the unit list of the current snapshot with a copy of us, nearest to hero first, cut to MaxUnits and
// UnitRadius. Before the first Observe there is no snapshot to attach to and it does nothing.
func (p *Publisher) SetUnits(us []Unit, hero Point) {
	keep := make([]Unit, 0, len(us))
	for _, u := range us {
		if d2 := dist2(u.X, u.Y, hero); d2 <= UnitRadius*UnitRadius {
			u.Mods = append([]string(nil), u.Mods...)
			keep = append(keep, u)
		}
	}
	sort.SliceStable(keep, func(i, j int) bool { return dist2(keep[i].X, keep[i].Y, hero) < dist2(keep[j].X, keep[j].Y, hero) })
	if len(keep) > MaxUnits {
		keep = keep[:MaxUnits]
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.have {
		return
	}
	p.snap.Unit = keep
}

func dist2(x, y int, c Point) int { return (x-c.X)*(x-c.X) + (y-c.Y)*(y-c.Y) }
