package data

// Difficulty indexes the per-difficulty arrays (Normal, Nightmare, Hell).
type Difficulty int

const (
	Normal Difficulty = iota
	Nightmare
	Hell
)

// DamageType indexes Monster.Res in monstats.txt column order.
type DamageType int

const (
	Physical  DamageType = iota // ResDm
	Magic                       // ResMa
	Fire                        // ResFi
	Lightning                   // ResLi
	Cold                        // ResCo
	Poison                      // ResPo
	numDamageTypes
)

func (d DamageType) String() string {
	return [...]string{"physical", "magic", "fire", "lightning", "cold", "poison"}[d]
}

// Monster is one monstats.txt row. The class id (the number the game stores
// in a monster unit's txtFileNo) is the hcIdx column.
type Monster struct {
	ID        int    // hcIdx: class id
	Key       string // Id column, e.g. "skeleton1"
	BaseKey   string // BaseId column
	NameKey   string // NameStr: key into the string tables
	Name      string // resolved display name (empty until a DB resolves it)
	Type      string // MonType, e.g. "skeleton"
	AI        string
	Enabled   bool
	NPC       bool
	Interact  bool
	Killable  bool
	Melee     bool // isMelee
	Ranged    bool // rangedtype
	Boss      bool
	PrimeEvil bool
	LUndead   bool
	HUndead   bool
	Demon     bool
	Flying    bool
	// Minion1/2 are monster keys spawned alongside; SetBoss makes this
	// monster the leader of its group.
	Minion1, Minion2   string
	SetBoss            bool
	MinGrp, MaxGrp     int
	PartyMin, PartyMax int
	Velocity, Run      int

	Level [3]int // Level, Level(N), Level(H): the monster level in areas that do not override it
	// Res is the resistance per difficulty and DamageType; 100 or more is
	// immune. Blank cells read as 0.
	Res [3][numDamageTypes]int
	// ToBlock per difficulty.
	ToBlock [3]int
	HPMin   [3]int
	HPMax   [3]int
	AC      [3]int
	Exp     [3]int
	// A1/A2/S1 damage ranges per difficulty (physical melee/ranged attack
	// 1, attack 2, special 1) plus their to-hit.
	A1Min, A1Max, A1TH [3]int
	A2Min, A2Max, A2TH [3]int
	S1Min, S1Max, S1TH [3]int
	// Elemental on-hit damage (up to 3 slots).
	Elem [3]MonElem
	// TreasureClass[diff][0..3]: normal, champion, unique, quest.
	TreasureClass [3][4]string
}

// MonElem is the elemental attack in monstats El1*/El2*/El3* (percent
// chance, damage range, duration, in the Normal-difficulty column set).
type MonElem struct {
	Mode, Type    string
	Pct, Min, Max int
}

// Immune reports whether the monster is immune to t at difficulty d.
func (m *Monster) Immune(t DamageType, d Difficulty) bool { return m.Res[d][t] >= 100 }

var monsterCols = []string{
	"Id", "hcIdx", "BaseId", "NameStr", "MonType", "AI", "enabled", "rangedtype", "minion1", "minion2",
	"SetBoss", "PartyMin", "PartyMax", "MinGrp", "MaxGrp", "Velocity", "Run", "Level", "Level(N)", "Level(H)",
	"isMelee", "npc", "interact", "lUndead", "hUndead", "demon", "flying", "boss", "primeevil", "killable",
	"ResDm", "ResMa", "ResFi", "ResLi", "ResCo", "ResPo",
	"ResDm(N)", "ResMa(N)", "ResFi(N)", "ResLi(N)", "ResCo(N)", "ResPo(N)",
	"ResDm(H)", "ResMa(H)", "ResFi(H)", "ResLi(H)", "ResCo(H)", "ResPo(H)",
	"ToBlock", "ToBlock(N)", "ToBlock(H)",
	"minHP", "maxHP", "AC", "Exp", "A1MinD", "A1MaxD", "A1TH", "A2MinD", "A2MaxD", "A2TH", "S1MinD", "S1MaxD", "S1TH",
	"MinHP(N)", "MaxHP(N)", "AC(N)", "Exp(N)", "A1MinD(N)", "A1MaxD(N)", "A1TH(N)", "A2MinD(N)", "A2MaxD(N)", "A2TH(N)", "S1MinD(N)", "S1MaxD(N)", "S1TH(N)",
	"MinHP(H)", "MaxHP(H)", "AC(H)", "Exp(H)", "A1MinD(H)", "A1MaxD(H)", "A1TH(H)", "A2MinD(H)", "A2MaxD(H)", "A2TH(H)", "S1MinD(H)", "S1MaxD(H)", "S1TH(H)",
	"El1Mode", "El1Type", "El1Pct", "El1MinD", "El1MaxD", "El2Mode", "El2Type", "El2Pct", "El2MinD", "El2MaxD", "El3Mode", "El3Type", "El3Pct", "El3MinD", "El3MaxD",
	"TreasureClass1", "TreasureClass2", "TreasureClass3", "TreasureClass4",
	"TreasureClass1(N)", "TreasureClass2(N)", "TreasureClass3(N)", "TreasureClass4(N)",
	"TreasureClass1(H)", "TreasureClass2(H)", "TreasureClass3(H)", "TreasureClass4(H)",
}

