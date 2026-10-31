package d2dsrc

import (
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Tables resolves d2d's string identifiers to the numeric ids our code uses. All methods answer ok=false
// for an unknown name; the adapter then reports the field unknown instead of inventing an id. A nil Tables
// is allowed: every class and item row is then unknown.
type Tables interface {
	// ItemRow returns the index into data.DB.Items (the unit's dwTxtFileNo) for a 3-4 letter item code.
	ItemRow(code string) (int, bool)
	// MonsterClass resolves a monstats key ("fallen1") or display name ("Kashya") to the class id.
	MonsterClass(name string) (int, bool)
}

// DBTables adapts the extracted game tables.
type DBTables struct {
	db   *data.DB
	rows map[string]int
}

// NewDBTables indexes db.Items by code (first row wins, like db.ItemByCode).
func NewDBTables(db *data.DB) *DBTables {
	t := &DBTables{db: db, rows: map[string]int{}}
	for i, it := range db.Items {
		if it == nil {
			continue
		}
		if _, dup := t.rows[it.Code]; !dup {
			t.rows[it.Code] = i
		}
	}
	return t
}

func (t *DBTables) ItemRow(code string) (int, bool) { n, ok := t.rows[code]; return n, ok }

func (t *DBTables) MonsterClass(name string) (int, bool) {
	if m := t.db.MonsterByName(name); m != nil {
		return m.ID, true
	}
	return 0, false
}

// Monster mode codes (MonMode.txt order, S: standard 1.14d tables; d2d's own code strings are the same two
// letter codes, V as strings seen in the spike: NU A1 GH DT DD WL). Numeric ids equal d2.MonModeDeath=0,
// MonModeStand=1, MonModeDead=12 (our code's existing constants). Any code not listed is "unknown mode" (-1).
var monModes = map[string]int{
	"DT": 0, "NU": 1, "WL": 2, "GH": 3, "A1": 4, "A2": 5, "BL": 6, "SC": 7,
	"S1": 8, "S2": 9, "S3": 10, "S4": 11, "DD": 12, "KB": 13, "SQ": 14, "RN": 15,
}

// Player mode codes (PlrMode.txt order, S). d2d's player mode strings come from its own table (kModeCode,
// order U): codes are matched by name, not by d2d's numbers.
var plrModes = map[string]int{
	"DT": 0, "NU": 1, "WL": 2, "RN": 3, "GH": 4, "TN": 5, "TW": 6, "A1": 7, "A2": 8, "BL": 9,
	"SC": 10, "TH": 11, "KK": 12, "S1": 13, "S2": 14, "S3": 15, "S4": 16, "DD": 17, "SQ": 18, "KB": 19,
}

// classSkillBase is the first skill id of each class (S: Skills.txt, d2.Class order). The dump lists
// base levels in the class's Skills.txt order (30 entries, U: ascending id order within the class).
var classSkillBase = [...]int{6, 36, 66, 96, 126, 221, 251}

var classByName = map[string]d2.Class{
	"amazon": d2.Amazon, "sorceress": d2.Sorceress, "necromancer": d2.Necromancer, "paladin": d2.Paladin,
	"barbarian": d2.Barbarian, "druid": d2.Druid, "assassin": d2.Assassin,
}

func classOf(name string) (d2.Class, bool) { c, ok := classByName[strings.ToLower(name)]; return c, ok }

// ActOf returns the 0-based act of a Levels.txt id (S: act boundaries by town ids 1, 40, 75, 103, 109).
func ActOf(area int) int {
	switch {
	case area >= 109:
		return 4
	case area >= 103:
		return 3
	case area >= 75:
		return 2
	case area >= 40:
		return 1
	}
	return 0
}

// mercClass maps d2d's merc code (the "state merc=" suffix and dump merc.code) to a monstats class. V only
// RG (Rogue, V docs/d2d-probes.md section 1); the other codes are not known, so they stay unknown.
var mercClass = map[string]int{"RG": d2.MercClasses[0]}
