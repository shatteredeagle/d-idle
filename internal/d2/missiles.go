package d2

import "strings"

// Missile units (type 3) are already read by Reader.Units from the client table (rvaClientUnits, D2BS Unit.cpp);
// Class is the missiles.txt id and OwnerID/OwnerType the shooter. That table was UNVERIFIED live when written (no
// missile in flight while recording). Everything here is isolated so an unknown answer is always safe: no missile,
// no dodge.

// MissileUnits reads the missiles in flight. ok is false when the table could not be read (torn read, bad pointer),
// in which case callers must act as if there were none.
func (r *Reader) MissileUnits() (ms []Unit, ok bool) {
	ms, err := r.Units(TypeMissile)
	if err != nil {
		return nil, false
	}
	return ms, true
}

// MissileKind groups the missiles worth sidestepping.
type MissileKind int

const (
	MissileOther MissileKind = iota
	MissileFireWall
	MissileLightning
	MissileHydra
	MissileChaos
)

func (k MissileKind) String() string {
	return [...]string{"other", "firewall", "lightning", "hydra", "chaos"}[k]
}

// MissileKindByName classifies a missiles.txt "Missile" name. The substrings are UNVERIFIED guesses at the 1.14d
// names (firewall, lightning bolt/nova/chain, hydra missiles, chaos orb); the live check lists the names
// seen in flight. Unknown names are MissileOther.
func MissileKindByName(name string) MissileKind {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "firewall"):
		return MissileFireWall
	case strings.Contains(n, "hydra"):
		return MissileHydra
	case strings.Contains(n, "chaos"):
		return MissileChaos
	case strings.Contains(n, "lightning"):
		return MissileLightning
	}
	return MissileOther
}

// CurseIDs are the states.txt ids of the curses that matter to a player. The defaults are S (memory of the stock
// 1.14d states.txt, only poison 2 and frozen armor 10 are V live); resolve them by name from internal/data when the
// tables are loaded (ResolveCurseIDs), which makes them V against the real file.
type CurseIDs struct {
	Amplify, Decrepify, LowerResist, Weaken, IronMaiden int
}

// DefaultCurseIDs: S, stock states.txt order. -1 would mean unknown; these are the values to fall back on.
var DefaultCurseIDs = CurseIDs{Amplify: 9, Weaken: 19, IronMaiden: 55, Decrepify: 56, LowerResist: 61}

// ResolveCurseIDs looks the states up by their states.txt names (amplifydam, decrepify, lowerresist, weaken,
// ironmaiden). A name the lookup does not know keeps its default.
func ResolveCurseIDs(byName func(string) (int, bool)) CurseIDs {
	ids := DefaultCurseIDs
	for ptr, name := range map[*int]string{&ids.Amplify: "amplifydam", &ids.Decrepify: "decrepify",
		&ids.LowerResist: "lowerresist", &ids.Weaken: "weaken", &ids.IronMaiden: "ironmaiden"} {
		if id, ok := byName(name); ok {
			*ptr = id
		}
	}
	return ids
}

// Curses is the set of enemy curses on the player.
type Curses struct {
	Amplify, Decrepify, LowerResist, Weaken, IronMaiden bool
}

// Any reports whether any curse is on.
func (c Curses) Any() bool {
	return c.Amplify || c.Decrepify || c.LowerResist || c.Weaken || c.IronMaiden
}

// Curses reads the curse states of the player (empty when the state array was unreadable: U, see slStateFlags).
func (p *Player) Curses(ids CurseIDs) Curses {
	if p == nil {
		return Curses{}
	}
	return Curses{
		Amplify: p.States.Has(ids.Amplify), Decrepify: p.States.Has(ids.Decrepify),
		LowerResist: p.States.Has(ids.LowerResist), Weaken: p.States.Has(ids.Weaken), IronMaiden: p.States.Has(ids.IronMaiden),
	}
}
