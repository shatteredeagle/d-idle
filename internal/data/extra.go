package data

import "strings"

// Hireling is one hireling.txt row: a mercenary type for one act, difficulty
// and starting level, with per-level growth. Act is 1-based and Difficulty
// 1-based as in the file (V by reading the real table: Normal rows carry 1).
type Hireling struct {
	Name                                                                           string // Hireling column, e.g. "Rogue Scout"
	SubType                                                                        string // e.g. "Fire - Normal"
	ID                                                                             int    // Id column
	Class                                                                          int    // monstats hcIdx of the merc unit
	Act                                                                            int
	Difficulty                                                                     int
	Level                                                                          int // character level this row applies from
	Seller                                                                         int // hcIdx of the NPC that hires it out
	Gold                                                                           int // hire cost factor
	ExpPerLvl                                                                      int
	HP, HPPerLvl, Defense, DefPerLvl, Str, StrPerLvl, Dex, DexPerLvl, AR, ARPerLvl int
	DmgMin, DmgMax, DmgPerLvl                                                      int
	Resist, ResistPerLvl                                                           int
	WType                                                                          []string
	HireDesc                                                                       string
	DefaultChance                                                                  int
	Skills                                                                         []HireSkill
}

// HireSkill is one Skill<n> block of a hireling row.
type HireSkill struct {
	Name         string // skills.txt Skill key, e.g. "Defiance"
	Mode         int
	Chance       int
	ChancePerLvl int
	Level        int
	LvlPerLvl    int
}

var hirelingCols = append([]string{
	"Hireling", "SubType", "Id", "Class", "Act", "Difficulty", "Level", "Seller", "Gold", "Exp/Lvl", "HP", "HP/Lvl",
	"Defense", "Def/Lvl", "Str", "Str/Lvl", "Dex", "Dex/Lvl", "AR", "AR/Lvl", "Dmg-Min", "Dmg-Max", "Dmg/Lvl",
	"Resist", "Resist/Lvl", "WType1", "WType2", "HireDesc", "DefaultChance",
}, hireSkillCols()...)

func hireSkillCols() []string {
	var c []string
	for k := 1; k <= 6; k++ {
		s := itoa(k)
		c = append(c, "Skill"+s, "Mode"+s, "Chance"+s, "ChancePerLvl"+s, "Level"+s, "LvlPerLvl"+s)
	}
	return c
}

func loadHirelings(t *Table) []*Hireling {
	var out []*Hireling
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("Hireling") == "" || r.Str("Hireling") == "Expansion" {
			continue
		}
		h := &Hireling{
			Name: r.Str("Hireling"), SubType: r.Str("SubType"), ID: r.Int("Id"), Class: r.Int("Class"), Act: r.Int("Act"),
			Difficulty: r.Int("Difficulty"), Level: r.Int("Level"), Seller: r.Int("Seller"), Gold: r.Int("Gold"),
			ExpPerLvl: r.Int("Exp/Lvl"), HP: r.Int("HP"), HPPerLvl: r.Int("HP/Lvl"), Defense: r.Int("Defense"), DefPerLvl: r.Int("Def/Lvl"),
			Str: r.Int("Str"), StrPerLvl: r.Int("Str/Lvl"), Dex: r.Int("Dex"), DexPerLvl: r.Int("Dex/Lvl"), AR: r.Int("AR"), ARPerLvl: r.Int("AR/Lvl"),
			DmgMin: r.Int("Dmg-Min"), DmgMax: r.Int("Dmg-Max"), DmgPerLvl: r.Int("Dmg/Lvl"),
			Resist: r.Int("Resist"), ResistPerLvl: r.Int("Resist/Lvl"), HireDesc: r.Str("HireDesc"), DefaultChance: r.Int("DefaultChance"),
		}
		for _, w := range []string{"WType1", "WType2"} {
			if s := r.Str(w); s != "" {
				h.WType = append(h.WType, s)
			}
		}
		for k := 1; k <= 6; k++ {
			s := itoa(k)
			if n := r.Str("Skill" + s); n != "" {
				h.Skills = append(h.Skills, HireSkill{n, r.Int("Mode" + s), r.Int("Chance" + s), r.Int("ChancePerLvl" + s), r.Int("Level" + s), r.Int("LvlPerLvl" + s)})
			}
		}
		out = append(out, h)
	}
	return out
}

