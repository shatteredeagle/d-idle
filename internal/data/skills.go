package data

import (
	"math"
	"regexp"
	"strings"
)

// Skill is one skills.txt row (joined with skilldesc.txt for name and tab
// position). Calc/damage fields are kept as raw strings or numbers: the
// formulas ("par8", "skill('Fire Ball'.blvl)") are an expression language
// the combat code can evaluate when it needs exact numbers.
type Skill struct {
	ID                             int
	Key                            string // skill column, e.g. "Fire Bolt"
	Name                           string // display name via skilldesc "str name"
	Class                          string // charclass: ama sor nec pal bar dru ass, "" for generic
	DescKey                        string // skilldesc column
	Page                           int    // skilldesc SkillPage (1-3 tab), 0 if none
	Row                            int    // skilldesc SkillRow (1-6)
	Column                         int    // skilldesc SkillColumn (1-3)
	ReqLevel                       int
	MaxLevel                       int
	ReqSkills                      [3]string
	ReqStr, ReqDex, ReqInt, ReqVit int

	Range     string // none, h2h, rng, both
	Passive   bool
	Aura      bool
	Periodic  bool
	InTown    bool
	Summon    string
	Delay     int // cooldown in frames (25 fps), 0 none
	LeftSkill bool
	Missile   string // srvmissile

	// Mana per D2: cost = max(MinMana, (Mana + LvlMana*(lvl-1)) << ManaShift / 256).
	Mana, LvlMana, ManaShift, MinMana int

	// Weapon-physical damage: percent of weapon damage etc.
	SrcDam               int
	HitShift             int
	MinDam, MaxDam       int
	MinLevDam, MaxLevDam [5]int
	DmgSymPerCalc        string
	// Elemental damage.
	EType          string // fire, cold, ltng, pois, mag, ...
	EMin, EMax     int
	EMinLev        [5]int
	EMaxLev        [5]int
	EDmgSymPerCalc string // synergy percent-per-level expression for elemental damage
	ELen           int
	ELevLen        [3]int
	Calc           [4]string // calc1..calc4 (radius, duration, ...)
	Param          [8]int    // Param1..8 values (par1..par8 in formulas)
}

var skillCols = []string{
	"skill", "Id", "charclass", "skilldesc", "reqlevel", "maxlvl", "reqstr", "reqdex", "reqint", "reqvit",
	"reqskill1", "reqskill2", "reqskill3", "range", "passive", "aura", "periodic", "InTown", "summon", "delay", "leftskill", "srvmissile",
	"mana", "lvlmana", "manashift", "minmana",
	"SrcDam", "HitShift", "MinDam", "MinLevDam1", "MinLevDam2", "MinLevDam3", "MinLevDam4", "MinLevDam5",
	"MaxDam", "MaxLevDam1", "MaxLevDam2", "MaxLevDam3", "MaxLevDam4", "MaxLevDam5", "DmgSymPerCalc",
	"EType", "EMin", "EMinLev1", "EMinLev2", "EMinLev3", "EMinLev4", "EMinLev5",
	"EMax", "EMaxLev1", "EMaxLev2", "EMaxLev3", "EMaxLev4", "EMaxLev5", "EDmgSymPerCalc", "ELen", "ELevLen1", "ELevLen2", "ELevLen3",
	"calc1", "calc2", "calc3", "calc4", "Param1", "Param2", "Param3", "Param4", "Param5", "Param6", "Param7", "Param8",
}

var skillDescCols = []string{"skilldesc", "SkillPage", "SkillRow", "SkillColumn", "str name"}

