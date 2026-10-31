package town

import (
	"d2lod-bot/internal/data"
)

// Unit classes. NPC classes are monstats hcIdx values; tests in acts_test.go check them against the real
// tables when /tmp/d2mpq exists. Act 1 NPC and object classes are VERIFIED LIVE (docs/map.md);
// the other acts come from the game data and Kolbot constants and are UNVERIFIED in the running client.
const (
	ObjStash    = 267 // "bank", VERIFIED LIVE in the Rogue Encampment (positions come from presets at runtime, never from constants)
	ObjPortal   = 59  // town portal object (objects.txt "Portal", subclass 4); UNVERIFIED live
	ObjWPAct1   = 119 // VERIFIED LIVE
	ObjWPAct2   = 156
	ObjWPAct3   = 237
	ObjWPAct4   = 398
	ObjWPAct5   = 429
	TypeMonster = 1
	TypeObject  = 2
)

// Act describes one town: who does what and where to stand when the unit table does not list the unit yet.
type Act struct {
	No        int   // 1..5
	TownLevel int   // levels.txt id of the town
	Heal      []int // NPC classes that heal when spoken to
	Shop      []int // sell potions, scrolls, tomes (first entry preferred)
	Repair    []int
	Gamble    []int
	Merc      []int
	Keys      []int
	Cain      []int
	Sell      []int // vendors tried in order when selling junk
	Waypoint  int   // object class
}

var cain = []int{146, 244, 245, 246, 265, 520}

// Acts is the table, index = act-1.
var Acts = [5]Act{
	{No: 1, TownLevel: 1, Heal: []int{148}, Shop: []int{148}, Repair: []int{154}, Gamble: []int{147}, Merc: []int{150},
		Keys: []int{148}, Cain: cain, Sell: []int{154, 147, 148}, Waypoint: ObjWPAct1},
	{No: 2, TownLevel: 40, Heal: []int{178, 176}, Shop: []int{177}, Repair: []int{178}, Gamble: []int{199}, Merc: []int{198},
		Keys: []int{202}, Cain: cain, Sell: []int{178, 199, 177}, Waypoint: ObjWPAct2},
	{No: 3, TownLevel: 75, Heal: []int{255}, Shop: []int{255}, Repair: []int{253}, Gamble: []int{254}, Merc: []int{252},
		Keys: []int{253}, Cain: cain, Sell: []int{253, 254, 255}, Waypoint: ObjWPAct3},
	{No: 4, TownLevel: 103, Heal: []int{405}, Shop: []int{405}, Repair: []int{257}, Gamble: []int{405}, Merc: nil,
		Keys: []int{405}, Cain: cain, Sell: []int{257, 405}, Waypoint: ObjWPAct4},
	{No: 5, TownLevel: 109, Heal: []int{513}, Shop: []int{513}, Repair: []int{511}, Gamble: []int{512}, Merc: []int{515},
		Keys: []int{513}, Cain: cain, Sell: []int{511, 512, 513}, Waypoint: ObjWPAct5},
}

// ActOfLevel returns the act table for a town level id (1, 40, 75, 103, 109), or nil.
func ActOfLevel(level int) *Act {
	for i := range Acts {
		if Acts[i].TownLevel == level {
			return &Acts[i]
		}
	}
	return nil
}

// ActByNo returns the table for act 1..5.
func ActByNo(no int) *Act {
	if no < 1 || no > 5 {
		return nil
	}
	return &Acts[no-1]
}

// hasClass reports whether c is in the list.
func hasClass(list []int, c int) bool {
	for _, x := range list {
		if x == c {
			return true
		}
	}
	return false
}

// IsTown reports whether the table's level id is a town in the data package's list.
func (a Act) IsTown() bool { return data.IsTown(a.TownLevel) }
