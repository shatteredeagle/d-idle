package data

import "strings"

// TreasureClass is a treasureclassex.txt row: what a monster or chest drops.
type TreasureClass struct {
	Name                     string
	Group                    int
	Level                    int
	Picks                    int // negative means "all items, each rolled independently"
	Unique, Set, Rare, Magic int // quality chance modifiers
	NoDrop                   int
	Items                    []TCItem
	TotalProb                int // sum of Prob + NoDrop, computed
}

// TCItem is one entry: an item code, item type code or another treasure
// class name, with its relative weight.
type TCItem struct {
	Name string
	Prob int
}

var tcCols = func() []string {
	c := []string{"Treasure Class", "group", "level", "Picks", "Unique", "Set", "Rare", "Magic", "NoDrop"}
	for i := 1; i <= 10; i++ {
		c = append(c, "Item"+itoa(i), "Prob"+itoa(i))
	}
	return c
}()

func loadTreasure(t *Table) []*TreasureClass {
	var out []*TreasureClass
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		name := r.Str("Treasure Class")
		if name == "" || name == "Expansion" {
			continue
		}
		tc := &TreasureClass{
			Name: name, Group: r.Int("group"), Level: r.Int("level"), Picks: r.Int("Picks"),
			Unique: r.Int("Unique"), Set: r.Int("Set"), Rare: r.Int("Rare"), Magic: r.Int("Magic"), NoDrop: r.Int("NoDrop"),
		}
		tc.TotalProb = tc.NoDrop
		for k := 1; k <= 10; k++ {
			it := r.Str("Item" + itoa(k))
			if it == "" {
				continue
			}
			p := r.Int("Prob" + itoa(k))
			tc.Items = append(tc.Items, TCItem{it, p})
			tc.TotalProb += p
		}
		out = append(out, tc)
	}
	return out
}

// Prop is an item property reference as unique/set/runeword rows list them:
// a properties.txt code with optional parameter and min/max values.
type Prop struct {
	Code     string
	Param    string
	Min, Max int
}

func readProps(r Row, codeFmt, parFmt, minFmt, maxFmt string, n int) []Prop {
	var out []Prop
	for k := 1; k <= n; k++ {
		s := itoa(k)
		code := r.Str(strings.Replace(codeFmt, "#", s, 1))
		if code == "" {
			continue
		}
		out = append(out, Prop{code, r.Str(strings.Replace(parFmt, "#", s, 1)), r.Int(strings.Replace(minFmt, "#", s, 1)), r.Int(strings.Replace(maxFmt, "#", s, 1))})
	}
	return out
}

func propCols(code, par, min, max string, n int) []string {
	var c []string
	for k := 1; k <= n; k++ {
		s := itoa(k)
		c = append(c, strings.Replace(code, "#", s, 1), strings.Replace(par, "#", s, 1), strings.Replace(min, "#", s, 1), strings.Replace(max, "#", s, 1))
	}
	return c
}

// UniqueItem is a uniqueitems.txt row.
type UniqueItem struct {
	Index    string // also the string-table key for the name
	Name     string // resolved display name
	Enabled  bool
	Ladder   bool
	Rarity   int
	Level    int
	LevelReq int
	Code     string // base item code
	Props    []Prop
}

var uniqueCols = append([]string{"index", "version", "enabled", "ladder", "rarity", "lvl", "lvl req", "code"}, propCols("prop#", "par#", "min#", "max#", 12)...)

func loadUniques(t *Table) []*UniqueItem {
	var out []*UniqueItem
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		idx := r.Str("index")
		if idx == "" || idx == "Expansion" {
			continue
		}
		out = append(out, &UniqueItem{
			Index: idx, Enabled: r.Bool("enabled"), Ladder: r.Bool("ladder"), Rarity: r.Int("rarity"),
			Level: r.Int("lvl"), LevelReq: r.Int("lvl req"), Code: r.Str("code"),
			Props: readProps(r, "prop#", "par#", "min#", "max#", 12),
		})
	}
	return out
}

// SetItem is a setitems.txt row. Only the always-on properties are loaded
// (prop1..prop9); the partial-set bonuses (aprop*) are skipped.
type SetItem struct {
	Index    string
	Name     string
	Set      string // set index (sets.txt)
	Code     string
	Level    int
	LevelReq int
	Rarity   int
	Props    []Prop
}