func loadSkills(t, desc *Table) []*Skill {
	type d struct {
		page, row, col int
		name           string
	}
	descs := map[string]d{}
	if desc != nil {
		for i := 0; i < desc.Len(); i++ {
			r := desc.Row(i)
			descs[strings.ToLower(r.Str("skilldesc"))] = d{r.Int("SkillPage"), r.Int("SkillRow"), r.Int("SkillColumn"), r.Str("str name")}
		}
	}
	var out []*Skill
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		key := r.Str("skill")
		if key == "" || !t.Has("Id") || r.Str("Id") == "" {
			continue
		}
		s := &Skill{
			ID: r.Int("Id"), Key: key, Class: r.Str("charclass"), DescKey: r.Str("skilldesc"),
			ReqLevel: r.Int("reqlevel"), MaxLevel: r.Int("maxlvl"),
			ReqStr: r.Int("reqstr"), ReqDex: r.Int("reqdex"), ReqInt: r.Int("reqint"), ReqVit: r.Int("reqvit"),
			ReqSkills: [3]string{r.Str("reqskill1"), r.Str("reqskill2"), r.Str("reqskill3")},
			Range:     r.Str("range"), Passive: r.Bool("passive"), Aura: r.Bool("aura"), Periodic: r.Bool("periodic"),
			InTown: r.Bool("InTown"), Summon: r.Str("summon"), Delay: r.Int("delay"), LeftSkill: r.Bool("leftskill"),
			Missile: r.Str("srvmissile"),
			Mana:    r.Int("mana"), LvlMana: r.Int("lvlmana"), ManaShift: r.Int("manashift"), MinMana: r.Int("minmana"),
			SrcDam: r.Int("SrcDam"), HitShift: r.Int("HitShift"), MinDam: r.Int("MinDam"), MaxDam: r.Int("MaxDam"),
			DmgSymPerCalc: r.Str("DmgSymPerCalc"),
			EType:         r.Str("EType"), EMin: r.Int("EMin"), EMax: r.Int("EMax"), EDmgSymPerCalc: r.Str("EDmgSymPerCalc"),
			ELen: r.Int("ELen"),
		}
		for k := 0; k < 5; k++ {
			n := itoa(k + 1)
			s.MinLevDam[k], s.MaxLevDam[k] = r.Int("MinLevDam"+n), r.Int("MaxLevDam"+n)
			s.EMinLev[k], s.EMaxLev[k] = r.Int("EMinLev"+n), r.Int("EMaxLev"+n)
		}
		for k := 0; k < 3; k++ {
			s.ELevLen[k] = r.Int("ELevLen" + itoa(k+1))
		}
		for k := 0; k < 4; k++ {
			s.Calc[k] = r.Str("calc" + itoa(k+1))
		}
		for k := 0; k < 8; k++ {
			s.Param[k] = r.Int("Param" + itoa(k+1))
		}
		if dd, ok := descs[strings.ToLower(s.DescKey)]; ok {
			s.Page, s.Row, s.Column = dd.page, dd.row, dd.col
			s.Name = dd.name // string key until the DB resolves it
		}
		out = append(out, s)
	}
	return out
}

// ManaCost returns the mana cost at skill level lvl (>=1), as the game
// computes it. The shift/256 form yields fractional mana (Fire Bolt level 1
// costs 2.5), so the result is a float.
func (s *Skill) ManaCost(lvl int) float64 {
	if lvl < 1 {
		lvl = 1
	}
	c := float64((s.Mana+s.LvlMana*(lvl-1))<<uint(s.ManaShift)) / 256
	return math.Max(c, float64(s.MinMana))
}

var synRe = regexp.MustCompile(`skill\('([^']+)'\.`)

// Synergies returns the skill keys referenced by the synergy expressions
// (DmgSymPerCalc / EDmgSymPerCalc), i.e. the skills that add damage to this one.
func (s *Skill) Synergies() []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range []string{s.DmgSymPerCalc, s.EDmgSymPerCalc} {
		for _, m := range synRe.FindAllStringSubmatch(e, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				out = append(out, m[1])
			}
		}
	}
	return out
}

// CharClass is a player class (charstats.txt row).
type CharClass struct {
	ID                          int // 0 Amazon .. 6 Assassin (row order)
	Name                        string
	Code                        string // three-letter skills.txt charclass code, set by the DB
	Str, Dex, Int, Vit, Stamina int
	HPAdd                       int
	// Per-level and per-point gains are stored in quarter points: 4 means 1 per level.
	LifePerLevel, StaminaPerLevel, ManaPerLevel int
	LifePerVit, StaminaPerVit, ManaPerEnergy    int
	StatPerLevel                                int
	ToHitFactor                                 int
	WalkVelocity, RunVelocity, RunDrain         int
	BlockFactor                                 int
	StartSkill                                  string
	Skills                                      []string // Skill 1..10: class skills with level-1 bonus/onboarding
	Items                                       []StartItem
}

// StartItem is a charstats starting-inventory entry.
type StartItem struct {
	Code     string
	Location string
	Count    int
}

