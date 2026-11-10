package status

// Item qualities understood by the browser (colors in web/live/lib.js).
const (
	QLow, QNormal, QSuperior, QMagic, QRare, QUnique, QSet, QCrafted, QRuneword = "low", "normal", "superior", "magic", "rare", "unique", "set", "crafted", "runeword"
)

// Equipment slot keys.
var EquipSlots = []string{"head", "neck", "torso", "rhand", "lhand", "rring", "lring", "belt", "feet", "gloves"}

type Item struct {
	ID         uint32    `json:"id"`
	Code       string    `json:"code"` // base item code, e.g. "lsd"
	Name       string    `json:"name"` // may be empty: a Namer fills it
	Quality    string    `json:"quality"`
	Ethereal   bool      `json:"ethereal,omitempty"`
	Sockets    int       `json:"sockets,omitempty"`
	Filled     int       `json:"filled,omitempty"`
	Level      int       `json:"level,omitempty"`
	Identified bool      `json:"identified"`
	Dur        int       `json:"dur,omitempty"`    // current durability (worn items; 0 with MaxDur > 0 means broken)
	MaxDur     int       `json:"maxDur,omitempty"` // maximum durability, 0 = the item has none or it was not read
	Props      []string  `json:"props,omitempty"`  // tooltip lines
	Art        string    `json:"art,omitempty"`    // inventory icon name (lower case, no extension) for /watch/art/item-<art>.png; empty = unknown, draw the cell
	Tip        []string  `json:"tip,omitempty"`    // the game's own tooltip lines as d2d prints them, name first (S: d2d "items" verb; U: colour codes, line order)
	Base       *ItemBase `json:"base,omitempty"`   // base-type facts from the game tables (data.ItemBase), the fallback when Tip is empty
	Sell       int       `json:"sell,omitempty"`   // vendor sell price in gold (loot.SellValue: a lower bound for magic, rare, set and unique items)
	X          int       `json:"x"`                // grid cell (belt: column, row)
	Y          int       `json:"y"`
	W          int       `json:"w"`
	H          int       `json:"h"`
}

// ItemBase is what the game tables say about an item's base type (docs/inventory-tooltip.md). Zero fields are absent.
type ItemBase struct {
	Name         string `json:"name,omitempty"`
	MinAC        int    `json:"minAC,omitempty"`
	MaxAC        int    `json:"maxAC,omitempty"`
	Block        int    `json:"block,omitempty"`
	MinDam       int    `json:"minDam,omitempty"`
	MaxDam       int    `json:"maxDam,omitempty"`
	TwoMinDam    int    `json:"twoMinDam,omitempty"`
	TwoMaxDam    int    `json:"twoMaxDam,omitempty"`
	ReqStr       int    `json:"reqStr,omitempty"`
	ReqDex       int    `json:"reqDex,omitempty"`
	ReqLvl       int    `json:"reqLvl,omitempty"`
	Speed        int    `json:"speed,omitempty"`
	MaxSockets   int    `json:"maxSockets,omitempty"`
	Dur          int    `json:"dur,omitempty"`
	TwoHandedDam bool   `json:"twoHanded,omitempty"`
}

type InventoryData struct {
	Equipment map[string]*Item `json:"equipment"`
	Inventory []Item           `json:"inventory"` // 10 x 4
	Stash     []Item           `json:"stash"`     // 6 x 8
	Cube      []Item           `json:"cube,omitempty"`
	Belt      []Item           `json:"belt"` // X column 0..3, Y row 0..3 (0 = bottom row)
	Gold      int              `json:"gold"`
	StashGold int              `json:"stashGold"`
	BeltRows  int              `json:"beltRows"`
}

type InventorySource interface{ Inventory() InventoryData }

// Namer produces display names for items. The DATA agent will provide the real one.
type Namer interface{ ItemName(Item) string }

// CodeNamer is the trivial fallback: it shows the base code.
type CodeNamer struct{}

func (CodeNamer) ItemName(it Item) string { return it.Code }

func NewInventoryPanel(src InventorySource, namer Namer) Panel {
	if namer == nil {
		namer = CodeNamer{}
	}
	fill := func(its []Item) {
		for i := range its {
			if its[i].Name == "" {
				its[i].Name = namer.ItemName(its[i])
			}
		}
	}
	return Panel{ID: "inventory", Title: "Inventory", Order: 30, Every: 2e9, Snapshot: func() any {
		d := src.Inventory()
		fill(d.Inventory)
		fill(d.Stash)
		fill(d.Cube)
		fill(d.Belt)
		for _, it := range d.Equipment {
			if it != nil && it.Name == "" {
				it.Name = namer.ItemName(*it)
			}
		}
		return d
	}}
}
