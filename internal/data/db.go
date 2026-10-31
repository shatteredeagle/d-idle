package data

import (
	"fmt"
	"strings"
)

// DB holds every table the bot uses, parsed and indexed. Build one with
// Open (extract-if-stale then load), Load (from a cache directory) or
// LoadInstall (straight from the MPQs, nothing written).
//
// All slices and the structs they point at are shared: treat them as
// read-only. A DB is safe for concurrent readers.
type DB struct {
	Monsters     []*Monster
	SuperUniques []*SuperUnique
	Levels       []*Level
	Skills       []*Skill
	Classes      []*CharClass
	Exp          *Experience
	ItemTypes    []*ItemType
	Items        []*ItemBase // weapons, armor, misc in file order (Kind tells which)
	Objects      []*Object
	Inventories  []*Inventory
	Belts        []*Belt
	Treasure     []*TreasureClass
	Uniques      []*UniqueItem
	SetItems     []*SetItem
	Runewords    []*Runeword
	Cube         []*CubeRecipe
	MonLevels    []*MonLevel      // by monster level (index = level)
	Difficulties []DifficultyInfo // Normal, Nightmare, Hell
	Hirelings    []*Hireling
	StatCosts    []*StatCost // itemstatcost.txt
	Properties   []*Property
	Prefixes     []*Affix
	Suffixes     []*Affix
	States       []*State
	Missiles     []*Missile
	Warps        []*LevelWarp // lvlwarp.txt

	strs map[string]string

	monByID, monByKey, monByName map[string]*Monster
	monByClass                   map[int]*Monster
	superByID                    map[int]*SuperUnique
	superByKey                   map[string]*SuperUnique
	levelByID                    map[int]*Level
	levelByName                  map[string]*Level
	skillByID                    map[int]*Skill
	skillByKey                   map[string]*Skill
	itemByCode                   map[string]*ItemBase
	typeByCode                   map[string]*ItemType
	objByID                      map[int]*Object
	invByClass                   map[string]*Inventory
	beltByName                   map[string]*Belt
	tcByName                     map[string]*TreasureClass
	uniqByName                   map[string]*UniqueItem
	setByName                    map[string]*SetItem
	rwByName                     map[string]*Runeword
	statByID                     map[int]*StatCost
	statByName                   map[string]*StatCost
	propByCode                   map[string]*Property
	stateByID                    map[int]*State
	stateByName                  map[string]*State
	missileByID                  map[int]*Missile
	missileByName                map[string]*Missile
	warpByID                     map[int]*LevelWarp
}

// sources are the trimmed tables by name plus the merged string table.
type sources struct {
	tables map[string][]byte
	strs   map[string]string
}

func (s *sources) table(name string) *Table {
	if b, ok := s.tables[name]; ok {
		return ParseTable(b)
	}
	return nil
}