var setItemCols = append([]string{"index", "set", "item", "rarity", "lvl", "lvl req"}, propCols("prop#", "par#", "min#", "max#", 9)...)

func loadSetItems(t *Table) []*SetItem {
	var out []*SetItem
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		idx := r.Str("index")
		if idx == "" || idx == "Expansion" {
			continue
		}
		out = append(out, &SetItem{
			Index: idx, Set: r.Str("set"), Code: r.Str("item"), Level: r.Int("lvl"), LevelReq: r.Int("lvl req"), Rarity: r.Int("rarity"),
			Props: readProps(r, "prop#", "par#", "min#", "max#", 9),
		})
	}
	return out
}

// Runeword is a runes.txt row.
type Runeword struct {
	ID           string // Name column: internal id, e.g. "Runeword12"
	NameKey      string // Rune Name: string table key
	Name         string // resolved display name
	Complete     bool
	Server       bool
	Ladder       bool     // ladder-only: the 1.14 file flags this in the "server" column (a "ladder" column is honoured too)
	ItemTypes    []string // itype1..6: allowed base item types
	ExcludeTypes []string // etype1..3
	Runes        []string // Rune1..6: misc.txt codes, in socket order
	Props        []Prop
}

var runewordCols = append([]string{
	"Name", "Rune Name", "complete", "server", "ladder", "itype1", "itype2", "itype3", "itype4", "itype5", "itype6",
	"etype1", "etype2", "etype3", "Rune1", "Rune2", "Rune3", "Rune4", "Rune5", "Rune6",
}, propCols("T1Code#", "T1Param#", "T1Min#", "T1Max#", 7)...)

func loadRunewords(t *Table) []*Runeword {
	var out []*Runeword
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("Name") == "" || r.Str("Rune Name") == "" {
			continue
		}
		w := &Runeword{
			ID: r.Str("Name"), NameKey: r.Str("Rune Name"), Complete: r.Bool("complete"), Server: r.Bool("server"), Ladder: r.Bool("server") || r.Bool("ladder"),
			Props: readProps(r, "T1Code#", "T1Param#", "T1Min#", "T1Max#", 7),
		}
		for k := 1; k <= 6; k++ {
			if s := r.Str("itype" + itoa(k)); s != "" {
				w.ItemTypes = append(w.ItemTypes, s)
			}
			if s := r.Str("Rune" + itoa(k)); s != "" {
				w.Runes = append(w.Runes, s)
			}
		}
		for k := 1; k <= 3; k++ {
			if s := r.Str("etype" + itoa(k)); s != "" {
				w.ExcludeTypes = append(w.ExcludeTypes, s)
			}
		}
		out = append(out, w)
	}
	return out
}

// CubeRecipe is a cubemain.txt row (Horadric Cube recipe). Inputs and
// outputs use the cube's mini-language: item codes, type codes, qualifiers
// ("hiq", "low", "sock", "uni", "qty=3", ...) comma-separated. Output
// modifiers (mod1..) are not loaded.
type CubeRecipe struct {
	Description      string
	Enabled          bool
	Ladder           bool
	MinDiff          int
	Op, Param, Value int
	Class            string
	NumInputs        int
	Inputs           []string
	Output           string
	OutputB          string
	OutputC          string
}

var cubeCols = []string{
	"description", "enabled", "ladder", "min diff", "op", "param", "value", "class", "numinputs",
	"input 1", "input 2", "input 3", "input 4", "input 5", "input 6", "input 7", "output", "output b", "output c",
}

func loadCube(t *Table) []*CubeRecipe {
	var out []*CubeRecipe
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("description") == "" || r.Str("description") == "Expansion" {
			continue
		}
		c := &CubeRecipe{
			Description: r.Str("description"), Enabled: r.Bool("enabled"), Ladder: r.Bool("ladder"), MinDiff: r.Int("min diff"),
			Op: r.Int("op"), Param: r.Int("param"), Value: r.Int("value"), Class: r.Str("class"), NumInputs: r.Int("numinputs"),
			Output: r.Str("output"), OutputB: r.Str("output b"), OutputC: r.Str("output c"),
		}
		for k := 1; k <= 7; k++ {
			if s := r.Str("input " + itoa(k)); s != "" {
				c.Inputs = append(c.Inputs, s)
			}
		}
		out = append(out, c)
	}
	return out
}