// Aura returns the name of the hireling's aura skill (the first skill that
// skills.txt marks as an aura), or "" when it has none.
func (h *Hireling) Aura(db *DB) string {
	for _, s := range h.Skills {
		if sk := db.SkillByName(s.Name); sk != nil && sk.Aura {
			return s.Name
		}
	}
	return ""
}

// StatCost is an itemstatcost.txt row: how a stat is stored in memory, in
// the .d2s save and what it displays as.
type StatCost struct {
	Name         string
	ID           int
	Signed       bool
	SendBits     int
	SendParam    int
	Saved        bool
	Add          int
	Multiply     int
	Divide       int
	ValShift     int // in-memory value is the displayed value << ValShift (life, mana, stamina: 8)
	SaveBits     int // width in the .d2s item/char stat list
	SaveAdd      int // added to the stored value to get the real value
	SaveParam    int
	CSvSigned    bool
	CSvBits      int // width in the character stat block of a .d2s
	CSvParam     int
	Op, OpParam  int
	OpBase       string
	OpStats      []string
	DescPriority int
	DescFunc     int
	DescVal      int
	DescStrPos   string
	DescStrNeg   string
	DescStr2     string
}

var statCostCols = []string{
	"Stat", "ID", "Signed", "Send Bits", "Send Param Bits", "Saved", "CSvSigned", "CSvBits", "CSvParam", "Add", "Multiply", "Divide",
	"ValShift", "Save Bits", "Save Add", "Save Param Bits", "op", "op param", "op base", "op stat1", "op stat2", "op stat3",
	"descpriority", "descfunc", "descval", "descstrpos", "descstrneg", "descstr2",
}

func loadStatCosts(t *Table) []*StatCost {
	var out []*StatCost
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("Stat") == "" || r.Str("Stat") == "Expansion" {
			continue
		}
		s := &StatCost{
			Name: r.Str("Stat"), ID: r.Int("ID"), Signed: r.Bool("Signed"), SendBits: r.Int("Send Bits"), SendParam: r.Int("Send Param Bits"),
			Saved: r.Bool("Saved"), Add: r.Int("Add"), Multiply: r.Int("Multiply"), Divide: r.Int("Divide"), ValShift: r.Int("ValShift"),
			SaveBits: r.Int("Save Bits"), SaveAdd: r.Int("Save Add"), SaveParam: r.Int("Save Param Bits"),
			CSvSigned: r.Bool("CSvSigned"), CSvBits: r.Int("CSvBits"), CSvParam: r.Int("CSvParam"),
			Op: r.Int("op"), OpParam: r.Int("op param"), OpBase: r.Str("op base"),
			DescPriority: r.Int("descpriority"), DescFunc: r.Int("descfunc"), DescVal: r.Int("descval"),
			DescStrPos: r.Str("descstrpos"), DescStrNeg: r.Str("descstrneg"), DescStr2: r.Str("descstr2"),
		}
		for k := 1; k <= 3; k++ {
			if v := r.Str("op stat" + itoa(k)); v != "" {
				s.OpStats = append(s.OpStats, v)
			}
		}
		out = append(out, s)
	}
	return out
}

// FromMemory converts an in-memory stat value to its displayed value
// (ValShift undone: life/mana/stamina are stored 256 times larger).
func (s *StatCost) FromMemory(v int) int { return v >> uint(s.ValShift) }

// ToMemory is the inverse of FromMemory.
func (s *StatCost) ToMemory(v int) int { return v << uint(s.ValShift) }

// Property is a properties.txt row: how an item property code (as used by
// uniques, runewords and affixes) maps to stats. Up to seven stat slots.
type Property struct {
	Code  string
	Slots []PropSlot
}

// PropSlot is one set/val/func/stat group of a Property.
type PropSlot struct {
	Set  int
	Val  string
	Func int
	Stat string // itemstatcost Stat name; empty for func-only slots
}

var propertyCols = func() []string {
	c := []string{"code"}
	for k := 1; k <= 7; k++ {
		s := itoa(k)
		c = append(c, "set"+s, "val"+s, "func"+s, "stat"+s)
	}
	return c
}()