func buildDB(s *sources) (*DB, error) {
	db := &DB{strs: s.strs}
	need := func(name string) (*Table, error) {
		t := s.table(name)
		if t == nil {
			return nil, fmt.Errorf("data: table %s missing", name)
		}
		return t, nil
	}
	var t *Table
	var err error
	if t, err = need("monstats"); err != nil {
		return nil, err
	}
	db.Monsters = loadMonsters(t)
	if t, err = need("levels"); err != nil {
		return nil, err
	}
	db.Levels = loadLevels(t)
	if t, err = need("skills"); err != nil {
		return nil, err
	}
	db.Skills = loadSkills(t, s.table("skilldesc"))
	if t, err = need("charstats"); err != nil {
		return nil, err
	}
	db.Classes = loadCharClasses(t)
	if t, err = need("experience"); err != nil {
		return nil, err
	}
	db.Exp = loadExperience(t)
	if t, err = need("itemtypes"); err != nil {
		return nil, err
	}
	db.ItemTypes = loadItemTypes(t)
	for i, n := range []string{"weapons", "armor", "misc"} {
		if t, err = need(n); err != nil {
			return nil, err
		}
		db.Items = append(db.Items, loadItemBases(t, ItemKind(i))...)
	}
	// Optional tables: a missing one leaves the slice empty.
	if t = s.table("objects"); t != nil {
		db.Objects = loadObjects(t)
	}
	if t = s.table("superuniques"); t != nil {
		db.SuperUniques = loadSuperUniques(t)
	}
	if t = s.table("inventory"); t != nil {
		db.Inventories = loadInventory(t)
	}
	if t = s.table("belts"); t != nil {
		db.Belts = loadBelts(t)
	}
	if t = s.table("treasureclassex"); t != nil {
		db.Treasure = loadTreasure(t)
	}
	if t = s.table("uniqueitems"); t != nil {
		db.Uniques = loadUniques(t)
	}
	if t = s.table("setitems"); t != nil {
		db.SetItems = loadSetItems(t)
	}
	if t = s.table("runes"); t != nil {
		db.Runewords = loadRunewords(t)
	}
	if t = s.table("cubemain"); t != nil {
		db.Cube = loadCube(t)
	}
	if t = s.table("monlvl"); t != nil {
		db.MonLevels = loadMonLevels(t)
	}
	if t = s.table("difficultylevels"); t != nil {
		db.Difficulties = loadDifficulties(t)
	}
	if t = s.table("hireling"); t != nil {
		db.Hirelings = loadHirelings(t)
	}
	if t = s.table("itemstatcost"); t != nil {
		db.StatCosts = loadStatCosts(t)
	}
	if t = s.table("properties"); t != nil {
		db.Properties = loadProperties(t)
	}
	if t = s.table("magicprefix"); t != nil {
		db.Prefixes = loadAffixes(t, false)
	}
	if t = s.table("magicsuffix"); t != nil {
		db.Suffixes = loadAffixes(t, true)
	}
	if t = s.table("states"); t != nil {
		db.States = loadStates(t)
	}
	if t = s.table("missiles"); t != nil {
		db.Missiles = loadMissiles(t)
	}
	if t = s.table("lvlwarp"); t != nil {
		db.Warps = loadWarps(t)
	}
	db.index()
	db.indexExtra()
	return db, nil
}

func lc(s string) string { return strings.ToLower(s) }

func (db *DB) index() {
	db.monByClass = map[int]*Monster{}
	db.monByKey = map[string]*Monster{}
	db.monByName = map[string]*Monster{}
	for _, m := range db.Monsters {
		m.Name = db.nameOf(m.NameKey, m.Key)
		db.monByClass[m.ID] = m
		db.monByKey[lc(m.Key)] = m
		if _, dup := db.monByName[lc(m.Name)]; !dup {
			db.monByName[lc(m.Name)] = m
		}
	}
	db.superByID = map[int]*SuperUnique{}
	db.superByKey = map[string]*SuperUnique{}
	for _, u := range db.SuperUniques {
		u.Name = db.nameOf(u.NameKey, u.Key)
		db.superByID[u.ID] = u
		db.superByKey[lc(u.Key)] = u
	}
	db.levelByID = map[int]*Level{}
	db.levelByName = map[string]*Level{}
	for _, l := range db.Levels {
		db.levelByID[l.ID] = l
		db.levelByName[lc(l.Name)] = l
		db.levelByName[lc(l.Key)] = l
	}
	db.skillByID = map[int]*Skill{}
	db.skillByKey = map[string]*Skill{}
	for _, s := range db.Skills {
		s.Name = db.nameOf(s.Name, s.Key)
		db.skillByID[s.ID] = s
		db.skillByKey[lc(s.Key)] = s
	}
	db.itemByCode = map[string]*ItemBase{}
	for _, it := range db.Items {
		it.Display = db.nameOf(it.NameKey, it.Name)
		if _, dup := db.itemByCode[it.Code]; !dup {
			db.itemByCode[it.Code] = it
		}
	}
	db.typeByCode = map[string]*ItemType{}
	for _, t := range db.ItemTypes {
		if _, dup := db.typeByCode[t.Code]; !dup {
			db.typeByCode[t.Code] = t
		}
	}
	db.objByID = map[int]*Object{}
	for _, o := range db.Objects {
		db.objByID[o.ID] = o
	}
	db.invByClass = map[string]*Inventory{}
	for _, v := range db.Inventories {
		db.invByClass[lc(v.Class)] = v
	}
	db.beltByName = map[string]*Belt{}
	for _, b := range db.Belts {
		db.beltByName[b.Name] = b
	}
	db.tcByName = map[string]*TreasureClass{}
	for _, tc := range db.Treasure {
		db.tcByName[lc(tc.Name)] = tc
	}
	db.uniqByName = map[string]*UniqueItem{}
	for _, u := range db.Uniques {
		u.Name = db.nameOf(u.Index, u.Index)
		db.uniqByName[lc(u.Name)] = u
	}
	db.setByName = map[string]*SetItem{}
	for _, s := range db.SetItems {
		s.Name = db.nameOf(s.Index, s.Index)
		db.setByName[lc(s.Name)] = s
	}
	db.rwByName = map[string]*Runeword{}
	for _, w := range db.Runewords {
		w.Name = db.nameOf(w.NameKey, w.NameKey)
		db.rwByName[lc(w.Name)] = w
	}
}