var charCols = []string{
	"class", "str", "dex", "int", "vit", "stamina", "hpadd",
	"LifePerLevel", "StaminaPerLevel", "ManaPerLevel", "LifePerVitality", "StaminaPerVitality", "ManaPerMagic", "StatPerLevel",
	"ToHitFactor", "WalkVelocity", "RunVelocity", "RunDrain", "BlockFactor", "StartSkill",
	"Skill 1", "Skill 2", "Skill 3", "Skill 4", "Skill 5", "Skill 6", "Skill 7", "Skill 8", "Skill 9", "Skill 10",
	"item1", "item1loc", "item1count", "item2", "item2loc", "item2count", "item3", "item3loc", "item3count",
	"item4", "item4loc", "item4count", "item5", "item5loc", "item5count", "item6", "item6loc", "item6count",
	"item7", "item7loc", "item7count", "item8", "item8loc", "item8count", "item9", "item9loc", "item9count",
	"item10", "item10loc", "item10count",
}

func loadCharClasses(t *Table) []*CharClass {
	var out []*CharClass
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		name := r.Str("class")
		if name == "" || name == "Expansion" {
			continue
		}
		c := &CharClass{
			ID: len(out), Name: name,
			Str: r.Int("str"), Dex: r.Int("dex"), Int: r.Int("int"), Vit: r.Int("vit"), Stamina: r.Int("stamina"), HPAdd: r.Int("hpadd"),
			LifePerLevel: r.Int("LifePerLevel"), StaminaPerLevel: r.Int("StaminaPerLevel"), ManaPerLevel: r.Int("ManaPerLevel"),
			LifePerVit: r.Int("LifePerVitality"), StaminaPerVit: r.Int("StaminaPerVitality"), ManaPerEnergy: r.Int("ManaPerMagic"),
			StatPerLevel: r.Int("StatPerLevel"), ToHitFactor: r.Int("ToHitFactor"),
			WalkVelocity: r.Int("WalkVelocity"), RunVelocity: r.Int("RunVelocity"), RunDrain: r.Int("RunDrain"),
			BlockFactor: r.Int("BlockFactor"), StartSkill: r.Str("StartSkill"),
		}
		// skills.txt charclass codes are the first three letters of the class name.
		if len(name) >= 3 {
			c.Code = strings.ToLower(name[:3])
		}
		for k := 1; k <= 10; k++ {
			if s := r.Str("Skill " + itoa(k)); s != "" {
				c.Skills = append(c.Skills, s)
			}
			n := "item" + itoa(k)
			if code := r.Str(n); code != "" && code != "0" {
				c.Items = append(c.Items, StartItem{code, r.Str(n + "loc"), r.Int(n + "count")})
			}
		}
		out = append(out, c)
	}
	return out
}

// Experience is the experience.txt table (the Amazon column; vanilla has
// identical columns for every class). Row[n] is the total experience needed
// to reach level n+1, so Row[0] = 0 and Row[1] = 500 (level 2).
type Experience struct {
	Row      [100]int64
	Ratio    [100]int // ExpRatio per row (1024 = full experience)
	MaxLevel int
}

var expCols = []string{"Level", "Amazon", "ExpRatio"}

func loadExperience(t *Table) *Experience {
	e := &Experience{MaxLevel: 99}
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		lv := r.Str("Level")
		if lv == "MaxLvl" {
			if m := r.Int("Amazon"); m > 0 {
				e.MaxLevel = m
			}
			continue
		}
		n := r.Int("Level")
		if n < 0 || n >= 100 {
			continue
		}
		e.Row[n] = int64(r.Int("Amazon"))
		e.Ratio[n] = r.Int("ExpRatio")
	}
	return e
}

// ToReach returns the total experience needed to reach level (1..99).
func (e *Experience) ToReach(level int) int64 {
	if level <= 1 {
		return 0
	}
	if level > e.MaxLevel {
		level = e.MaxLevel
	}
	return e.Row[level-1]
}

// LevelFor returns the character level for a total experience value.
func (e *Experience) LevelFor(exp int64) int {
	lv := 1
	for l := 2; l <= e.MaxLevel; l++ {
		if exp >= e.Row[l-1] {
			lv = l
		}
	}
	return lv
}