func loadProperties(t *Table) []*Property {
	var out []*Property
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("code") == "" || r.Str("code") == "Expansion" {
			continue
		}
		p := &Property{Code: r.Str("code")}
		for k := 1; k <= 7; k++ {
			s := itoa(k)
			f, st := r.Int("func"+s), r.Str("stat"+s)
			if f == 0 && st == "" {
				continue
			}
			p.Slots = append(p.Slots, PropSlot{r.Int("set" + s), r.Str("val" + s), f, st})
		}
		out = append(out, p)
	}
	return out
}

// Affix is a magicprefix.txt or magicsuffix.txt row.
type Affix struct {
	Name          string // key into the string tables
	Display       string // resolved display text
	Suffix        bool
	Version       int
	Spawnable     bool
	Rare          bool
	Level         int
	MaxLevel      int
	LevelReq      int
	ClassSpecific string
	Class         string
	ClassLevelReq int
	Frequency     int
	Group         int
	Mods          []Prop   // mod1..3 (Param, Min, Max)
	Include       []string // itype1..7
	Exclude       []string // etype1..5
}

var affixCols = append([]string{
	"Name", "version", "spawnable", "rare", "level", "maxlevel", "levelreq", "classspecific", "class", "classlevelreq", "frequency", "group",
	"itype1", "itype2", "itype3", "itype4", "itype5", "itype6", "itype7", "etype1", "etype2", "etype3", "etype4", "etype5",
}, propCols("mod#code", "mod#param", "mod#min", "mod#max", 3)...)

func loadAffixes(t *Table, suffix bool) []*Affix {
	var out []*Affix
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("Name") == "" || r.Str("Name") == "Expansion" {
			continue
		}
		a := &Affix{
			Name: r.Str("Name"), Suffix: suffix, Version: r.Int("version"), Spawnable: r.Bool("spawnable"), Rare: r.Bool("rare"),
			Level: r.Int("level"), MaxLevel: r.Int("maxlevel"), LevelReq: r.Int("levelreq"), ClassSpecific: r.Str("classspecific"),
			Class: r.Str("class"), ClassLevelReq: r.Int("classlevelreq"), Frequency: r.Int("frequency"), Group: r.Int("group"),
			Mods: readProps(r, "mod#code", "mod#param", "mod#min", "mod#max", 3),
		}
		for k := 1; k <= 7; k++ {
			if s := r.Str("itype" + itoa(k)); s != "" {
				a.Include = append(a.Include, s)
			}
		}
		for k := 1; k <= 5; k++ {
			if s := r.Str("etype" + itoa(k)); s != "" {
				a.Exclude = append(a.Exclude, s)
			}
		}
		out = append(out, a)
	}
	return out
}

// State is a states.txt row: a status effect a unit can be in. Only the
// columns useful for recognising effects are kept.
type State struct {
	Name                       string
	ID                         int
	Group                      int
	Aura                       bool
	Curse                      bool
	Curable                    bool
	Active                     bool
	Restrict                   bool
	Hide                       bool
	Shatter                    bool
	Life                       bool // life-bar colour state
	Stat                       string
	Missile                    string
	Skill                      string
	ItemType                   string
	Overlay                    []string // overlay1..4
	ColorPri                   int
	NoSend                     bool
	RemHit                     bool
	PlrStayDeath, MonStayDeath bool
}

var stateCols = []string{
	"state", "id", "group", "aura", "curse", "curable", "active", "restrict", "hide", "shatter", "life", "stat", "missile", "skill", "itemtype",
	"overlay1", "overlay2", "overlay3", "overlay4", "colorpri", "nosend", "remhit", "plrstaydeath", "monstaydeath",
}