// nameOf resolves a string-table key to display text, falling back to
// fallback when the key is empty or unknown.
func (db *DB) nameOf(key, fallback string) string {
	if key != "" {
		if v, ok := db.strs[key]; ok {
			return CleanString(v)
		}
	}
	return fallback
}

// Str returns the raw string-table text for key (colour codes intact).
func (db *DB) Str(key string) (string, bool) { v, ok := db.strs[key]; return v, ok }

// Text returns the display text for key with colour codes removed, or key
// itself when unknown.
func (db *DB) Text(key string) string { return db.nameOf(key, key) }

// MonsterByClassID looks a monster up by class id (the unit's txtFileNo).
func (db *DB) MonsterByClassID(id int) *Monster { return db.monByClass[id] }

// MonsterByName looks up by internal key ("skeleton1") or display name
// ("Skeleton"), case-insensitively; the first class with a display name wins.
func (db *DB) MonsterByName(name string) *Monster {
	if m := db.monByKey[lc(name)]; m != nil {
		return m
	}
	return db.monByName[lc(name)]
}

// SuperUniqueByID is by superuniques hcIdx (the value the game stores for a
// super unique's uniqueno).
func (db *DB) SuperUniqueByID(id int) *SuperUnique { return db.superByID[id] }

// SuperUniqueByName accepts the key ("Bishibosh") or display name.
func (db *DB) SuperUniqueByName(name string) *SuperUnique {
	if u := db.superByKey[lc(name)]; u != nil {
		return u
	}
	for _, u := range db.SuperUniques {
		if strings.EqualFold(u.Name, name) {
			return u
		}
	}
	return nil
}

// AreaByID returns the level (area) with the given id.
func (db *DB) AreaByID(id int) *Level { return db.levelByID[id] }

// AreaByName accepts the display name ("Blood Moor") or the key
// ("Act 1 - Wilderness 1").
func (db *DB) AreaByName(name string) *Level { return db.levelByName[lc(name)] }

// SkillByID returns the skill with the given id.
func (db *DB) SkillByID(id int) *Skill { return db.skillByID[id] }

// SkillByName accepts the key ("Fire Bolt") or display name.
func (db *DB) SkillByName(name string) *Skill { return db.skillByKey[lc(name)] }

// ExpToLevel is the total experience needed to reach level (1..99).
func (db *DB) ExpToLevel(level int) int64 { return db.Exp.ToReach(level) }