var diffSuffix = [3]string{"", "(N)", "(H)"}

func loadMonsters(t *Table) []*Monster {
	var out []*Monster
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		key := r.Str("Id")
		hc := r.Str("hcIdx")
		// The "Expansion" separator row has no hcIdx and is not a class.
		if key == "" || hc == "" {
			continue
		}
		m := &Monster{
			ID: r.Int("hcIdx"), Key: key, BaseKey: r.Str("BaseId"), NameKey: r.Str("NameStr"),
			Type: r.Str("MonType"), AI: r.Str("AI"), Enabled: r.Bool("enabled"), Ranged: r.Bool("rangedtype"),
			Minion1: r.Str("minion1"), Minion2: r.Str("minion2"), SetBoss: r.Bool("SetBoss"),
			PartyMin: r.Int("PartyMin"), PartyMax: r.Int("PartyMax"), MinGrp: r.Int("MinGrp"), MaxGrp: r.Int("MaxGrp"),
			Velocity: r.Int("Velocity"), Run: r.Int("Run"),
			Melee: r.Bool("isMelee"), NPC: r.Bool("npc"), Interact: r.Bool("interact"), Killable: r.Bool("killable"),
			Boss: r.Bool("boss"), PrimeEvil: r.Bool("primeevil"), LUndead: r.Bool("lUndead"), HUndead: r.Bool("hUndead"),
			Demon: r.Bool("demon"), Flying: r.Bool("flying"),
		}
		res := [...]string{"ResDm", "ResMa", "ResFi", "ResLi", "ResCo", "ResPo"}
		for d := 0; d < 3; d++ {
			s := diffSuffix[d]
			m.Level[d] = r.Int("Level" + s)
			for k, n := range res {
				m.Res[d][k] = r.Int(n + s)
			}
			m.ToBlock[d] = r.Int("ToBlock" + s)
			m.HPMin[d], m.HPMax[d] = r.Int("MinHP"+s), r.Int("MaxHP"+s)
			m.AC[d], m.Exp[d] = r.Int("AC"+s), r.Int("Exp"+s)
			m.A1Min[d], m.A1Max[d], m.A1TH[d] = r.Int("A1MinD"+s), r.Int("A1MaxD"+s), r.Int("A1TH"+s)
			m.A2Min[d], m.A2Max[d], m.A2TH[d] = r.Int("A2MinD"+s), r.Int("A2MaxD"+s), r.Int("A2TH"+s)
			m.S1Min[d], m.S1Max[d], m.S1TH[d] = r.Int("S1MinD"+s), r.Int("S1MaxD"+s), r.Int("S1TH"+s)
			for k := 0; k < 4; k++ {
				m.TreasureClass[d][k] = r.Str("TreasureClass" + string(rune('1'+k)) + s)
			}
		}
		for k := 0; k < 3; k++ {
			p := "El" + string(rune('1'+k))
			m.Elem[k] = MonElem{r.Str(p + "Mode"), r.Str(p + "Type"), r.Int(p + "Pct"), r.Int(p + "MinD"), r.Int(p + "MaxD")}
		}
		out = append(out, m)
	}
	return out
}

// SuperUnique is a superuniques.txt row (named boss/unique monsters).
type SuperUnique struct {
	ID      int    // hcIdx
	Key     string // Superunique column
	NameKey string // Name: string table key
	Name    string
	Class   string // monster Key this unique is based on
	Mods    [3]string
	MinGrp  int
	MaxGrp  int
	AutoPos bool
	Stacks  bool
	TC      [3]string // treasure class per difficulty
}

var superUniqueCols = []string{"Superunique", "Name", "Class", "hcIdx", "Mod1", "Mod2", "Mod3", "MinGrp", "MaxGrp", "AutoPos", "Stacks", "TC", "TC(N)", "TC(H)"}

func loadSuperUniques(t *Table) []*SuperUnique {
	var out []*SuperUnique
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("Superunique") == "" {
			continue
		}
		out = append(out, &SuperUnique{
			ID: r.Int("hcIdx"), Key: r.Str("Superunique"), NameKey: r.Str("Name"), Class: r.Str("Class"),
			Mods:   [3]string{r.Str("Mod1"), r.Str("Mod2"), r.Str("Mod3")},
			MinGrp: r.Int("MinGrp"), MaxGrp: r.Int("MaxGrp"), AutoPos: r.Bool("AutoPos"), Stacks: r.Bool("Stacks"),
			TC: [3]string{r.Str("TC"), r.Str("TC(N)"), r.Str("TC(H)")},
		})
	}
	return out
}