func loadStates(t *Table) []*State {
	var out []*State
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("state") == "" || r.Str("state") == "Expansion" {
			continue
		}
		s := &State{
			Name: r.Str("state"), ID: r.Int("id"), Group: r.Int("group"), Aura: r.Bool("aura"), Curse: r.Bool("curse"), Curable: r.Bool("curable"),
			Active: r.Bool("active"), Restrict: r.Bool("restrict"), Hide: r.Bool("hide"), Shatter: r.Bool("shatter"), Life: r.Bool("life"),
			Stat: r.Str("stat"), Missile: r.Str("missile"), Skill: r.Str("skill"), ItemType: r.Str("itemtype"), ColorPri: r.Int("colorpri"),
			NoSend: r.Bool("nosend"), RemHit: r.Bool("remhit"), PlrStayDeath: r.Bool("plrstaydeath"), MonStayDeath: r.Bool("monstaydeath"),
		}
		for k := 1; k <= 4; k++ {
			if o := r.Str("overlay" + itoa(k)); o != "" {
				s.Overlay = append(s.Overlay, o)
			}
		}
		out = append(out, s)
	}
	return out
}

// Missile is a missiles.txt row (projectile): speed, range, collision.
type Missile struct {
	Name                                    string
	ID                                      int
	Vel                                     int
	MaxVel                                  int
	VelLev                                  int
	Accel                                   int
	Range                                   int // base range in game frames, see RangeAt
	LevRange                                int // extra range per skill level
	Size                                    int
	Skill                                   string // Skill column: owning skills.txt skill
	Pierce                                  bool
	CollideType, CollideKill, CollideFriend bool
	Explosion, Town, ToHit, Holy, KnockBack bool
	MinDamage, MaxDamage                    int
}

var missileCols = []string{
	"Missile", "Id", "Vel", "MaxVel", "VelLev", "Accel", "Range", "LevRange", "Size", "Skill", "Pierce",
	"CollideType", "CollideKill", "CollideFriend", "Explosion", "Town", "ToHit", "Holy", "KnockBack", "MinDamage", "MaxDamage",
}

func loadMissiles(t *Table) []*Missile {
	var out []*Missile
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("Missile") == "" || r.Str("Missile") == "Expansion" {
			continue
		}
		out = append(out, &Missile{
			Name: r.Str("Missile"), ID: r.Int("Id"), Vel: r.Int("Vel"), MaxVel: r.Int("MaxVel"), VelLev: r.Int("VelLev"), Accel: r.Int("Accel"),
			Range: r.Int("Range"), LevRange: r.Int("LevRange"), Size: r.Int("Size"), Skill: r.Str("Skill"), Pierce: r.Bool("Pierce"),
			CollideType: r.Bool("CollideType"), CollideKill: r.Bool("CollideKill"), CollideFriend: r.Bool("CollideFriend"),
			Explosion: r.Bool("Explosion"), Town: r.Bool("Town"), ToHit: r.Bool("ToHit"), Holy: r.Bool("Holy"), KnockBack: r.Bool("KnockBack"),
			MinDamage: r.Int("MinDamage"), MaxDamage: r.Int("MaxDamage"),
		})
	}
	return out
}

// RangeAt is the raw range column extended by LevRange per skill level
// above 1, in the file's own unit. UNVERIFIED: the game's frames-per-tile
// conversion and the real stop distance (a 0 Range means "until it hits")
// are not checked live.
func (m *Missile) RangeAt(skillLevel int) int {
	if skillLevel < 1 {
		skillLevel = 1
	}
	return m.Range + m.LevRange*(skillLevel-1)
}

// LevelWarp is a lvlwarp.txt row: the click region and arrival offset of a
// level-to-level warp tile (used by the area exit objects).
type LevelWarp struct {
	Name                 string
	ID                   int
	SelectX, SelectY     int
	SelectDX, SelectDY   int
	ExitWalkX, ExitWalkY int
	OffsetX, OffsetY     int
	LitVersion           bool
	Tiles                int
	Direction            string
}

var warpCols = []string{
	"Name", "Id", "SelectX", "SelectY", "SelectDX", "SelectDY", "ExitWalkX", "ExitWalkY", "OffsetX", "OffsetY", "LitVersion", "Tiles", "Direction",
}

func loadWarps(t *Table) []*LevelWarp {
	var out []*LevelWarp
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("Name") == "" || r.Str("Name") == "Expansion" {
			continue
		}
		out = append(out, &LevelWarp{
			Name: r.Str("Name"), ID: r.Int("Id"), SelectX: r.Int("SelectX"), SelectY: r.Int("SelectY"), SelectDX: r.Int("SelectDX"),
			SelectDY: r.Int("SelectDY"), ExitWalkX: r.Int("ExitWalkX"), ExitWalkY: r.Int("ExitWalkY"), OffsetX: r.Int("OffsetX"),
			OffsetY: r.Int("OffsetY"), LitVersion: r.Bool("LitVersion"), Tiles: r.Int("Tiles"), Direction: r.Str("Direction"),
		})
	}
	return out
}