// ClassByName returns the player class ("Sorceress"), case-insensitive.
func (db *DB) ClassByName(name string) *CharClass {
	for _, c := range db.Classes {
		if strings.EqualFold(c.Name, name) {
			return c
		}
	}
	return nil
}

// ItemByCode returns the base item for a 3-4 character item code ("hax", "r01").
func (db *DB) ItemByCode(code string) *ItemBase { return db.itemByCode[code] }

// ItemType returns the item type for a type code ("swor").
func (db *DB) ItemType(code string) *ItemType { return db.typeByCode[code] }

// IsA reports whether item type typ is, or inherits through Equiv1/Equiv2
// from, ancestor (e.g. IsA("swor","weap") is true).
func (db *DB) IsA(typ, ancestor string) bool {
	seen := map[string]bool{}
	var walk func(string) bool
	walk = func(c string) bool {
		if c == "" || seen[c] {
			return false
		}
		if c == ancestor {
			return true
		}
		seen[c] = true
		t := db.typeByCode[c]
		if t == nil {
			return false
		}
		return walk(t.Equiv[0]) || walk(t.Equiv[1])
	}
	return walk(typ)
}

// ObjectByID returns the objects.txt row.
func (db *DB) ObjectByID(id int) *Object { return db.objByID[id] }

// InventoryLayout returns a panel layout by inventory.txt class name, e.g.
// "Sorceress", "Bank Page 1", "Transmogrify Box Page 1", "Hireling".
func (db *DB) InventoryLayout(class string) *Inventory { return db.invByClass[lc(class)] }

// BeltLayout returns a belt type's slot boxes ("belt", "sash", "default").
func (db *DB) BeltLayout(name string) *Belt { return db.beltByName[lc(name)] }

// TreasureClassByName returns a treasure class by name.
func (db *DB) TreasureClassByName(name string) *TreasureClass { return db.tcByName[lc(name)] }

// UniqueByName finds a unique item by display name ("Harlequin Crest").
func (db *DB) UniqueByName(name string) *UniqueItem { return db.uniqByName[lc(name)] }

// SetItemByName finds a set item by display name.
func (db *DB) SetItemByName(name string) *SetItem { return db.setByName[lc(name)] }

// RunewordByName finds a runeword by display name ("Spirit").
func (db *DB) RunewordByName(name string) *Runeword { return db.rwByName[lc(name)] }

// Town returns the town area id of a 0-based act (1, 40, 75, 103, 109).
func Town(act int) int { return [...]int{1, 40, 75, 103, 109}[act] }

// IsTown reports whether area id is one of the five towns.
func IsTown(area int) bool {
	switch area {
	case 1, 40, 75, 103, 109:
		return true
	}
	return false
}

// MonLevelRow returns the monlvl.txt scaling row for a monster level, or nil.
func (db *DB) MonLevelRow(level int) *MonLevel {
	if level < 0 || level >= len(db.MonLevels) {
		return nil
	}
	return db.MonLevels[level]
}

// ScaledHP estimates a monster's maximum life at difficulty d and monster
// level mlvl: monstats MaxHP times the monlvl HP percentage. UNVERIFIED
// against a live monster; bosses and uniques add their own multipliers.
func (db *DB) ScaledHP(m *Monster, d Difficulty, mlvl int) int {
	row := db.MonLevelRow(mlvl)
	if row == nil {
		return m.HPMax[d]
	}
	return m.HPMax[d] * row.HP[d] / 100
}

// BeltByIndex returns belts.txt row i (an armor.txt belt item's "belt"
// column, see ItemBase.BeltIndex; the first set of rows is for the 640x480
// UI, a second set with a "2" suffix for the other).
func (db *DB) BeltByIndex(i int) *Belt {
	if i < 0 || i >= len(db.Belts) {
		return nil
	}
	return db.Belts[i]
}
