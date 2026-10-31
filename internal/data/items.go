package data

// ItemType is an itemtypes.txt row. Types form a DAG through Equiv1/Equiv2
// (e.g. "swor" -> "mele" -> "weap"); DB.IsA walks it.
type ItemType struct {
	Name      string
	Code      string
	Equiv     [2]string
	Body      bool
	BodyLocs  [2]string
	Shoots    string // ammo type a launcher uses
	Quiver    string // launcher type an ammo belongs to
	Throwable bool
	AutoStack bool
	Charm     bool
	Gem       bool
	Beltable  bool
	Class     string // class restriction code, "" for none
	MaxSock   [3]int // max sockets for ilvl <25, 25-39, >=40
	StorePage string
}

var itemTypeCols = []string{
	"ItemType", "Code", "Equiv1", "Equiv2", "Body", "BodyLoc1", "BodyLoc2", "Shoots", "Quiver", "Throwable",
	"AutoStack", "Charm", "Gem", "Beltable", "MaxSock1", "MaxSock25", "MaxSock40", "Class", "StorePage",
}

func loadItemTypes(t *Table) []*ItemType {
	var out []*ItemType
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		code := r.Str("Code")
		if code == "" {
			continue
		}
		out = append(out, &ItemType{
			Name: r.Str("ItemType"), Code: code, Equiv: [2]string{r.Str("Equiv1"), r.Str("Equiv2")},
			Body: r.Bool("Body"), BodyLocs: [2]string{r.Str("BodyLoc1"), r.Str("BodyLoc2")},
			Shoots: r.Str("Shoots"), Quiver: r.Str("Quiver"), Throwable: r.Bool("Throwable"),
			AutoStack: r.Bool("AutoStack"), Charm: r.Bool("Charm"), Gem: r.Bool("Gem"), Beltable: r.Bool("Beltable"),
			Class: r.Str("Class"), MaxSock: [3]int{r.Int("MaxSock1"), r.Int("MaxSock25"), r.Int("MaxSock40")},
			StorePage: r.Str("StorePage"),
		})
	}
	return out
}

// ItemKind says which of weapons/armor/misc.txt a base item came from.
type ItemKind int

const (
	KindWeapon ItemKind = iota
	KindArmor
	KindMisc
)

// ItemBase is one row of weapons.txt, armor.txt or misc.txt. The 4-character
// Code is the key the game stores in an item's dwFileIndex lookups and in
// the d2s item code.
type ItemBase struct {
	Kind                          ItemKind
	Index                         int    // row index within its own file (the game's txtFileNo for that kind)
	Name                          string // name column (English-ish internal name)
	Code                          string
	NameKey                       string // namestr: string table key
	Display                       string // resolved display name
	Type                          string // itemtypes code
	Type2                         string
	Width                         int // invwidth: inventory grid cells
	Height                        int // invheight
	Stackable                     bool
	MinStack, MaxStack            int
	Level, LevelReq               int
	ReqStr, ReqDex                int
	Cost                          int
	Durability                    int
	NoDurability                  bool
	Sockets                       int // gemsockets: max sockets this base can take
	Speed                         int
	Rarity                        int
	Spawnable                     bool
	Quest                         bool
	Useable                       bool
	Throwable                     bool
	AutoBelt                      bool // autobelt: potions that go to the belt when picked up
	BeltIndex                     int  // "belt" column: for belt armor, the belts.txt row index
	NormCode, UberCode, UltraCode string
	// Weapon damage (weapons.txt).
	MinDam, MaxDam       int
	TwoMinDam, TwoMaxDam int // 2-handed damage
	TwoHanded            bool
	WClass               string
	// Armor (armor.txt).
	MinAC, MaxAC int
	Block        int
	// Misc (misc.txt) spell: potion/scroll effect type and state.
	Spell string
	State string
	Len   int
	Stat1 string
	Calc1 string
}

var itemCols = []string{
	"name", "code", "namestr", "type", "type2", "invwidth", "invheight", "stackable", "minstack", "maxstack",
	"level", "levelreq", "reqstr", "reqdex", "cost", "durability", "nodurability", "gemsockets", "speed", "rarity", "spawnable",
	"quest", "useable", "throwable", "belt", "autobelt", "normcode", "ubercode", "ultracode",
	"mindam", "maxdam", "2handmindam", "2handmaxdam", "2handed", "wclass", "minac", "maxac", "block",
	"pSpell", "state", "len", "stat1", "calc1",
}