// HirelingByID returns the first hireling.txt row with the given Id, or nil.
func (db *DB) HirelingByID(id int) *Hireling {
	for _, h := range db.Hirelings {
		if h.ID == id {
			return h
		}
	}
	return nil
}

// HirelingsFor lists the rows for a 1-based act and difficulty (0-based
// Difficulty value + 1 in the file), in file order (ascending level).
func (db *DB) HirelingsFor(act int, d Difficulty) []*Hireling {
	var out []*Hireling
	for _, h := range db.Hirelings {
		if h.Act == act && h.Difficulty == int(d)+1 {
			out = append(out, h)
		}
	}
	return out
}

// StatByID returns the itemstatcost row for a stat id.
func (db *DB) StatByID(id int) *StatCost { return db.statByID[id] }

// StatByName returns the itemstatcost row by its Stat name ("strength").
func (db *DB) StatByName(name string) *StatCost { return db.statByName[lc(name)] }

// PropertyByCode returns the properties.txt row for a property code ("res-all").
func (db *DB) PropertyByCode(code string) *Property { return db.propByCode[lc(code)] }

// AffixByName returns the first prefix or suffix with the given key or
// display name.
func (db *DB) AffixByName(name string, suffix bool) *Affix {
	l := db.Prefixes
	if suffix {
		l = db.Suffixes
	}
	for _, a := range l {
		if strings.EqualFold(a.Name, name) || strings.EqualFold(a.Display, name) {
			return a
		}
	}
	return nil
}

// StateByID returns the states.txt row with the given id.
func (db *DB) StateByID(id int) *State { return db.stateByID[id] }

// StateByName returns the state by name ("freeze").
func (db *DB) StateByName(name string) *State { return db.stateByName[lc(name)] }

// MissileByID returns the missiles.txt row with the given id.
func (db *DB) MissileByID(id int) *Missile { return db.missileByID[id] }

// MissileByName returns the missile by name ("firebolt").
func (db *DB) MissileByName(name string) *Missile { return db.missileByName[lc(name)] }

// WarpByID returns the lvlwarp.txt row with the given id.
func (db *DB) WarpByID(id int) *LevelWarp { return db.warpByID[id] }

// RunewordAvailableOffline reports whether the named runeword can be made in
// offline single player: it must be complete (enabled) and not ladder-only.
// The runes.txt column "server" is what 1.14 uses for the ladder flag (its
// 23 flagged rows are exactly the 1.10 ladder-only list, V by reading the
// file). That the game really refuses them offline is S (community
// knowledge), UNVERIFIED in game.
func (db *DB) RunewordAvailableOffline(name string) bool {
	w := db.RunewordByName(name)
	return w != nil && w.Complete && !w.Ladder
}

func (db *DB) indexExtra() {
	db.statByID, db.statByName = map[int]*StatCost{}, map[string]*StatCost{}
	for _, s := range db.StatCosts {
		db.statByID[s.ID] = s
		db.statByName[lc(s.Name)] = s
	}
	db.propByCode = map[string]*Property{}
	for _, p := range db.Properties {
		db.propByCode[lc(p.Code)] = p
	}
	for _, l := range [][]*Affix{db.Prefixes, db.Suffixes} {
		for _, a := range l {
			a.Display = db.nameOf(a.Name, a.Name)
		}
	}
	db.stateByID, db.stateByName = map[int]*State{}, map[string]*State{}
	for _, s := range db.States {
		db.stateByID[s.ID] = s
		db.stateByName[lc(s.Name)] = s
	}
	db.missileByID, db.missileByName = map[int]*Missile{}, map[string]*Missile{}
	for _, m := range db.Missiles {
		db.missileByID[m.ID] = m
		db.missileByName[lc(m.Name)] = m
	}
	db.warpByID = map[int]*LevelWarp{}
	for _, w := range db.Warps {
		db.warpByID[w.ID] = w
	}
}
