package combat

import (
	"strings"

	"d2lod-bot/internal/data"
)

// PotionKind classifies a belt item.
type PotionKind int

const (
	PotionNone PotionKind = iota
	PotionHealing
	PotionMana
	PotionRejuv
	PotionAntidote // "yps": cures poison (boss fights, docs/boss.md)
)

func (k PotionKind) String() string {
	return [...]string{"none", "healing", "mana", "rejuvenation", "antidote"}[k]
}

// ItemClass is what the fight code needs to know about an item base.
type ItemClass struct {
	Potion PotionKind
	Tier   int  // 1 (minor) .. 5 (super); rejuvenation 1 (small) or 2 (full)
	Portal bool // Scroll or Tome of Town Portal
	Tome   bool // the tome (a Portal item whose quantity stat is the scroll count; an empty tome has no stat list)
}

// Catalog is the static game data the fight code consults. *data.DB is adapted by NewCatalog; tests supply maps.
type Catalog interface {
	Monster(class int) *data.Monster
	MonsterName(m *data.Monster) string
	Skill(id int) *data.Skill
	SkillByName(name string) *data.Skill
	Item(class int) ItemClass
}

// NewCatalog adapts a data.DB. A nil DB gives an empty catalog (every monster unknown, no skills).
func NewCatalog(db *data.DB) Catalog { return dbCatalog{db} }

type dbCatalog struct{ db *data.DB }

func (c dbCatalog) Monster(class int) *data.Monster {
	if c.db == nil {
		return nil
	}
	return c.db.MonsterByClassID(class)
}

func (c dbCatalog) MonsterName(m *data.Monster) string {
	if m == nil {
		return ""
	}
	if m.Name != "" {
		return m.Name
	}
	if c.db != nil {
		if s := c.db.Text(m.NameKey); s != "" {
			return s
		}
	}
	return m.Key
}

func (c dbCatalog) Skill(id int) *data.Skill {
	if c.db == nil {
		return nil
	}
	return c.db.SkillByID(id)
}

func (c dbCatalog) SkillByName(name string) *data.Skill {
	if c.db == nil {
		return nil
	}
	return c.db.SkillByName(name)
}

// Item maps UnitAny.dwTxtFileNo to a class. VERIFIED against the real tables: db.Items is weapons, armor, misc
// in file order and index 587 is "hp1" (the minor healing potion seen in the live belt), 529 is "tsc".
func (c dbCatalog) Item(class int) ItemClass {
	if c.db == nil || class < 0 || class >= len(c.db.Items) {
		return ItemClass{}
	}
	return classifyCode(c.db.Items[class].Code)
}

func classifyCode(code string) ItemClass {
	switch {
	case len(code) == 3 && strings.HasPrefix(code, "hp") && code[2] >= '1' && code[2] <= '5':
		return ItemClass{Potion: PotionHealing, Tier: int(code[2] - '0')}
	case len(code) == 3 && strings.HasPrefix(code, "mp") && code[2] >= '1' && code[2] <= '5':
		return ItemClass{Potion: PotionMana, Tier: int(code[2] - '0')}
	case code == "rvs":
		return ItemClass{Potion: PotionRejuv, Tier: 1}
	case code == "rvl":
		return ItemClass{Potion: PotionRejuv, Tier: 2}
	case code == "yps":
		return ItemClass{Potion: PotionAntidote, Tier: 1}
	case code == "tsc":
		return ItemClass{Portal: true}
	case code == "tbk":
		return ItemClass{Portal: true, Tome: true}
	}
	return ItemClass{}
}

// MapCatalog is a hand-built Catalog for tests and for running without game data.
type MapCatalog struct {
	Monsters map[int]*data.Monster
	Skills   map[int]*data.Skill
	Items    map[int]ItemClass
}

func (c MapCatalog) Monster(class int) *data.Monster { return c.Monsters[class] }
func (c MapCatalog) MonsterName(m *data.Monster) string {
	if m == nil {
		return ""
	}
	if m.Name != "" {
		return m.Name
	}
	return m.Key
}
func (c MapCatalog) Skill(id int) *data.Skill { return c.Skills[id] }
func (c MapCatalog) SkillByName(name string) *data.Skill {
	for _, s := range c.Skills {
		if strings.EqualFold(s.Key, name) || strings.EqualFold(s.Name, name) {
			return s
		}
	}
	return nil
}
func (c MapCatalog) Item(class int) ItemClass { return c.Items[class] }