func loadItemBases(t *Table, kind ItemKind) []*ItemBase {
	var out []*ItemBase
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		code := r.Str("code")
		if code == "" {
			continue
		}
		b := &ItemBase{
			Kind: kind, Index: i, Name: r.Str("name"), Code: code, NameKey: r.Str("namestr"),
			Type: r.Str("type"), Type2: r.Str("type2"), Width: r.Int("invwidth"), Height: r.Int("invheight"),
			Stackable: r.Bool("stackable"), MinStack: r.Int("minstack"), MaxStack: r.Int("maxstack"),
			Level: r.Int("level"), LevelReq: r.Int("levelreq"), ReqStr: r.Int("reqstr"), ReqDex: r.Int("reqdex"),
			Cost: r.Int("cost"), Durability: r.Int("durability"), NoDurability: r.Bool("nodurability"),
			Sockets: r.Int("gemsockets"), Speed: r.Int("speed"), Rarity: r.Int("rarity"), Spawnable: r.Bool("spawnable"),
			Quest: r.Bool("quest"), Useable: r.Bool("useable"), Throwable: r.Bool("throwable"),
			AutoBelt: r.Bool("autobelt"), BeltIndex: r.Int("belt"),
			NormCode: r.Str("normcode"), UberCode: r.Str("ubercode"), UltraCode: r.Str("ultracode"),
			MinDam: r.Int("mindam"), MaxDam: r.Int("maxdam"), TwoMinDam: r.Int("2handmindam"), TwoMaxDam: r.Int("2handmaxdam"),
			TwoHanded: r.Bool("2handed"), WClass: r.Str("wclass"),
			MinAC: r.Int("minac"), MaxAC: r.Int("maxac"), Block: r.Int("block"),
			Spell: r.Str("pSpell"), State: r.Str("state"), Len: r.Int("len"), Stat1: r.Str("stat1"), Calc1: r.Str("calc1"),
		}
		out = append(out, b)
	}
	return out
}

// Gem/rune lookup: runes are misc.txt items of type "rune" with codes r01..r33.

// Object is one objects.txt row (chests, shrines, waypoints, doors, portals).
type Object struct {
	ID           int
	Name         string
	Token        string
	SizeX, SizeY int
	// Selectable0 is true when the object can be clicked in its first mode.
	Selectable     bool
	IsDoor         bool
	BlocksVis      bool
	SubClass       int // bit flags: 1 shrine, 2 obelisk, 4 portal, 8 container, 16 well, 32 waypoint? raw; see docs
	Lockable       bool
	Act            int
	OperateRange   int
	ShrineFunction int
	Restore        bool
	AutoMap        int
	// Left/Top/Width/Height: clickable box in pixels (raw).
	Left, Top, Width, Height int
	OpenWarp                 bool
	Parm                     [8]int
}

var objectCols = []string{
	"Name", "Id", "Token", "SizeX", "SizeY", "Selectable0", "IsDoor", "BlocksVis", "SubClass", "Lockable", "Act",
	"OperateRange", "ShrineFunction", "Restore", "AutoMap", "Left", "Top", "Width", "Height", "OpenWarp",
	"Parm0", "Parm1", "Parm2", "Parm3", "Parm4", "Parm5", "Parm6", "Parm7",
}

func loadObjects(t *Table) []*Object {
	var out []*Object
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("Name") == "" || r.Str("Id") == "" {
			continue
		}
		o := &Object{
			ID: r.Int("Id"), Name: r.Str("Name"), Token: r.Str("Token"), SizeX: r.Int("SizeX"), SizeY: r.Int("SizeY"),
			Selectable: r.Bool("Selectable0"), IsDoor: r.Bool("IsDoor"), BlocksVis: r.Bool("BlocksVis"),
			SubClass: r.Int("SubClass"), Lockable: r.Bool("Lockable"), Act: r.Int("Act"),
			OperateRange: r.Int("OperateRange"), ShrineFunction: r.Int("ShrineFunction"), Restore: r.Bool("Restore"),
			AutoMap: r.Int("AutoMap"), Left: r.Int("Left"), Top: r.Int("Top"), Width: r.Int("Width"), Height: r.Int("Height"),
			OpenWarp: r.Bool("OpenWarp"),
		}
		for k := range o.Parm {
			o.Parm[k] = r.Int("Parm" + itoa(k))
		}
		out = append(out, o)
	}
	return out
}
