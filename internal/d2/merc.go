package d2

// MercClasses are the monstats classes of the four hireling units: Rogue Scout 271, Desert Mercenary 338, Iron Wolf 359,
// Barbarian 561. V: Class column of the real 1.14d hireling.txt (docs/merc-farm.md 1.1); live only the Rogue (271) was
// seen hired (docs/friendly.md), 338/359/561 are U. Class 560 appears in no hireling row and is deliberately absent.
var MercClasses = [...]int{271, 338, 359, 561}

// IsMercClass reports whether c is a hireling unit class.
func IsMercClass(c int) bool {
	for _, k := range MercClasses {
		if k == c {
			return true
		}
	}
	return false
}

// Merc is the player's hired mercenary as read from the monster list.
//
// The unit owner words read 0/0 for the hired Rogue (V docs/friendly.md), so the merc is found by class, not by owner.
// In offline single player no other hireling exists next to the player (U: a hostile unit sharing one of these classes
// would be taken for a merc; none is known), and the nearest unit wins when several are listed.
type Merc struct {
	Present bool // a merc-class unit is in the unit table
	Alive   bool // Present and not dying or dead (mode check; U that a dead merc stays listed, it may vanish: then Present is false)
	Class   int
	ID      uint32
	X, Y    int
	// LifePct is stat 6 / stat 7 * 100 (0 when absent). V only for town NPCs; U for a damaged merc (docs/merc-farm.md 1.4).
	LifePct int
	// Level is stat 12 of the unit (U: never read on a live merc; 0 when unknown).
	Level int
	// Name is the unit's own name when it carries one (unique-style names); the hire window names ("Pratham") are
	// not read by any known memory path (U).
	Name string
	// Aura is not readable from memory: hire window text only (V docs/town.md). Always empty; callers that hired a
	// merc keep the wanted aura themselves.
	Aura string
}

// FindMerc picks the merc out of a snapshot's monster list. The zero Merc means none listed.
func FindMerc(s *Snapshot) Merc {
	if s == nil {
		return Merc{}
	}
	var best *Unit
	bestD := -1
	for i := range s.Monsters {
		u := &s.Monsters[i]
		if !IsMercClass(u.Class) {
			continue
		}
		d := 0
		if s.Player != nil {
			d = max(abs(u.X-s.Player.X), abs(u.Y-s.Player.Y))
		}
		// a living unit beats a corpse, then the nearer one
		if best == nil || (u.Alive() && !best.Alive()) || (u.Alive() == best.Alive() && d < bestD) {
			best, bestD = u, d
		}
	}
	if best == nil {
		return Merc{}
	}
	m := Merc{Present: true, Alive: best.Alive(), Class: best.Class, ID: best.ID, X: best.X, Y: best.Y}
	if best.Monster != nil {
		m.LifePct, m.Name, m.Level = best.Monster.LifePct, best.Monster.Name, best.Monster.Level
	}
	if !m.Alive {
		m.LifePct = 0
	}
	return m
}
