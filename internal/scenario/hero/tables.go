package hero

import (
	"strings"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/points"
)

// DBTables answers Tables from the game's tables.
func DBTables(db *data.DB) Tables { return dbTables{db} }

type dbTables struct{ db *data.DB }

func (t dbTables) SkillID(name string) (int, bool) {
	s := t.db.SkillByName(name)
	if s == nil {
		return 0, false
	}
	return s.ID, true
}

func (t dbTables) SkillIndex(class string, id int) (int, bool) {
	pt := points.Tables{Skills: t.db.Skills, Classes: t.db.Classes}
	c := pt.Class(class)
	if c == nil {
		return 0, false
	}
	return points.ClassSkillIndex(pt, c.Code, id)
}

func (t dbTables) Rule(class string) leveler.Rule {
	return leveler.DBRule(t.db, leveler.ClassCode(t.db, class))
}

// StarterWeapon is the first charstats starting item worn in the right arm (U: the location column value "rarm" is
// the 1.14d charstats.txt one; not read back from a live table here).
func (t dbTables) StarterWeapon(class string) string {
	c := t.db.ClassByName(class)
	if c == nil {
		return ""
	}
	for _, it := range c.Items {
		if strings.EqualFold(it.Location, "rarm") {
			return it.Code
		}
	}
	return ""
}
